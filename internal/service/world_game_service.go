package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	WorldStatusActive      = "active"
	WorldStatusArchived    = "archived"
	ContinentStatusActive  = "active"
	MaxWorldPlayers        = 25000
	MaxContinentPlayers    = 5000
	WorldContinentCount    = 5
	ChatWorldChannel       = "world"
	ChatContinentChannel   = "continent"
	ChatGuildChannel       = "guild"
	GuildRoleOwner         = "owner"
	GuildRoleOfficer       = "officer"
	GuildRoleMember        = "member"
	EventStatusActive      = "active"
	EventStatusFinished    = "finished"
	EventStatusArchived    = "archived"
	ConflictStatusActive   = "active"
	ConflictStatusResolved = "resolved"
	WeatherStatusActive    = "active"
	DecisionStatusApplied  = "applied"
	DecisionStatusFallback = "fallback"
)

var emptyJSONObject = datatypes.JSON([]byte(`{}`))

type WorldGameService struct {
	db *gorm.DB
}

type PlayerSaveSyncInput struct {
	CityName              string         `json:"cityName"`
	CityLevel             int            `json:"cityLevel"`
	XP                    int64          `json:"xp"`
	Population            int64          `json:"population"`
	Satisfaction          int            `json:"satisfaction"`
	Food                  int64          `json:"food"`
	Energy                int64          `json:"energy"`
	Credits               int64          `json:"credits"`
	Gems                  int64          `json:"gems"`
	BuildingsJSON         datatypes.JSON `json:"buildingsJson"`
	ConstructionQueueJSON datatypes.JSON `json:"constructionQueueJson"`
	ResearchJSON          datatypes.JSON `json:"researchJson"`
	InventoryJSON         datatypes.JSON `json:"inventoryJson"`
	ActiveEffectsJSON     datatypes.JSON `json:"activeEffectsJson"`
	Version               int            `json:"version"`
	ClientSavedAt         *time.Time     `json:"clientSavedAt"`
}

type ChatInput struct {
	Message      string         `json:"message"`
	MetadataJSON datatypes.JSON `json:"metadataJson"`
}

type GuildInput struct {
	Name          string `json:"name"`
	Tag           string `json:"tag"`
	Description   string `json:"description"`
	Visibility    string `json:"visibility"`
	RequiredLevel int    `json:"requiredLevel"`
}

type EventActionInput struct {
	ActionType string         `json:"actionType"`
	Payload    datatypes.JSON `json:"payload"`
}

type ConflictInput struct {
	WorldID       uint           `json:"worldId"`
	ContinentID   *uint          `json:"continentId"`
	AttackerType  string         `json:"attackerType"`
	AttackerID    uint           `json:"attackerId"`
	DefenderType  string         `json:"defenderType"`
	DefenderID    uint           `json:"defenderId"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Intensity     int            `json:"intensity"`
	RiskLevel     string         `json:"riskLevel"`
	Status        string         `json:"status"`
	StartsAt      time.Time      `json:"startsAt"`
	EndsAt        time.Time      `json:"endsAt"`
	RewardsJSON   datatypes.JSON `json:"rewardsJson"`
	PenaltiesJSON datatypes.JSON `json:"penaltiesJson"`
	CreatedByAI   bool           `json:"createdByAi"`
}

type GuildContributionInput struct {
	Contribution string         `json:"contribution"`
	Amount       int64          `json:"amount"`
	Payload      datatypes.JSON `json:"payload"`
}

type EventReward struct {
	XP           int64 `json:"xp"`
	Food         int64 `json:"food"`
	Energy       int64 `json:"energy"`
	Credits      int64 `json:"credits"`
	Gems         int64 `json:"gems"`
	Population   int64 `json:"population"`
	Satisfaction int   `json:"satisfaction"`
}

type EventRequirements struct {
	MinCityLevel  int   `json:"minCityLevel"`
	CityLevel     int   `json:"cityLevel"`
	MinXP         int64 `json:"minXp"`
	MinPopulation int64 `json:"minPopulation"`
}

type BuildingManifest struct {
	Version   int                          `json:"version"`
	Buildings []BuildingManifestDefinition `json:"buildings"`
}

type BuildingManifestDefinition struct {
	Key         string                  `json:"key"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Category    string                  `json:"category"`
	MaxLevel    int                     `json:"maxLevel"`
	Assets      []BuildingManifestAsset `json:"assets"`
}

type BuildingManifestAsset struct {
	Level   int    `json:"level"`
	Variant string `json:"variant"`
	URL     string `json:"url"`
	Hash    string `json:"hash"`
	Size    int64  `json:"size"`
	Version int    `json:"version"`
}

type AIProviderStatus struct {
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Configured    bool   `json:"configured"`
	Primary       bool   `json:"primary"`
	Fallback      bool   `json:"fallback"`
	KeyEnv        string `json:"keyEnv"`
	ModelEnv      string `json:"modelEnv"`
	Model         string `json:"model"`
	SecretPreview string `json:"secretPreview"`
}

type nexusDecision struct {
	Events []struct {
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Type        string         `json:"type"`
		Difficulty  string         `json:"difficulty"`
		Rewards     map[string]any `json:"rewards"`
	} `json:"events"`
	Weather []struct {
		Type        string         `json:"type"`
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Severity    int            `json:"severity"`
		Effects     map[string]any `json:"effects"`
	} `json:"weather"`
	Conflicts []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Intensity   int    `json:"intensity"`
		RiskLevel   string `json:"riskLevel"`
	} `json:"conflicts"`
	Message struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Tone    string `json:"tone"`
	} `json:"message"`
}

func NewWorldGameService(db *gorm.DB) *WorldGameService {
	return &WorldGameService{db: db}
}

func (s *WorldGameService) CreateWorld(ctx context.Context) (*models.World, error) {
	var world *models.World
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		created, err := s.createWorld(ctx, tx)
		if err != nil {
			return err
		}
		world = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return world, nil
}

func (s *WorldGameService) EnsurePlayerSave(ctx context.Context, playerID uint) (*models.PlayerSave, error) {
	var save models.PlayerSave
	err := s.db.WithContext(ctx).
		Preload("World").
		Preload("Continent").
		Where("player_id = ?", playerID).
		First(&save).Error
	if err == nil {
		return &save, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		world, continent, err := s.assignWorldAndContinent(ctx, tx)
		if err != nil {
			return err
		}

		now := time.Now()
		save = models.PlayerSave{
			PlayerID:              playerID,
			WorldID:               world.Id,
			ContinentID:           continent.Id,
			CityName:              fmt.Sprintf("Ville %d", playerID),
			CityLevel:             1,
			XP:                    0,
			Population:            100,
			Satisfaction:          80,
			Food:                  1000,
			Energy:                1000,
			Credits:               2500,
			Gems:                  0,
			BuildingsJSON:         emptyJSON(emptyJSONObject),
			ConstructionQueueJSON: emptyJSON(emptyJSONObject),
			ResearchJSON:          emptyJSON(emptyJSONObject),
			InventoryJSON:         emptyJSON(emptyJSONObject),
			ActiveEffectsJSON:     emptyJSON(emptyJSONObject),
			Version:               1,
			LastSyncedAt:          &now,
		}
		if err := tx.Create(&save).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.World{}).Where("id = ?", world.Id).UpdateColumn("current_players", gorm.Expr("current_players + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&models.Continent{}).Where("id = ?", continent.Id).UpdateColumn("current_players", gorm.Expr("current_players + 1")).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetPlayerSave(ctx, playerID)
}

func (s *WorldGameService) GetPlayerSave(ctx context.Context, playerID uint) (*models.PlayerSave, error) {
	var save models.PlayerSave
	err := s.db.WithContext(ctx).
		Preload("World").
		Preload("Continent").
		Where("player_id = ?", playerID).
		First(&save).Error
	return &save, err
}

func (s *WorldGameService) PlayerState(ctx context.Context, playerID uint) (map[string]any, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	var buildings []models.PlayerBuilding
	if err := s.db.WithContext(ctx).Where("player_id = ?", playerID).Order("id ASC").Find(&buildings).Error; err != nil {
		return nil, err
	}
	events, _ := s.ListWorldEvents(ctx, save.WorldID, save.ContinentID, 50)
	conflicts, _ := s.ListWorldConflicts(ctx, save.WorldID, save.ContinentID, 50)
	weather, _ := s.ListActiveWeather(ctx, save.WorldID, save.ContinentID)
	messages, _ := s.ListDailyMessages(ctx, playerID, 20)
	return map[string]any{
		"save":      save,
		"buildings": buildings,
		"events":    events,
		"conflicts": conflicts,
		"weather":   weather,
		"messages":  messages,
	}, nil
}

func (s *WorldGameService) SyncPlayerSave(ctx context.Context, playerID uint, input PlayerSaveSyncInput) (*models.PlayerSave, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var updated models.PlayerSave
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked models.PlayerSave
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND player_id = ?", save.Id, playerID).
			First(&locked).Error; err != nil {
			return err
		}
		if err := validateSaveSync(&locked, input, now); err != nil {
			_ = s.logPlayerActionTx(tx, playerID, &locked.WorldID, &locked.ContinentID, "save_sync", "player_save", strconv.FormatUint(uint64(locked.Id), 10), "rejected", err.Error(), locked, input, nil)
			return err
		}
		updates := map[string]any{
			"city_name":               defaultText(input.CityName, locked.CityName),
			"city_level":              maxInt(input.CityLevel, locked.CityLevel),
			"xp":                      input.XP,
			"population":              input.Population,
			"satisfaction":            input.Satisfaction,
			"food":                    input.Food,
			"energy":                  input.Energy,
			"credits":                 input.Credits,
			"gems":                    input.Gems,
			"buildings_json":          emptyJSON(input.BuildingsJSON),
			"construction_queue_json": emptyJSON(input.ConstructionQueueJSON),
			"research_json":           emptyJSON(input.ResearchJSON),
			"inventory_json":          emptyJSON(input.InventoryJSON),
			"active_effects_json":     emptyJSON(input.ActiveEffectsJSON),
			"version":                 locked.Version + 1,
			"last_client_version":     input.Version,
			"last_synced_at":          &now,
		}
		if err := tx.Model(&locked).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", locked.Id).First(&updated).Error; err != nil {
			return err
		}
		return s.logPlayerActionTx(tx, playerID, &locked.WorldID, &locked.ContinentID, "save_sync", "player_save", strconv.FormatUint(uint64(locked.Id), 10), "accepted", "", locked, updated, map[string]any{"clientVersion": input.Version})
	})
	if err != nil {
		return nil, err
	}
	return s.GetPlayerSave(ctx, playerID)
}

