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
		&models.RolePlayQuestScene{},
		&models.RolePlayQuestSceneImage{},
		&models.RolePlayImagePromptJob{},
		&models.RolePlayImagePromptJobItem{},
		&models.RolePlayQuestRun{},
		&models.RolePlayCharacter{},
		&models.RolePlayHeroImage{},
		&models.RolePlaySession{},
		&models.RolePlaySessionTurn{},
		&models.AIUsageRecord{},
		&models.NexusCoinPlan{},
		&models.CoopParty{},
		&models.CoopPartyMember{},
		&models.LiveSession{},
		&models.LiveEvent{},
		&models.RPGArcMap{},
		&models.RPGSessionMapState{},
		&models.RPGCoopMapState{},
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
		&models.GuildQuest{},
		&models.GuildWar{},
		&models.GuildWarContribution{},
		&models.GuildResearch{},
		&models.GuildDiplomacy{},
		&models.AIWorldFaction{},
		&models.Conflict{},
		&models.ConflictAction{},
		&models.WeatherEvent{},
		&models.GameEvent{},
		&models.GameEventParticipation{},
		&models.GameEventClaim{},
		&models.DailyAIMessage{},
		&models.AIWorldDecision{},
		&models.DailyTask{},
		&models.WorldRoutineSnapshot{},
		&models.PlayerWorldMetric{},
		&models.ArmyUnit{},
		&models.ArmyTrainingQueue{},
		&models.ResourceDefinition{},
		&models.ResearchTreeDefinition{},
		&models.ResearchNodeDefinition{},
		&models.BuildingDefinition{},
		&models.BuildingAsset{},
		&models.BuildingCatalogVersion{},
		&models.AdminAuditLog{},
		&models.DailyTask{},
		&models.MarketOffer{}, // IA Global + player continental market offers (source, direction, continent_id)
		// Nexus Games translation models (POINT 01)
		&models.TranslationDomain{},
		&models.TranslationKey{},
		&models.TranslationValue{},
		&models.TranslationBatch{},
		&models.TranslationImport{},
		&models.TranslationImportRow{},
		&models.TranslationImportError{},
		&models.TranslationMissingLog{},
		&models.UserLocalePreference{},
	)
	_ = db.Exec(`
		UPDATE role_play_quest_templates
		SET is_published = 1
		WHERE status = 'published' AND (is_published = 0 OR is_published IS NULL)
	`).Error

	if err := seedDefaultBuildingDefinitions(db); err != nil {
		panic(fmt.Sprintf("failed to seed default building catalog: %v", err))
	}
	if err := seedDefaultResearchSystem(db); err != nil {
		panic(fmt.Sprintf("failed to seed research system: %v", err))
	}
	return db
}

