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
	DecisionStatusDryRun   = "dry_run"
	DecisionStatusEnabled  = "enabled"
	DecisionStatusDisabled = "disabled"
	SimulationCycleManual  = "manual"
	SimulationCycleLight   = "light"
	SimulationCycleHourly  = "continental"
	SimulationCycleDaily   = "daily"
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

type SaveSyncConflictError struct {
	Message       string         `json:"message"`
	ServerVersion int            `json:"serverVersion"`
	ClientVersion int            `json:"clientVersion"`
	ServerSave    map[string]any `json:"serverSave"`
}

func (e *SaveSyncConflictError) Error() string {
	if e == nil {
		return "save conflict"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return fmt.Sprintf("save conflict: server version %d client version %d", e.ServerVersion, e.ClientVersion)
}

func AsSaveSyncConflict(err error) (*SaveSyncConflictError, bool) {
	if err == nil {
		return nil, false
	}
	var conflict *SaveSyncConflictError
	if errors.As(err, &conflict) {
		return conflict, true
	}
	return nil, false
}

type PlayerCityCreateInput struct {
	CityName       string `json:"cityName"`
	Continent      string `json:"continent"`
	Specialization string `json:"specialization"`
}

type PlayerActionSyncItem struct {
	IdempotencyKey string         `json:"idempotencyKey"`
	Type           string         `json:"type"`
	Payload        datatypes.JSON `json:"payload"`
}

type PlayerActionsSyncInput struct {
	ClientSaveVersion int                    `json:"clientSaveVersion"`
	Actions           []PlayerActionSyncItem `json:"actions"`
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
	Key             string                  `json:"key"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Category        string                  `json:"category"`
	ResearchTreeKey string                  `json:"researchTreeKey"`
	MaxLevel        int                     `json:"maxLevel"`
	Assets          []BuildingManifestAsset `json:"assets"`
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

type FourPillarActionCounts struct {
	ConflictAccepted  int `json:"conflictAccepted"`
	ConflictRejected  int `json:"conflictRejected"`
	DiplomacyAccepted int `json:"diplomacyAccepted"`
	DiplomacyRejected int `json:"diplomacyRejected"`
	CommerceAccepted  int `json:"commerceAccepted"`
	CommerceRejected  int `json:"commerceRejected"`
	WeatherAccepted   int `json:"weatherAccepted"`
	WeatherRejected   int `json:"weatherRejected"`
}

type FourPillarPressure struct {
	ActiveConflicts        int `json:"activeConflicts"`
	AverageConflictRisk    int `json:"averageConflictRisk"`
	ActiveWeather          int `json:"activeWeather"`
	AverageWeatherSeverity int `json:"averageWeatherSeverity"`
}

type FourPillarScore struct {
	Popularity     int                    `json:"popularity"`
	Stability      int                    `json:"stability"`
	Sustainability int                    `json:"sustainability"`
	ConflictScore  int                    `json:"conflictScore"`
	DiplomacyScore int                    `json:"diplomacyScore"`
	CommerceScore  int                    `json:"commerceScore"`
	WeatherScore   int                    `json:"weatherScore"`
	Inputs         map[string]any         `json:"inputs"`
	Actions        FourPillarActionCounts `json:"actions"`
	Pressure       FourPillarPressure     `json:"pressure"`
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

func (s *WorldGameService) ReconcileWorldPopulationCounts(ctx context.Context) (map[string]any, error) {
	type countRow struct {
		ID    uint
		Total int
	}
	worldRows := []countRow{}
	continentRows := []countRow{}
	if err := s.db.WithContext(ctx).Model(&models.PlayerSave{}).
		Select("world_id as id, COUNT(*) as total").
		Group("world_id").
		Scan(&worldRows).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&models.PlayerSave{}).
		Select("continent_id as id, COUNT(*) as total").
		Group("continent_id").
		Scan(&continentRows).Error; err != nil {
		return nil, err
	}
	worldCounts := map[uint]int{}
	for _, row := range worldRows {
		worldCounts[row.ID] = row.Total
	}
	continentCounts := map[uint]int{}
	for _, row := range continentRows {
		continentCounts[row.ID] = row.Total
	}
	updatedWorlds := int64(0)
	updatedContinents := int64(0)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var worlds []models.World
		if err := tx.Find(&worlds).Error; err != nil {
			return err
		}
		for _, world := range worlds {
			count := worldCounts[world.Id]
			if world.CurrentPlayers != count {
				if err := tx.Model(&models.World{}).Where("id = ?", world.Id).Update("current_players", count).Error; err != nil {
					return err
				}
				updatedWorlds++
			}
		}
		var continents []models.Continent
		if err := tx.Find(&continents).Error; err != nil {
			return err
		}
		for _, continent := range continents {
			count := continentCounts[continent.Id]
			if continent.CurrentPlayers != count {
				if err := tx.Model(&models.Continent{}).Where("id = ?", continent.Id).Update("current_players", count).Error; err != nil {
					return err
				}
				updatedContinents++
			}
		}
		return nil
	})
	return map[string]any{"updatedWorlds": updatedWorlds, "updatedContinents": updatedContinents}, err
}

func (s *WorldGameService) ArchiveEmptyWorlds(ctx context.Context) (map[string]any, error) {
	if _, err := s.ReconcileWorldPopulationCounts(ctx); err != nil {
		return nil, err
	}
	var keep models.World
	keepErr := s.db.WithContext(ctx).
		Where("status = ? AND current_players = 0", WorldStatusActive).
		Order("id ASC").
		First(&keep).Error
	query := s.db.WithContext(ctx).Model(&models.World{}).
		Where("status = ? AND current_players = 0", WorldStatusActive)
	if keepErr == nil {
		query = query.Where("id <> ?", keep.Id)
	}
	result := query.Update("status", WorldStatusArchived)
	if result.Error != nil {
		return nil, result.Error
	}
	return map[string]any{"archivedWorlds": result.RowsAffected, "keptEmptyWorldId": keep.Id}, nil
}

func (s *WorldGameService) EnsurePlayerSave(ctx context.Context, playerID uint) (*models.PlayerSave, error) {
	if err := s.ensurePlayerUserRecord(ctx, playerID); err != nil {
		return nil, err
	}

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

func (s *WorldGameService) ensurePlayerUserRecord(ctx context.Context, playerID uint) error {
	if playerID == 0 {
		return fmt.Errorf("invalid player id")
	}

	var user models.Users
	err := s.db.WithContext(ctx).Unscoped().First(&user, playerID).Error
	if err == nil {
		if user.DeletedAt.Valid {
			if restoreErr := s.db.WithContext(ctx).
				Model(&models.Users{}).
				Unscoped().
				Where("id = ?", playerID).
				Update("deleted_at", nil).Error; restoreErr != nil {
				return restoreErr
			}
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	placeholder := models.Users{
		Id:       playerID,
		Pseudo:   fmt.Sprintf("Player %d", playerID),
		Password: "purged-account",
		Email:    fmt.Sprintf("player-%d@local.invalid", playerID),
		Avatar:   "",
		Xp:       0,
		Coin:     0,
	}
	return s.db.WithContext(ctx).Create(&placeholder).Error
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
	save, err = s.reconcilePopulationState(ctx, playerID)
	if err != nil {
		return nil, err
	}
	var player models.Users
	if err := s.db.WithContext(ctx).First(&player, playerID).Error; err != nil {
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
		"pseudo": player.Pseudo,
		"player": map[string]any{
			"id":     strconv.FormatUint(uint64(player.Id), 10),
			"pseudo": player.Pseudo,
			"guild":  nil,
		},
		"save":      save,
		"buildings": buildings,
		"events":    events,
		"conflicts": conflicts,
		"weather":   weather,
		"messages":  messages,
	}, nil
}

func (s *WorldGameService) reconcilePopulationState(ctx context.Context, playerID uint) (*models.PlayerSave, error) {
	var out models.PlayerSave
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("player_id = ?", playerID).
			First(&save).Error; err != nil {
			return err
		}

		housingCapacity := int64(0)
		buildings := []map[string]any{}
		if len(save.BuildingsJSON) > 0 {
			_ = json.Unmarshal(save.BuildingsJSON, &buildings)
		}
		for _, item := range buildings {
			key := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["buildingKey"])))
			if !isHousingBuildingKey(key) {
				continue
			}
			level := int64(intFromAny(item["level"]))
			if level < 0 {
				level = 0
			}
			housingCapacity += level * 50
		}

		var activeConflicts int64
		if err := tx.Model(&models.Conflict{}).
			Where("world_id = ? AND status = ? AND (continent_id IS NULL OR continent_id = ?)", save.WorldID, ConflictStatusActive, save.ContinentID).
			Count(&activeConflicts).Error; err != nil {
			return err
		}

		var weatherEvents []models.WeatherEvent
		if err := tx.Where("world_id = ? AND continent_id = ? AND starts_at <= ? AND ends_at >= ?", save.WorldID, save.ContinentID, time.Now(), time.Now()).
			Find(&weatherEvents).Error; err != nil {
			return err
		}
		weatherPressure := int64(0)
		for _, event := range weatherEvents {
			weatherPressure += int64(clamp(event.Severity, 0, 100))
		}
		if len(weatherEvents) > 0 {
			weatherPressure = weatherPressure / int64(len(weatherEvents))
		}

		currentPop := save.Population
		targetPop := housingCapacity
		if targetPop < 0 {
			targetPop = 0
		}

		maxDelta := int64(25)
		if currentPop > 0 {
			delta := int64(float64(currentPop) * 0.03)
			if delta > maxDelta {
				maxDelta = delta
			}
		}

		foodEnough := save.Food >= maxInt64(100, currentPop/4)
		energyEnough := save.Energy >= maxInt64(100, currentPop/4)
		stableEnough := save.Satisfaction >= 35

		nextPop := currentPop
		if targetPop == 0 {
			nextPop = 0
		} else if currentPop < targetPop && foodEnough && energyEnough && stableEnough {
			grow := targetPop - currentPop
			if grow > maxDelta {
				grow = maxDelta
			}
			nextPop = currentPop + grow
		} else if currentPop > targetPop {
			shrink := currentPop - targetPop
			maxShrink := maxDelta * 2
			if shrink > maxShrink {
				shrink = maxShrink
			}
			nextPop = currentPop - shrink
		}

		crisisPressure := int64(clamp(int(activeConflicts*20+weatherPressure), 0, 100))
		if crisisPressure > 0 && nextPop > 0 {
			loss := (nextPop * crisisPressure) / 500
			if loss < 1 {
				loss = 1
			}
			nextPop -= loss
		}
		if nextPop < 0 {
			nextPop = 0
		}
		if nextPop > targetPop {
			nextPop = targetPop
		}

		nextSatisfaction := save.Satisfaction
		if crisisPressure >= 60 {
			nextSatisfaction = clamp(nextSatisfaction-2, 0, 100)
		} else if crisisPressure <= 15 && foodEnough && energyEnough {
			nextSatisfaction = clamp(nextSatisfaction+1, 0, 100)
		}

		if nextPop != save.Population || nextSatisfaction != save.Satisfaction {
			now := time.Now().UTC()
			if err := tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(map[string]any{
				"population":     nextPop,
				"satisfaction":   nextSatisfaction,
				"version":        save.Version + 1,
				"last_synced_at": &now,
			}).Error; err != nil {
				return err
			}
		}

		refreshed, err := s.GetPlayerSave(ctx, playerID)
		if err != nil {
			return err
		}
		out = *refreshed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func isHousingBuildingKey(key string) bool {
	normalized := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(key)), "-", "_")
	switch normalized {
	case "habitation", "housing", "house", "residence", "residential":
		return true
	default:
		return false
	}
}

func (s *WorldGameService) CreatePlayerCity(ctx context.Context, playerID uint, input PlayerCityCreateInput) (map[string]any, error) {
	cityName := strings.TrimSpace(input.CityName)
	if cityName == "" {
		cityName = fmt.Sprintf("Ville %d", playerID)
	}
	if len([]rune(cityName)) > 120 {
		return nil, fmt.Errorf("city name too long")
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		base, err := s.EnsurePlayerSave(ctx, playerID)
		if err != nil {
			return err
		}
		var save models.PlayerSave
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", base.Id).First(&save).Error; err != nil {
			return err
		}
		defaultCityName := fmt.Sprintf("Ville %d", playerID)
		if strings.TrimSpace(save.CityName) != "" && save.CityName != defaultCityName {
			return fmt.Errorf("player already has an active city")
		}

		starterBuildings := []map[string]any{
			{"buildingKey": "habitation", "level": 1, "positionX": 1, "positionY": 1, "state": "placed", "lastCollectedAt": now},
			{"buildingKey": "solar_park", "level": 1, "positionX": 2, "positionY": 1, "state": "placed", "lastCollectedAt": now},
		}
		buildingsJSON, _ := json.Marshal(starterBuildings)

		activeEffects := map[string]any{}
		if len(save.ActiveEffectsJSON) > 0 {
			_ = json.Unmarshal(save.ActiveEffectsJSON, &activeEffects)
		}
		if activeEffects == nil {
			activeEffects = map[string]any{}
		}
		if strings.TrimSpace(input.Specialization) != "" {
			activeEffects["specialization"] = strings.TrimSpace(input.Specialization)
		}
		if strings.TrimSpace(input.Continent) != "" {
			activeEffects["continent"] = strings.TrimSpace(input.Continent)
		}
		activeEffectsJSON, _ := json.Marshal(activeEffects)

		updates := map[string]any{
			"city_name":               cityName,
			"buildings_json":          datatypes.JSON(buildingsJSON),
			"construction_queue_json": emptyJSONObject,
			"active_effects_json":     datatypes.JSON(activeEffectsJSON),
			"version":                 save.Version + 1,
			"last_synced_at":          &now,
		}
		if err := tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(updates).Error; err != nil {
			return err
		}

		if err := tx.Where("player_id = ?", playerID).Delete(&models.PlayerBuilding{}).Error; err != nil {
			return err
		}
		rows := []models.PlayerBuilding{
			{PlayerID: playerID, BuildingKey: "habitation", Level: 1, PositionX: 1, PositionY: 1, State: "placed", LastCollectedAt: &now},
			{PlayerID: playerID, BuildingKey: "solar_park", Level: 1, PositionX: 2, PositionY: 1, State: "placed", LastCollectedAt: &now},
		}
		for _, row := range rows {
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.PlayerState(ctx, playerID)
}

func (s *WorldGameService) SyncPlayerActions(ctx context.Context, playerID uint, input PlayerActionsSyncInput) (map[string]any, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if input.ClientSaveVersion > 0 && input.ClientSaveVersion != save.Version {
		return nil, &SaveSyncConflictError{
			Message:       "Save version conflict",
			ServerVersion: save.Version,
			ClientVersion: input.ClientSaveVersion,
			ServerSave: map[string]any{
				"id":        save.Id,
				"playerId":  save.PlayerID,
				"version":   save.Version,
				"cityName":  save.CityName,
				"cityLevel": save.CityLevel,
			},
		}
	}

	accepted := make([]map[string]any, 0, len(input.Actions))
	rejected := make([]map[string]any, 0)

	for _, action := range input.Actions {
		idempotencyKey := strings.TrimSpace(action.IdempotencyKey)
		actionType := strings.TrimSpace(action.Type)
		if idempotencyKey == "" || actionType == "" {
			rejected = append(rejected, map[string]any{
				"idempotencyKey": idempotencyKey,
				"type":           actionType,
				"reason":         "missing idempotencyKey or type",
			})
			continue
		}

		var existing models.PlayerActionLog
		err := s.db.WithContext(ctx).
			Where("player_id = ? AND action = ? AND target_id = ? AND status = ?", playerID, "client_action_sync", idempotencyKey, "accepted").
			Order("id DESC").
			First(&existing).Error
		if err == nil {
			accepted = append(accepted, map[string]any{
				"idempotencyKey": idempotencyKey,
				"type":           actionType,
				"idempotent":     true,
				"status":         "accepted",
			})
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if actionType == "player.city.create" {
			payload := map[string]any{}
			if len(action.Payload) > 0 {
				_ = json.Unmarshal(action.Payload, &payload)
			}
			_, err := s.CreatePlayerCity(ctx, playerID, PlayerCityCreateInput{
				CityName:       strings.TrimSpace(fmt.Sprint(payload["cityName"])),
				Continent:      strings.TrimSpace(fmt.Sprint(payload["continent"])),
				Specialization: strings.TrimSpace(fmt.Sprint(payload["specialization"])),
			})
			if err != nil {
				rejected = append(rejected, map[string]any{
					"idempotencyKey": idempotencyKey,
					"type":           actionType,
					"reason":         err.Error(),
				})
				continue
			}
		} else if !isSupportedSyncedAction(actionType) {
			rejected = append(rejected, map[string]any{
				"idempotencyKey": idempotencyKey,
				"type":           actionType,
				"reason":         "unsupported action type",
			})
			continue
		}

		if err := s.logPlayerAction(ctx, playerID, &save.WorldID, &save.ContinentID, "client_action_sync", actionType, idempotencyKey, "accepted", "", nil, nil, map[string]any{
			"type": actionType,
		}); err != nil {
			return nil, err
		}
		accepted = append(accepted, map[string]any{
			"idempotencyKey": idempotencyKey,
			"type":           actionType,
			"idempotent":     false,
			"status":         "accepted",
		})
	}

	updatedSave, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"accepted":      accepted,
		"rejected":      rejected,
		"save":          updatedSave,
		"serverVersion": updatedSave.Version,
	}, nil
}

func isSupportedSyncedAction(actionType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(actionType))
	return strings.HasPrefix(normalized, "construction.") || strings.HasPrefix(normalized, "diplomacy.") || strings.HasPrefix(normalized, "commerce.") || strings.HasPrefix(normalized, "weather.")
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
		if input.Version != locked.Version {
			return &SaveSyncConflictError{
				Message:       "Save version conflict",
				ServerVersion: locked.Version,
				ClientVersion: input.Version,
				ServerSave: map[string]any{
					"id":                    locked.Id,
					"playerId":              locked.PlayerID,
					"worldId":               locked.WorldID,
					"continentId":           locked.ContinentID,
					"cityName":              locked.CityName,
					"cityLevel":             locked.CityLevel,
					"xp":                    locked.XP,
					"population":            locked.Population,
					"satisfaction":          locked.Satisfaction,
					"food":                  locked.Food,
					"energy":                locked.Energy,
					"credits":               locked.Credits,
					"gems":                  locked.Gems,
					"buildingsJson":         locked.BuildingsJSON,
					"constructionQueueJson": locked.ConstructionQueueJSON,
					"researchJson":          locked.ResearchJSON,
					"inventoryJson":         locked.InventoryJSON,
					"activeEffectsJson":     locked.ActiveEffectsJSON,
					"version":               locked.Version,
					"lastSyncedAt":          locked.LastSyncedAt,
				},
			}
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
	if err != nil {
		return nil, err
	}

	now := time.Now()
	active := make([]models.GameEvent, 0, len(events))
	inactive := make([]models.GameEvent, 0, len(events))
	for _, event := range events {
		if event.EndsAt.IsZero() {
			continue
		}
		if event.EndsAt.After(now) {
			active = append(active, event)
			continue
		}
		inactive = append(inactive, event)
	}

	return append(active, inactive...), nil
}

func (s *WorldGameService) ListWorldConflicts(ctx context.Context, worldID uint, continentID uint, limit int) ([]models.Conflict, error) {
	var conflicts []models.Conflict
	err := s.db.WithContext(ctx).
		Where("world_id = ? AND status = ? AND (continent_id IS NULL OR continent_id = ?)", worldID, ConflictStatusActive, continentID).
		Order("intensity DESC, starts_at DESC").
		Limit(limitOrDefault(limit)).
		Find(&conflicts).Error
	if err != nil {
		return nil, err
	}

	now := time.Now()
	active := make([]models.Conflict, 0, len(conflicts))
	inactive := make([]models.Conflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		if conflict.EndsAt.IsZero() {
			continue
		}
		if conflict.EndsAt.After(now) {
			active = append(active, conflict)
			continue
		}
		inactive = append(inactive, conflict)
	}

	return append(active, inactive...), nil
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
			Key:             building.Key,
			Name:            building.Name,
			Description:     building.Description,
			Category:        building.Category,
			ResearchTreeKey: building.ResearchTreeKey,
			MaxLevel:        building.MaxLevel,
			Assets:          []BuildingManifestAsset{},
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

func (s *WorldGameService) GenerateWorldFourPageRoutine(ctx context.Context, worldID uint, forcedBy string) (*models.WorldRoutineSnapshot, *models.AIWorldDecision, error) {
	world, err := s.loadWorld(ctx, worldID)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	snapshot, metrics, err := s.buildFourPageRoutineSnapshot(ctx, *world, now)
	if err != nil {
		return nil, nil, err
	}
	aiOutput, providerName, modelName, status, callErr := s.callNEXUSRoutine(ctx, *world, snapshot)
	if callErr != nil && status == "" {
		status = DecisionStatusFallback
	}
	if status == "" {
		status = DecisionStatusApplied
	}
	if callErr != nil {
		snapshot["aiError"] = callErr.Error()
	}
	inputJSON := mustJSON(map[string]any{
		"routine":  "WORLD_IA_ROUTINE_4_PAGES",
		"worldId":  world.Id,
		"forcedBy": forcedBy,
		"snapshot": snapshot,
	})
	outputJSON := mustJSON(aiOutput)
	metricsJSON := mustJSON(metrics)
	applied := map[string]any{
		"routine":       "WORLD_IA_ROUTINE_4_PAGES",
		"worldId":       world.Id,
		"metricsCount":  len(metrics),
		"generatedAt":   now,
		"conflictCount": len(asSliceMap(snapshot["conflicts"])),
		"weatherCount":  len(asSliceMap(snapshot["weather"])),
	}
	var routine models.WorldRoutineSnapshot
	var decision models.AIWorldDecision
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		decision = models.AIWorldDecision{
			WorldID:            world.Id,
			Type:               "routine_4_pages",
			InputSnapshotJSON:  inputJSON,
			OutputDecisionJSON: outputJSON,
			AppliedChangesJSON: mustJSON(applied),
			Provider:           providerName,
			Model:              modelName,
			Status:             status,
			IsActive:           true,
		}
		if callErr != nil {
			decision.Error = callErr.Error()
		}
		if err := tx.Create(&decision).Error; err != nil {
			return err
		}
		routine = models.WorldRoutineSnapshot{
			WorldID:        world.Id,
			RoutineVersion: "WORLD_IA_ROUTINE_4_PAGES",
			SnapshotJSON:   mustJSON(snapshot),
			AIOutputJSON:   outputJSON,
			MetricsJSON:    metricsJSON,
			Provider:       providerName,
			Model:          modelName,
			Status:         status,
		}
		if callErr != nil {
			routine.Error = callErr.Error()
		}
		return tx.Create(&routine).Error
	})
	return &routine, &decision, err
}

func (s *WorldGameService) LatestWorldFourPageRoutine(ctx context.Context, worldID uint) (*models.WorldRoutineSnapshot, error) {
	var routine models.WorldRoutineSnapshot
	err := s.db.WithContext(ctx).
		Where("world_id = ? AND routine_version = ?", worldID, "WORLD_IA_ROUTINE_4_PAGES").
		Order("created_at DESC").
		First(&routine).Error
	return &routine, err
}

func (s *WorldGameService) PlayerFourPillarMetrics(ctx context.Context, playerID uint) (*models.PlayerWorldMetric, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if err := s.CalculateAndStorePlayerWorldMetric(ctx, *save); err != nil {
		return nil, err
	}
	var metric models.PlayerWorldMetric
	err = s.db.WithContext(ctx).Where("player_id = ? AND world_id = ?", save.PlayerID, save.WorldID).First(&metric).Error
	return &metric, err
}

func (s *WorldGameService) CalculateAndStorePlayerWorldMetric(ctx context.Context, save models.PlayerSave) error {
	counts, err := s.playerFourPillarActionCounts(ctx, save.PlayerID, save.WorldID, time.Now().Add(-14*24*time.Hour))
	if err != nil {
		return err
	}
	pressure, err := s.playerFourPillarPressure(ctx, save.WorldID, save.ContinentID)
	if err != nil {
		return err
	}
	score := scorePlayerFourPillars(save, counts, pressure)
	now := time.Now().UTC()
	metric := models.PlayerWorldMetric{
		PlayerID:       save.PlayerID,
		WorldID:        save.WorldID,
		ContinentID:    save.ContinentID,
		Popularity:     score.Popularity,
		Stability:      score.Stability,
		Sustainability: score.Sustainability,
		ConflictScore:  score.ConflictScore,
		DiplomacyScore: score.DiplomacyScore,
		CommerceScore:  score.CommerceScore,
		WeatherScore:   score.WeatherScore,
		InputJSON:      mustJSON(score),
		GeneratedAt:    now,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "player_id"}, {Name: "world_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"continent_id",
			"popularity",
			"stability",
			"sustainability",
			"conflict_score",
			"diplomacy_score",
			"commerce_score",
			"weather_score",
			"input_json",
			"generated_at",
			"updated_at",
		}),
	}).Create(&metric).Error
}

func (s *WorldGameService) SimulateWorld(ctx context.Context, worldID uint, forcedBy string) (*models.AIWorldDecision, error) {
	return s.SimulateWorldCycle(ctx, worldID, forcedBy, SimulationCycleManual)
}

func (s *WorldGameService) SimulateWorldCycle(ctx context.Context, worldID uint, forcedBy string, cycleType string) (*models.AIWorldDecision, error) {
	cycleType = normalizeSimulationCycle(cycleType)
	var world models.World
	query := s.db.WithContext(ctx).Preload("Continents", func(db *gorm.DB) *gorm.DB {
		return db.Order("`index` ASC")
	})
	if worldID == 0 {
		query = query.Where("status = ?", WorldStatusActive).Order("id ASC")
	}
	var err error
	if worldID == 0 {
		err = query.First(&world).Error
	} else {
		err = query.First(&world, worldID).Error
	}
	if err != nil {
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
		"world":     world,
		"cycleType": cycleType,
		"forcedBy":  forcedBy,
		"now":       time.Now().Format(time.RFC3339),
	}
	inputJSON := mustJSON(snapshot)
	decision, providerName, modelName, status, callErr := s.callNEXUS(ctx, world)
	outputJSON := mustJSON(decision)
	now := time.Now()
	applied := map[string]any{}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(world.Continents) == 0 {
			return fmt.Errorf("world has no continents")
		}
		continent := pickSimulationContinent(world, cycleType)
		createEvents := cycleType == SimulationCycleManual || cycleType == SimulationCycleHourly || cycleType == SimulationCycleDaily
		createWeather := cycleType == SimulationCycleManual || cycleType == SimulationCycleHourly || cycleType == SimulationCycleDaily
		createConflicts := cycleType == SimulationCycleManual || cycleType == SimulationCycleHourly || cycleType == SimulationCycleDaily
		sendMessages := cycleType == SimulationCycleManual || cycleType == SimulationCycleDaily
		if createEvents && len(decision.Events) > 0 {
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
		if createWeather && len(decision.Weather) > 0 {
			weather := decision.Weather[0]
			starts := now
			ends := starts.Add(3 * time.Hour)
			effects := mustJSON(defaultMap(weather.Effects, defaultWeatherEffects(continent.AIBehaviorProfile)))
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
		if createConflicts && len(decision.Conflicts) > 0 {
			conflict := decision.Conflicts[0]
			starts := now
			ends := starts.Add(6 * time.Hour)
			faction, err := s.ensureAIWorldFactionTx(ctx, tx, world.Id, &continent.Id, "Novus")
			if err != nil {
				return err
			}
			conflictInput := ConflictInput{
				WorldID:       world.Id,
				ContinentID:   &continent.Id,
				AttackerType:  "ai_faction",
				AttackerID:    faction.Id,
				DefenderType:  "continent",
				DefenderID:    continent.Id,
				Title:         defaultText(conflict.Title, fmt.Sprintf("%s -> %s", faction.Name, continent.Name)),
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
				AttackerID:    conflictInput.AttackerID,
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
		if sendMessages && strings.TrimSpace(decision.Message.Message) != "" {
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
		tensionDelta, weatherDelta := simulationRiskDeltas(continent.AIBehaviorProfile, cycleType)
		updates := map[string]any{
			"current_cycle":        world.CurrentCycle + 1,
			"global_tension_level": clamp(world.GlobalTensionLevel+tensionDelta, 0, 100),
			"global_weather_risk":  clamp(world.GlobalWeatherRisk+weatherDelta, 0, 100),
			"last_simulation_at":   &now,
			"global_economic_state": defaultText(world.GlobalEconomicState, profileEconomicState(
				continent.AIBehaviorProfile,
				world.GlobalEconomicState,
			)),
		}
		if sendMessages {
			updates["last_daily_message_at"] = &now
		}
		return tx.Model(&models.World{}).Where("id = ?", world.Id).Updates(updates).Error
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
		ContinentID:        decisionContinentID(world, cycleType),
		Type:               "simulation_" + cycleType,
		InputSnapshotJSON:  inputJSON,
		OutputDecisionJSON: outputJSON,
		AppliedChangesJSON: mustJSON(applied),
		Provider:           providerName,
		Model:              modelName,
		Status:             status,
		IsActive:           true,
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

func (s *WorldGameService) DryRunWorldSimulation(ctx context.Context, worldID uint, cycleType string) (*models.AIWorldDecision, error) {
	cycleType = normalizeSimulationCycle(cycleType)
	var world models.World
	query := s.db.WithContext(ctx).Preload("Continents", func(db *gorm.DB) *gorm.DB {
		return db.Order("`index` ASC")
	})
	if worldID == 0 {
		query = query.Where("status = ?", WorldStatusActive).Order("id ASC")
	}
	if worldID == 0 {
		err := query.First(&world).Error
		if err != nil {
			return nil, err
		}
	} else if err := query.First(&world, worldID).Error; err != nil {
		return nil, err
	}
	decision, providerName, modelName, status, callErr := s.callNEXUS(ctx, world)
	continent := pickSimulationContinent(world, cycleType)
	tensionDelta, weatherDelta := simulationRiskDeltas(continent.AIBehaviorProfile, cycleType)
	applied := map[string]any{
		"dryRun":        true,
		"cycleType":     cycleType,
		"targetWorld":   world.Id,
		"targetProfile": continent.AIBehaviorProfile,
		"targetContinent": map[string]any{
			"id":    continent.Id,
			"name":  continent.Name,
			"index": continent.Index,
		},
		"wouldCreateEvents":    cycleType != SimulationCycleLight && len(decision.Events) > 0,
		"wouldCreateWeather":   cycleType != SimulationCycleLight && len(decision.Weather) > 0,
		"wouldCreateConflicts": cycleType != SimulationCycleLight && len(decision.Conflicts) > 0,
		"wouldSendMessages":    (cycleType == SimulationCycleManual || cycleType == SimulationCycleDaily) && strings.TrimSpace(decision.Message.Message) != "",
		"tensionDelta":         tensionDelta,
		"weatherDelta":         weatherDelta,
	}
	row := &models.AIWorldDecision{
		WorldID:            world.Id,
		ContinentID:        &continent.Id,
		Type:               "simulation_" + cycleType,
		InputSnapshotJSON:  mustJSON(map[string]any{"world": world, "cycleType": cycleType, "dryRun": true}),
		OutputDecisionJSON: mustJSON(decision),
		AppliedChangesJSON: mustJSON(applied),
		Provider:           providerName,
		Model:              modelName,
		Status:             DecisionStatusDryRun,
		IsActive:           true,
	}
	if callErr != nil {
		row.Error = callErr.Error()
	} else if status != "" && status != DecisionStatusApplied {
		row.Error = status
	}
	return row, nil
}

func (s *WorldGameService) loadWorld(ctx context.Context, worldID uint) (*models.World, error) {
	var world models.World
	query := s.db.WithContext(ctx).Preload("Continents", func(db *gorm.DB) *gorm.DB {
		return db.Order("`index` ASC")
	})
	if worldID == 0 {
		query = query.Where("status = ?", WorldStatusActive).Order("id ASC")
		if err := query.First(&world).Error; err != nil {
			return nil, err
		}
		return &world, nil
	}
	if err := query.First(&world, worldID).Error; err != nil {
		return nil, err
	}
	return &world, nil
}

func (s *WorldGameService) buildFourPageRoutineSnapshot(ctx context.Context, world models.World, now time.Time) (map[string]any, []models.PlayerWorldMetric, error) {
	var conflicts []models.Conflict
	if err := s.db.WithContext(ctx).
		Where("world_id = ? AND status = ?", world.Id, ConflictStatusActive).
		Order("intensity DESC, ends_at ASC").
		Limit(100).
		Find(&conflicts).Error; err != nil {
		return nil, nil, err
	}
	var weather []models.WeatherEvent
	if err := s.db.WithContext(ctx).
		Where("world_id = ? AND ends_at >= ?", world.Id, now).
		Order("severity DESC, ends_at ASC").
		Limit(100).
		Find(&weather).Error; err != nil {
		return nil, nil, err
	}
	var events []models.GameEvent
	if err := s.db.WithContext(ctx).
		Where("world_id = ? AND status = ?", world.Id, EventStatusActive).
		Order("starts_at ASC").
		Limit(100).
		Find(&events).Error; err != nil {
		return nil, nil, err
	}
	var saves []models.PlayerSave
	if err := s.db.WithContext(ctx).
		Where("world_id = ?", world.Id).
		Order("updated_at DESC").
		Limit(1000).
		Find(&saves).Error; err != nil {
		return nil, nil, err
	}
	metrics := make([]models.PlayerWorldMetric, 0, len(saves))
	for _, save := range saves {
		if err := s.CalculateAndStorePlayerWorldMetric(ctx, save); err != nil {
			return nil, nil, err
		}
		var metric models.PlayerWorldMetric
		if err := s.db.WithContext(ctx).Where("player_id = ? AND world_id = ?", save.PlayerID, save.WorldID).First(&metric).Error; err == nil {
			metrics = append(metrics, metric)
		}
	}
	continents := make([]map[string]any, 0, len(world.Continents))
	for _, continent := range world.Continents {
		continents = append(continents, map[string]any{
			"id":                continent.Id,
			"name":              defaultText(continent.Name, "Non disponible"),
			"index":             continent.Index,
			"players":           continent.CurrentPlayers,
			"tension":           clamp(continent.TensionLevel, 0, 100),
			"climateState":      defaultText(continent.ClimateState, "Non disponible"),
			"politicalState":    defaultText(continent.PoliticalState, "Non disponible"),
			"economicState":     defaultText(continent.EconomicState, "Non disponible"),
			"aiBehaviorProfile": defaultText(continent.AIBehaviorProfile, "Non disponible"),
		})
	}
	return map[string]any{
		"generatedAt": now,
		"world": map[string]any{
			"id":                 world.Id,
			"name":               defaultText(world.Name, "Non disponible"),
			"status":             defaultText(world.Status, "Non disponible"),
			"currentPlayers":     world.CurrentPlayers,
			"maxPlayers":         world.MaxPlayers,
			"currentCycle":       world.CurrentCycle,
			"globalTensionLevel": clamp(world.GlobalTensionLevel, 0, 100),
			"globalWeatherRisk":  clamp(world.GlobalWeatherRisk, 0, 100),
			"globalEconomicState": defaultText(
				world.GlobalEconomicState,
				"Non disponible",
			),
		},
		"continents":      continents,
		"conflicts":       s.routineConflictItems(ctx, conflicts),
		"diplomacy":       s.routineDiplomacy(world),
		"commerce":        s.routineCommerce(ctx, conflicts),
		"weather":         s.routineWeather(ctx, weather),
		"events":          routineEventItems(events),
		"playerMetrics":   routineMetricItems(metrics),
		"metricsSummary":  routineMetricSummary(metrics),
		"qualityControls": routineQualityControls(conflicts, weather, metrics),
	}, metrics, nil
}

func (s *WorldGameService) routineConflictItems(ctx context.Context, conflicts []models.Conflict) []map[string]any {
	items := make([]map[string]any, 0, len(conflicts))
	for _, conflict := range conflicts {
		items = append(items, map[string]any{
			"id":           strconv.FormatUint(uint64(conflict.Id), 10),
			"title":        defaultText(conflict.Title, "Non disponible"),
			"description":  defaultText(conflict.Description, "Non disponible"),
			"attackerName": s.conflictEntityName(ctx, conflict.AttackerType, conflict.AttackerID),
			"defenderName": s.conflictEntityName(ctx, conflict.DefenderType, conflict.DefenderID),
			"intensity":    clamp(conflict.Intensity, 0, 100),
			"riskLevel":    normalizeWorldRiskLevel(conflict.RiskLevel, conflict.Intensity),
			"status":       defaultText(conflict.Status, "Non disponible"),
			"startsAt":     conflict.StartsAt,
			"endsAt":       conflict.EndsAt,
		})
	}
	return items
}

func (s *WorldGameService) routineDiplomacy(world models.World) map[string]any {
	relations := make([]map[string]any, 0, len(world.Continents))
	allies := 0
	hostiles := 0
	for _, continent := range world.Continents {
		score := clamp(100-continent.TensionLevel, 0, 100)
		stance := diplomacyStanceFromScore(score)
		if stance == "allie" {
			allies++
		}
		if stance == "hostile" {
			hostiles++
		}
		relations = append(relations, map[string]any{
			"id":             strconv.FormatUint(uint64(continent.Id), 10),
			"faction":        defaultText(continent.Name, "Non disponible"),
			"continentId":    continent.Id,
			"score":          score,
			"stance":         stance,
			"politicalState": defaultText(continent.PoliticalState, "Non disponible"),
		})
	}
	return map[string]any{
		"relations": relations,
		"summary": map[string]any{
			"allies":       allies,
			"hostiles":     hostiles,
			"total":        len(relations),
			"emissaries":   map[string]any{"available": 0, "total": 0, "cooldownSeconds": 0},
			"dominancePct": averageRelationScore(relations),
		},
	}
}

func (s *WorldGameService) routineCommerce(ctx context.Context, conflicts []models.Conflict) map[string]any {
	routes := make([]map[string]any, 0, len(conflicts))
	var total int64
	active := 0
	for _, conflict := range conflicts {
		origin := s.conflictEntityName(ctx, conflict.AttackerType, conflict.AttackerID)
		destination := s.conflictEntityName(ctx, conflict.DefenderType, conflict.DefenderID)
		route := "Non disponible"
		if origin != "Non disponible" && destination != "Non disponible" {
			route = origin + " -> " + destination
		}
		volume := int64(1000000 * (100 - clamp(conflict.Intensity, 0, 100)) / 100)
		status := "active"
		if conflict.Intensity >= 70 {
			status = "perturbe"
		} else {
			active++
		}
		total += volume
		routes = append(routes, map[string]any{
			"id":         strconv.FormatUint(uint64(conflict.Id), 10),
			"route":      route,
			"cargo":      "Flux inter-regions",
			"volume":     volume,
			"efficiency": clamp(100-conflict.Intensity, 0, 100),
			"status":     status,
		})
	}
	return map[string]any{
		"routes": routes,
		"summary": map[string]any{
			"totalVolume":  total,
			"routesCount":  len(routes),
			"activeRoutes": active,
		},
	}
}

func (s *WorldGameService) routineWeather(ctx context.Context, events []models.WeatherEvent) map[string]any {
	items := make([]map[string]any, 0, len(events))
	total := 0
	for _, event := range events {
		severity := clamp(event.Severity, 0, 100)
		total += severity
		items = append(items, map[string]any{
			"id":          strconv.FormatUint(uint64(event.Id), 10),
			"type":        defaultText(event.Type, "Non disponible"),
			"title":       defaultText(event.Title, "Non disponible"),
			"description": defaultText(event.Description, "Non disponible"),
			"severity":    severity,
			"region":      s.continentName(ctx, event.ContinentID),
			"startsAt":    event.StartsAt,
			"endsAt":      event.EndsAt,
		})
	}
	avg := 0
	if len(events) > 0 {
		avg = total / len(events)
	}
	return map[string]any{
		"events": items,
		"summary": map[string]any{
			"activeEvents":    len(events),
			"averageSeverity": avg,
			"globalRiskLabel": normalizeWorldRiskLevel("", avg),
		},
	}
}

func (s *WorldGameService) conflictEntityName(ctx context.Context, entityType string, entityID uint) string {
	if entityID == 0 {
		return "Non disponible"
	}
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case "continent":
		return s.continentName(ctx, entityID)
	case "ai_faction", "faction":
		var faction models.AIWorldFaction
		if err := s.db.WithContext(ctx).First(&faction, entityID).Error; err == nil {
			return defaultText(faction.Name, "Non disponible")
		}
	case "guild":
		var guild models.Guild
		if err := s.db.WithContext(ctx).First(&guild, entityID).Error; err == nil {
			return defaultText(guild.Name, "Non disponible")
		}
	case "player":
		var save models.PlayerSave
		if err := s.db.WithContext(ctx).Where("player_id = ?", entityID).First(&save).Error; err == nil {
			return defaultText(save.CityName, "Non disponible")
		}
	}
	return "Non disponible"
}

func (s *WorldGameService) continentName(ctx context.Context, continentID uint) string {
	var continent models.Continent
	if continentID == 0 || s.db.WithContext(ctx).First(&continent, continentID).Error != nil {
		return "Non disponible"
	}
	return defaultText(continent.Name, "Non disponible")
}

func (s *WorldGameService) playerFourPillarActionCounts(ctx context.Context, playerID uint, worldID uint, since time.Time) (FourPillarActionCounts, error) {
	var logs []models.PlayerActionLog
	err := s.db.WithContext(ctx).
		Where("player_id = ? AND world_id = ? AND created_at >= ?", playerID, worldID, since).
		Find(&logs).Error
	if err != nil {
		return FourPillarActionCounts{}, err
	}
	var counts FourPillarActionCounts
	for _, item := range logs {
		accepted := item.Status == "accepted"
		action := strings.ToLower(item.Action)
		switch {
		case strings.Contains(action, "conflict"):
			if accepted {
				counts.ConflictAccepted++
			} else {
				counts.ConflictRejected++
			}
		case strings.Contains(action, "diplomacy"):
			if accepted {
				counts.DiplomacyAccepted++
			} else {
				counts.DiplomacyRejected++
			}
		case strings.Contains(action, "commerce"):
			if accepted {
				counts.CommerceAccepted++
			} else {
				counts.CommerceRejected++
			}
		case strings.Contains(action, "weather"):
			if accepted {
				counts.WeatherAccepted++
			} else {
				counts.WeatherRejected++
			}
		}
	}
	return counts, nil
}

func (s *WorldGameService) playerFourPillarPressure(ctx context.Context, worldID uint, continentID uint) (FourPillarPressure, error) {
	var conflicts []models.Conflict
	if err := s.db.WithContext(ctx).
		Where("world_id = ? AND status = ? AND (continent_id IS NULL OR continent_id = ?)", worldID, ConflictStatusActive, continentID).
		Find(&conflicts).Error; err != nil {
		return FourPillarPressure{}, err
	}
	var weather []models.WeatherEvent
	if err := s.db.WithContext(ctx).
		Where("world_id = ? AND continent_id = ? AND ends_at >= ?", worldID, continentID, time.Now()).
		Find(&weather).Error; err != nil {
		return FourPillarPressure{}, err
	}
	pressure := FourPillarPressure{ActiveConflicts: len(conflicts), ActiveWeather: len(weather)}
	for _, conflict := range conflicts {
		pressure.AverageConflictRisk += clamp(conflict.Intensity, 0, 100)
	}
	if len(conflicts) > 0 {
		pressure.AverageConflictRisk = pressure.AverageConflictRisk / len(conflicts)
	}
	for _, event := range weather {
		pressure.AverageWeatherSeverity += clamp(event.Severity, 0, 100)
	}
	if len(weather) > 0 {
		pressure.AverageWeatherSeverity = pressure.AverageWeatherSeverity / len(weather)
	}
	return pressure, nil
}

func scorePlayerFourPillars(save models.PlayerSave, counts FourPillarActionCounts, pressure FourPillarPressure) FourPillarScore {
	resourceBase := 0
	if save.Population > 0 {
		resourceBase = int(clamp(int((save.Food+save.Energy+save.Credits/2)/save.Population), 0, 100))
	}
	conflictScore := clamp(55+counts.ConflictAccepted*8-counts.ConflictRejected*15-pressure.AverageConflictRisk/3-pressure.ActiveConflicts*4, 0, 100)
	diplomacyScore := clamp(50+counts.DiplomacyAccepted*10-counts.DiplomacyRejected*20+save.Satisfaction/5, 0, 100)
	commerceScore := clamp(45+counts.CommerceAccepted*9-counts.CommerceRejected*15+resourceBase/3+int(save.Credits/10000), 0, 100)
	weatherScore := clamp(60+counts.WeatherAccepted*8-counts.WeatherRejected*18-pressure.AverageWeatherSeverity/3-pressure.ActiveWeather*5+resourceBase/4, 0, 100)
	popularity := clamp((save.Satisfaction*2+diplomacyScore+commerceScore+int(save.CityLevel)*3)/5, 0, 100)
	stability := clamp((save.Satisfaction+conflictScore+weatherScore+resourceBase)/4, 0, 100)
	sustainability := clamp((weatherScore*2+resourceBase+minInt(100, int(save.Energy/1000))+minInt(100, int(save.Food/1000)))/5, 0, 100)
	return FourPillarScore{
		Popularity:     popularity,
		Stability:      stability,
		Sustainability: sustainability,
		ConflictScore:  conflictScore,
		DiplomacyScore: diplomacyScore,
		CommerceScore:  commerceScore,
		WeatherScore:   weatherScore,
		Actions:        counts,
		Pressure:       pressure,
		Inputs: map[string]any{
			"cityLevel":    save.CityLevel,
			"satisfaction": save.Satisfaction,
			"population":   save.Population,
			"food":         save.Food,
			"energy":       save.Energy,
			"credits":      save.Credits,
		},
	}
}

func routineEventItems(events []models.GameEvent) []map[string]any {
	items := make([]map[string]any, 0, len(events))
	for _, event := range events {
		items = append(items, map[string]any{
			"id":         strconv.FormatUint(uint64(event.Id), 10),
			"title":      defaultText(event.Title, "Non disponible"),
			"type":       defaultText(event.Type, "Non disponible"),
			"difficulty": defaultText(event.Difficulty, "Non disponible"),
			"status":     defaultText(event.Status, "Non disponible"),
			"startsAt":   event.StartsAt,
			"endsAt":     event.EndsAt,
		})
	}
	return items
}

func routineMetricItems(metrics []models.PlayerWorldMetric) []map[string]any {
	items := make([]map[string]any, 0, len(metrics))
	for _, metric := range metrics {
		items = append(items, map[string]any{
			"playerId":       metric.PlayerID,
			"worldId":        metric.WorldID,
			"continentId":    metric.ContinentID,
			"popularity":     clamp(metric.Popularity, 0, 100),
			"stability":      clamp(metric.Stability, 0, 100),
			"sustainability": clamp(metric.Sustainability, 0, 100),
			"conflictScore":  clamp(metric.ConflictScore, 0, 100),
			"diplomacyScore": clamp(metric.DiplomacyScore, 0, 100),
			"commerceScore":  clamp(metric.CommerceScore, 0, 100),
			"weatherScore":   clamp(metric.WeatherScore, 0, 100),
			"generatedAt":    metric.GeneratedAt,
		})
	}
	return items
}

func routineMetricSummary(metrics []models.PlayerWorldMetric) map[string]any {
	summary := map[string]any{"players": len(metrics), "popularity": 0, "stability": 0, "sustainability": 0}
	if len(metrics) == 0 {
		return summary
	}
	popularity := 0
	stability := 0
	sustainability := 0
	for _, metric := range metrics {
		popularity += clamp(metric.Popularity, 0, 100)
		stability += clamp(metric.Stability, 0, 100)
		sustainability += clamp(metric.Sustainability, 0, 100)
	}
	summary["popularity"] = popularity / len(metrics)
	summary["stability"] = stability / len(metrics)
	summary["sustainability"] = sustainability / len(metrics)
	return summary
}

func routineQualityControls(conflicts []models.Conflict, weather []models.WeatherEvent, metrics []models.PlayerWorldMetric) map[string]any {
	return map[string]any{
		"conflicts":        len(conflicts),
		"weather":          len(weather),
		"playerMetrics":    len(metrics),
		"fallbackNumeric":  0,
		"fallbackText":     "Non disponible",
		"clampRangeMin":    0,
		"clampRangeMax":    100,
		"listsNeverNull":   true,
		"stableIdsEnabled": true,
	}
}

func averageRelationScore(relations []map[string]any) int {
	if len(relations) == 0 {
		return 0
	}
	total := 0
	for _, relation := range relations {
		score, _ := relation["score"].(int)
		total += clamp(score, 0, 100)
	}
	return total / len(relations)
}

func normalizeWorldRiskLevel(value string, score int) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "surveille", "modere", "eleve", "critique":
		return strings.ToLower(strings.TrimSpace(value))
	case "low":
		return "surveille"
	case "medium":
		return "modere"
	case "high":
		return "eleve"
	case "critical":
		return "critique"
	default:
		if score >= 80 {
			return "critique"
		}
		if score >= 60 {
			return "eleve"
		}
		if score >= 35 {
			return "modere"
		}
		return "surveille"
	}
}

func diplomacyStanceFromScore(score int) string {
	if score >= 80 {
		return "allie"
	}
	if score >= 60 {
		return "neutre"
	}
	if score >= 35 {
		return "independant"
	}
	return "hostile"
}

func asSliceMap(value any) []map[string]any {
	if items, ok := value.([]map[string]any); ok {
		return items
	}
	return []map[string]any{}
}

func normalizeSimulationCycle(cycleType string) string {
	switch strings.ToLower(strings.TrimSpace(cycleType)) {
	case SimulationCycleLight, "15m", "quarter":
		return SimulationCycleLight
	case SimulationCycleHourly, "hourly", "1h", "continent":
		return SimulationCycleHourly
	case SimulationCycleDaily, "24h", "day":
		return SimulationCycleDaily
	default:
		return SimulationCycleManual
	}
}

func pickSimulationContinent(world models.World, cycleType string) models.Continent {
	if len(world.Continents) == 0 {
		return models.Continent{}
	}
	index := world.CurrentCycle % len(world.Continents)
	if cycleType == SimulationCycleDaily {
		var selected models.Continent
		for i, continent := range world.Continents {
			if i == 0 || continent.TensionLevel > selected.TensionLevel {
				selected = continent
			}
		}
		if selected.Id != 0 {
			return selected
		}
	}
	if cycleType == SimulationCycleHourly {
		var selected models.Continent
		for i, continent := range world.Continents {
			if i == 0 || continent.CurrentPlayers < selected.CurrentPlayers {
				selected = continent
			}
		}
		if selected.Id != 0 {
			return selected
		}
	}
	return world.Continents[index]
}

func decisionContinentID(world models.World, cycleType string) *uint {
	continent := pickSimulationContinent(world, cycleType)
	if continent.Id == 0 {
		return nil
	}
	id := continent.Id
	return &id
}

func simulationRiskDeltas(profile string, cycleType string) (int, int) {
	tension := 1
	weather := 1
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "militaire":
		tension = 5
	case "commercial":
		tension = -1
	case "diplomatique":
		tension = -2
	case "instable":
		tension = 6
		weather = 3
	case "écologique", "ecologique":
		weather = 5
	case "technologique":
		tension = 2
		weather = 2
	}
	switch cycleType {
	case SimulationCycleLight:
		tension = clamp(tension, -2, 2)
		weather = clamp(weather, 0, 2)
	case SimulationCycleDaily:
		tension *= 2
		weather *= 2
	}
	return tension, weather
}

func defaultWeatherEffects(profile string) map[string]any {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "commercial":
		return map[string]any{"credits": -8, "constructionCost": 12}
	case "écologique", "ecologique":
		return map[string]any{"food": -12, "satisfaction": -6}
	case "technologique":
		return map[string]any{"energy": -12, "research": -5}
	case "militaire":
		return map[string]any{"energy": -8, "defense": -5}
	default:
		return map[string]any{"energy": -10, "satisfaction": -5}
	}
}

func profileEconomicState(profile string, current string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "commercial":
		return "expansion"
	case "instable":
		return "volatile"
	case "technologique":
		return "innovation"
	default:
		return "stable"
	}
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
	names := []string{"Nordrealm", "Valoria", "Aurora", "Sylvanie", "Dravon"}
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

func (s *WorldGameService) ensureAIWorldFactionTx(ctx context.Context, tx *gorm.DB, worldID uint, continentID *uint, name string) (*models.AIWorldFaction, error) {
	name = defaultText(name, "Novus")
	var faction models.AIWorldFaction
	query := tx.WithContext(ctx).Where("world_id = ? AND name = ?", worldID, name)
	if continentID != nil {
		query = query.Where("continent_id = ?", *continentID)
	}
	err := query.First(&faction).Error
	if err == nil {
		return &faction, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	faction = models.AIWorldFaction{
		WorldID:           worldID,
		ContinentID:       continentID,
		Name:              name,
		Type:              "antagonist",
		Aggressiveness:    65,
		Diplomacy:         25,
		Economy:           40,
		MilitaryPower:     60,
		ClimateResistance: 45,
		Status:            "active",
	}
	if err := tx.WithContext(ctx).Create(&faction).Error; err != nil {
		return nil, err
	}
	return &faction, nil
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

func (s *WorldGameService) callNEXUSRoutine(ctx context.Context, world models.World, snapshot map[string]any) (map[string]any, string, string, string, error) {
	providers := []string{defaultText(os.Getenv("WORLD_AI_PRIMARY_PROVIDER"), "mistral"), defaultText(os.Getenv("WORLD_AI_FALLBACK_PROVIDER"), "openai")}
	payload, _ := json.Marshal(snapshot)
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
			{Role: "system", Content: nexusRoutineSystemPrompt()},
			{Role: "user", Content: string(payload)},
		})
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		var decision map[string]any
		if err := json.Unmarshal([]byte(extractJSONObject(response)), &decision); err != nil {
			lastErr = err
			continue
		}
		status := DecisionStatusApplied
		if index > 0 {
			status = "fallback_provider"
		}
		return sanitizeRoutineAIOutput(decision), normalizeProviderName(cfg.Name), cfg.Model, status, nil
	}
	return deterministicRoutineDecision(world, snapshot), defaultText(providers[0], "deterministic"), "", DecisionStatusFallback, lastErr
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

func nexusRoutineSystemPrompt() string {
	return `Tu es NEXUS, l'intelligence centrale antagoniste du jeu.
Tu analyses exclusivement les donnees serveur fournies pour les 4 piliers: conflits, diplomatie, commerce, meteo.
Tu dois produire une decision par monde, jamais par donnees inventees.
Si une donnee manque, retourne 0 ou "Non disponible".
Retourne uniquement un JSON valide:
{
  "summary":"string",
  "threatLevel":0,
  "worldMood":"string",
  "pillarDecisions":{
    "conflicts":{"assessment":"string","actions":["string"]},
    "diplomacy":{"assessment":"string","actions":["string"]},
    "commerce":{"assessment":"string","actions":["string"]},
    "weather":{"assessment":"string","actions":["string"]}
  },
  "playerMetricPolicy":{"popularity":"string","stability":"string","sustainability":"string"},
  "alerts":["string"]
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

func deterministicRoutineDecision(world models.World, snapshot map[string]any) map[string]any {
	threat := clamp(world.GlobalTensionLevel+world.GlobalWeatherRisk/2, 0, 100)
	return map[string]any{
		"summary":     fmt.Sprintf("Routine NEXUS generee pour %s avec donnees serveur uniquement.", defaultText(world.Name, "Non disponible")),
		"threatLevel": threat,
		"worldMood":   normalizeWorldRiskLevel("", threat),
		"pillarDecisions": map[string]any{
			"conflicts": map[string]any{"assessment": "Surveillance des conflits actifs.", "actions": []string{}},
			"diplomacy": map[string]any{"assessment": "Relations calculees depuis les continents.", "actions": []string{}},
			"commerce":  map[string]any{"assessment": "Routes derivees des pressions serveur.", "actions": []string{}},
			"weather":   map[string]any{"assessment": "Risques meteo actifs consolides.", "actions": []string{}},
		},
		"playerMetricPolicy": map[string]any{
			"popularity":     "Satisfaction, diplomatie et commerce.",
			"stability":      "Conflits, meteo, ressources et satisfaction.",
			"sustainability": "Meteo, nourriture, energie et pression locale.",
		},
		"alerts": routineFallbackAlerts(snapshot),
	}
}

func sanitizeRoutineAIOutput(output map[string]any) map[string]any {
	if output == nil {
		return map[string]any{}
	}
	if _, ok := output["summary"].(string); !ok {
		output["summary"] = "Non disponible"
	}
	output["threatLevel"] = clamp(intFromAny(output["threatLevel"]), 0, 100)
	if _, ok := output["worldMood"].(string); !ok {
		output["worldMood"] = normalizeWorldRiskLevel("", intFromAny(output["threatLevel"]))
	}
	if _, ok := output["alerts"].([]any); !ok {
		output["alerts"] = []string{}
	}
	return output
}

func routineFallbackAlerts(snapshot map[string]any) []string {
	alerts := []string{}
	if quality, ok := snapshot["qualityControls"].(map[string]any); ok {
		if count, ok := quality["conflicts"].(int); ok && count == 0 {
			alerts = append(alerts, "Aucun conflit actif")
		}
		if count, ok := quality["weather"].(int); ok && count == 0 {
			alerts = append(alerts, "Aucune meteo active")
		}
	}
	return alerts
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
		seconds = 60
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

func minInt(max int, value int) int {
	if value > max {
		return max
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

func intFromAny(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
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

func (s *WorldGameService) LogPlayerWorldAction(ctx context.Context, playerID uint, worldID uint, continentID uint, action string, targetType string, targetID string, status string, errorMessage string, metadata any) error {
	return s.logPlayerAction(ctx, playerID, &worldID, &continentID, action, targetType, targetID, status, errorMessage, nil, nil, metadata)
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