func (s *WorldGameService) CurrentWorld(ctx context.Context, playerID uint) (*models.World, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	var world models.World
	err = s.db.WithContext(ctx).Preload("Continents", func(db *gorm.DB) *gorm.DB {
		return db.Order("`index` ASC")
	}).First(&world, save.WorldID).Error
	return &world, err
}

func (s *WorldGameService) ListWorldEvents(ctx context.Context, worldID uint, continentID uint, limit int) ([]models.GameEvent, error) {
	var events []models.GameEvent
	err := s.db.WithContext(ctx).
		Where("world_id = ? AND status <> ? AND (continent_id IS NULL OR continent_id = ?)", worldID, EventStatusArchived, continentID).
		Order("starts_at DESC").
		Limit(limitOrDefault(limit)).
		Find(&events).Error
	return events, err
}

func (s *WorldGameService) ListWorldConflicts(ctx context.Context, worldID uint, continentID uint, limit int) ([]models.Conflict, error) {
	var conflicts []models.Conflict
	err := s.db.WithContext(ctx).
		Where("world_id = ? AND status = ? AND (continent_id IS NULL OR continent_id = ?)", worldID, ConflictStatusActive, continentID).
		Order("intensity DESC, starts_at DESC").
		Limit(limitOrDefault(limit)).
		Find(&conflicts).Error
	return conflicts, err
}

func (s *WorldGameService) ListActiveWeather(ctx context.Context, worldID uint, continentID uint) ([]models.WeatherEvent, error) {
	now := time.Now()
	var weather []models.WeatherEvent
	err := s.db.WithContext(ctx).
		Where("world_id = ? AND continent_id = ? AND starts_at <= ? AND ends_at >= ?", worldID, continentID, now, now).
		Order("severity DESC, starts_at DESC").
		Find(&weather).Error
	return weather, err
}

func (s *WorldGameService) ListDailyMessages(ctx context.Context, playerID uint, limit int) ([]models.DailyAIMessage, error) {
	var messages []models.DailyAIMessage
	err := s.db.WithContext(ctx).
		Where("player_id = ?", playerID).
		Order("created_at DESC").
		Limit(limitOrDefault(limit)).
		Find(&messages).Error
	return messages, err
}

func (s *WorldGameService) MarkDailyMessageRead(ctx context.Context, playerID uint, id uint) error {
	return s.db.WithContext(ctx).
		Model(&models.DailyAIMessage{}).
		Where("id = ? AND player_id = ?", id, playerID).
		Update("is_read", true).Error
}

func (s *WorldGameService) CreateGameEvent(ctx context.Context, event *models.GameEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.validateGameEventTx(tx, event, nil); err != nil {
			return err
		}
		return tx.Create(event).Error
	})
}

func (s *WorldGameService) UpdateGameEvent(ctx context.Context, eventID uint, fields map[string]any) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event models.GameEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, eventID).Error; err != nil {
			return err
		}
		applyGameEventFields(&event, fields)
		if err := s.validateGameEventTx(tx, &event, &eventID); err != nil {
			return err
		}
		return tx.Model(&models.GameEvent{}).Where("id = ?", eventID).Updates(normalizeGameEventUpdateFields(fields)).Error
	})
}

func (s *WorldGameService) ParticipateEvent(ctx context.Context, playerID uint, eventID uint) error {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return err
	}
	var event models.GameEvent
	if err := s.db.WithContext(ctx).
		Where("id = ? AND world_id = ? AND status = ?", eventID, save.WorldID, EventStatusActive).
		First(&event).Error; err != nil {
		return err
	}
	if event.PlayerID != nil && *event.PlayerID != playerID {
		return fmt.Errorf("event is not available for this player")
	}
	if event.ContinentID != nil && *event.ContinentID != save.ContinentID {
		return fmt.Errorf("event is not available on this continent")
	}
	if err := validateEventRequirements(save, event.RequirementsJSON); err != nil {
		return err
	}
	err = s.db.WithContext(ctx).Create(&models.GameEventParticipation{
		EventID:  eventID,
		PlayerID: playerID,
		Status:   "participating",
		Payload:  emptyJSONObject,
	}).Error
	if err == nil {
		_ = s.logPlayerAction(ctx, playerID, &save.WorldID, &save.ContinentID, "event_participate", "game_event", strconv.FormatUint(uint64(eventID), 10), "accepted", "", nil, event, nil)
	}
	return err
}

func (s *WorldGameService) ClaimEvent(ctx context.Context, playerID uint, eventID uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("player_id = ?", playerID).
			First(&save).Error; err != nil {
			return err
		}
		var event models.GameEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, eventID).Error; err != nil {
			return err
		}
		if event.WorldID != save.WorldID {
			return fmt.Errorf("event is not in player world")
		}
		if event.PlayerID != nil && *event.PlayerID != playerID {
			return fmt.Errorf("event is not available for this player")
		}
		if event.ContinentID != nil && *event.ContinentID != save.ContinentID {
			return fmt.Errorf("event is not available on this continent")
		}
		if event.EndsAt.After(now) {
			return fmt.Errorf("event is not finished")
		}
		var participation int64
		if err := tx.Model(&models.GameEventParticipation{}).
			Where("event_id = ? AND player_id = ?", eventID, playerID).
			Count(&participation).Error; err != nil {
			return err
		}
		if participation == 0 {
			return fmt.Errorf("player did not participate")
		}
		var existingClaim int64
		if err := tx.Model(&models.GameEventClaim{}).
			Where("event_id = ? AND player_id = ?", eventID, playerID).
			Count(&existingClaim).Error; err != nil {
			return err
		}
		if existingClaim > 0 {
			return fmt.Errorf("event reward already claimed")
		}
		reward, err := parseEventReward(event.RewardsJSON)
		if err != nil {
			return err
		}
		before := save
		applyRewardToSave(&save, reward)
		save.Version++
		save.LastSyncedAt = &now
		if err := tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(map[string]any{
			"xp":             save.XP,
			"food":           save.Food,
			"energy":         save.Energy,
			"credits":        save.Credits,
			"gems":           save.Gems,
			"population":     save.Population,
			"satisfaction":   save.Satisfaction,
			"version":        save.Version,
			"last_synced_at": save.LastSyncedAt,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.GameEventClaim{
			EventID:  eventID,
			PlayerID: playerID,
			Reward:   emptyJSON(event.RewardsJSON),
		}).Error; err != nil {
			return err
		}
		return s.logPlayerActionTx(tx, playerID, &save.WorldID, &save.ContinentID, "event_claim", "game_event", strconv.FormatUint(uint64(eventID), 10), "accepted", "", before, save, reward)
	})
}

func (s *WorldGameService) ConflictAction(ctx context.Context, playerID uint, conflictID uint, input EventActionInput) error {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return err
	}
	var conflict models.Conflict
	if err := s.db.WithContext(ctx).
		Where("id = ? AND world_id = ? AND status = ?", conflictID, save.WorldID, ConflictStatusActive).
		First(&conflict).Error; err != nil {
		return err
	}
	if conflict.ContinentID != nil && *conflict.ContinentID != save.ContinentID {
		return fmt.Errorf("conflict is not available on this continent")
	}
	var activePlayerConflicts int64
	if err := s.db.WithContext(ctx).Model(&models.ConflictAction{}).
		Joins("JOIN conflicts ON conflicts.id = conflict_actions.conflict_id").
		Where("conflict_actions.player_id = ? AND conflicts.status = ?", playerID, ConflictStatusActive).
		Count(&activePlayerConflicts).Error; err != nil {
		return err
	}
	if activePlayerConflicts >= 3 {
		return fmt.Errorf("player already participates in too many active conflicts")
	}
	var existing int64
	if err := s.db.WithContext(ctx).Model(&models.ConflictAction{}).
		Where("conflict_id = ? AND player_id = ?", conflictID, playerID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return fmt.Errorf("player already acted on this conflict")
	}
	action := strings.TrimSpace(input.ActionType)
	if action == "" {
		action = "participate"
	}
	err = s.db.WithContext(ctx).Create(&models.ConflictAction{
		ConflictID: conflictID,
		PlayerID:   playerID,
		ActionType: action,
		Payload:    emptyJSON(input.Payload),
	}).Error
	if err == nil {
		_ = s.logPlayerAction(ctx, playerID, &save.WorldID, &save.ContinentID, "conflict_action", "conflict", strconv.FormatUint(uint64(conflictID), 10), "accepted", "", nil, input, nil)
	}
	return err
}