func seedDefaultBuildingDefinitions(db *gorm.DB) error {
	defaults := []models.BuildingDefinition{
		{
			Key:                    "habitation",
			Name:                   "Habitation",
			Description:            "Des modules d'habitat qui font grandir la cité sans la faire craquer: plus de citoyens, plus de vie, plus de pression à gérer.",
			Category:               "residential",
			ResearchTreeKey:        "construction_genie_civil",
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
			Description:            "Un champ de panneaux et de batteries qui garde les lumières allumées quand le continent entre en crise.",
			Category:               "energy",
			ResearchTreeKey:        "durabilite_energie",
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
			Description:            "Des tours de culture sous verrière: de la nourriture locale, même quand les routes commerciales brûlent.",
			Category:               "food",
			ResearchTreeKey:        "durabilite_energie",
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
			Description:            "Le cerveau scientifique de la ville, là où les prototypes deviennent des avantages décisifs.",
			Category:               "research",
			ResearchTreeKey:        "technologie_science",
			MaxLevel:               30,
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
			Description:            "Un noyau d'IA qui lit les signaux faibles, anticipe les crises et donne une longueur d'avance au joueur.",
			Category:               "ai",
			ResearchTreeKey:        "technologie_science",
			MaxLevel:               30,
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
			Description:            "Un maillage de tourelles, drones et boucliers pour tenir la ville quand les factions testent tes murs.",
			Category:               "defense",
			ResearchTreeKey:        "defense_militaire",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":780,"energy":70,"durationMinutes":10}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.21,"energyMultiplier":1.13,"durationMultiplier":1.09}`),
			EffectsJSON:            jsonValue(`{"defense":75,"conflictRisk":-3}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":2}`),
			IsActive:               true,
			SortOrder:              60,
		},
		{
			Key:                    "city_hall",
			Name:                   "Hotel de ville",
			Description:            "Le centre de commandement civil: lois, ordres d'urgence et décisions qui gardent la population soudée.",
			Category:               "civic",
			ResearchTreeKey:        "stabilite_civile",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":1000,"energy":40,"durationMinutes":15}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.2,"energyMultiplier":1.1,"durationMultiplier":1.1}`),
			EffectsJSON:            jsonValue(`{"authority":20,"stability":3}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":1}`),
			IsActive:               true,
			SortOrder:              70,
		},
		{
			Key:                    "trade_hub",
			Name:                   "Hub commercial",
			Description:            "Le coeur marchand de la cité, où les convois, les contrats et les paris économiques se transforment en crédits.",
			Category:               "economy",
			ResearchTreeKey:        "prosperite_economique",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":1150,"energy":55,"durationMinutes":16}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.21,"energyMultiplier":1.12,"durationMultiplier":1.1}`),
			EffectsJSON:            jsonValue(`{"creditsProduction":80,"commerce":25}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":2}`),
			IsActive:               true,
			SortOrder:              80,
		},
		{
			Key:                    "diplomacy_center",
			Name:                   "Centre diplomatique",
			Description:            "Salons feutrés, canaux secrets et tables de négociation pour gagner sans toujours tirer le premier.",
			Category:               "diplomacy",
			ResearchTreeKey:        "diplomatie_influence",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":1200,"energy":60,"durationMinutes":18}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.22,"energyMultiplier":1.13,"durationMultiplier":1.11}`),
			EffectsJSON:            jsonValue(`{"diplomacy":25,"influence":20}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":3}`),
			IsActive:               true,
			SortOrder:              90,
		},
		{
			Key:                    "engineering_office",
			Name:                   "Bureau d'ingenierie",
			Description:            "Des ingénieurs, des plans et des machines lourdes pour bâtir plus haut, plus vite et plus solide.",
			Category:               "engineering",
			ResearchTreeKey:        "construction_genie_civil",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":1050,"energy":70,"durationMinutes":15}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.2,"energyMultiplier":1.15,"durationMultiplier":1.1}`),
			EffectsJSON:            jsonValue(`{"engineering":30,"constructionSpeed":2}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":2}`),
			IsActive:               true,
			SortOrder:              100,
		},
		{
			Key:                    "barracks",
			Name:                   "Caserne",
			Description:            "La forge des troupes: chaque niveau ouvre des unités plus lourdes et prépare la ville aux vrais conflits.",
			Category:               "military",
			ResearchTreeKey:        "defense_militaire",
			MaxLevel:               30,
			BaseCostJSON:           jsonValue(`{"credits":1200,"food":250,"energy":180,"materials":350,"durationMinutes":20}`),
			LevelCostFormulaJSON:   jsonValue(`{"creditsMultiplier":1.28,"foodMultiplier":1.28,"energyMultiplier":1.28,"materialsMultiplier":1.28,"durationMultiplier":1.22,"durationMaxDays":30}`),
			EffectsJSON:            jsonValue(`{"unlocks":{"1":"infanterie_legere","3":"infanterie_lourde","5":"tireur_longue_portee","8":"unite_logistique","12":"unite_elite","15":"multi_interventions","20":"reduced_conflict_losses","25":"global_morale_bonus","30":"global_strategic_bonus"}}`),
			UnlockRequirementsJSON: jsonValue(`{"cityLevel":2}`),
			IsActive:               true,
			SortOrder:              110,
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
		if seed.Description != "" && existing.Description != seed.Description {
			updates["description"] = seed.Description
		}
		if !existing.IsActive {
			updates["is_active"] = true
		}
		if existing.MaxLevel < 30 {
			updates["max_level"] = 30
		}
		if existing.ResearchTreeKey == "" && seed.ResearchTreeKey != "" {
			updates["research_tree_key"] = seed.ResearchTreeKey
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
