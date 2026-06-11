package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/models"
	saimodels "cgwm/battle/internal/nexus_game/server_ai/models"

	"gorm.io/gorm"
)

// WorldService handles world creation, continent assignment for factions/players,
// capacity checks, and proportional distribution using Redis heavily for performance and locking.
type WorldService struct {
	db    *gorm.DB
	redis *cache.RedisService
}

type WorldPlayersQuery struct {
	Limit       int
	Offset      int
	Search      string
	WorldID     uint
	ContinentID uint
}

type DeleteWorldPlayerResult struct {
	ProfileID   uint           `json:"profileId"`
	UserID      uint           `json:"userId"`
	WorldID     uint           `json:"worldId"`
	ContinentID uint           `json:"continentId"`
	Pseudo      string         `json:"pseudo"`
	Deleted     map[string]int `json:"deleted"`
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
// Falls back to DB Faction.WorldID/ContinentID if Redis faction keys are missing (e.g. old factions, Redis flush).
// This prevents 400 on first player profile save when faction Redis mapping is absent.
func (s *WorldService) AssignPlayerToContinent(ctx context.Context, userID, factionID uint) (worldID, continentID uint, err error) {
	// 1. Try Redis first (fast path set at faction creation).
	fContStr, ok, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:faction:%d:continent", factionID))
	if ok {
		fmt.Sscanf(fContStr, "%d", &continentID)
	}
	fWorldStr, _, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:faction:%d:world", factionID))
	if ok && fWorldStr != "" {
		fmt.Sscanf(fWorldStr, "%d", &worldID)
	}

	// 2. Fallback to DB on the Faction if Redis keys missing (common cause of the "record not found" / assignment 400 on first profile save).
	if continentID == 0 || worldID == 0 {
		var f models.Faction
		if err := s.db.Where("id = ?", factionID).First(&f).Error; err == nil {
			if f.ContinentID != 0 {
				continentID = f.ContinentID
			}
			if f.WorldID != 0 {
				worldID = f.WorldID
			}
			// Re-populate Redis for future calls (self-healing).
			if continentID != 0 {
				_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:faction:%d:continent", factionID), fmt.Sprintf("%d", continentID), 0)
			}
			if worldID != 0 {
				_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:faction:%d:world", factionID), fmt.Sprintf("%d", worldID), 0)
			}
		}
	}

	if continentID == 0 || worldID == 0 {
		return 0, 0, fmt.Errorf("faction %d not assigned to continent (no Redis keys and no DB fallback on Faction)", factionID)
	}

	// Check capacity using Redis (with DB fallback for count if needed).
	pKey := fmt.Sprintf("nexus:continent:%d:players", continentID)
	pCountStr, _, _ := s.redis.GetString(ctx, pKey)
	pCount := 0
	fmt.Sscanf(pCountStr, "%d", &pCount)

	if pCount >= 500 {
		return 0, 0, fmt.Errorf("faction continent is full (max 500 players)")
	}

	// Increment count.
	newCount := pCount + 1
	_ = s.redis.SetString(ctx, pKey, fmt.Sprintf("%d", newCount), 0)

	// Caller (profile handler) will persist WorldID/ContinentID on the ProfileGamer.
	return worldID, continentID, nil
}

// ListWorlds returns worlds with continent summary (capacities from Redis).
func (s *WorldService) ListWorlds(ctx context.Context) ([]map[string]interface{}, error) {
	if s.db == nil {
		return nil, errors.New("database unavailable")
	}
	if _, err := s.RepairMissingProfileWorldAssignments(ctx); err != nil {
		return nil, err
	}

	var worlds []models.World
	if err := s.db.WithContext(ctx).Order("id asc").Find(&worlds).Error; err != nil {
		return nil, err
	}

	result := []map[string]interface{}{}
	for _, w := range worlds {
		var conts []models.Continent
		if err := s.db.WithContext(ctx).Where("world_id = ?", w.ID).Order("id asc").Find(&conts).Error; err != nil {
			return nil, err
		}
		contSummary := []map[string]interface{}{}
		worldPlayersCount := int64(0)
		for _, c := range conts {
			pStr, _, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:continent:%d:players", c.ID))
			fStr, _, _ := s.redis.GetString(ctx, fmt.Sprintf("nexus:continent:%d:factions", c.ID))
			dbPlayersCount := int64(0)
			_ = s.db.WithContext(ctx).
				Model(&models.ProfileGamer{}).
				Where("world_id = ? AND continent_id = ?", w.ID, c.ID).
				Count(&dbPlayersCount).Error
			worldPlayersCount += dbPlayersCount
			if !isNonNegativeIntString(pStr) {
				pStr = fmt.Sprintf("%d", dbPlayersCount)
			}
			dbFactionsCount := int64(0)
			_ = s.db.WithContext(ctx).
				Model(&models.Faction{}).
				Where("world_id = ? AND continent_id = ?", w.ID, c.ID).
				Count(&dbFactionsCount).Error
			if !isNonNegativeIntString(fStr) {
				fStr = fmt.Sprintf("%d", dbFactionsCount)
			}

			// Liste des joueurs pour la gestion (from DB ProfileGamer)
			var profiles []models.ProfileGamer
			s.db.WithContext(ctx).
				Where("world_id = ? AND continent_id = ?", w.ID, c.ID).
				Order("created_at desc").
				Limit(25).
				Find(&profiles)
			playerList := []map[string]interface{}{}
			factionNames, err := s.factionNamesByID(ctx, profiles)
			if err != nil {
				return nil, err
			}
			for _, pr := range profiles {
				playerList = append(playerList, map[string]interface{}{
					"profile_id":   pr.ID,
					"user_id":      pr.UserID,
					"pseudo":       pr.Pseudo,
					"city_name":    pr.CityName,
					"level":        pr.Level,
					"power":        pr.Power,
					"faction_id":   pr.FactionID,
					"faction_name": factionNames[pr.FactionID],
					"assigned_at":  pr.CreatedAt.Format("2006-01-02 15:04"),
				})
			}
			contSummary = append(contSummary, map[string]interface{}{
				"id":            c.ID,
				"name":          c.Name,
				"players":       pStr,
				"players_count": dbPlayersCount,
				"factions":      fStr,
				"max_players":   c.MaxPlayers,
				"max_factions":  c.MaxFactions,
				"players_list":  playerList,
			})
		}
		result = append(result, map[string]interface{}{
			"id":            w.ID,
			"name":          w.Name,
			"continents":    contSummary,
			"is_active":     w.IsActive,
			"players_count": worldPlayersCount,
		})
	}
	return result, nil
}

func (s *WorldService) GetWorldDetail(ctx context.Context, worldID uint) (map[string]interface{}, error) {
	if s.db == nil {
		return nil, errors.New("database unavailable")
	}

	var world models.World
	if err := s.db.WithContext(ctx).First(&world, worldID).Error; err != nil {
		return nil, err
	}

	worlds, err := s.ListWorlds(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range worlds {
		if id, ok := item["id"].(uint); ok && id == world.ID {
			return item, nil
		}
	}
	return map[string]interface{}{
		"id":        world.ID,
		"name":      world.Name,
		"is_active": world.IsActive,
	}, nil
}

func (s *WorldService) ListPlayersByWorld(ctx context.Context, worldID uint, query WorldPlayersQuery) (map[string]interface{}, error) {
	if s.db == nil {
		return nil, errors.New("database unavailable")
	}
	if _, err := s.RepairMissingProfileWorldAssignments(ctx); err != nil {
		return nil, err
	}

	if query.Limit <= 0 || query.Limit > 200 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	query.Search = strings.TrimSpace(query.Search)

	var world models.World
	if err := s.db.WithContext(ctx).First(&world, worldID).Error; err != nil {
		return nil, err
	}

	var continents []models.Continent
	if err := s.db.WithContext(ctx).Where("world_id = ?", worldID).Order("id asc").Find(&continents).Error; err != nil {
		return nil, err
	}
	continentNames := make(map[uint]string, len(continents))
	continentSummary := make([]map[string]interface{}, 0, len(continents))
	for _, continent := range continents {
		continentNames[continent.ID] = continent.Name
		var count int64
		_ = s.db.WithContext(ctx).
			Model(&models.ProfileGamer{}).
			Where("world_id = ? AND continent_id = ?", worldID, continent.ID).
			Count(&count).Error
		continentSummary = append(continentSummary, map[string]interface{}{
			"id":            continent.ID,
			"name":          continent.Name,
			"players_count": count,
			"max_players":   continent.MaxPlayers,
		})
	}

	base := s.db.WithContext(ctx).Model(&models.ProfileGamer{}).Where("world_id = ?", worldID)
	if query.ContinentID != 0 {
		base = base.Where("continent_id = ?", query.ContinentID)
	}
	if query.Search != "" {
		like := "%" + query.Search + "%"
		base = base.Where("pseudo LIKE ? OR city_name LIKE ?", like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	var profiles []models.ProfileGamer
	if err := base.Order("created_at desc").Limit(query.Limit).Offset(query.Offset).Find(&profiles).Error; err != nil {
		return nil, err
	}

	factionNames, err := s.factionNamesByID(ctx, profiles)
	if err != nil {
		return nil, err
	}

	players := make([]map[string]interface{}, 0, len(profiles))
	for _, profile := range profiles {
		players = append(players, map[string]interface{}{
			"profile_id":      profile.ID,
			"user_id":         profile.UserID,
			"pseudo":          profile.Pseudo,
			"city_name":       profile.CityName,
			"level":           profile.Level,
			"power":           profile.Power,
			"world_id":        profile.WorldID,
			"continent_id":    profile.ContinentID,
			"continent_name":  continentNames[profile.ContinentID],
			"faction_id":      profile.FactionID,
			"faction_name":    factionNames[profile.FactionID],
			"avatar_id":       profile.AvatarID,
			"ia_companion_id": profile.IACompanionID,
			"profile_created": profile.CreatedAt,
			"profile_updated": profile.UpdatedAt,
			"assigned_at":     profile.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	return map[string]interface{}{
		"world": map[string]interface{}{
			"id":        world.ID,
			"name":      world.Name,
			"is_active": world.IsActive,
		},
		"continents": continentSummary,
		"players":    players,
		"pagination": map[string]interface{}{
			"limit":  query.Limit,
			"offset": query.Offset,
			"total":  total,
		},
		"filters": map[string]interface{}{
			"continent_id": query.ContinentID,
			"search":       query.Search,
		},
	}, nil
}

func (s *WorldService) ListWorldPlayers(ctx context.Context, query WorldPlayersQuery, assignment string) (map[string]interface{}, error) {
	if s.db == nil {
		return nil, errors.New("database unavailable")
	}
	repaired, err := s.RepairMissingProfileWorldAssignments(ctx)
	if err != nil {
		return nil, err
	}

	if query.Limit <= 0 || query.Limit > 200 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	query.Search = strings.TrimSpace(query.Search)
	assignment = strings.TrimSpace(strings.ToLower(assignment))
	if assignment == "" {
		assignment = "all"
	}

	base := s.db.WithContext(ctx).Model(&models.ProfileGamer{})
	if query.WorldID != 0 {
		base = base.Where("world_id = ?", query.WorldID)
	}
	if query.ContinentID != 0 {
		base = base.Where("continent_id = ?", query.ContinentID)
	}
	switch assignment {
	case "assigned":
		base = base.Where("world_id IS NOT NULL AND world_id <> 0 AND continent_id IS NOT NULL AND continent_id <> 0")
	case "unassigned":
		base = base.Where("world_id IS NULL OR world_id = 0 OR continent_id IS NULL OR continent_id = 0")
	}
	if query.Search != "" {
		like := "%" + query.Search + "%"
		base = base.Where("pseudo LIKE ? OR city_name LIKE ?", like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	var profiles []models.ProfileGamer
	if err := base.Order("created_at desc").Limit(query.Limit).Offset(query.Offset).Find(&profiles).Error; err != nil {
		return nil, err
	}

	worldNames, err := s.worldNamesByID(ctx, profiles)
	if err != nil {
		return nil, err
	}
	continentNames, err := s.continentNamesByID(ctx, profiles)
	if err != nil {
		return nil, err
	}
	factionNames, err := s.factionNamesByID(ctx, profiles)
	if err != nil {
		return nil, err
	}

	players := make([]map[string]interface{}, 0, len(profiles))
	for _, profile := range profiles {
		players = append(players, profileWorldPlayerPayload(profile, worldNames[profile.WorldID], continentNames[profile.ContinentID], factionNames[profile.FactionID]))
	}

	var assignedCount int64
	var unassignedCount int64
	_ = s.db.WithContext(ctx).Model(&models.ProfileGamer{}).Where("world_id IS NOT NULL AND world_id <> 0 AND continent_id IS NOT NULL AND continent_id <> 0").Count(&assignedCount).Error
	_ = s.db.WithContext(ctx).Model(&models.ProfileGamer{}).Where("world_id IS NULL OR world_id = 0 OR continent_id IS NULL OR continent_id = 0").Count(&unassignedCount).Error

	return map[string]interface{}{
		"players": players,
		"pagination": map[string]interface{}{
			"limit":  query.Limit,
			"offset": query.Offset,
			"total":  total,
		},
		"filters": map[string]interface{}{
			"world_id":     query.WorldID,
			"continent_id": query.ContinentID,
			"search":       query.Search,
			"assignment":   assignment,
		},
		"summary": map[string]interface{}{
			"assigned_count":           assignedCount,
			"unassigned_count":         unassignedCount,
			"repaired_on_this_request": repaired,
		},
	}, nil
}

func (s *WorldService) DeleteWorldPlayer(ctx context.Context, worldID uint, profileID uint) (DeleteWorldPlayerResult, error) {
	result := DeleteWorldPlayerResult{Deleted: map[string]int{}}
	if s.db == nil {
		return result, errors.New("database unavailable")
	}
	if worldID == 0 || profileID == 0 {
		return result, errors.New("world id and profile id are required")
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var profile models.ProfileGamer
		if err := tx.First(&profile, profileID).Error; err != nil {
			return err
		}
		if profile.WorldID != worldID {
			return fmt.Errorf("profile %d is not assigned to world %d", profileID, worldID)
		}

		result.ProfileID = profile.ID
		result.UserID = profile.UserID
		result.WorldID = profile.WorldID
		result.ContinentID = profile.ContinentID
		result.Pseudo = profile.Pseudo

		deleteWhere := func(label string, model interface{}, query string, args ...interface{}) error {
			res := tx.Unscoped().Where(query, args...).Delete(model)
			if res.Error != nil {
				return res.Error
			}
			result.Deleted[label] = int(res.RowsAffected)
			return nil
		}

		if err := deleteWhere("player_buildings", &models.PlayerBuilding{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("player_units", &models.PlayerUnit{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("player_research", &models.PlayerResearch{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("player_resources", &models.PlayerResource{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("player_city_stats", &models.PlayerCityStats{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("resource_transactions", &models.ResourceTransaction{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("daily_grant_claims", &models.DailyGrantClaim{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("initial_allocation_logs", &models.InitialAllocationLog{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("daily_plans", &models.DailyPlan{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("mmo_ia_agents", &models.MmoIAAgent{}, "profile_gamer_id = ?", profile.ID); err != nil {
			return err
		}
		if err := deleteWhere("server_ai_player_memory", &saimodels.ServerAIPlayerMemory{}, "user_id = ?", profile.UserID); err != nil {
			return err
		}
		if err := deleteWhere("server_ai_attacks", &saimodels.ServerAIAttack{}, "target_user_id = ?", profile.UserID); err != nil {
			return err
		}
		if err := deleteWhere("server_ai_sabotages", &saimodels.ServerAISabotage{}, "target_user_id = ?", profile.UserID); err != nil {
			return err
		}
		if err := deleteWhere("server_ai_espionage", &saimodels.ServerAIEspionage{}, "target_user_id = ?", profile.UserID); err != nil {
			return err
		}
		if err := deleteWhere("avatars", &models.Avatar{}, "player_id = ?", profile.UserID); err != nil {
			return err
		}
		if err := deleteWhere("ia_companions", &models.IACompanion{}, "player_id = ?", profile.UserID); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&models.ProfileGamer{}, profile.ID).Error; err != nil {
			return err
		}
		result.Deleted["profile_gamers"] = 1
		return nil
	})
	if err != nil {
		return result, err
	}

	s.refreshContinentPlayerCounter(ctx, result.ContinentID)
	return result, nil
}

func (s *WorldService) GetWorldPlayerDetail(ctx context.Context, worldID uint, profileID uint) (map[string]interface{}, error) {
	if s.db == nil {
		return nil, errors.New("database unavailable")
	}
	if worldID == 0 || profileID == 0 {
		return nil, errors.New("world id and profile id are required")
	}

	db := s.db.WithContext(ctx)
	var profile models.ProfileGamer
	if err := db.First(&profile, profileID).Error; err != nil {
		return nil, err
	}
	if profile.WorldID != worldID {
		return nil, fmt.Errorf("profile %d is not assigned to world %d", profileID, worldID)
	}
	_ = NewResourceService(s.db).SyncBuildingProduction(ctx, profile.ID, true)
	_ = db.First(&profile, profileID).Error

	var world models.World
	_ = db.First(&world, profile.WorldID).Error
	var continent models.Continent
	_ = db.First(&continent, profile.ContinentID).Error
	var faction models.Faction
	if profile.FactionID != 0 {
		_ = db.First(&faction, profile.FactionID).Error
	}
	var selectedAvatar models.Avatar
	if profile.AvatarID != 0 {
		_ = db.First(&selectedAvatar, profile.AvatarID).Error
	}
	var selectedCompanion models.IACompanion
	if profile.IACompanionID != 0 {
		_ = db.First(&selectedCompanion, profile.IACompanionID).Error
	}

	var resources []models.PlayerResource
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("resource_code ASC").Find(&resources).Error; err != nil {
		return nil, err
	}
	var resourceCatalog []models.ResourceCatalog
	if err := db.Where("is_active = ?", true).Order("sort_order ASC, code ASC").Find(&resourceCatalog).Error; err != nil {
		return nil, err
	}

	var cityStats models.PlayerCityStats
	cityStatsFound := db.Where("profile_gamer_id = ?", profile.ID).First(&cityStats).Error == nil

	var buildings []models.PlayerBuilding
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("content_id ASC, level DESC").Find(&buildings).Error; err != nil {
		return nil, err
	}
	buildingIDs := make([]string, 0, len(buildings))
	for _, building := range buildings {
		buildingIDs = append(buildingIDs, building.ContentID)
	}
	buildingDefs := make(map[string]models.BuildingDefinition)
	if len(buildingIDs) > 0 {
		var defs []models.BuildingDefinition
		if err := db.Where("content_id IN ?", uniqueStrings(buildingIDs)).Find(&defs).Error; err != nil {
			return nil, err
		}
		for _, def := range defs {
			buildingDefs[def.ContentID] = def
		}
	}
	buildingPayload := make([]map[string]interface{}, 0, len(buildings))
	constructionQueue := make([]map[string]interface{}, 0)
	for _, building := range buildings {
		item := map[string]interface{}{"playerBuilding": building}
		if def, ok := buildingDefs[building.ContentID]; ok {
			item["definition"] = def
		}
		buildingPayload = append(buildingPayload, item)
		if building.IsConstructing {
			constructionQueue = append(constructionQueue, item)
		}
	}

	var units []models.PlayerUnit
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("content_id ASC").Find(&units).Error; err != nil {
		return nil, err
	}
	unitIDs := make([]string, 0, len(units))
	for _, unit := range units {
		unitIDs = append(unitIDs, unit.ContentID)
	}
	unitDefs := make(map[string]models.UnitDefinition)
	if len(unitIDs) > 0 {
		var defs []models.UnitDefinition
		if err := db.Where("content_id IN ?", uniqueStrings(unitIDs)).Find(&defs).Error; err != nil {
			return nil, err
		}
		for _, def := range defs {
			unitDefs[def.ContentID] = def
		}
	}
	unitPayload := make([]map[string]interface{}, 0, len(units))
	for _, unit := range units {
		item := map[string]interface{}{"playerUnit": unit}
		if def, ok := unitDefs[unit.ContentID]; ok {
			item["definition"] = def
		}
		unitPayload = append(unitPayload, item)
	}

	var research []models.PlayerResearch
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("completed_at DESC").Find(&research).Error; err != nil {
		return nil, err
	}
	researchIDs := make([]string, 0, len(research))
	for _, item := range research {
		researchIDs = append(researchIDs, item.ContentID)
	}
	researchDefs := make(map[string]models.ResearchDefinition)
	if len(researchIDs) > 0 {
		var defs []models.ResearchDefinition
		if err := db.Where("content_id IN ?", uniqueStrings(researchIDs)).Find(&defs).Error; err != nil {
			return nil, err
		}
		for _, def := range defs {
			researchDefs[def.ContentID] = def
		}
	}
	researchPayload := make([]map[string]interface{}, 0, len(research))
	for _, playerResearch := range research {
		item := map[string]interface{}{"playerResearch": playerResearch}
		if def, ok := researchDefs[playerResearch.ContentID]; ok {
			item["definition"] = def
		}
		researchPayload = append(researchPayload, item)
	}

	var avatars []models.Avatar
	if err := db.Where("player_id = ?", profile.UserID).Order("created_at DESC").Find(&avatars).Error; err != nil {
		return nil, err
	}
	var companions []models.IACompanion
	if err := db.Where("player_id = ?", profile.UserID).Order("created_at DESC").Find(&companions).Error; err != nil {
		return nil, err
	}
	var agents []models.MmoIAAgent
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("role ASC, created_at DESC").Find(&agents).Error; err != nil {
		return nil, err
	}

	var transactions []models.ResourceTransaction
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("created_at DESC").Limit(50).Find(&transactions).Error; err != nil {
		return nil, err
	}
	var claims []models.DailyGrantClaim
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("created_at DESC").Limit(30).Find(&claims).Error; err != nil {
		return nil, err
	}
	var dailyPlans []models.DailyPlan
	if err := db.Where("profile_gamer_id = ?", profile.ID).Order("generated_at DESC, created_at DESC").Limit(10).Find(&dailyPlans).Error; err != nil {
		return nil, err
	}
	var allocation models.InitialAllocationLog
	allocationFound := db.Where("profile_gamer_id = ?", profile.ID).First(&allocation).Error == nil

	var memory saimodels.ServerAIPlayerMemory
	memoryFound := db.Where("world_id = ? AND user_id = ?", profile.WorldID, profile.UserID).First(&memory).Error == nil
	var attacks []saimodels.ServerAIAttack
	if err := db.Where("world_id = ? AND target_user_id = ?", profile.WorldID, profile.UserID).Order("created_at DESC").Limit(25).Find(&attacks).Error; err != nil {
		return nil, err
	}
	var sabotages []saimodels.ServerAISabotage
	if err := db.Where("world_id = ? AND target_user_id = ?", profile.WorldID, profile.UserID).Order("created_at DESC").Limit(25).Find(&sabotages).Error; err != nil {
		return nil, err
	}
	var espionage []saimodels.ServerAIEspionage
	if err := db.Where("world_id = ? AND target_user_id = ?", profile.WorldID, profile.UserID).Order("created_at DESC").Limit(25).Find(&espionage).Error; err != nil {
		return nil, err
	}
	var aiOutputs []models.AIOutput
	linkedTypes := []string{"profile", "player", "profile_gamer", "user"}
	linkedIDs := []uint{profile.ID, profile.UserID}
	if err := db.Where("(world_id = ? AND linked_id IN ?) OR (linked_id IN ? AND linked_type IN ?)", profile.WorldID, linkedIDs, linkedIDs, linkedTypes).
		Order("created_at DESC").
		Limit(25).
		Find(&aiOutputs).Error; err != nil {
		return nil, err
	}
	var aiCallLogs []saimodels.ServerAICallLog
	if err := db.Where("linked_id IN ? AND linked_type IN ?", linkedIDs, linkedTypes).
		Order("created_at DESC").
		Limit(25).
		Find(&aiCallLogs).Error; err != nil {
		return nil, err
	}
	var adminActions []saimodels.ServerAIAdminAction
	if err := db.Where("user_id = ? OR (target_id IN ? AND target_type IN ?)", profile.UserID, linkedIDs, linkedTypes).
		Order("created_at DESC").
		Limit(25).
		Find(&adminActions).Error; err != nil {
		return nil, err
	}

	city := map[string]interface{}{
		"name":                profile.CityName,
		"level":               profile.Level,
		"power":               profile.Power,
		"population":          profile.Population,
		"populationCapacity":  profile.PopulationCapacity,
		"morale":              profile.Morale,
		"energyProduction":    profile.EnergyProduction,
		"energyConsumption":   profile.EnergyConsumption,
		"energyBalance":       profile.EnergyBalance,
		"energyStored":        profile.EnergyStored,
		"security":            profile.Security,
		"buildingsCount":      len(buildings),
		"unitsKindsCount":     len(units),
		"researchCount":       len(research),
		"constructionInQueue": len(constructionQueue),
	}
	if cityStatsFound {
		city["cityStats"] = cityStats
		city["populationFree"] = max(0, profile.PopulationCapacity-profile.Population)
		city["populationGrowthPerHour"] = cityStats.PopulationGrowthHour
		city["populationRemainder"] = cityStats.PopulationRemainder
		city["lastPopulationSyncAt"] = cityStats.LastPopulationSyncAt
		city["foodProduction"] = cityStats.FoodProduction
		city["foodConsumption"] = cityStats.FoodConsumption
		city["foodBalance"] = cityStats.FoodBalance
	}

	identity := map[string]interface{}{
		"world":             world,
		"continent":         continent,
		"faction":           faction,
		"selectedAvatar":    selectedAvatar,
		"selectedCompanion": selectedCompanion,
	}

	serverAI := map[string]interface{}{
		"attacks":    attacks,
		"sabotages":  sabotages,
		"espionage":  espionage,
		"aiOutputs":  aiOutputs,
		"aiCallLogs": aiCallLogs,
		"hasMemory":  memoryFound,
		"memory":     nil,
		"limits":     map[string]int{"transactions": 50, "dailyGrantClaims": 30, "dailyPlans": 10, "serverAIEvents": 25},
		"queriedFor": map[string]uint{"worldId": profile.WorldID, "profileId": profile.ID, "userId": profile.UserID},
	}
	if memoryFound {
		serverAI["memory"] = memory
	}

	payload := map[string]interface{}{
		"profile":              profile,
		"identity":             identity,
		"city":                 city,
		"resources":            resources,
		"resourceCatalog":      resourceCatalog,
		"buildings":            buildingPayload,
		"units":                unitPayload,
		"research":             researchPayload,
		"queues":               map[string]interface{}{"construction": constructionQueue},
		"avatars":              avatars,
		"companions":           companions,
		"agents":               agents,
		"resourceTransactions": transactions,
		"dailyGrantClaims":     claims,
		"dailyPlans":           dailyPlans,
		"adminActions":         adminActions,
		"serverAI":             serverAI,
		"generatedAt":          time.Now().UTC(),
	}
	if allocationFound {
		payload["initialAllocation"] = allocation
	}

	return payload, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (s *WorldService) refreshContinentPlayerCounter(ctx context.Context, continentID uint) {
	if s.redis == nil || s.db == nil || continentID == 0 {
		return
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.ProfileGamer{}).Where("continent_id = ?", continentID).Count(&count).Error; err != nil {
		return
	}
	_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:continent:%d:players", continentID), fmt.Sprintf("%d", count), 0)
}

func (s *WorldService) RepairMissingProfileWorldAssignments(ctx context.Context) (int, error) {
	if s.db == nil {
		return 0, errors.New("database unavailable")
	}
	if _, err := s.RepairMissingFactionWorldAssignments(ctx); err != nil {
		return 0, err
	}

	var profiles []models.ProfileGamer
	if err := s.db.WithContext(ctx).
		Where("world_id IS NULL OR world_id = 0 OR continent_id IS NULL OR continent_id = 0").
		Find(&profiles).Error; err != nil {
		return 0, err
	}

	repaired := 0
	for _, profile := range profiles {
		worldID := profile.WorldID
		continentID := profile.ContinentID

		if continentID != 0 && worldID == 0 {
			var continent models.Continent
			if err := s.db.WithContext(ctx).First(&continent, continentID).Error; err == nil && continent.WorldID != 0 {
				worldID = continent.WorldID
			}
		}

		if profile.FactionID != 0 && (worldID == 0 || continentID == 0) {
			var faction models.Faction
			if err := s.db.WithContext(ctx).First(&faction, profile.FactionID).Error; err == nil {
				if worldID == 0 {
					worldID = faction.WorldID
				}
				if continentID == 0 {
					continentID = faction.ContinentID
				}
			}
		}

		if continentID != 0 && worldID == 0 {
			var continent models.Continent
			if err := s.db.WithContext(ctx).First(&continent, continentID).Error; err == nil && continent.WorldID != 0 {
				worldID = continent.WorldID
			}
		}

		if worldID == 0 || continentID == 0 || (worldID == profile.WorldID && continentID == profile.ContinentID) {
			continue
		}

		if err := s.db.WithContext(ctx).
			Model(&models.ProfileGamer{}).
			Where("id = ?", profile.ID).
			Updates(map[string]interface{}{
				"world_id":     worldID,
				"continent_id": continentID,
			}).Error; err != nil {
			return repaired, err
		}
		repaired++
	}

	return repaired, nil
}

func (s *WorldService) RepairMissingFactionWorldAssignments(ctx context.Context) (int, error) {
	if s.db == nil {
		return 0, errors.New("database unavailable")
	}

	var factions []models.Faction
	if err := s.db.WithContext(ctx).
		Where("world_id IS NULL OR world_id = 0 OR continent_id IS NULL OR continent_id = 0").
		Order("id asc").
		Find(&factions).Error; err != nil {
		return 0, err
	}
	if len(factions) == 0 {
		return 0, nil
	}

	continents, err := s.availableContinentsForFactionRepair(ctx)
	if err != nil {
		return 0, err
	}
	if len(continents) == 0 {
		world, err := s.CreateWorld(ctx)
		if err != nil {
			return 0, err
		}
		continents, err = s.availableContinentsForFactionRepair(ctx)
		if err != nil {
			return 0, err
		}
		if len(continents) == 0 {
			return 0, fmt.Errorf("world %d has no continent for faction assignment", world.ID)
		}
	}

	factionCounts := make(map[uint]int)
	for _, continent := range continents {
		var count int64
		_ = s.db.WithContext(ctx).
			Model(&models.Faction{}).
			Where("world_id = ? AND continent_id = ?", continent.WorldID, continent.ID).
			Count(&count).Error
		factionCounts[continent.ID] = int(count)
	}

	repaired := 0
	for _, faction := range factions {
		continentIndex := -1
		for index, continent := range continents {
			if factionCounts[continent.ID] < continent.MaxFactions {
				continentIndex = index
				break
			}
		}
		if continentIndex < 0 {
			return repaired, errors.New("no continent capacity available for faction repair")
		}

		continent := continents[continentIndex]
		if err := s.db.WithContext(ctx).
			Model(&models.Faction{}).
			Where("id = ?", faction.ID).
			Updates(map[string]interface{}{
				"world_id":     continent.WorldID,
				"continent_id": continent.ID,
			}).Error; err != nil {
			return repaired, err
		}
		factionCounts[continent.ID]++
		_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:faction:%d:world", faction.ID), fmt.Sprintf("%d", continent.WorldID), 0)
		_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:faction:%d:continent", faction.ID), fmt.Sprintf("%d", continent.ID), 0)
		_ = s.redis.SetString(ctx, fmt.Sprintf("nexus:continent:%d:factions", continent.ID), fmt.Sprintf("%d", factionCounts[continent.ID]), 0)
		repaired++
	}

	return repaired, nil
}

func (s *WorldService) availableContinentsForFactionRepair(ctx context.Context) ([]models.Continent, error) {
	var worlds []models.World
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("id asc").Find(&worlds).Error; err != nil {
		return nil, err
	}
	if len(worlds) == 0 {
		return nil, nil
	}
	worldIDs := make([]uint, 0, len(worlds))
	for _, world := range worlds {
		worldIDs = append(worldIDs, world.ID)
	}

	var continents []models.Continent
	if err := s.db.WithContext(ctx).
		Where("world_id IN ?", worldIDs).
		Order("world_id asc, id asc").
		Find(&continents).Error; err != nil {
		return nil, err
	}
	return continents, nil
}

func profileWorldPlayerPayload(profile models.ProfileGamer, worldName string, continentName string, factionName string) map[string]interface{} {
	assignmentStatus := "assigned"
	if profile.WorldID == 0 || profile.ContinentID == 0 {
		assignmentStatus = "unassigned"
	}
	return map[string]interface{}{
		"profile_id":        profile.ID,
		"user_id":           profile.UserID,
		"pseudo":            profile.Pseudo,
		"city_name":         profile.CityName,
		"level":             profile.Level,
		"power":             profile.Power,
		"world_id":          profile.WorldID,
		"world_name":        worldName,
		"continent_id":      profile.ContinentID,
		"continent_name":    continentName,
		"faction_id":        profile.FactionID,
		"faction_name":      factionName,
		"avatar_id":         profile.AvatarID,
		"ia_companion_id":   profile.IACompanionID,
		"assignment_status": assignmentStatus,
		"profile_created":   profile.CreatedAt,
		"profile_updated":   profile.UpdatedAt,
		"assigned_at":       profile.CreatedAt.Format("2006-01-02 15:04"),
	}
}

func isNonNegativeIntString(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func (s *WorldService) factionNamesByID(ctx context.Context, profiles []models.ProfileGamer) (map[uint]string, error) {
	ids := make([]uint, 0)
	seen := make(map[uint]struct{})
	for _, profile := range profiles {
		if profile.FactionID == 0 {
			continue
		}
		if _, exists := seen[profile.FactionID]; exists {
			continue
		}
		seen[profile.FactionID] = struct{}{}
		ids = append(ids, profile.FactionID)
	}
	if len(ids) == 0 {
		return map[uint]string{}, nil
	}

	var factions []models.Faction
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&factions).Error; err != nil {
		return nil, err
	}
	names := make(map[uint]string, len(factions))
	for _, faction := range factions {
		names[faction.ID] = faction.Name
	}
	return names, nil
}

func (s *WorldService) worldNamesByID(ctx context.Context, profiles []models.ProfileGamer) (map[uint]string, error) {
	ids := make([]uint, 0)
	seen := make(map[uint]struct{})
	for _, profile := range profiles {
		if profile.WorldID == 0 {
			continue
		}
		if _, exists := seen[profile.WorldID]; exists {
			continue
		}
		seen[profile.WorldID] = struct{}{}
		ids = append(ids, profile.WorldID)
	}
	if len(ids) == 0 {
		return map[uint]string{}, nil
	}

	var worlds []models.World
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&worlds).Error; err != nil {
		return nil, err
	}
	names := make(map[uint]string, len(worlds))
	for _, world := range worlds {
		names[world.ID] = world.Name
	}
	return names, nil
}

func (s *WorldService) continentNamesByID(ctx context.Context, profiles []models.ProfileGamer) (map[uint]string, error) {
	ids := make([]uint, 0)
	seen := make(map[uint]struct{})
	for _, profile := range profiles {
		if profile.ContinentID == 0 {
			continue
		}
		if _, exists := seen[profile.ContinentID]; exists {
			continue
		}
		seen[profile.ContinentID] = struct{}{}
		ids = append(ids, profile.ContinentID)
	}
	if len(ids) == 0 {
		return map[uint]string{}, nil
	}

	var continents []models.Continent
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&continents).Error; err != nil {
		return nil, err
	}
	names := make(map[uint]string, len(continents))
	for _, continent := range continents {
		names[continent.ID] = continent.Name
	}
	return names, nil
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