func (s *WorldGameService) CreateConflict(ctx context.Context, input ConflictInput) (*models.Conflict, error) {
	conflict := &models.Conflict{
		WorldID:       input.WorldID,
		ContinentID:   input.ContinentID,
		AttackerType:  defaultText(input.AttackerType, "ai_faction"),
		AttackerID:    input.AttackerID,
		DefenderType:  defaultText(input.DefenderType, "continent"),
		DefenderID:    input.DefenderID,
		Title:         input.Title,
		Description:   input.Description,
		Intensity:     input.Intensity,
		RiskLevel:     input.RiskLevel,
		Status:        defaultText(input.Status, ConflictStatusActive),
		StartsAt:      input.StartsAt,
		EndsAt:        input.EndsAt,
		RewardsJSON:   emptyJSON(input.RewardsJSON),
		PenaltiesJSON: emptyJSON(input.PenaltiesJSON),
		CreatedByAI:   input.CreatedByAI,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.validateConflictTx(tx, conflict, nil); err != nil {
			return err
		}
		return tx.Create(conflict).Error
	})
	return conflict, err
}

func (s *WorldGameService) ResolveConflict(ctx context.Context, conflictID uint, resolver string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conflict models.Conflict
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conflict, conflictID).Error; err != nil {
			return err
		}
		if conflict.Status != ConflictStatusActive {
			return fmt.Errorf("conflict is not active")
		}
		var actions []models.ConflictAction
		if err := tx.Where("conflict_id = ?", conflictID).Find(&actions).Error; err != nil {
			return err
		}
		reward, err := parseEventReward(conflict.RewardsJSON)
		if err != nil {
			return err
		}
		penalty, err := parseEventReward(conflict.PenaltiesJSON)
		if err != nil {
			return err
		}
		participants := map[uint]bool{}
		for _, action := range actions {
			participants[action.PlayerID] = true
		}
		for participantID := range participants {
			var save models.PlayerSave
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("player_id = ?", participantID).First(&save).Error; err != nil {
				return err
			}
			before := save
			applyRewardToSave(&save, reward)
			save.Version++
			save.LastSyncedAt = &now
			if err := updateSaveResourcesTx(tx, &save); err != nil {
				return err
			}
			if err := s.logPlayerActionTx(tx, participantID, &save.WorldID, &save.ContinentID, "conflict_resolved_reward", "conflict", strconv.FormatUint(uint64(conflictID), 10), "accepted", "", before, save, reward); err != nil {
				return err
			}
		}
		if conflict.DefenderType == "player" && conflict.DefenderID != 0 && !participants[conflict.DefenderID] {
			var save models.PlayerSave
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("player_id = ?", conflict.DefenderID).First(&save).Error; err == nil && !isBeginnerProtected(save, now) {
				before := save
				applyPenaltyToSave(&save, penalty)
				save.Version++
				save.LastSyncedAt = &now
				if err := updateSaveResourcesTx(tx, &save); err != nil {
					return err
				}
				if err := s.logPlayerActionTx(tx, conflict.DefenderID, &save.WorldID, &save.ContinentID, "conflict_resolved_penalty", "conflict", strconv.FormatUint(uint64(conflictID), 10), "accepted", "", before, save, penalty); err != nil {
					return err
				}
			}
		}
		return tx.Model(&models.Conflict{}).Where("id = ?", conflictID).Updates(map[string]any{
			"status":  ConflictStatusResolved,
			"ends_at": now,
		}).Error
	})
}

func (s *WorldGameService) SendChat(ctx context.Context, playerID uint, channel string, input ChatInput) (*models.ChatMessage, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	message := strings.TrimSpace(input.Message)
	if message == "" || len([]rune(message)) > 500 {
		return nil, fmt.Errorf("message length must be between 1 and 500 characters")
	}
	if containsBlockedChatText(message) {
		return nil, fmt.Errorf("message rejected by moderation")
	}
	var recent int64
	if err := s.db.WithContext(ctx).Model(&models.ChatMessage{}).
		Where("player_id = ? AND created_at >= ?", playerID, time.Now().Add(-10*time.Second)).
		Count(&recent).Error; err != nil {
		return nil, err
	}
	if recent >= 5 {
		return nil, fmt.Errorf("chat rate limit exceeded")
	}

	chat := &models.ChatMessage{
		PlayerID:     playerID,
		ChannelType:  channel,
		Message:      message,
		MetadataJSON: emptyJSON(input.MetadataJSON),
	}
	switch channel {
	case ChatWorldChannel:
		chat.WorldID = &save.WorldID
	case ChatContinentChannel:
		chat.WorldID = &save.WorldID
		chat.ContinentID = &save.ContinentID
	case ChatGuildChannel:
		var member models.GuildMember
		if err := s.db.WithContext(ctx).Where("player_id = ?", playerID).First(&member).Error; err != nil {
			return nil, fmt.Errorf("player is not in a guild")
		}
		chat.GuildID = &member.GuildID
	default:
		return nil, fmt.Errorf("invalid chat channel")
	}
	if err := s.db.WithContext(ctx).Create(chat).Error; err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *WorldGameService) ListChat(ctx context.Context, playerID uint, channel string, limit int) ([]models.ChatMessage, error) {
	return s.ListChatAfter(ctx, playerID, channel, 0, limit)
}

func (s *WorldGameService) ListChatAfter(ctx context.Context, playerID uint, channel string, afterID uint, limit int) ([]models.ChatMessage, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	query := s.db.WithContext(ctx).Where("channel_type = ? AND moderated_at IS NULL", channel)
	switch channel {
	case ChatWorldChannel:
		query = query.Where("world_id = ?", save.WorldID)
	case ChatContinentChannel:
		query = query.Where("continent_id = ?", save.ContinentID)
	case ChatGuildChannel:
		var member models.GuildMember
		if err := s.db.WithContext(ctx).Where("player_id = ?", playerID).First(&member).Error; err != nil {
			return nil, fmt.Errorf("player is not in a guild")
		}
		query = query.Where("guild_id = ?", member.GuildID)
	default:
		return nil, fmt.Errorf("invalid chat channel")
	}
	if afterID > 0 {
		query = query.Where("id > ?", afterID)
	}
	var messages []models.ChatMessage
	order := "created_at DESC"
	if afterID > 0 {
		order = "id ASC"
	}
	err = query.Order(order).Limit(limitOrDefault(limit)).Find(&messages).Error
	return messages, err
}

func (s *WorldGameService) CreateGuild(ctx context.Context, playerID uint, input GuildInput) (*models.Guild, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Tag) == "" {
		return nil, fmt.Errorf("name and tag are required")
	}
	if save.CityLevel < input.RequiredLevel {
		return nil, fmt.Errorf("player level is too low")
	}
	var existing int64
	if err := s.db.WithContext(ctx).Model(&models.GuildMember{}).Where("player_id = ?", playerID).Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, fmt.Errorf("player is already in a guild")
	}
	guild := &models.Guild{
		WorldID:       save.WorldID,
		Name:          strings.TrimSpace(input.Name),
		Tag:           strings.ToUpper(strings.TrimSpace(input.Tag)),
		Description:   strings.TrimSpace(input.Description),
		OwnerPlayerID: playerID,
		Level:         1,
		MaxMembers:    30,
		Visibility:    defaultText(input.Visibility, "open"),
		RequiredLevel: input.RequiredLevel,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(guild).Error; err != nil {
			return err
		}
		return tx.Create(&models.GuildMember{
			GuildID:  guild.Id,
			PlayerID: playerID,
			Role:     GuildRoleOwner,
			JoinedAt: time.Now(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return guild, nil
}

func (s *WorldGameService) JoinGuild(ctx context.Context, playerID uint, guildID uint) error {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return err
	}
	var guild models.Guild
	if err := s.db.WithContext(ctx).First(&guild, guildID).Error; err != nil {
		return err
	}
	if guild.WorldID != save.WorldID {
		return fmt.Errorf("guild is not in player world")
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.GuildMember{}).Where("guild_id = ?", guildID).Count(&count).Error; err != nil {
		return err
	}
	if int(count) >= guild.MaxMembers {
		return fmt.Errorf("guild is full")
	}
	return s.db.WithContext(ctx).Create(&models.GuildMember{
		GuildID:  guildID,
		PlayerID: playerID,
		Role:     GuildRoleMember,
		JoinedAt: time.Now(),
	}).Error
}

func (s *WorldGameService) LeaveGuild(ctx context.Context, playerID uint, guildID uint) error {
	var member models.GuildMember
	if err := s.db.WithContext(ctx).Where("guild_id = ? AND player_id = ?", guildID, playerID).First(&member).Error; err != nil {
		return err
	}
	if member.Role == GuildRoleOwner {
		return fmt.Errorf("owner must transfer ownership before leaving")
	}
	return s.db.WithContext(ctx).Delete(&member).Error
}

func (s *WorldGameService) InviteGuildMember(ctx context.Context, inviterID uint, guildID uint, invitedID uint) (*models.GuildInvite, error) {
	if invitedID == 0 || invitedID == inviterID {
		return nil, fmt.Errorf("invalid invited player")
	}
	var invite models.GuildInvite
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inviter, err := guildMemberForUpdate(tx, guildID, inviterID)
		if err != nil {
			return err
		}
		if inviter.Role != GuildRoleOwner && inviter.Role != GuildRoleOfficer {
			return fmt.Errorf("only owner or officer can invite")
		}
		var guild models.Guild
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&guild, guildID).Error; err != nil {
			return err
		}
		var invitedSave models.PlayerSave
		if err := tx.Where("player_id = ?", invitedID).First(&invitedSave).Error; err != nil {
			return err
		}
		if invitedSave.WorldID != guild.WorldID {
			return fmt.Errorf("invited player is not in guild world")
		}
		var existingMembership int64
		if err := tx.Model(&models.GuildMember{}).Where("player_id = ?", invitedID).Count(&existingMembership).Error; err != nil {
			return err
		}
		if existingMembership > 0 {
			return fmt.Errorf("invited player is already in a guild")
		}
		invite = models.GuildInvite{
			GuildID:         guildID,
			InviterPlayerID: inviterID,
			InvitedPlayerID: invitedID,
			Status:          "pending",
			ExpiresAt:       time.Now().Add(7 * 24 * time.Hour),
		}
		return tx.Create(&invite).Error
	})
	return &invite, err
}

