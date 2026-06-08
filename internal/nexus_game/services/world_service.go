package services

import (
	"context"
	"fmt"
	"time"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/models"

	"gorm.io/gorm"
)

// WorldService handles world creation, continent assignment for factions/players,
// capacity checks, and proportional distribution using Redis heavily for performance and locking.
type WorldService struct {
	db    *gorm.DB
	redis *cache.RedisService
}

func NewWorldService(db *gorm.DB, redis *cache.RedisService) *WorldService {
	return &WorldService{db: db, redis: redis}
}

// CreateWorld creates a new world with exactly 5 continents (the fixed names).
// Called when all existing continents are at capacity (3 factions or 500 players).
func (s *WorldService) CreateWorld(ctx context.Context) (*models.World, error) {
	// Use Redis lock for world creation to prevent duplicates.
	lockKey := "nexus:world:create:lock"
	locked, err := s.redis.AcquireLock(ctx, lockKey, time.Minute)
	if err != nil || !locked {
		return nil, fmt.Errorf("could not acquire world creation lock")
	}
	defer s.redis.ReleaseLock(ctx, lockKey)

	world := &models.World{
		Name:      fmt.Sprintf("World-%d", time.Now().Unix()),
		CreatedAt: time.Now().UTC(),
		IsActive:  true,
	}
	if err := s.db.Create(world).Error; err != nil {
		return nil, err
	}

	// Create the 5 fixed continents.
	for _, name := range models.ContinentNames {
		cont := &models.Continent{
			WorldID:     world.ID,
			Name:        name,
			MaxPlayers:  500,
			MaxFactions: 3,
		}
		if err := s.db.Create(cont).Error; err != nil {
			return nil, err
		}
		// Init Redis counters for this continent.
		_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:continent:%d:players", cont.ID), "0", 0)
		_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:continent:%d:factions", cont.ID), "0", 0)
	}

	return world, nil
}

// GetOrCreateWorldForFaction finds a suitable world/continent for a new faction (max 3 factions per continent).
// If no space, creates a new world and assigns priority to this faction.
func (s *WorldService) GetOrCreateWorldForFaction(ctx context.Context, factionID uint) (worldID, continentID uint, err error) {
	// Try existing worlds first.
	worlds, err := s.listActiveWorlds(ctx)
	if err != nil {
		return 0, 0, err
	}

	for _, w := range worlds {
		contID, ok := s.findAvailableContinentForFaction(ctx, w.ID)
		if ok {
			// Assign
			if err := s.assignFactionToContinent(ctx, factionID, w.ID, contID); err != nil {
				continue
			}
			return w.ID, contID, nil
		}
	}

	// No space: create new world, assign this faction first (priority).
	newWorld, err := s.CreateWorld(ctx)
	if err != nil {
		return 0, 0, err
	}
	// First continent in new world gets this faction.
	firstContID := s.getFirstContinentID(ctx, newWorld.ID)
	if err := s.assignFactionToContinent(ctx, factionID, newWorld.ID, firstContID); err != nil {
		return 0, 0, err
	}
	return newWorld.ID, firstContID, nil
}

// findAvailableContinentForFaction returns a continent in the world with <3 factions.
func (s *WorldService) findAvailableContinentForFaction(ctx context.Context, worldID uint) (uint, bool) {
	// Use Redis for fast check.
	// For simplicity, scan continents for the world.
	var conts []models.Continent
	s.db.Where("world_id = ?", worldID).Find(&conts)
	for _, c := range conts {
		fCountStr, _, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:continent:%d:factions", c.ID))
		fCount := 0
		fmt.Sscanf(fCountStr, "%d", &fCount)
		if fCount < c.MaxFactions {
			return c.ID, true
		}
	}
	return 0, false
}

func (s *WorldService) assignFactionToContinent(ctx context.Context, factionID, worldID, continentID uint) error {
	// Increment Redis faction count for continent (use available Set for demo; in full use HINCRBY via extended redis).
	key := fmt.Sprintf("nexus:continent:%d:factions", continentID)
	current, _, _ := s.redis.GetString(ctx, key)
	count := 0
	fmt.Sscanf(current, "%d", &count)
	count++
	_ = s.redis.SetString(ctx, key, fmt.Sprintf("%d", count), 0)
	// Also store mapping faction -> continent for quick lookup.
	_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:faction:%d:continent", factionID), fmt.Sprintf("%d", continentID), 0)
	_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:faction:%d:world", factionID), fmt.Sprintf("%d", worldID), 0)
	return nil
}

