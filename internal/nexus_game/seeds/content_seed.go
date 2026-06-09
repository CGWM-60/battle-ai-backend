package seeds

import (
	"errors"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"

	"gorm.io/gorm"
)

// SeedInitialBuildings seeds the first few from NEXUS GAME CONTENT REFERENCE v2.0 for dev/testing.
// Run once or on admin command. Expand with all 20 + units + research later.
// Use the admin CRUD / upload-asset to manage full catalog + images in prod.

func SeedInitialBuildings(db *gorm.DB, svc *services.ContentService) error {
	buildings := []models.BuildingDefinition{
		{
			ContentID:            "building_modular_habitat",
			Domain:               "building",
			Type:                 "habitation",
			NameKey:              "building.modular_habitat.name",
			DescriptionKey:       "building.modular_habitat.description",
			AssetID:              "building_modular_habitat_tier1.jpg",
			AssetsByTier:         map[string]string{"tier1": "building_modular_habitat_tier1.jpg"},
			MaxLevel:             30,
			Rarity:               "common",
			NexusLevelRequired:   1,
			CostBaseCredits:      100,
			CostBaseMetal:        200,
			DurationBaseSeconds:  60,
			WorkersMin:           2,
			WorkersMax:           8,
			AIAgentSlots:         1,
			EffectsJSON:          "[]",
			BalanceVersion:       "v2.0.0",
			IsPublished:          true,
		},
		{
			ContentID:            "building_solar_plant",
			Domain:               "building",
			Type:                 "énergie",
			NameKey:              "building.solar_plant.name",
			DescriptionKey:       "building.solar_plant.description",
			AssetID:              "building_solar_plant_tier1.jpg",
			AssetsByTier:         map[string]string{"tier1": "building_solar_plant_tier1.jpg"},
			MaxLevel:             30,
			Rarity:               "common",
			NexusLevelRequired:   1,
			CostBaseCredits:      150,
			CostBaseMetal:        300,
			DurationBaseSeconds:  90,
			WorkersMin:           1,
			WorkersMax:           4,
			AIAgentSlots:         1,
			EffectsJSON:          "[]",
			BalanceVersion:       "v2.0.0",
			IsPublished:          true,
		},
		{
			ContentID:            "building_vertical_farm",
			Domain:               "building",
			Type:                 "production",
			NameKey:              "building.vertical_farm.name",
			DescriptionKey:       "building.vertical_farm.description",
			AssetID:              "building_vertical_farm_tier1.jpg",
			AssetsByTier:         map[string]string{"tier1": "building_vertical_farm_tier1.jpg"},
			MaxLevel:             30,
			Rarity:               "common",
			NexusLevelRequired:   1,
			CostBaseCredits:      120,
			CostBaseMetal:        250,
			CostBaseData:         50,
			DurationBaseSeconds:  90,
			WorkersMin:           2,
			WorkersMax:           6,
			AIAgentSlots:         1,
			EffectsJSON:          "[]",
			BalanceVersion:       "v2.0.0",
			IsPublished:          true,
		},
		// More from reference (samples; add the rest via admin CRUD or expand seed)
		{
			ContentID:            "building_composite_mine",
			Domain:               "building",
			Type:                 "production",
			NameKey:              "building.composite_mine.name",
			DescriptionKey:       "building.composite_mine.description",
			AssetID:              "building_composite_mine_tier1.jpg",
			AssetsByTier:         map[string]string{"tier1": "building_composite_mine_tier1.jpg"},
			MaxLevel:             30,
			Rarity:               "common",
			NexusLevelRequired:   1,
			CostBaseCredits:      200,
			CostBaseMetal:        350,
			DurationBaseSeconds:  120,
			WorkersMin:           3,
			WorkersMax:           10,
			AIAgentSlots:         1,
			EffectsJSON:          `[{"effectType":"resourceProduction","target":"metal","valueBase":100,"isPercentage":false}]`,
			BalanceVersion:       "v2.0.0",
			IsPublished:          true,
		},
		{
			ContentID:            "building_ai_center",
			Domain:               "building",
			Type:                 "ia",
			NameKey:              "building.ai_center.name",
			DescriptionKey:       "building.ai_center.description",
			AssetID:              "building_ai_center_tier1.jpg",
			AssetsByTier:         map[string]string{"tier1": "building_ai_center_tier1.jpg"},
			MaxLevel:             30,
			Rarity:               "uncommon",
			NexusLevelRequired:   4,
			CostBaseCredits:      600,
			CostBaseMetal:        800,
			CostBaseData:         400,
			DurationBaseSeconds:  1800,
			WorkersMin:           2,
			WorkersMax:           8,
			AIAgentSlots:         6,
			EffectsJSON:          `[{"effectType":"aiAgentSlots","target":"maxAgents","valueBase":1,"valuePerLevel":0.2,"isPercentage":false}]`,
			BalanceVersion:       "v2.0.0",
			IsPublished:          true,
		},
		// TODO: add the remaining ~15 from reference (Raffinerie Quantum, Caserne, Labo Recherche, etc.)
	}

	for i := range buildings {
		b := buildings[i]
		var existing models.BuildingDefinition
		if err := db.Where("content_id = ?", b.ContentID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = db.Create(&b).Error
			}
		} else {
			b.ID = existing.ID
			_ = db.Save(&b).Error
		}
	}
	return nil
}