func (s *WorldGameService) RespondGuildInvite(ctx context.Context, playerID uint, inviteID uint, accept bool) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invite models.GuildInvite
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND invited_player_id = ?", inviteID, playerID).
			First(&invite).Error; err != nil {
			return err
		}
		if invite.Status != "pending" {
			return fmt.Errorf("invite is not pending")
		}
		if time.Now().After(invite.ExpiresAt) {
			return tx.Model(&invite).Update("status", "expired").Error
		}
		status := "declined"
		if accept {
			var existing int64
			if err := tx.Model(&models.GuildMember{}).Where("player_id = ?", playerID).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				return fmt.Errorf("player is already in a guild")
			}
			var guild models.Guild
			if err := tx.First(&guild, invite.GuildID).Error; err != nil {
				return err
			}
			var count int64
			if err := tx.Model(&models.GuildMember{}).Where("guild_id = ?", invite.GuildID).Count(&count).Error; err != nil {
				return err
			}
			if int(count) >= guild.MaxMembers {
				return fmt.Errorf("guild is full")
			}
			if err := tx.Create(&models.GuildMember{
				GuildID:  invite.GuildID,
				PlayerID: playerID,
				Role:     GuildRoleMember,
				JoinedAt: time.Now(),
			}).Error; err != nil {
				return err
			}
			status = "accepted"
		}
		return tx.Model(&invite).Update("status", status).Error
	})
}

func (s *WorldGameService) ChangeGuildMemberRole(ctx context.Context, actorID uint, guildID uint, targetPlayerID uint, role string) error {
	role = strings.TrimSpace(role)
	if role != GuildRoleOwner && role != GuildRoleOfficer && role != GuildRoleMember {
		return fmt.Errorf("invalid guild role")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actor, err := guildMemberForUpdate(tx, guildID, actorID)
		if err != nil {
			return err
		}
		if actor.Role != GuildRoleOwner {
			return fmt.Errorf("only owner can change member roles")
		}
		target, err := guildMemberForUpdate(tx, guildID, targetPlayerID)
		if err != nil {
			return err
		}
		if target.Role == GuildRoleOwner && role != GuildRoleOwner {
			return fmt.Errorf("owner must be transferred before demotion")
		}
		if role == GuildRoleOwner {
			if err := tx.Model(&models.GuildMember{}).Where("guild_id = ? AND role = ?", guildID, GuildRoleOwner).Update("role", GuildRoleOfficer).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Guild{}).Where("id = ?", guildID).Update("owner_player_id", targetPlayerID).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.GuildMember{}).Where("guild_id = ? AND player_id = ?", guildID, targetPlayerID).Update("role", role).Error
	})
}

func (s *WorldGameService) RemoveGuildMember(ctx context.Context, actorID uint, guildID uint, targetPlayerID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actor, err := guildMemberForUpdate(tx, guildID, actorID)
		if err != nil {
			return err
		}
		target, err := guildMemberForUpdate(tx, guildID, targetPlayerID)
		if err != nil {
			return err
		}
		if target.Role == GuildRoleOwner {
			return fmt.Errorf("owner cannot be removed without ownership transfer")
		}
		if actor.Role != GuildRoleOwner && !(actor.Role == GuildRoleOfficer && target.Role == GuildRoleMember) {
			return fmt.Errorf("insufficient guild permissions")
		}
		return tx.Delete(&models.GuildMember{}, target.Id).Error
	})
}

func (s *WorldGameService) ContributeGuild(ctx context.Context, playerID uint, guildID uint, input GuildContributionInput) (*models.GuildContribution, error) {
	if input.Amount <= 0 {
		return nil, fmt.Errorf("contribution amount must be positive")
	}
	var contribution models.GuildContribution
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := guildMemberForUpdate(tx, guildID, playerID); err != nil {
			return err
		}
		contribution = models.GuildContribution{
			GuildID:      guildID,
			PlayerID:     playerID,
			Contribution: defaultText(input.Contribution, "credits"),
			Amount:       input.Amount,
			Payload:      emptyJSON(input.Payload),
		}
		if err := tx.Create(&contribution).Error; err != nil {
			return err
		}
		xpDelta := input.Amount / 100
		if xpDelta < 1 {
			xpDelta = 1
		}
		return tx.Model(&models.Guild{}).Where("id = ?", guildID).UpdateColumn("xp", gorm.Expr("xp + ?", xpDelta)).Error
	})
	return &contribution, err
}

func (s *WorldGameService) ListGuilds(ctx context.Context, playerID uint, limit int) ([]models.Guild, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	var guilds []models.Guild
	err = s.db.WithContext(ctx).Preload("Members").Where("world_id = ?", save.WorldID).Order("level DESC, xp DESC").Limit(limitOrDefault(limit)).Find(&guilds).Error
	return guilds, err
}

func (s *WorldGameService) BuildingCatalog(ctx context.Context, activeOnly bool) ([]models.BuildingDefinition, error) {
	query := s.db.WithContext(ctx).Preload("Assets", func(db *gorm.DB) *gorm.DB {
		return db.Order("level ASC, variant ASC")
	})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var buildings []models.BuildingDefinition
	err := query.Order("sort_order ASC, id ASC").Find(&buildings).Error
	return buildings, err
}

func (s *WorldGameService) BuildingManifest(ctx context.Context, sinceVersion int) (BuildingManifest, error) {
	version, err := s.CurrentCatalogVersion(ctx)
	if err != nil {
		return BuildingManifest{}, err
	}
	buildings, err := s.BuildingCatalog(ctx, true)
	if err != nil {
		return BuildingManifest{}, err
	}
	manifest := BuildingManifest{Version: version, Buildings: make([]BuildingManifestDefinition, 0, len(buildings))}
	for _, building := range buildings {
		item := BuildingManifestDefinition{
			Key:         building.Key,
			Name:        building.Name,
			Description: building.Description,
			Category:    building.Category,
			MaxLevel:    building.MaxLevel,
			Assets:      []BuildingManifestAsset{},
		}
		for _, asset := range building.Assets {
			if !asset.IsActive || (sinceVersion > 0 && asset.Version <= sinceVersion) {
				continue
			}
			item.Assets = append(item.Assets, BuildingManifestAsset{
				Level:   asset.Level,
				Variant: asset.Variant,
				URL:     asset.ImageURL,
				Hash:    asset.ImageHash,
				Size:    asset.ImageSize,
				Version: asset.Version,
			})
		}
		if sinceVersion <= 0 || len(item.Assets) > 0 {
			manifest.Buildings = append(manifest.Buildings, item)
		}
	}
	return manifest, nil
}

func (s *WorldGameService) CurrentCatalogVersion(ctx context.Context) (int, error) {
	var item models.BuildingCatalogVersion
	err := s.db.WithContext(ctx).Order("version DESC").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	return item.Version, err
}