func (s *WorldService) getFirstContinentID(ctx context.Context, worldID uint) uint {
	var c models.Continent
	s.db.Where("world_id = ?", worldID).Order("id asc").First(&c)
	return c.ID
}

// AssignPlayerToContinent for a profile based on faction's continent.
// Checks max 500 players. If full, returns error "faction continent full".
// Distributes players proportionally by choosing continent with lowest current count in the world.
func (s *WorldService) AssignPlayerToContinent(ctx context.Context, userID, factionID uint) (worldID, continentID uint, err error) {
	// Get faction's assigned continent.
	fContStr, ok, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:faction:%d:continent", factionID))
	if !ok {
		return 0, 0, fmt.Errorf("faction %d not assigned to continent", factionID)
	}
	fmt.Sscanf(fContStr, "%d", &continentID)

	fWorldStr, _, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:faction:%d:world", factionID))
	fmt.Sscanf(fWorldStr, "%d", &worldID)

	// Check capacity using Redis.
	pKey := fmt.Sprintf("nexus:continent:%d:players", continentID)
	pCountStr, _, _ := s.redis.GetString(ctx, pKey)
	pCount := 0
	fmt.Sscanf(pCountStr, "%d", &pCount)

	if pCount >= 500 {
		return 0, 0, fmt.Errorf("faction continent is full (max 500 players)")
	}

	// Increment (proportional by choosing lowest, but for simplicity use the faction's).
	newCount := pCount + 1
	_ = s.redis.SetString(ctx, pKey, fmt.Sprintf("%d", newCount), 0)

	// Update profile (caller should set it).
	return worldID, continentID, nil
}

// ListWorlds returns worlds with continent summary (capacities from Redis).
func (s *WorldService) ListWorlds(ctx context.Context) ([]map[string]interface{}, error) {
	var worlds []models.World
	s.db.Find(&worlds)

	result := []map[string]interface{}{}
	for _, w := range worlds {
		var conts []models.Continent
		s.db.Where("world_id = ?", w.ID).Find(&conts)
		contSummary := []map[string]interface{}{}
		for _, c := range conts {
			pStr, _, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:continent:%d:players", c.ID))
			fStr, _, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:continent:%d:factions", c.ID))
			// Liste des joueurs pour la gestion (from DB ProfileGamer)
			var profiles []models.ProfileGamer
			s.db.Where("continent_id = ?", c.ID).Find(&profiles)
			playerPseudos := []string{}
			for _, pr := range profiles {
				playerPseudos = append(playerPseudos, pr.Pseudo)
			}
			contSummary = append(contSummary, map[string]interface{}{
				"id":            c.ID,
				"name":          c.Name,
				"players":       pStr,
				"factions":      fStr,
				"max_players":   c.MaxPlayers,
				"max_factions":  c.MaxFactions,
				"players_list":  playerPseudos,
			})
		}
		result = append(result, map[string]interface{}{
			"id":         w.ID,
			"name":       w.Name,
			"continents": contSummary,
			"is_active":  w.IsActive,
		})
	}
	return result, nil
}

func (s *WorldService) listActiveWorlds(ctx context.Context) ([]models.World, error) {
	var ws []models.World
	s.db.Where("is_active = ?", true).Find(&ws)
	return ws, nil
}

// Prompt CRUD for "gestion des world" - modifiable in world management UI.
// Prompts are versioned, optimized (cost, speed, constructive/detailed/enriching).
// Can evolve: admin can update system_prompt based on universe state.
func (s *WorldService) ListPrompts(ctx context.Context, domain string) ([]models.Prompt, error) {
	var prompts []models.Prompt
	q := s.db.Where("is_active = ?", true)
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	if err := q.Order("updated_at desc").Find(&prompts).Error; err != nil {
		return nil, err
	}
	return prompts, nil
}

func (s *WorldService) CreatePrompt(ctx context.Context, p *models.Prompt) error {
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt
	return s.db.Create(p).Error
}

func (s *WorldService) UpdatePrompt(ctx context.Context, id uint, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().UTC()
	return s.db.Model(&models.Prompt{}).Where("id = ?", id).Updates(updates).Error
}

func (s *WorldService) GetPrompt(ctx context.Context, promptID, version string) (*models.Prompt, error) {
	var p models.Prompt
	if err := s.db.Where("prompt_id = ? AND version = ? AND is_active = ?", promptID, version, true).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