func SeedInitialUnits(db *gorm.DB, svc *services.ContentService) error {
	units := []models.UnitDefinition{
		{
			ContentID:           "unit_milicien_nexus",
			Domain:              "unit",
			Type:                "infantry",
			NameKey:             "unit.milicien_nexus.name",
			DescriptionKey:      "unit.milicien_nexus.description",
			AssetID:             "unit_milicien_nexus_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "unit_milicien_nexus_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "common",
			HealthBase:          100,
			AttackBase:          20,
			DefenseBase:         15,
			SpeedBase:           5,
			TrainingTimeBaseSeconds: 300,
			UpkeepBase:          2,
			EffectsJSON:         "[]",
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "unit_drone_sentinelle",
			Domain:              "unit",
			Type:                "drone",
			NameKey:             "unit.drone_sentinelle.name",
			DescriptionKey:      "unit.drone_sentinelle.description",
			AssetID:             "unit_drone_sentinelle_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "unit_drone_sentinelle_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "common",
			HealthBase:          50,
			AttackBase:          10,
			DefenseBase:         30,
			SpeedBase:           8,
			TrainingTimeBaseSeconds: 180,
			UpkeepBase:          1,
			EffectsJSON:         "[]",
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		// TODO: add the other 13 from reference §5 (Éclaireur, Fantassin, Artillerie, Titan, etc.) with full stats per level, counters.
	}

	for i := range units {
		u := units[i]
		var existing models.UnitDefinition
		if err := db.Where("content_id = ?", u.ContentID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = db.Create(&u).Error
			}
		} else {
			u.ID = existing.ID
			_ = db.Save(&u).Error
		}
	}
	return nil
}

func SeedInitialResearch(db *gorm.DB, svc *services.ContentService) error {
	research := []models.ResearchDefinition{
		{
			ContentID:           "research_economy_tier1",
			Domain:              "research",
			Branch:              "economy",
			NameKey:             "research.economy_tier1.name",
			DescriptionKey:      "research.economy_tier1.description",
			AssetID:             "research_economy_tier1.jpg",
			Tier:                1,
			Rarity:              "common",
			CostBaseCredits:     500,
			CostBaseData:        200,
			DurationBaseSeconds: 3600,
			EffectsJSON:         `[{"effectType":"resourceProduction","target":"credits","valueBase":10,"isPercentage":true}]`,
			PrerequisitesJSON:   "[]",
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "research_military_tier1",
			Domain:              "research",
			Branch:              "military",
			NameKey:             "research.military_tier1.name",
			DescriptionKey:      "research.military_tier1.description",
			AssetID:             "research_military_tier1.jpg",
			Tier:                1,
			Rarity:              "common",
			CostBaseCredits:     400,
			CostBaseData:        150,
			DurationBaseSeconds: 2700,
			EffectsJSON:         `[{"effectType":"statBonus","target":"unitAttack","valueBase":5,"isPercentage":true}]`,
			PrerequisitesJSON:   "[]",
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		// TODO: add all 11 branches x 7 tiers from reference §6 with dependencies, per-level formulas, etc.
	}

	for i := range research {
		r := research[i]
		var existing models.ResearchDefinition
		if err := db.Where("content_id = ?", r.ContentID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = db.Create(&r).Error
			}
		} else {
			r.ID = existing.ID
			_ = db.Save(&r).Error
		}
	}
	return nil
}