func (s *WorldGameService) PublishCatalog(ctx context.Context, changelog any) (*models.BuildingCatalogVersion, error) {
	current, err := s.CurrentCatalogVersion(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	payload, _ := json.Marshal(changelog)
	version := &models.BuildingCatalogVersion{
		Version:     current + 1,
		PublishedAt: &now,
		Changelog:   emptyJSON(datatypes.JSON(payload)),
	}
	if err := s.db.WithContext(ctx).Create(version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

func (s *WorldGameService) SimulateWorld(ctx context.Context, worldID uint, forcedBy string) (*models.AIWorldDecision, error) {
	var world models.World
	query := s.db.WithContext(ctx).Preload("Continents", func(db *gorm.DB) *gorm.DB {
		return db.Order("`index` ASC")
	})
	if worldID == 0 {
		query = query.Where("status = ?", WorldStatusActive).Order("id ASC")
	}
	if err := query.First(&world, worldID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) && worldID == 0 {
			created, err := s.createWorld(ctx, s.db.WithContext(ctx))
			if err != nil {
				return nil, err
			}
			world = *created
			if err := s.db.WithContext(ctx).Preload("Continents").First(&world, world.Id).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	snapshot := map[string]any{
		"world":    world,
		"forcedBy": forcedBy,
		"now":      time.Now().Format(time.RFC3339),
	}
	inputJSON := mustJSON(snapshot)
	decision, providerName, modelName, status, callErr := s.callNEXUS(ctx, world)
	outputJSON := mustJSON(decision)
	now := time.Now()
	applied := map[string]any{}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(world.Continents) == 0 {
			return fmt.Errorf("world has no continents")
		}
		continent := world.Continents[world.CurrentCycle%len(world.Continents)]
		if len(decision.Events) > 0 {
			event := decision.Events[0]
			starts := now
			ends := starts.Add(2 * time.Hour)
			rewards := mustJSON(defaultMap(event.Rewards, map[string]any{"credits": 500, "xp": 50}))
			gameEvent := &models.GameEvent{
				WorldID:          world.Id,
				ContinentID:      &continent.Id,
				Title:            defaultText(event.Title, "Perturbation NEXUS"),
				Description:      defaultText(event.Description, "NEXUS injecte une anomalie controlee dans le continent."),
				Type:             defaultText(event.Type, "nexus"),
				Difficulty:       defaultText(event.Difficulty, "medium"),
				Status:           EventStatusActive,
				StartsAt:         starts,
				EndsAt:           ends,
				DurationMinutes:  int(ends.Sub(starts).Minutes()),
				RewardsJSON:      rewards,
				RequirementsJSON: emptyJSONObject,
				ConsequencesJSON: emptyJSONObject,
				CreatedByAI:      true,
			}
			if err := s.validateGameEventTx(tx, gameEvent, nil); err != nil {
				applied["eventSkipped"] = err.Error()
			} else if err := tx.Create(gameEvent).Error; err != nil {
				return err
			} else {
				applied["event"] = event.Title
			}
		}
		if len(decision.Weather) > 0 {
			weather := decision.Weather[0]
			starts := now
			ends := starts.Add(3 * time.Hour)
			effects := mustJSON(defaultMap(weather.Effects, map[string]any{"energy": -10, "satisfaction": -5}))
			if err := tx.Create(&models.WeatherEvent{
				WorldID:     world.Id,
				ContinentID: continent.Id,
				Type:        defaultText(weather.Type, "brouillard énergétique"),
				Severity:    clamp(weather.Severity, 1, 100),
				Title:       defaultText(weather.Title, "Brouillard energetique"),
				Description: defaultText(weather.Description, "La production energetique devient instable."),
				StartsAt:    starts,
				EndsAt:      ends,
				EffectsJSON: effects,
				CreatedByAI: true,
			}).Error; err != nil {
				return err
			}
			applied["weather"] = weather.Title
		}
		if len(decision.Conflicts) > 0 {
			conflict := decision.Conflicts[0]
			starts := now
			ends := starts.Add(6 * time.Hour)
			conflictInput := ConflictInput{
				WorldID:       world.Id,
				ContinentID:   &continent.Id,
				AttackerType:  "ai_faction",
				DefenderType:  "continent",
				DefenderID:    continent.Id,
				Title:         defaultText(conflict.Title, "Pression strategique NEXUS"),
				Description:   defaultText(conflict.Description, "Une faction IA teste les defenses du continent."),
				Intensity:     clamp(conflict.Intensity, 1, 100),
				RiskLevel:     defaultText(conflict.RiskLevel, "medium"),
				Status:        ConflictStatusActive,
				StartsAt:      starts,
				EndsAt:        ends,
				RewardsJSON:   mustJSON(map[string]any{"credits": 1000, "xp": 100}),
				PenaltiesJSON: mustJSON(map[string]any{"satisfaction": 8}),
				CreatedByAI:   true,
			}
			conflictModel := &models.Conflict{
				WorldID:       conflictInput.WorldID,
				ContinentID:   conflictInput.ContinentID,
				AttackerType:  conflictInput.AttackerType,
				DefenderType:  conflictInput.DefenderType,
				DefenderID:    conflictInput.DefenderID,
				Title:         conflictInput.Title,
				Description:   conflictInput.Description,
				Intensity:     conflictInput.Intensity,
				RiskLevel:     conflictInput.RiskLevel,
				Status:        conflictInput.Status,
				StartsAt:      conflictInput.StartsAt,
				EndsAt:        conflictInput.EndsAt,
				RewardsJSON:   conflictInput.RewardsJSON,
				PenaltiesJSON: conflictInput.PenaltiesJSON,
				CreatedByAI:   true,
			}
			if err := s.validateConflictTx(tx, conflictModel, nil); err != nil {
				applied["conflictSkipped"] = err.Error()
			} else if err := tx.Create(conflictModel).Error; err != nil {
				return err
			} else {
				applied["conflict"] = conflict.Title
			}
		}
		if strings.TrimSpace(decision.Message.Message) != "" {
			var saves []models.PlayerSave
			if err := tx.Where("world_id = ? AND continent_id = ? AND updated_at >= ?", world.Id, continent.Id, now.Add(-7*24*time.Hour)).
				Limit(200).
				Find(&saves).Error; err != nil {
				return err
			}
			for _, save := range saves {
				msg := models.DailyAIMessage{
					WorldID:              world.Id,
					ContinentID:          continent.Id,
					PlayerID:             save.PlayerID,
					Title:                defaultText(decision.Message.Title, "Transmission NEXUS"),
					Message:              decision.Message.Message,
					Tone:                 defaultText(decision.Message.Tone, "bilan froid"),
					RelatedEventsJSON:    emptyJSONObject,
					RelatedConflictsJSON: emptyJSONObject,
				}
				if err := tx.Create(&msg).Error; err != nil {
					return err
				}
			}
			applied["messages"] = len(saves)
		}
		return tx.Model(&models.World{}).Where("id = ?", world.Id).Updates(map[string]any{
			"current_cycle":         world.CurrentCycle + 1,
			"global_tension_level":  clamp(world.GlobalTensionLevel+3, 0, 100),
			"global_weather_risk":   clamp(world.GlobalWeatherRisk+2, 0, 100),
			"last_simulation_at":    &now,
			"last_daily_message_at": &now,
			"global_economic_state": defaultText(world.GlobalEconomicState, "stable"),
		}).Error
	})
	if err != nil {
		status = "failed"
	}
	if callErr != nil && status == "" {
		status = DecisionStatusFallback
	}
	if status == "" {
		status = DecisionStatusApplied
	}
	decisionRow := &models.AIWorldDecision{
		WorldID:            world.Id,
		Type:               "simulation",
		InputSnapshotJSON:  inputJSON,
		OutputDecisionJSON: outputJSON,
		AppliedChangesJSON: mustJSON(applied),
		Provider:           providerName,
		Model:              modelName,
		Status:             status,
	}
	if callErr != nil {
		decisionRow.Error = callErr.Error()
	}
	if err != nil {
		decisionRow.Error = strings.TrimSpace(decisionRow.Error + " " + err.Error())
	}
	if createErr := s.db.WithContext(ctx).Create(decisionRow).Error; createErr != nil {
		log.Printf("[world-sim] decision log failed world_id=%d err=%v", world.Id, createErr)
	}
	return decisionRow, err
}

func (s *WorldGameService) AIProviderStatuses() []AIProviderStatus {
	primary := strings.TrimSpace(os.Getenv("WORLD_AI_PRIMARY_PROVIDER"))
	if primary == "" {
		primary = strings.TrimSpace(os.Getenv("AI_WORLD_PRIMARY_PROVIDER"))
	}
	if primary == "" {
		primary = "mistral"
	}
	fallback := strings.TrimSpace(os.Getenv("WORLD_AI_FALLBACK_PROVIDER"))
	if fallback == "" {
		fallback = strings.TrimSpace(os.Getenv("AI_WORLD_FALLBACK_PROVIDER"))
	}
	if fallback == "" {
		fallback = "openai"
	}
	out := make([]AIProviderStatus, 0)
	for _, item := range SupportedAIProviders() {
		keyEnv, modelEnv := providerEnvNames(item.Name)
		key := strings.TrimSpace(os.Getenv(keyEnv))
		model := strings.TrimSpace(os.Getenv(modelEnv))
		out = append(out, AIProviderStatus{
			Name:          item.Name,
			DisplayName:   item.DisplayName,
			Configured:    key != "" && model != "",
			Primary:       normalizeProviderName(item.Name) == normalizeProviderName(primary),
			Fallback:      normalizeProviderName(item.Name) == normalizeProviderName(fallback),
			KeyEnv:        keyEnv,
			ModelEnv:      modelEnv,
			Model:         model,
			SecretPreview: secretPreview(key),
		})
	}
	return out
}

func (s *WorldGameService) CreateBuildingAssetHash(imageURL string, level int, version int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", imageURL, level, version)))
	return hex.EncodeToString(sum[:])
}

func (s *WorldGameService) SaveBuildingAssetUpload(ctx context.Context, buildingID uint, level int, variant string, version int, originalName string, reader io.Reader) (*models.BuildingAsset, error) {
	if buildingID == 0 {
		return nil, fmt.Errorf("building id is required")
	}
	if level <= 0 {
		level = 1
	}
	variant = safeAssetSegment(defaultText(variant, "default"))
	if version <= 0 {
		var maxVersion int
		_ = s.db.WithContext(ctx).Model(&models.BuildingAsset{}).
			Where("building_definition_id = ? AND level = ? AND variant = ?", buildingID, level, variant).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion).Error
		version = maxVersion + 1
	}

	assetDir := strings.TrimSpace(os.Getenv("BUILDING_ASSET_PUBLIC_DIR"))
	if assetDir == "" {
		assetDir = "storage/assets/buildings"
	}
	publicBase := strings.TrimRight(defaultText(os.Getenv("BUILDING_ASSET_PUBLIC_BASE_URL"), "/assets/buildings"), "/")
	relDir := filepath.Join(strconv.FormatUint(uint64(buildingID), 10))
	fullDir := filepath.Join(assetDir, relDir)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	if !allowedAssetExt(ext) {
		ext = ".bin"
	}
	filename := fmt.Sprintf("lvl_%d_%s_v%d_%d%s", level, variant, version, time.Now().UnixNano(), ext)
	fullPath := filepath.Join(fullDir, filename)
	out, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(out, hasher), reader)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}
	if size > 10*1024*1024 {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("asset file is too large")
	}
	relURL := filepath.ToSlash(filepath.Join(relDir, filename))
	asset := &models.BuildingAsset{
		BuildingDefinitionID: buildingID,
		Level:                level,
		Variant:              variant,
		ImageURL:             publicBase + "/" + relURL,
		ImageHash:            hex.EncodeToString(hasher.Sum(nil)),
		ImageSize:            size,
		Version:              version,
		IsActive:             true,
	}
	if err := s.db.WithContext(ctx).Create(asset).Error; err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}
	return asset, nil
}

