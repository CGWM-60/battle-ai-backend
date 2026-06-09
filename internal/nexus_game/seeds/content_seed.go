package seeds

import (
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
			BalanceVersion:       "v2.0.0",
			IsPublished:          true,
		},
		// TODO: add the other 17 from the reference (Mine, Raffinerie, Centre IA, Caserne, etc.)
		// Use POST /admin/content/buildings or the seed for full list + upload real tiered assets via CRUD.
	}

	for i := range buildings {
		_ = svc.CreateOrUpdateBuilding(&buildings[i])
	}
	return nil
}
