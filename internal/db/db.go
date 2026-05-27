package db

import (
	"cgwm/battle/internal/models"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func DbConnect() *gorm.DB {
	host := getEnv("DB_HOST", "127.0.0.1")
	port := getEnv("DB_PORT", "3306")
	name := getEnv("DB_NAME", "battleia")
	user := getEnv("DB_USER", "battleia")
	password := getEnv("DB_PASSWORD", "battleia")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		host,
		port,
		name,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed to get database pool: %v", err))
	}
	sqlDB.SetMaxOpenConns(getEnvInt("DB_MAX_OPEN_CONNS", 25))
	sqlDB.SetMaxIdleConns(getEnvInt("DB_MAX_IDLE_CONNS", 10))
	sqlDB.SetConnMaxLifetime(time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute)

	db.AutoMigrate(
		&models.Users{},
		&models.IAProfile{},
		&models.QuestIaBattle{},
		&models.BattleSave{},
		&models.BattleSaveTurn{},
		&models.BattleArena{},
		&models.BattleArenaMember{},
		&models.RolePlayQuestTemplate{},
		&models.RolePlayQuestArc{},
		&models.RolePlayQuestChapter{},
		&models.RolePlayQuestRun{},
		&models.RolePlaySession{},
		&models.RolePlaySessionTurn{},
		&models.AIUsageRecord{},
		&models.NexusCoinPlan{},
		&models.CoopParty{},
		&models.CoopPartyMember{},
		&models.LiveSession{},
		&models.LiveEvent{},
		&models.World{},
		&models.Continent{},
		&models.PlayerSave{},
		&models.PlayerBuilding{},
		&models.PlayerActionLog{},
		&models.ChatMessage{},
		&models.Guild{},
		&models.GuildMember{},
		&models.GuildInvite{},
		&models.GuildContribution{},
		&models.AIWorldFaction{},
		&models.Conflict{},
		&models.ConflictAction{},
		&models.WeatherEvent{},
		&models.GameEvent{},
		&models.GameEventParticipation{},
		&models.GameEventClaim{},
		&models.DailyAIMessage{},
		&models.AIWorldDecision{},
		&models.WorldRoutineSnapshot{},
		&models.PlayerWorldMetric{},
		&models.BuildingDefinition{},
		&models.BuildingAsset{},
		&models.BuildingCatalogVersion{},
		&models.AdminAuditLog{},
	)
	if err := seedDefaultBuildingDefinitions(db); err != nil {
		panic(fmt.Sprintf("failed to seed default building catalog: %v", err))
	}
	return db
}

func seedDefaultBuildingDefinitions(db *gorm.DB) error {
	defaults := []models.BuildingDefinition{
		{
			Key:                    "habitation",
			Name:                   "Habitation",
			Description:            "Logements modulaires pour stabiliser la population urbaine.",
			Category:               "residential",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":450,"energy":25,"durationMinutes":5}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.18,"energyMultiplier":1.1,"durationMultiplier":1.08}`),
			EffectsJSON:            jsonValue(`{"populationCapacity":120,"satisfaction":2}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":1}`),
			IsActive:               true,
			SortOrder:              10,
		},
		{
			Key:                    "solar_park",
			Name:                   "Parc solaire",
			Description:            "Infrastructure d'energie propre adaptee aux villes futuristes.",
			Category:               "energy",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":650,"durationMinutes":7}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.2,"durationMultiplier":1.1}`),
			EffectsJSON:            jsonValue(`{"energyProduction":90,"pollution":-2}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":1}`),
			IsActive:               true,
			SortOrder:              20,
		},
		{
			Key:                    "vertical_farm",
			Name:                   "Ferme verticale",
			Description:            "Production alimentaire dense pour continents sous pression.",
			Category:               "food",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":520,"energy":35,"durationMinutes":6}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.18,"energyMultiplier":1.12,"durationMultiplier":1.08}`),
			EffectsJSON:            jsonValue(`{"foodProduction":85,"satisfaction":1}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":1}`),
			IsActive:               true,
			SortOrder:              30,
		},
		{
			Key:                    "research_center",
			Name:                   "Centre de recherche",
			Description:            "Laboratoire avance pour accelerer les technologies urbaines.",
			Category:               "research",
			MaxLevel:               25,
			BaseCostJSON:           jsonValue(`{"credits":900,"energy":80,"durationMinutes":12}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.22,"energyMultiplier":1.14,"durationMultiplier":1.1}`),
			EffectsJSON:            jsonValue(`{"research":60,"xp":15}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":2}`),
			IsActive:               true,
			SortOrder:              40,
		},
		{
			Key:                    "ai_center",
			Name:                   "Centre IA",
			Description:            "Noeud de calcul urbain capable d'analyser les crises NEXUS.",
			Category:               "ai",
			MaxLevel:               20,
			BaseCostJSON:           jsonValue(`{"credits":1250,"energy":120,"durationMinutes":18}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.25,"energyMultiplier":1.16,"durationMultiplier":1.12}`),
			EffectsJSON:            jsonValue(`{"aiInsight":1,"stability":2}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":3}`),
			IsActive:               true,
			SortOrder:              50,
		},
		{
			Key:                    "defense_grid",
			Name:                   "Grille de defense",
			Description:            "Reseau defensif contre conflits, sabotage et pression des factions.",
			Category:               "defense",
			MaxLevel:               25,
			BaseCostJSON:           jsonValue(`{"credits":780,"energy":70,"durationMinutes":10}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.21,"energyMultiplier":1.13,"durationMultiplier":1.09}`),
			EffectsJSON:            jsonValue(`{"defense":75,"conflictRisk":-3}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":2}`),
			IsActive:               true,
			SortOrder:              60,
		},
	}

	for _, seed := range defaults {
		var existing models.BuildingDefinition
		err := db.Unscoped().Where("`key` = ?", seed.Key).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&seed).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		updates := map[string]any{}
		if !existing.IsActive {
			updates["is_active"] = true
		}
		if existing.DeletedAt.Valid {
			updates["deleted_at"] = nil
		}
		if len(updates) > 0 {
			if err := db.Unscoped().Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func jsonValue(raw string) datatypes.JSON {
	return datatypes.JSON([]byte(raw))
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(getEnv(key, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