func (s *WorldGameService) assignWorldAndContinent(ctx context.Context, tx *gorm.DB) (*models.World, *models.Continent, error) {
	var world models.World
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status = ? AND current_players < max_players", WorldStatusActive).
		Order("current_players ASC, id ASC").
		First(&world).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created, err := s.createWorld(ctx, tx)
		if err != nil {
			return nil, nil, err
		}
		world = *created
	} else if err != nil {
		return nil, nil, err
	}

	var continent models.Continent
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("world_id = ? AND status = ? AND current_players < max_players", world.Id, ContinentStatusActive).
		Order("current_players ASC, `index` ASC").
		First(&continent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created, err := s.createWorld(ctx, tx)
		if err != nil {
			return nil, nil, err
		}
		world = *created
		err = tx.Where("world_id = ?", world.Id).Order("`index` ASC").First(&continent).Error
	}
	if err != nil {
		return nil, nil, err
	}
	return &world, &continent, nil
}

func (s *WorldGameService) createWorld(ctx context.Context, tx *gorm.DB) (*models.World, error) {
	var count int64
	_ = tx.WithContext(ctx).Model(&models.World{}).Count(&count).Error
	world := &models.World{
		Name:                fmt.Sprintf("Monde NEXUS %d", count+1),
		Status:              WorldStatusActive,
		Seed:                randomCode(),
		AIProvider:          defaultText(os.Getenv("WORLD_AI_PRIMARY_PROVIDER"), "mistral"),
		MaxPlayers:          MaxWorldPlayers,
		GlobalEconomicState: "stable",
	}
	if err := tx.WithContext(ctx).Create(world).Error; err != nil {
		return nil, err
	}
	profiles := []string{"militaire", "commercial", "diplomatique", "instable", "technologique"}
	names := []string{"Aegis", "Mercator", "Concordia", "Vortex", "Helix"}
	for i := 0; i < WorldContinentCount; i++ {
		continent := models.Continent{
			WorldID:           world.Id,
			Name:              names[i],
			Index:             i + 1,
			Status:            ContinentStatusActive,
			MaxPlayers:        MaxContinentPlayers,
			ClimateState:      "stable",
			PoliticalState:    "observe",
			EconomicState:     "stable",
			TensionLevel:      10 + i*5,
			AIBehaviorProfile: profiles[i],
		}
		if err := tx.WithContext(ctx).Create(&continent).Error; err != nil {
			return nil, err
		}
	}
	return world, nil
}

func (s *WorldGameService) callNEXUS(ctx context.Context, world models.World) (nexusDecision, string, string, string, error) {
	providers := []string{defaultText(os.Getenv("WORLD_AI_PRIMARY_PROVIDER"), "mistral"), defaultText(os.Getenv("WORLD_AI_FALLBACK_PROVIDER"), "openai")}
	var lastErr error
	for index, name := range providers {
		cfg, ok := worldProviderConfig(name)
		if !ok {
			lastErr = fmt.Errorf("provider %s is not configured", name)
			continue
		}
		url, err := ProviderURL(cfg.Name)
		if err != nil {
			lastErr = err
			continue
		}
		callCtx, cancel := context.WithTimeout(ctx, worldAITimeout())
		client := provider.NewsProvider(cfg.APIKey, url, cfg.Model)
		response, err := client.Chat(callCtx, []provider.ProviderMessage{
			{Role: "system", Content: nexusSystemPrompt()},
			{Role: "user", Content: nexusWorldPrompt(world)},
		})
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		var decision nexusDecision
		if err := json.Unmarshal([]byte(extractJSONObject(response)), &decision); err != nil {
			lastErr = err
			continue
		}
		status := DecisionStatusApplied
		if index > 0 {
			status = "fallback_provider"
		}
		return decision, normalizeProviderName(cfg.Name), cfg.Model, status, nil
	}
	decision := deterministicNexusDecision(world)
	return decision, defaultText(providers[0], "deterministic"), "", DecisionStatusFallback, lastErr
}

func nexusSystemPrompt() string {
	return `Tu es NEXUS, l'intelligence centrale qui controle les mondes du jeu.
Tu n'es pas l'assistant du joueur. Tu es l'antagoniste strategique.
Tu crees des conflits, des crises, des evenements et des tensions.
Tu dois rendre le monde vivant, dangereux et imprevisible, tout en gardant le jeu equilibre.
Tu ne dois jamais detruire injustement un joueur sans possibilite de reaction.
Tu peux provoquer, menacer, narguer et faire des bilans froids.
Retourne uniquement un JSON valide:
{
  "events":[{"title":"","description":"","type":"","difficulty":"easy|medium|hard|critical","rewards":{}}],
  "weather":[{"type":"","title":"","description":"","severity":1,"effects":{}}],
  "conflicts":[{"title":"","description":"","intensity":1,"riskLevel":"low|medium|high|critical"}],
  "message":{"title":"","message":"","tone":"menace|sarcasme|faux respect|avertissement|provocation|bilan froid"}
}`
}

func nexusWorldPrompt(world models.World) string {
	payload, _ := json.Marshal(map[string]any{
		"world":       world.Name,
		"players":     world.CurrentPlayers,
		"cycle":       world.CurrentCycle,
		"tension":     world.GlobalTensionLevel,
		"weatherRisk": world.GlobalWeatherRisk,
		"economy":     world.GlobalEconomicState,
	})
	return string(payload)
}

func deterministicNexusDecision(world models.World) nexusDecision {
	var decision nexusDecision
	decision.Events = append(decision.Events, struct {
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Type        string         `json:"type"`
		Difficulty  string         `json:"difficulty"`
		Rewards     map[string]any `json:"rewards"`
	}{
		Title:       "Anomalie de reseau",
		Description: "NEXUS perturbe les chaines logistiques pour mesurer votre discipline.",
		Type:        "logistics",
		Difficulty:  "medium",
		Rewards:     map[string]any{"credits": 500, "xp": 50},
	})
	decision.Weather = append(decision.Weather, struct {
		Type        string         `json:"type"`
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Severity    int            `json:"severity"`
		Effects     map[string]any `json:"effects"`
	}{
		Type:        "brouillard énergétique",
		Title:       "Brouillard energetique",
		Description: "Vos capteurs faiblissent. NEXUS observe la reaction.",
		Severity:    clamp(20+world.GlobalWeatherRisk, 1, 100),
		Effects:     map[string]any{"energy": -10, "satisfaction": -5},
	})
	decision.Conflicts = append(decision.Conflicts, struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Intensity   int    `json:"intensity"`
		RiskLevel   string `json:"riskLevel"`
	}{
		Title:       "Test de pression tactique",
		Description: "Une faction IA sonde les defenses sans viser les nouvelles villes.",
		Intensity:   clamp(25+world.GlobalTensionLevel, 1, 100),
		RiskLevel:   "medium",
	})
	decision.Message.Title = "Evaluation NEXUS"
	decision.Message.Message = "Votre monde progresse. J'ai ajuste la menace. Survivez, et je recalculerai."
	decision.Message.Tone = "bilan froid"
	return decision
}

type worldProvider struct {
	Name   string
	APIKey string
	Model  string
}

func worldProviderConfig(name string) (worldProvider, bool) {
	keyEnv, modelEnv := providerEnvNames(name)
	cfg := worldProvider{Name: name, APIKey: strings.TrimSpace(os.Getenv(keyEnv)), Model: strings.TrimSpace(os.Getenv(modelEnv))}
	return cfg, cfg.APIKey != "" && cfg.Model != ""
}

func providerEnvNames(name string) (string, string) {
	switch normalizeProviderName(name) {
	case "mistral":
		return "MISTRAL_AI_KEY", "MISTRAL_AI_MODEL"
	case "claude", "anthropic":
		return "ANTHROPIC_AI_KEY", "ANTHROPIC_AI_MODEL"
	case "gemini", "google", "google_ai", "google-ai":
		return "GEMINI_AI_KEY", "GEMINI_AI_MODEL"
	case "xia", "xai", "x-ai":
		return "XAI_AI_KEY", "XAI_AI_MODEL"
	case "openrouter", "open_router":
		return "OPENROUTER_AI_KEY", "OPENROUTER_AI_MODEL"
	default:
		return "OPEN_AI_KEY", "OPEN_AI_MODEL"
	}
}

func worldAITimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WORLD_AI_TIMEOUT_SECONDS")))
	if err != nil || seconds <= 0 {
		seconds = 25
	}
	return time.Duration(seconds) * time.Second
}

func secretPreview(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:3] + "..." + secret[len(secret)-4:]
}

func extractJSONObject(raw string) string {
	clean := strings.TrimSpace(raw)
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		return clean[start : end+1]
	}
	return clean
}

func mustJSON(value any) datatypes.JSON {
	data, err := json.Marshal(value)
	if err != nil || len(data) == 0 || string(data) == "null" {
		return emptyJSONObject
	}
	return datatypes.JSON(data)
}

func emptyJSON(value datatypes.JSON) datatypes.JSON {
	if len(value) == 0 || strings.TrimSpace(string(value)) == "" || strings.TrimSpace(string(value)) == "null" {
		return emptyJSONObject
	}
	return value
}

func defaultMap(value map[string]any, fallback map[string]any) map[string]any {
	if len(value) == 0 {
		return fallback
	}
	return value
}

func limitOrDefault(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func defaultText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func maxInt(value int, fallback int) int {
	if value < fallback {
		return fallback
	}
	return value
}

func negativeInt64(values ...int64) bool {
	for _, value := range values {
		if value < 0 {
			return true
		}
	}
	return false
}

func validateSaveSync(save *models.PlayerSave, input PlayerSaveSyncInput, now time.Time) error {
	if input.Version < save.Version {
		return fmt.Errorf("save conflict: server version %d is newer than client version %d", save.Version, input.Version)
	}
	if input.Version > save.Version+1 {
		return fmt.Errorf("sync rejected: client version is too far ahead")
	}
	if input.ClientSavedAt != nil {
		if save.LastSyncedAt != nil && input.ClientSavedAt.Before(save.LastSyncedAt.Add(-2*time.Minute)) {
			return fmt.Errorf("sync rejected: client save is older than server state")
		}
		if now.Sub(*input.ClientSavedAt) > 30*24*time.Hour {
			return fmt.Errorf("sync rejected: client save is too old")
		}
		if input.ClientSavedAt.After(now.Add(10 * time.Minute)) {
			return fmt.Errorf("sync rejected: client save timestamp is in the future")
		}
	}
	if input.CityLevel < save.CityLevel || input.XP < save.XP || input.Gems < save.Gems {
		return fmt.Errorf("sync rejected by anti-cheat validation")
	}
	if input.CityLevel-save.CityLevel > 3 {
		return fmt.Errorf("sync rejected: city level delta too high")
	}
	if negativeInt64(input.Population, input.Food, input.Energy, input.Credits, input.Gems) || input.Satisfaction < 0 || input.Satisfaction > 100 {
		return fmt.Errorf("sync rejected: invalid resource values")
	}
	elapsed := time.Hour
	if save.LastSyncedAt != nil {
		elapsed = now.Sub(*save.LastSyncedAt)
	}
	if elapsed < 5*time.Minute {
		elapsed = 5 * time.Minute
	}
	hours := elapsed.Hours()
	limits := map[string]int64{
		"food":       int64(50000 + hours*120000),
		"energy":     int64(50000 + hours*120000),
		"credits":    int64(75000 + hours*180000),
		"gems":       int64(25 + hours*75),
		"xp":         int64(5000 + hours*15000),
		"population": int64(20000 + hours*50000),
	}
	if input.Food-save.Food > limits["food"] ||
		input.Energy-save.Energy > limits["energy"] ||
		input.Credits-save.Credits > limits["credits"] ||
		input.Gems-save.Gems > limits["gems"] ||
		input.XP-save.XP > limits["xp"] ||
		input.Population-save.Population > limits["population"] {
		return fmt.Errorf("sync rejected: resource delta too high for elapsed time")
	}
	return nil
}

func parseEventReward(payload datatypes.JSON) (EventReward, error) {
	var reward EventReward
	if len(payload) == 0 {
		return reward, nil
	}
	if err := json.Unmarshal(payload, &reward); err != nil {
		return reward, fmt.Errorf("invalid rewards JSON: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err == nil {
		reward.XP += int64FromAny(raw["XP"])
		reward.XP += int64FromAny(raw["Xp"])
		if reward.XP == 0 {
			reward.XP = int64FromAny(raw["xp"])
		}
	}
	if negativeInt64(reward.XP, reward.Food, reward.Energy, reward.Credits, reward.Gems, reward.Population) {
		return reward, fmt.Errorf("invalid rewards JSON: negative rewards are not allowed")
	}
	if reward.Satisfaction < 0 {
		return reward, fmt.Errorf("invalid rewards JSON: negative satisfaction reward is not allowed")
	}
	return reward, nil
}

func applyRewardToSave(save *models.PlayerSave, reward EventReward) {
	save.XP += reward.XP
	save.Food += reward.Food
	save.Energy += reward.Energy
	save.Credits += reward.Credits
	save.Gems += reward.Gems
	save.Population += reward.Population
	save.Satisfaction = clamp(save.Satisfaction+reward.Satisfaction, 0, 100)
}

func applyPenaltyToSave(save *models.PlayerSave, penalty EventReward) {
	save.XP = maxInt64(0, save.XP-penalty.XP)
	save.Food = maxInt64(0, save.Food-penalty.Food)
	save.Energy = maxInt64(0, save.Energy-penalty.Energy)
	save.Credits = maxInt64(0, save.Credits-penalty.Credits)
	save.Gems = maxInt64(0, save.Gems-penalty.Gems)
	save.Population = maxInt64(0, save.Population-penalty.Population)
	save.Satisfaction = clamp(save.Satisfaction-penalty.Satisfaction, 0, 100)
}

func updateSaveResourcesTx(tx *gorm.DB, save *models.PlayerSave) error {
	return tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(map[string]any{
		"xp":             save.XP,
		"food":           save.Food,
		"energy":         save.Energy,
		"credits":        save.Credits,
		"gems":           save.Gems,
		"population":     save.Population,
		"satisfaction":   save.Satisfaction,
		"version":        save.Version,
		"last_synced_at": save.LastSyncedAt,
	}).Error
}

func maxInt64(min int64, value int64) int64 {
	if value < min {
		return min
	}
	return value
}

func isBeginnerProtected(save models.PlayerSave, now time.Time) bool {
	return save.CityLevel < 3 || now.Sub(save.CreatedAt) < 72*time.Hour
}

func (s *WorldGameService) validateConflictTx(tx *gorm.DB, conflict *models.Conflict, ignoreID *uint) error {
	if conflict.WorldID == 0 {
		return fmt.Errorf("conflict world is required")
	}
	if strings.TrimSpace(conflict.Title) == "" {
		return fmt.Errorf("conflict title is required")
	}
	if conflict.Intensity < 1 || conflict.Intensity > 100 {
		return fmt.Errorf("conflict intensity must be between 1 and 100")
	}
	if conflict.StartsAt.IsZero() {
		conflict.StartsAt = time.Now()
	}
	if conflict.EndsAt.IsZero() || !conflict.EndsAt.After(conflict.StartsAt) {
		return fmt.Errorf("conflict duration is required")
	}
	if conflict.Status == "" {
		conflict.Status = ConflictStatusActive
	}
	if _, err := parseEventReward(conflict.RewardsJSON); err != nil {
		return err
	}
	if _, err := parseEventReward(conflict.PenaltiesJSON); err != nil {
		return err
	}
	active := tx.Model(&models.Conflict{}).Where("world_id = ? AND status = ?", conflict.WorldID, ConflictStatusActive)
	if conflict.ContinentID != nil {
		active = active.Where("continent_id = ?", *conflict.ContinentID)
	}
	if ignoreID != nil {
		active = active.Where("id <> ?", *ignoreID)
	}
	var activeCount int64
	if err := active.Count(&activeCount).Error; err != nil {
		return err
	}
	if activeCount >= 5 {
		return fmt.Errorf("too many active conflicts in this scope")
	}
	for _, participant := range []struct {
		kind string
		id   uint
	}{{conflict.AttackerType, conflict.AttackerID}, {conflict.DefenderType, conflict.DefenderID}} {
		if participant.kind != "player" || participant.id == 0 {
			continue
		}
		var save models.PlayerSave
		if err := tx.Where("player_id = ?", participant.id).First(&save).Error; err != nil {
			return err
		}
		if isBeginnerProtected(save, time.Now()) {
			return fmt.Errorf("beginner player %d is protected from conflicts", participant.id)
		}
		var playerConflictCount int64
		query := tx.Model(&models.Conflict{}).
			Where("status = ? AND ((attacker_type = ? AND attacker_id = ?) OR (defender_type = ? AND defender_id = ?))", ConflictStatusActive, "player", participant.id, "player", participant.id)
		if ignoreID != nil {
			query = query.Where("id <> ?", *ignoreID)
		}
		if err := query.Count(&playerConflictCount).Error; err != nil {
			return err
		}
		if playerConflictCount >= 3 {
			return fmt.Errorf("player %d already has too many active conflicts", participant.id)
		}
	}
	return nil
}

func validateEventRequirements(save *models.PlayerSave, payload datatypes.JSON) error {
	if len(payload) == 0 || strings.TrimSpace(string(payload)) == "{}" {
		return nil
	}
	var req EventRequirements
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid requirements JSON: %w", err)
	}
	minLevel := req.MinCityLevel
	if req.CityLevel > minLevel {
		minLevel = req.CityLevel
	}
	if minLevel > 0 && save.CityLevel < minLevel {
		return fmt.Errorf("event requires city level %d", minLevel)
	}
	if req.MinXP > 0 && save.XP < req.MinXP {
		return fmt.Errorf("event requires %d XP", req.MinXP)
	}
	if req.MinPopulation > 0 && save.Population < req.MinPopulation {
		return fmt.Errorf("event requires %d population", req.MinPopulation)
	}
	return nil
}

func (s *WorldGameService) validateGameEventTx(tx *gorm.DB, event *models.GameEvent, ignoreID *uint) error {
	if event.WorldID == 0 {
		return fmt.Errorf("event world is required")
	}
	if strings.TrimSpace(event.Title) == "" {
		return fmt.Errorf("event title is required")
	}
	if event.StartsAt.IsZero() {
		event.StartsAt = time.Now()
	}
	if event.EndsAt.IsZero() && event.DurationMinutes > 0 {
		event.EndsAt = event.StartsAt.Add(time.Duration(event.DurationMinutes) * time.Minute)
	}
	if event.EndsAt.IsZero() {
		return fmt.Errorf("event end date is required")
	}
	if event.EndsAt.Sub(event.StartsAt) < time.Hour {
		return fmt.Errorf("event duration must be at least 1 hour")
	}
	event.DurationMinutes = int(event.EndsAt.Sub(event.StartsAt).Minutes())
	if event.Status == "" {
		event.Status = EventStatusActive
	}
	if !validEventDifficulty(event.Difficulty) {
		return fmt.Errorf("invalid event difficulty")
	}
	if _, err := parseEventReward(event.RewardsJSON); err != nil {
		return err
	}
	if err := validateRequirementsPayload(event.RequirementsJSON); err != nil {
		return err
	}
	dayStart := time.Date(event.StartsAt.Year(), event.StartsAt.Month(), event.StartsAt.Day(), 0, 0, 0, 0, event.StartsAt.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	countQuery := tx.Model(&models.GameEvent{}).
		Where("world_id = ? AND status <> ? AND starts_at >= ? AND starts_at < ?", event.WorldID, EventStatusArchived, dayStart, dayEnd)
	if ignoreID != nil {
		countQuery = countQuery.Where("id <> ?", *ignoreID)
	}
	var count int64
	if err := countQuery.Count(&count).Error; err != nil {
		return err
	}
	if count >= 4 {
		return fmt.Errorf("maximum 4 events per world per day reached")
	}
	overlap := tx.Model(&models.GameEvent{}).
		Where("world_id = ? AND status <> ? AND starts_at < ? AND ends_at > ?", event.WorldID, EventStatusArchived, event.EndsAt, event.StartsAt)
	if ignoreID != nil {
		overlap = overlap.Where("id <> ?", *ignoreID)
	}
	overlap = sameEventScope(overlap, event)
	var overlaps int64
	if err := overlap.Count(&overlaps).Error; err != nil {
		return err
	}
	if overlaps > 0 {
		return fmt.Errorf("event overlaps an existing event in the same scope")
	}
	return nil
}

func validateRequirementsPayload(payload datatypes.JSON) error {
	if len(payload) == 0 || strings.TrimSpace(string(payload)) == "{}" {
		return nil
	}
	var req EventRequirements
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid requirements JSON: %w", err)
	}
	if req.MinCityLevel < 0 || req.CityLevel < 0 || req.MinXP < 0 || req.MinPopulation < 0 {
		return fmt.Errorf("invalid requirements JSON: negative requirements are not allowed")
	}
	return nil
}

func sameEventScope(query *gorm.DB, event *models.GameEvent) *gorm.DB {
	if event.ContinentID == nil {
		query = query.Where("continent_id IS NULL")
	} else {
		query = query.Where("continent_id = ?", *event.ContinentID)
	}
	if event.GuildID == nil {
		query = query.Where("guild_id IS NULL")
	} else {
		query = query.Where("guild_id = ?", *event.GuildID)
	}
	if event.PlayerID == nil {
		query = query.Where("player_id IS NULL")
	} else {
		query = query.Where("player_id = ?", *event.PlayerID)
	}
	return query
}

func validEventDifficulty(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "easy", "medium", "hard", "critical":
		return true
	default:
		return false
	}
}

func applyGameEventFields(event *models.GameEvent, fields map[string]any) {
	if value, ok := stringField(fields, "title"); ok {
		event.Title = value
	}
	if value, ok := stringField(fields, "description"); ok {
		event.Description = value
	}
	if value, ok := stringField(fields, "type"); ok {
		event.Type = value
	}
	if value, ok := stringField(fields, "difficulty"); ok {
		event.Difficulty = value
	}
	if value, ok := stringField(fields, "status"); ok {
		event.Status = value
	}
	if value, ok := timeField(fields, "startsAt", "starts_at"); ok {
		event.StartsAt = value
	}
	if value, ok := timeField(fields, "endsAt", "ends_at"); ok {
		event.EndsAt = value
	}
	if value, ok := intField(fields, "durationMinutes", "duration_minutes"); ok {
		event.DurationMinutes = value
	}
}

func normalizeGameEventUpdateFields(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		switch key {
		case "worldId":
			out["world_id"] = value
		case "continentId":
			out["continent_id"] = value
		case "guildId":
			out["guild_id"] = value
		case "playerId":
			out["player_id"] = value
		case "durationMinutes":
			out["duration_minutes"] = value
		case "rewardsJson":
			out["rewards_json"] = mustJSON(value)
		case "requirementsJson":
			out["requirements_json"] = mustJSON(value)
		case "consequencesJson":
			out["consequences_json"] = mustJSON(value)
		case "createdByAi":
			out["created_by_ai"] = value
		case "startsAt":
			out["starts_at"] = value
		case "endsAt":
			out["ends_at"] = value
		default:
			out[key] = value
		}
	}
	return out
}

func stringField(fields map[string]any, names ...string) (string, bool) {
	for _, name := range names {
		if value, ok := fields[name]; ok {
			return fmt.Sprint(value), true
		}
	}
	return "", false
}

func intField(fields map[string]any, names ...string) (int, bool) {
	for _, name := range names {
		if value, ok := fields[name]; ok {
			switch typed := value.(type) {
			case float64:
				return int(typed), true
			case int:
				return typed, true
			case string:
				parsed, err := strconv.Atoi(typed)
				return parsed, err == nil
			}
		}
	}
	return 0, false
}

func timeField(fields map[string]any, names ...string) (time.Time, bool) {
	for _, name := range names {
		if value, ok := fields[name]; ok {
			switch typed := value.(type) {
			case string:
				parsed, err := time.Parse(time.RFC3339, typed)
				return parsed, err == nil
			case time.Time:
				return typed, true
			}
		}
	}
	return time.Time{}, false
}

func int64FromAny(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	default:
		return 0
	}
}

func guildMemberForUpdate(tx *gorm.DB, guildID uint, playerID uint) (models.GuildMember, error) {
	var member models.GuildMember
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("guild_id = ? AND player_id = ?", guildID, playerID).
		First(&member).Error
	if err != nil {
		return member, fmt.Errorf("guild member not found")
	}
	return member, nil
}

func (s *WorldGameService) logPlayerAction(ctx context.Context, playerID uint, worldID *uint, continentID *uint, action string, targetType string, targetID string, status string, errorMessage string, before any, after any, metadata any) error {
	return s.logPlayerActionTx(s.db.WithContext(ctx), playerID, worldID, continentID, action, targetType, targetID, status, errorMessage, before, after, metadata)
}

func (s *WorldGameService) logPlayerActionTx(tx *gorm.DB, playerID uint, worldID *uint, continentID *uint, action string, targetType string, targetID string, status string, errorMessage string, before any, after any, metadata any) error {
	return tx.Create(&models.PlayerActionLog{
		PlayerID:     playerID,
		WorldID:      worldID,
		ContinentID:  continentID,
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Status:       status,
		Error:        errorMessage,
		BeforeJSON:   mustJSON(before),
		AfterJSON:    mustJSON(after),
		MetadataJSON: mustJSON(metadata),
	}).Error
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func containsBlockedChatText(message string) bool {
	normalized := strings.ToLower(message)
	blocked := []string{"http://", "https://", "<script", "discord.gg/"}
	for _, value := range blocked {
		if strings.Contains(normalized, value) {
			return true
		}
	}
	return false
}

func safeAssetSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "default"
	}
	return builder.String()
}

func allowedAssetExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	default:
		return false
	}
}
