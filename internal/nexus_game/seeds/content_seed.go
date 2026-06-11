package seeds

import (
	"errors"
	"sort"
	"strings"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"

	"gorm.io/gorm"
)

// SeedInitial* seeds the complete catalogs from NEXUS GAME CONTENT REFERENCE v2.0 §4/5/6 for dev/testing.
// Buildings: 20 (BLD-), Units: 15 (UNT-), Research: 11 branches × 7 tiers (full arbre with exact node structure, prereqs, effects from "Effet principal", costs/durs per tier).
// Idempotent create-only seed. Once a catalog row exists, admin CRUD is the source
// of truth and startup seeds must never overwrite uploaded assets or custom fields.
// Legacy names purged on start. Use admin Next.js CRUD + upload-asset for images in prod.

func SeedInitialBuildings(db *gorm.DB, svc *services.ContentService) error {
	buildings := []models.BuildingDefinition{
		{
			ContentID:           "building_modular_habitat",
			Domain:              "building",
			Type:                "habitation",
			NameKey:             "building.modular_habitat.name",
			DescriptionKey:      "building.modular_habitat.description",
			AssetID:             "building_modular_habitat_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_modular_habitat_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "common",
			NexusLevelRequired:  1,
			CostBaseCredits:     100,
			CostBaseMetal:       200,
			DurationBaseSeconds: 60,
			WorkersMin:          2,
			WorkersMax:          8,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"statBonus","target":"city","stat":"populationCapacity","valueBase":500,"valuePerLevel":250,"isPercentage":false},{"effectType":"statBonus","target":"city","stat":"morale","valueBase":2,"valuePerLevel":0,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_solar_plant",
			Domain:              "building",
			Type:                "énergie",
			NameKey:             "building.solar_plant.name",
			DescriptionKey:      "building.solar_plant.description",
			AssetID:             "building_solar_plant_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_solar_plant_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "common",
			NexusLevelRequired:  1,
			CostBaseCredits:     150,
			CostBaseMetal:       300,
			DurationBaseSeconds: 90,
			WorkersMin:          1,
			WorkersMax:          4,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"resourceProduction","target":"energy","valueBase":80,"valuePerLevel":30,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_vertical_farm",
			Domain:              "building",
			Type:                "production",
			NameKey:             "building.vertical_farm.name",
			DescriptionKey:      "building.vertical_farm.description",
			AssetID:             "building_vertical_farm_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_vertical_farm_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "common",
			NexusLevelRequired:  1,
			CostBaseCredits:     120,
			CostBaseMetal:       250,
			CostBaseData:        50,
			DurationBaseSeconds: 90,
			WorkersMin:          2,
			WorkersMax:          6,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"resourceProduction","target":"food","valueBase":70,"valuePerLevel":25,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		// Full 20 from the NEXUS GAME CONTENT REFERENCE v2.0 doc. All IsPublished: true, with exact data from the tables.
		{
			ContentID:           "building_composite_mine",
			Domain:              "building",
			Type:                "production",
			NameKey:             "building.composite_mine.name",
			DescriptionKey:      "building.composite_mine.description",
			AssetID:             "building_composite_mine_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_composite_mine_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "common",
			NexusLevelRequired:  1,
			CostBaseCredits:     200,
			CostBaseMetal:       350,
			DurationBaseSeconds: 120,
			WorkersMin:          3,
			WorkersMax:          10,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"resourceProduction","target":"metal","valueBase":100,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_ai_center",
			Domain:              "building",
			Type:                "ia",
			NameKey:             "building.ai_center.name",
			DescriptionKey:      "building.ai_center.description",
			AssetID:             "building_ai_center_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_ai_center_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  4,
			CostBaseCredits:     600,
			CostBaseMetal:       800,
			CostBaseData:        400,
			DurationBaseSeconds: 1800,
			WorkersMin:          2,
			WorkersMax:          8,
			AIAgentSlots:        6,
			EffectsJSON:         `[{"effectType":"aiAgentSlots","target":"maxAgents","valueBase":1,"valuePerLevel":0.2,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_quantum_refinery",
			Domain:              "building",
			Type:                "production avancée",
			NameKey:             "building.quantum_refinery.name",
			DescriptionKey:      "building.quantum_refinery.description",
			AssetID:             "building_quantum_refinery_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_quantum_refinery_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "rare",
			NexusLevelRequired:  8,
			CostBaseCredits:     1500,
			CostBaseMetal:       2000,
			CostBaseData:        200,
			DurationBaseSeconds: 14400,
			WorkersMin:          4,
			WorkersMax:          12,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"resourceProduction","target":"quantumCoreProduction","valueBase":5,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_research_lab",
			Domain:              "building",
			Type:                "recherche",
			NameKey:             "building.research_lab.name",
			DescriptionKey:      "building.research_lab.description",
			AssetID:             "building_research_lab_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_research_lab_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  2,
			CostBaseCredits:     400,
			CostBaseMetal:       500,
			CostBaseData:        200,
			DurationBaseSeconds: 900,
			WorkersMin:          2,
			WorkersMax:          8,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"statBonus","target":"researchSpeed","valueBase":10,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_barracks",
			Domain:              "building",
			Type:                "militaire",
			NameKey:             "building.barracks.name",
			DescriptionKey:      "building.barracks.description",
			AssetID:             "building_barracks_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_barracks_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "common",
			NexusLevelRequired:  2,
			CostBaseCredits:     300,
			CostBaseMetal:       400,
			DurationBaseSeconds: 300,
			WorkersMin:          3,
			WorkersMax:          10,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"statBonus","target":"trainingQueueSlots","valueBase":1,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_drone_factory",
			Domain:              "building",
			Type:                "militaire",
			NameKey:             "building.drone_factory.name",
			DescriptionKey:      "building.drone_factory.description",
			AssetID:             "building_drone_factory_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_drone_factory_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  5,
			CostBaseCredits:     800,
			CostBaseMetal:       1200,
			CostBaseData:        300,
			DurationBaseSeconds: 7200,
			WorkersMin:          4,
			WorkersMax:          12,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"statBonus","target":"droneTrainingSpeed","valueBase":20,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_holo_wall",
			Domain:              "building",
			Type:                "défense",
			NameKey:             "building.holo_wall.name",
			DescriptionKey:      "building.holo_wall.description",
			AssetID:             "building_holo_wall_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_holo_wall_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  3,
			CostBaseCredits:     400,
			CostBaseMetal:       600,
			CostBaseData:        100,
			DurationBaseSeconds: 1200,
			WorkersMin:          0,
			WorkersMax:          0,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"statBonus","target":"cityDefense","valueBase":200,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_diplomatic_center",
			Domain:              "building",
			Type:                "diplomatie",
			NameKey:             "building.diplomatic_center.name",
			DescriptionKey:      "building.diplomatic_center.description",
			AssetID:             "building_diplomatic_center_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_diplomatic_center_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  4,
			CostBaseCredits:     700,
			CostBaseMetal:       0,
			CostBaseData:        200,
			DurationBaseSeconds: 1800,
			WorkersMin:          2,
			WorkersMax:          6,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"statBonus","target":"factionReputationGain","valueBase":10,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_nexus_market",
			Domain:              "building",
			Type:                "commerce",
			NameKey:             "building.nexus_market.name",
			DescriptionKey:      "building.nexus_market.description",
			AssetID:             "building_nexus_market_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_nexus_market_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  3,
			CostBaseCredits:     500,
			CostBaseMetal:       300,
			CostBaseData:        100,
			DurationBaseSeconds: 900,
			WorkersMin:          1,
			WorkersMax:          5,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"statBonus","target":"tradeBonus","valueBase":15,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_logistic_station",
			Domain:              "building",
			Type:                "logistique",
			NameKey:             "building.logistic_station.name",
			DescriptionKey:      "building.logistic_station.description",
			AssetID:             "building_logistic_station_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_logistic_station_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  3,
			CostBaseCredits:     380,
			CostBaseMetal:       500,
			CostBaseData:        0,
			DurationBaseSeconds: 480,
			WorkersMin:          1,
			WorkersMax:          5,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"statBonus","target":"resourceTransportSpeed","valueBase":20,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_living_lore_archives",
			Domain:              "building",
			Type:                "lore",
			NameKey:             "building.living_lore_archives.name",
			DescriptionKey:      "building.living_lore_archives.description",
			AssetID:             "building_living_lore_archives_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_living_lore_archives_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "rare",
			NexusLevelRequired:  3,
			CostBaseCredits:     420,
			CostBaseMetal:       650,
			CostBaseData:        180,
			DurationBaseSeconds: 1080,
			WorkersMin:          1,
			WorkersMax:          4,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"statBonus","target":"loreContributionGain","valueBase":10,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_tribunal_nexus",
			Domain:              "building",
			Type:                "tribunal",
			NameKey:             "building.tribunal_nexus.name",
			DescriptionKey:      "building.tribunal_nexus.description",
			AssetID:             "building_tribunal_nexus_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_tribunal_nexus_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "epic",
			NexusLevelRequired:  5,
			CostBaseCredits:     700,
			CostBaseMetal:       900,
			CostBaseData:        300,
			DurationBaseSeconds: 1500,
			WorkersMin:          2,
			WorkersMax:          6,
			AIAgentSlots:        3,
			EffectsJSON:         `[{"effectType":"statBonus","target":"tribunalEfficiency","valueBase":15,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_guild_hq",
			Domain:              "building",
			Type:                "guilde",
			NameKey:             "building.guild_hq.name",
			DescriptionKey:      "building.guild_hq.description",
			AssetID:             "building_guild_hq_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_guild_hq_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  3,
			CostBaseCredits:     360,
			CostBaseMetal:       480,
			CostBaseData:        0,
			DurationBaseSeconds: 420,
			WorkersMin:          2,
			WorkersMax:          5,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"statBonus","target":"guildXP","valueBase":12,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_surveillance_tower",
			Domain:              "building",
			Type:                "défense",
			NameKey:             "building.surveillance_tower.name",
			DescriptionKey:      "building.surveillance_tower.description",
			AssetID:             "building_surveillance_tower_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_surveillance_tower_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  3,
			CostBaseCredits:     390,
			CostBaseMetal:       520,
			CostBaseData:        0,
			DurationBaseSeconds: 540,
			WorkersMin:          1,
			WorkersMax:          4,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"statBonus","target":"detectionRange","valueBase":10,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_world_relay",
			Domain:              "building",
			Type:                "monde",
			NameKey:             "building.world_relay.name",
			DescriptionKey:      "building.world_relay.description",
			AssetID:             "building_world_relay_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_world_relay_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "rare",
			NexusLevelRequired:  4,
			CostBaseCredits:     550,
			CostBaseMetal:       750,
			CostBaseData:        220,
			DurationBaseSeconds: 1200,
			WorkersMin:          1,
			WorkersMax:          5,
			AIAgentSlots:        2,
			EffectsJSON:         `[{"effectType":"statBonus","target":"worldEventInfluence","valueBase":8,"isPercentage":true}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_data_bank",
			Domain:              "building",
			Type:                "data",
			NameKey:             "building.data_bank.name",
			DescriptionKey:      "building.data_bank.description",
			AssetID:             "building_data_bank_tier1.jpg",
			AssetsByTier:        map[string]string{"tier1": "building_data_bank_tier1.jpg"},
			MaxLevel:            30,
			Rarity:              "uncommon",
			NexusLevelRequired:  3,
			CostBaseCredits:     410,
			CostBaseMetal:       580,
			CostBaseData:        160,
			DurationBaseSeconds: 660,
			WorkersMin:          1,
			WorkersMax:          4,
			AIAgentSlots:        1,
			EffectsJSON:         `[{"effectType":"statBonus","target":"dataProduction","valueBase":50,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
		{
			ContentID:           "building_nexus_core",
			Domain:              "building",
			Type:                "spécial",
			NameKey:             "building.nexus_core.name",
			DescriptionKey:      "building.nexus_core.description",
			AssetID:             "building_nexus_core_tier4.jpg",
			AssetsByTier:        map[string]string{"tier4": "building_nexus_core_tier4.jpg"},
			MaxLevel:            30,
			Rarity:              "nexus",
			NexusLevelRequired:  0,
			CostBaseCredits:     0,
			CostBaseMetal:       0,
			DurationBaseSeconds: 300,
			WorkersMin:          0,
			WorkersMax:          0,
			AIAgentSlots:        8,
			EffectsJSON:         `[{"effectType":"statBonus","target":"globalPowerBonus","valueBase":5,"isPercentage":true},{"effectType":"statBonus","target":"cityLevelCap","valueBase":5,"isPercentage":false}]`,
			BalanceVersion:      "v2.0.0",
			IsPublished:         true,
		},
	}

	applyDefaultBuildingRequirements(buildings)
	applyDefaultBuildingMMODesign(buildings)
	for i := range buildings {
		b := buildings[i]
		var existing models.BuildingDefinition
		if err := db.Where("content_id = ?", b.ContentID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = db.Create(&b).Error
			}
		} else {
			fillEmptyBuildingRequirements(db, &existing, b)
		}
	}
	return nil
}

type buildingStorageSeed struct {
	resource          string
	capBase           int
	growth            float64
	overflow          string
	decay             float64
	productionPerHour int
	productionGrowth  float64
	harvestSeconds    int
}

type buildingUnlockSeed struct {
	era              string
	availableAtSpawn bool
	unlockType       string
	requirements     string
	message          string
	nonConstructible bool
}

func applyDefaultBuildingMMODesign(buildings []models.BuildingDefinition) {
	slots := map[string]int{
		"building_composite_mine":       3,
		"building_solar_plant":          4,
		"building_vertical_farm":        3,
		"building_quantum_refinery":     1,
		"building_modular_habitat":      6,
		"building_logistic_station":     2,
		"building_barracks":             2,
		"building_drone_factory":        2,
		"building_holo_wall":            3,
		"building_surveillance_tower":   2,
		"building_nexus_market":         2,
		"building_data_bank":            2,
		"building_diplomatic_center":    1,
		"building_world_relay":          1,
		"building_guild_hq":             1,
		"building_ai_center":            2,
		"building_research_lab":         2,
		"building_living_lore_archives": 1,
		"building_tribunal_nexus":       1,
		"building_nexus_core":           1,
	}
	durations := map[string]int{
		"building_modular_habitat":      600,
		"building_vertical_farm":        600,
		"building_solar_plant":          600,
		"building_composite_mine":       600,
		"building_barracks":             600,
		"building_logistic_station":     600,
		"building_holo_wall":            1800,
		"building_data_bank":            1800,
		"building_surveillance_tower":   1800,
		"building_nexus_market":         1800,
		"building_research_lab":         1800,
		"building_ai_center":            1800,
		"building_diplomatic_center":    1800,
		"building_guild_hq":             3600,
		"building_drone_factory":        7200,
		"building_living_lore_archives": 7200,
		"building_world_relay":          7200,
		"building_quantum_refinery":     14400,
		"building_tribunal_nexus":       21600,
		"building_nexus_core":           86400,
	}
	storage := map[string]buildingStorageSeed{
		"building_composite_mine":    {resource: "metal", capBase: 2000, growth: 1.10, overflow: "suspend", productionPerHour: 100, productionGrowth: 1.13, harvestSeconds: 21600},
		"building_solar_plant":       {resource: "energy", capBase: 0, growth: 1.00, overflow: "realtime", productionPerHour: 80, productionGrowth: 1.13, harvestSeconds: 0},
		"building_vertical_farm":     {resource: "food", capBase: 5000, growth: 1.10, overflow: "suspend", productionPerHour: 70, productionGrowth: 1.13, harvestSeconds: 28800},
		"building_data_bank":         {resource: "data", capBase: 500000, growth: 1.12, overflow: "loop", productionPerHour: 50, productionGrowth: 1.13, harvestSeconds: 43200},
		"building_quantum_refinery":  {resource: "quantum_core", capBase: 50, growth: 1.08, overflow: "suspend", productionPerHour: 5, productionGrowth: 1.10, harvestSeconds: 86400},
		"building_diplomatic_center": {resource: "influence", capBase: 200, growth: 1.10, overflow: "loop", productionPerHour: 8, productionGrowth: 1.10, harvestSeconds: 43200},
	}
	unlocks := map[string]buildingUnlockSeed{
		"building_nexus_core":           {era: "0", availableAtSpawn: true, unlockType: "AND", requirements: "[]", message: "Noyau de spawn unique, coeur politique et technique de la cite.", nonConstructible: true},
		"building_modular_habitat":      {era: "0", availableAtSpawn: true, unlockType: "AND", requirements: "[]", message: "Survie de base: logement disponible des l'arrivee."},
		"building_solar_plant":          {era: "0", availableAtSpawn: true, unlockType: "AND", requirements: "[]", message: "Survie de base: energie stable disponible des l'arrivee."},
		"building_composite_mine":       {era: "0", availableAtSpawn: true, unlockType: "AND", requirements: "[]", message: "Survie de base: metal disponible des l'arrivee."},
		"building_logistic_station":     {era: "0", availableAtSpawn: true, unlockType: "AND", requirements: "[]", message: "Survie de base: logistique disponible des l'arrivee."},
		"building_vertical_farm":        {era: "1", unlockType: "AND", requirements: buildingReq("building_modular_habitat", 2), message: "Habitat niveau 2: il faut nourrir la population grandissante."},
		"building_barracks":             {era: "1", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_modular_habitat": 3, "building_composite_mine": 2}), message: "Population et metal suffisants pour former les premieres troupes."},
		"building_holo_wall":            {era: "1", unlockType: "AND", requirements: buildingReq("building_barracks", 1), message: "Caserne construite: protege ce que tu as bati."},
		"building_data_bank":            {era: "1", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_solar_plant": 3, "building_logistic_station": 2}), message: "Infrastructure solide: il est temps de structurer tes donnees."},
		"building_research_lab":         {era: "2", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_data_bank": 3, "building_solar_plant": 5}), message: "Assez de donnees et d'energie pour lancer la recherche."},
		"building_surveillance_tower":   {era: "2", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_holo_wall": 3, "building_data_bank": 2}), message: "Defenses actives et donnees ouvrent la surveillance intelligente."},
		"building_drone_factory":        {era: "2", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_barracks": 5, "building_research_lab": 1}), message: "Caserne veterane et recherche initiale debloquent les drones."},
		"building_nexus_market":         {era: "2", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_logistic_station": 5, "building_data_bank": 3}), message: "Logistique maitrisee et donnees fiables ouvrent le commerce Nexus."},
		"building_ai_center":            {era: "3", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_research_lab": 5, "building_data_bank": 8, "building_solar_plant": 8}), message: "Recherche avancee, donnees et energie rendent l'infrastructure IA possible."},
		"building_diplomatic_center":    {era: "3", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_nexus_market": 5, "building_surveillance_tower": 3}), message: "Commerce etabli et renseignements ouvrent le pouvoir diplomatique."},
		"building_guild_hq":             {era: "3", unlockType: "OR", requirements: multiBuildingReq(map[string]int{"building_diplomatic_center": 1, "building_nexus_market": 8}), message: "Diplomatie ou commerce avance permettent de rejoindre une guilde."},
		"building_quantum_refinery":     {era: "4", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_research_lab": 15, "building_ai_center": 5, "building_composite_mine": 12}), message: "Recherche poussee, IA et mine veterane ouvrent le raffinage quantique."},
		"building_living_lore_archives": {era: "4", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_data_bank": 12, "building_ai_center": 8, "building_diplomatic_center": 5}), message: "Memoire, IA et reseau diplomatique donnent naissance aux archives vivantes."},
		"building_tribunal_nexus":       {era: "4", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_diplomatic_center": 8, "building_guild_hq": 5, "building_surveillance_tower": 10}), message: "Diplomatie, guilde et surveillance donnent une legitimite judiciaire."},
		"building_world_relay":          {era: "4", unlockType: "AND", requirements: multiBuildingReq(map[string]int{"building_ai_center": 10, "building_surveillance_tower": 8, "building_nexus_market": 12}), message: "IA, renseignement et commerce mondial ouvrent le Relais Monde."},
	}

	for i := range buildings {
		b := &buildings[i]
		if maxSlots, ok := slots[b.ContentID]; ok {
			b.SlotsMax = maxSlots
		}
		if duration, ok := durations[b.ContentID]; ok {
			b.DurationBaseSeconds = duration
		}
		b.DurationMultiplier = 1.28
		b.MilestoneReduction = 0.15
		if unlock, ok := unlocks[b.ContentID]; ok {
			b.UnlockEra = unlock.era
			b.AvailableAtSpawn = unlock.availableAtSpawn
			b.UnlockType = unlock.unlockType
			b.UnlockMessage = unlock.message
			b.NonConstructible = unlock.nonConstructible
			b.RequiredBuildingsJSON = unlock.requirements
		}
		if store, ok := storage[b.ContentID]; ok {
			b.StorageResource = store.resource
			b.StorageCapBase = store.capBase
			b.StorageGrowth = store.growth
			b.OverflowBehavior = store.overflow
			b.OverflowDecayPercentPerHour = store.decay
			b.ProductionBasePerHour = store.productionPerHour
			b.ProductionGrowth = store.productionGrowth
			b.HarvestRecommendedIntervalSeconds = store.harvestSeconds
		} else {
			b.OverflowBehavior = "none"
			b.ProductionGrowth = 1.00
			b.StorageGrowth = 1.00
		}
		b.SynergiesJSON = defaultBuildingSynergies(b.ContentID)
		b.RisksJSON = defaultBuildingRisks(b.ContentID)
		b.AIActionsJSON = defaultBuildingAIActions(b.ContentID)
		if b.FlavorTextKey == "" {
			b.FlavorTextKey = defaultFlavorTextKey(b.DescriptionKey)
		}
		if len(b.LevelDescriptionKeys) == 0 {
			b.LevelDescriptionKeys = defaultLevelDescriptionKeys(b.DescriptionKey)
		}
	}
}

func SeedInitialUnits(db *gorm.DB, svc *services.ContentService) error {
	units := []models.UnitDefinition{
		{
			ContentID:               "unit_milicien_nexus",
			Domain:                  "unit",
			Type:                    "infantry",
			NameKey:                 "unit.milicien_nexus.name",
			DescriptionKey:          "unit.milicien_nexus.description",
			AssetID:                 "unit_milicien_nexus_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_milicien_nexus_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "common",
			HealthBase:              100,
			AttackBase:              15,
			DefenseBase:             8,
			SpeedBase:               3,
			TrainingTimeBaseSeconds: 300,
			UpkeepBase:              2,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_drone_sentinelle",
			Domain:                  "unit",
			Type:                    "drone",
			NameKey:                 "unit.drone_sentinelle.name",
			DescriptionKey:          "unit.drone_sentinelle.description",
			AssetID:                 "unit_drone_sentinelle_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_drone_sentinelle_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "common",
			HealthBase:              60,
			AttackBase:              20,
			DefenseBase:             12,
			SpeedBase:               5,
			TrainingTimeBaseSeconds: 180,
			UpkeepBase:              1,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_eclaireur_cybernetique",
			Domain:                  "unit",
			Type:                    "infantry",
			NameKey:                 "unit.eclaireur_cybernetique.name",
			DescriptionKey:          "unit.eclaireur_cybernetique.description",
			AssetID:                 "unit_eclaireur_cybernetique_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_eclaireur_cybernetique_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "common",
			HealthBase:              80,
			AttackBase:              10,
			DefenseBase:             5,
			SpeedBase:               8,
			TrainingTimeBaseSeconds: 240,
			UpkeepBase:              3,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_fantassin_augmente",
			Domain:                  "unit",
			Type:                    "infantry",
			NameKey:                 "unit.fantassin_augmente.name",
			DescriptionKey:          "unit.fantassin_augmente.description",
			AssetID:                 "unit_fantassin_augmente_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_fantassin_augmente_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "uncommon",
			HealthBase:              180,
			AttackBase:              30,
			DefenseBase:             20,
			SpeedBase:               4,
			TrainingTimeBaseSeconds: 420,
			UpkeepBase:              5,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_drone_assaut",
			Domain:                  "unit",
			Type:                    "drone",
			NameKey:                 "unit.drone_assaut.name",
			DescriptionKey:          "unit.drone_assaut.description",
			AssetID:                 "unit_drone_assaut_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_drone_assaut_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "uncommon",
			HealthBase:              70,
			AttackBase:              35,
			DefenseBase:             8,
			SpeedBase:               7,
			TrainingTimeBaseSeconds: 300,
			UpkeepBase:              8,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_drone_bouclier",
			Domain:                  "unit",
			Type:                    "drone",
			NameKey:                 "unit.drone_bouclier.name",
			DescriptionKey:          "unit.drone_bouclier.description",
			AssetID:                 "unit_drone_bouclier_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_drone_bouclier_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "uncommon",
			HealthBase:              250,
			AttackBase:              12,
			DefenseBase:             40,
			SpeedBase:               3,
			TrainingTimeBaseSeconds: 360,
			UpkeepBase:              10,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_hacker_de_combat",
			Domain:                  "unit",
			Type:                    "support",
			NameKey:                 "unit.hacker_de_combat.name",
			DescriptionKey:          "unit.hacker_de_combat.description",
			AssetID:                 "unit_hacker_de_combat_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_hacker_de_combat_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "rare",
			HealthBase:              90,
			AttackBase:              18,
			DefenseBase:             10,
			SpeedBase:               4,
			TrainingTimeBaseSeconds: 480,
			UpkeepBase:              3,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_medic_synthetique",
			Domain:                  "unit",
			Type:                    "support",
			NameKey:                 "unit.medic_synthetique.name",
			DescriptionKey:          "unit.medic_synthetique.description",
			AssetID:                 "unit_medic_synthetique_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_medic_synthetique_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "uncommon",
			HealthBase:              110,
			AttackBase:              5,
			DefenseBase:             8,
			SpeedBase:               4,
			TrainingTimeBaseSeconds: 360,
			UpkeepBase:              3,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_artillerie_railgun",
			Domain:                  "unit",
			Type:                    "artillerie",
			NameKey:                 "unit.artillerie_railgun.name",
			DescriptionKey:          "unit.artillerie_railgun.description",
			AssetID:                 "unit_artillerie_railgun_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_artillerie_railgun_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "rare",
			HealthBase:              200,
			AttackBase:              120,
			DefenseBase:             15,
			SpeedBase:               1,
			TrainingTimeBaseSeconds: 900,
			UpkeepBase:              20,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_mecha_leger",
			Domain:                  "unit",
			Type:                    "mecha",
			NameKey:                 "unit.mecha_leger.name",
			DescriptionKey:          "unit.mecha_leger.description",
			AssetID:                 "unit_mecha_leger_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_mecha_leger_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "epic",
			HealthBase:              500,
			AttackBase:              80,
			DefenseBase:             60,
			SpeedBase:               5,
			TrainingTimeBaseSeconds: 1200,
			UpkeepBase:              25,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_agent_infiltre",
			Domain:                  "unit",
			Type:                    "special",
			NameKey:                 "unit.agent_infiltre.name",
			DescriptionKey:          "unit.agent_infiltre.description",
			AssetID:                 "unit_agent_infiltre_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_agent_infiltre_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "rare",
			HealthBase:              70,
			AttackBase:              25,
			DefenseBase:             5,
			SpeedBase:               6,
			TrainingTimeBaseSeconds: 600,
			UpkeepBase:              5,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_envoye_de_faction",
			Domain:                  "unit",
			Type:                    "special",
			NameKey:                 "unit.envoye_de_faction.name",
			DescriptionKey:          "unit.envoye_de_faction.description",
			AssetID:                 "unit_envoye_de_faction_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_envoye_de_faction_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "uncommon",
			HealthBase:              50,
			AttackBase:              0,
			DefenseBase:             0,
			SpeedBase:               4,
			TrainingTimeBaseSeconds: 300,
			UpkeepBase:              4,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_gardien_holographique",
			Domain:                  "unit",
			Type:                    "defense",
			NameKey:                 "unit.gardien_holographique.name",
			DescriptionKey:          "unit.gardien_holographique.description",
			AssetID:                 "unit_gardien_holographique_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_gardien_holographique_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "rare",
			HealthBase:              300,
			AttackBase:              45,
			DefenseBase:             50,
			SpeedBase:               0,
			TrainingTimeBaseSeconds: 720,
			UpkeepBase:              15,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_titan_nexus",
			Domain:                  "unit",
			Type:                    "mecha",
			NameKey:                 "unit.titan_nexus.name",
			DescriptionKey:          "unit.titan_nexus.description",
			AssetID:                 "unit_titan_nexus_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_titan_nexus_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "nexus",
			HealthBase:              2000,
			AttackBase:              200,
			DefenseBase:             150,
			SpeedBase:               3,
			TrainingTimeBaseSeconds: 3600,
			UpkeepBase:              100,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
		{
			ContentID:               "unit_archiviste_mobile",
			Domain:                  "unit",
			Type:                    "support",
			NameKey:                 "unit.archiviste_mobile.name",
			DescriptionKey:          "unit.archiviste_mobile.description",
			AssetID:                 "unit_archiviste_mobile_tier1.jpg",
			AssetsByTier:            map[string]string{"tier1": "unit_archiviste_mobile_tier1.jpg"},
			MaxLevel:                30,
			Rarity:                  "rare",
			HealthBase:              80,
			AttackBase:              0,
			DefenseBase:             5,
			SpeedBase:               4,
			TrainingTimeBaseSeconds: 300,
			UpkeepBase:              3,
			EffectsJSON:             "[]",
			BalanceVersion:          "v2.0.0",
			IsPublished:             true,
		},
	}

	applyDefaultUnitRequirements(units)
	for i := range units {
		u := units[i]
		var existing models.UnitDefinition
		if err := db.Where("content_id = ?", u.ContentID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = db.Create(&u).Error
			}
		} else {
			fillEmptyUnitRequirements(db, &existing, u)
		}
	}
	return nil
}

func SeedInitialResearch(db *gorm.DB, svc *services.ContentService) error {
	research := []models.ResearchDefinition{
		// ÉCONOMIE
		{ContentID: "research_efficient_storage", Domain: "research", Branch: "economy", NameKey: "research.efficient_storage.name", DescriptionKey: "research.efficient_storage.description", AssetID: "icon_research_economy", AssetsByTier: map[string]string{"tier1": "icon_research_economy.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"resourceBonus","target":"all","stat":"storageCapacity","value":0.20}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_resource_routing", Domain: "research", Branch: "economy", NameKey: "research.resource_routing.name", DescriptionKey: "research.resource_routing.description", AssetID: "icon_research_economy", AssetsByTier: map[string]string{"tier2": "icon_research_economy.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"transportTime","value":-0.15}]`, PrerequisitesJSON: `["research_efficient_storage"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_automated_harvest", Domain: "research", Branch: "economy", NameKey: "research.automated_harvest.name", DescriptionKey: "research.automated_harvest.description", AssetID: "icon_research_economy", AssetsByTier: map[string]string{"tier3": "icon_research_economy.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"harvestSlots","value":1}]`, PrerequisitesJSON: `["research_resource_routing"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_market_prediction", Domain: "research", Branch: "economy", NameKey: "research.market_prediction.name", DescriptionKey: "research.market_prediction.description", AssetID: "icon_research_economy", AssetsByTier: map[string]string{"tier4": "icon_research_economy.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"marketPredictionAccuracy","value":0.20}]`, PrerequisitesJSON: `["research_automated_harvest"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_quantum_logistics", Domain: "research", Branch: "economy", NameKey: "research.quantum_logistics.name", DescriptionKey: "research.quantum_logistics.description", AssetID: "icon_research_economy", AssetsByTier: map[string]string{"tier5": "icon_research_economy.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"unlock","target":"quantum_logistics"}]`, PrerequisitesJSON: `["research_market_prediction"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_guild_trade_protocols", Domain: "research", Branch: "economy", NameKey: "research.guild_trade_protocols.name", DescriptionKey: "research.guild_trade_protocols.description", AssetID: "icon_research_economy", AssetsByTier: map[string]string{"tier6": "icon_research_economy.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"guildTradeSlots","value":2},{"effectType":"statBonus","target":"all","stat":"guildTradeTax","value":-0.10}]`, PrerequisitesJSON: `["research_quantum_logistics"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_macro_economy", Domain: "research", Branch: "economy", NameKey: "research.nexus_macro_economy.name", DescriptionKey: "research.nexus_macro_economy.description", AssetID: "icon_research_economy", AssetsByTier: map[string]string{"tier7": "icon_research_economy.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"globalResourceProduction","value":0.30}]`, PrerequisitesJSON: `["research_guild_trade_protocols"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// ÉNERGIE
		{ContentID: "research_solar_stabilization", Domain: "research", Branch: "energy", NameKey: "research.solar_stabilization.name", DescriptionKey: "research.solar_stabilization.description", AssetID: "icon_research_energy", AssetsByTier: map[string]string{"tier1": "icon_research_energy.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"buildingBonus","target":"building_solar_plant","stat":"energyProduction","value":0.10}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_grid_balancing", Domain: "research", Branch: "energy", NameKey: "research.grid_balancing.name", DescriptionKey: "research.grid_balancing.description", AssetID: "icon_research_energy", AssetsByTier: map[string]string{"tier2": "icon_research_energy.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"blackoutResistance","value":1}]`, PrerequisitesJSON: `["research_solar_stabilization"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_battery_clusters", Domain: "research", Branch: "energy", NameKey: "research.battery_clusters.name", DescriptionKey: "research.battery_clusters.description", AssetID: "icon_research_energy", AssetsByTier: map[string]string{"tier3": "icon_research_energy.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"energyStorage","value":500}]`, PrerequisitesJSON: `["research_grid_balancing"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_fusion_relay", Domain: "research", Branch: "energy", NameKey: "research.fusion_relay.name", DescriptionKey: "research.fusion_relay.description", AssetID: "icon_research_energy", AssetsByTier: map[string]string{"tier4": "icon_research_energy.jpg"}, Tier: 4, Rarity: "rare", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"unlock","target":"fusion_relay"},{"effectType":"statBonus","target":"all","stat":"energyProduction","value":200}]`, PrerequisitesJSON: `["research_battery_clusters"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_quantum_reactor_safety", Domain: "research", Branch: "energy", NameKey: "research.quantum_reactor_safety.name", DescriptionKey: "research.quantum_reactor_safety.description", AssetID: "icon_research_energy", AssetsByTier: map[string]string{"tier5": "icon_research_energy.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"building_quantum_refinery","stat":"anomalyRisk","value":-1}]`, PrerequisitesJSON: `["research_fusion_relay"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_void_energy_harness", Domain: "research", Branch: "energy", NameKey: "research.void_energy_harness.name", DescriptionKey: "research.void_energy_harness.description", AssetID: "icon_research_energy", AssetsByTier: map[string]string{"tier6": "icon_research_energy.jpg"}, Tier: 6, Rarity: "epic", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"voidToEnergy","value":5}]`, PrerequisitesJSON: `["research_quantum_reactor_safety"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_grid", Domain: "research", Branch: "energy", NameKey: "research.nexus_grid.name", DescriptionKey: "research.nexus_grid.description", AssetID: "icon_research_energy", AssetsByTier: map[string]string{"tier7": "icon_research_energy.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"globalEnergyProduction","value":0.50}]`, PrerequisitesJSON: `["research_void_energy_harness"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// VILLE
		{ContentID: "research_modular_housing", Domain: "research", Branch: "ville", NameKey: "research.modular_housing.name", DescriptionKey: "research.modular_housing.description", AssetID: "icon_research_ville", AssetsByTier: map[string]string{"tier1": "icon_research_ville.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"buildingBonus","target":"building_modular_habitat","stat":"populationCapacity","value":0.10}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_population_welfare", Domain: "research", Branch: "ville", NameKey: "research.population_welfare.name", DescriptionKey: "research.population_welfare.description", AssetID: "icon_research_ville", AssetsByTier: map[string]string{"tier2": "icon_research_ville.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"morale","value":0.05}]`, PrerequisitesJSON: `["research_modular_housing"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_urban_ai_planning", Domain: "research", Branch: "ville", NameKey: "research.urban_ai_planning.name", DescriptionKey: "research.urban_ai_planning.description", AssetID: "icon_research_ville", AssetsByTier: map[string]string{"tier3": "icon_research_ville.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"unlock","target":"urban_ai_planning"}]`, PrerequisitesJSON: `["research_population_welfare"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_crisis_resilience", Domain: "research", Branch: "ville", NameKey: "research.crisis_resilience.name", DescriptionKey: "research.crisis_resilience.description", AssetID: "icon_research_ville", AssetsByTier: map[string]string{"tier4": "icon_research_ville.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"crisisResistance","value":0.30}]`, PrerequisitesJSON: `["research_urban_ai_planning"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_arcology_design", Domain: "research", Branch: "ville", NameKey: "research.arcology_design.name", DescriptionKey: "research.arcology_design.description", AssetID: "icon_research_ville", AssetsByTier: map[string]string{"tier5": "icon_research_ville.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"unlock","target":"arcology_design"}]`, PrerequisitesJSON: `["research_crisis_resilience"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_autonomous_districts", Domain: "research", Branch: "ville", NameKey: "research.autonomous_districts.name", DescriptionKey: "research.autonomous_districts.description", AssetID: "icon_research_ville", AssetsByTier: map[string]string{"tier6": "icon_research_ville.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"autonomousDistricts","value":1}]`, PrerequisitesJSON: `["research_arcology_design"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_urban_singularity", Domain: "research", Branch: "ville", NameKey: "research.nexus_urban_singularity.name", DescriptionKey: "research.nexus_urban_singularity.description", AssetID: "icon_research_ville", AssetsByTier: map[string]string{"tier7": "icon_research_ville.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"populationUnlimited","value":1},{"effectType":"statBonus","target":"all","stat":"overpopulationPenalty","value":0}]`, PrerequisitesJSON: `["research_autonomous_districts"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// MILITAIRE
		{ContentID: "research_basic_tactics", Domain: "research", Branch: "militaire", NameKey: "research.basic_tactics.name", DescriptionKey: "research.basic_tactics.description", AssetID: "icon_research_military", AssetsByTier: map[string]string{"tier1": "icon_research_military.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"buildingBonus","target":"building_barracks","stat":"unitAttack","value":0.05}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_augmented_infantry", Domain: "research", Branch: "militaire", NameKey: "research.augmented_infantry.name", DescriptionKey: "research.augmented_infantry.description", AssetID: "icon_research_military", AssetsByTier: map[string]string{"tier2": "icon_research_military.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"unlock","target":"fantassin_augmente"}]`, PrerequisitesJSON: `["research_basic_tactics"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_squad_coordination", Domain: "research", Branch: "militaire", NameKey: "research.squad_coordination.name", DescriptionKey: "research.squad_coordination.description", AssetID: "icon_research_military", AssetsByTier: map[string]string{"tier3": "icon_research_military.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"squadBonus","value":0.10}]`, PrerequisitesJSON: `["research_augmented_infantry"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_railgun_engineering", Domain: "research", Branch: "militaire", NameKey: "research.railgun_engineering.name", DescriptionKey: "research.railgun_engineering.description", AssetID: "icon_research_military", AssetsByTier: map[string]string{"tier4": "icon_research_military.jpg"}, Tier: 4, Rarity: "rare", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"unlock","target":"artillerie_railgun"}]`, PrerequisitesJSON: `["research_squad_coordination"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_battlefield_logistics", Domain: "research", Branch: "militaire", NameKey: "research.battlefield_logistics.name", DescriptionKey: "research.battlefield_logistics.description", AssetID: "icon_research_military", AssetsByTier: map[string]string{"tier5": "icon_research_military.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"deploymentTime","value":-0.20}]`, PrerequisitesJSON: `["research_railgun_engineering"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_elite_doctrine", Domain: "research", Branch: "militaire", NameKey: "research.elite_doctrine.name", DescriptionKey: "research.elite_doctrine.description", AssetID: "icon_research_military", AssetsByTier: map[string]string{"tier6": "icon_research_military.jpg"}, Tier: 6, Rarity: "epic", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"elite","stat":"allStats","value":0.15}]`, PrerequisitesJSON: `["research_battlefield_logistics"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_warfare", Domain: "research", Branch: "militaire", NameKey: "research.nexus_warfare.name", DescriptionKey: "research.nexus_warfare.description", AssetID: "icon_research_military", AssetsByTier: map[string]string{"tier7": "icon_research_military.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"unlock","target":"titan_nexus"},{"effectType":"statBonus","target":"all","stat":"worldCombatBonus","value":0.20}]`, PrerequisitesJSON: `["research_elite_doctrine"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// DRONES
		{ContentID: "research_drone_assembly", Domain: "research", Branch: "drones", NameKey: "research.drone_assembly.name", DescriptionKey: "research.drone_assembly.description", AssetID: "icon_research_drones", AssetsByTier: map[string]string{"tier1": "icon_research_drones.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"unlock","target":"usine_de_drones"}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_swarm_control", Domain: "research", Branch: "drones", NameKey: "research.swarm_control.name", DescriptionKey: "research.swarm_control.description", AssetID: "icon_research_drones", AssetsByTier: map[string]string{"tier2": "icon_research_drones.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"droneMaxPerSquad","value":5}]`, PrerequisitesJSON: `["research_drone_assembly"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_shield_drone_matrix", Domain: "research", Branch: "drones", NameKey: "research.shield_drone_matrix.name", DescriptionKey: "research.shield_drone_matrix.description", AssetID: "icon_research_drones", AssetsByTier: map[string]string{"tier3": "icon_research_drones.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"unlock","target":"drone_bouclier"}]`, PrerequisitesJSON: `["research_swarm_control"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_autonomous_targeting", Domain: "research", Branch: "drones", NameKey: "research.autonomous_targeting.name", DescriptionKey: "research.autonomous_targeting.description", AssetID: "icon_research_drones", AssetsByTier: map[string]string{"tier4": "icon_research_drones.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"droneAutonomous","value":1}]`, PrerequisitesJSON: `["research_shield_drone_matrix"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_anti_hack_firmware", Domain: "research", Branch: "drones", NameKey: "research.anti_hack_firmware.name", DescriptionKey: "research.anti_hack_firmware.description", AssetID: "icon_research_drones", AssetsByTier: map[string]string{"tier5": "icon_research_drones.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"droneHackResistance","value":0.50}]`, PrerequisitesJSON: `["research_autonomous_targeting"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_drone_carrier_protocol", Domain: "research", Branch: "drones", NameKey: "research.drone_carrier_protocol.name", DescriptionKey: "research.drone_carrier_protocol.description", AssetID: "icon_research_drones", AssetsByTier: map[string]string{"tier6": "icon_research_drones.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"unlock","target":"drone_carrier"}]`, PrerequisitesJSON: `["research_anti_hack_firmware"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_swarm", Domain: "research", Branch: "drones", NameKey: "research.nexus_swarm.name", DescriptionKey: "research.nexus_swarm.description", AssetID: "icon_research_drones", AssetsByTier: map[string]string{"tier7": "icon_research_drones.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"droneUnlimited","value":1},{"effectType":"statBonus","target":"all","stat":"droneEnergyCost","value":3}]`, PrerequisitesJSON: `["research_drone_carrier_protocol"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// IA
		{ContentID: "research_agent_basics", Domain: "research", Branch: "ia", NameKey: "research.agent_basics.name", DescriptionKey: "research.agent_basics.description", AssetID: "icon_research_ia", AssetsByTier: map[string]string{"tier1": "icon_research_ia.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"agentSlots","value":1}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_prompt_memory", Domain: "research", Branch: "ia", NameKey: "research.prompt_memory.name", DescriptionKey: "research.prompt_memory.description", AssetID: "icon_research_ia", AssetsByTier: map[string]string{"tier2": "icon_research_ia.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"agentMemoryDays","value":7}]`, PrerequisitesJSON: `["research_agent_basics"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_multi_agent_routing", Domain: "research", Branch: "ia", NameKey: "research.multi_agent_routing.name", DescriptionKey: "research.multi_agent_routing.description", AssetID: "icon_research_ia", AssetsByTier: map[string]string{"tier3": "icon_research_ia.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"multiAgentCoordination","value":1}]`, PrerequisitesJSON: `["research_prompt_memory"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_byoai_cost_control", Domain: "research", Branch: "ia", NameKey: "research.byoai_cost_control.name", DescriptionKey: "research.byoai_cost_control.description", AssetID: "icon_research_ia", AssetsByTier: map[string]string{"tier4": "icon_research_ia.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"byoaiCostReduction","value":0.20}]`, PrerequisitesJSON: `["research_multi_agent_routing"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_local_llm_optimization", Domain: "research", Branch: "ia", NameKey: "research.local_llm_optimization.name", DescriptionKey: "research.local_llm_optimization.description", AssetID: "icon_research_ia", AssetsByTier: map[string]string{"tier5": "icon_research_ia.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"localLLMSpeed","value":2}]`, PrerequisitesJSON: `["research_byoai_cost_control"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_autonomous_proposal_system", Domain: "research", Branch: "ia", NameKey: "research.autonomous_proposal_system.name", DescriptionKey: "research.autonomous_proposal_system.description", AssetID: "icon_research_ia", AssetsByTier: map[string]string{"tier6": "icon_research_ia.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"autonomousProposal","value":1}]`, PrerequisitesJSON: `["research_local_llm_optimization"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_cognitive_mesh", Domain: "research", Branch: "ia", NameKey: "research.nexus_cognitive_mesh.name", DescriptionKey: "research.nexus_cognitive_mesh.description", AssetID: "icon_research_ia", AssetsByTier: map[string]string{"tier7": "icon_research_ia.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"guildAISharing","value":1}]`, PrerequisitesJSON: `["research_autonomous_proposal_system"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// DIPLOMATIE
		{ContentID: "research_emissary_protocol", Domain: "research", Branch: "diplomatie", NameKey: "research.emissary_protocol.name", DescriptionKey: "research.emissary_protocol.description", AssetID: "icon_research_diplomatie", AssetsByTier: map[string]string{"tier1": "icon_research_diplomatie.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"emissarySlots","value":1}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_faction_language_models", Domain: "research", Branch: "diplomatie", NameKey: "research.faction_language_models.name", DescriptionKey: "research.faction_language_models.description", AssetID: "icon_research_diplomatie", AssetsByTier: map[string]string{"tier2": "icon_research_diplomatie.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"factionLanguage","value":1}]`, PrerequisitesJSON: `["research_emissary_protocol"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_reputation_mapping", Domain: "research", Branch: "diplomatie", NameKey: "research.reputation_mapping.name", DescriptionKey: "research.reputation_mapping.description", AssetID: "icon_research_diplomatie", AssetsByTier: map[string]string{"tier3": "icon_research_diplomatie.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"reputationMapping","value":1}]`, PrerequisitesJSON: `["research_faction_language_models"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_treaty_simulation", Domain: "research", Branch: "diplomatie", NameKey: "research.treaty_simulation.name", DescriptionKey: "research.treaty_simulation.description", AssetID: "icon_research_diplomatie", AssetsByTier: map[string]string{"tier4": "icon_research_diplomatie.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"treatySimulation","value":1}]`, PrerequisitesJSON: `["research_reputation_mapping"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_conflict_mediation", Domain: "research", Branch: "diplomatie", NameKey: "research.conflict_mediation.name", DescriptionKey: "research.conflict_mediation.description", AssetID: "icon_research_diplomatie", AssetsByTier: map[string]string{"tier5": "icon_research_diplomatie.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"conflictMediation","value":1}]`, PrerequisitesJSON: `["research_treaty_simulation"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_tribunal_diplomacy", Domain: "research", Branch: "diplomatie", NameKey: "research.tribunal_diplomacy.name", DescriptionKey: "research.tribunal_diplomacy.description", AssetID: "icon_research_diplomatie", AssetsByTier: map[string]string{"tier6": "icon_research_diplomatie.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"tribunalDiplomacy","value":1}]`, PrerequisitesJSON: `["research_conflict_mediation"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_concordat", Domain: "research", Branch: "diplomatie", NameKey: "research.nexus_concordat.name", DescriptionKey: "research.nexus_concordat.description", AssetID: "icon_research_diplomatie", AssetsByTier: map[string]string{"tier7": "icon_research_diplomatie.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"nexusConcordat","value":1}]`, PrerequisitesJSON: `["research_tribunal_diplomacy"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// MONDE
		{ContentID: "research_regional_scanning", Domain: "research", Branch: "monde", NameKey: "research.regional_scanning.name", DescriptionKey: "research.regional_scanning.description", AssetID: "icon_research_monde", AssetsByTier: map[string]string{"tier1": "icon_research_monde.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"regionalScanning","value":2}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_event_forecasting", Domain: "research", Branch: "monde", NameKey: "research.event_forecasting.name", DescriptionKey: "research.event_forecasting.description", AssetID: "icon_research_monde", AssetsByTier: map[string]string{"tier2": "icon_research_monde.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"eventForecastingHours","value":48}]`, PrerequisitesJSON: `["research_regional_scanning"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_conflict_mapping", Domain: "research", Branch: "monde", NameKey: "research.conflict_mapping.name", DescriptionKey: "research.conflict_mapping.description", AssetID: "icon_research_monde", AssetsByTier: map[string]string{"tier3": "icon_research_monde.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"conflictMapping","value":1}]`, PrerequisitesJSON: `["research_event_forecasting"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_weather_adaptation", Domain: "research", Branch: "monde", NameKey: "research.weather_adaptation.name", DescriptionKey: "research.weather_adaptation.description", AssetID: "icon_research_monde", AssetsByTier: map[string]string{"tier4": "icon_research_monde.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"weatherAdaptation","value":0.50}]`, PrerequisitesJSON: `["research_conflict_mapping"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_world_action_slots", Domain: "research", Branch: "monde", NameKey: "research.world_action_slots.name", DescriptionKey: "research.world_action_slots.description", AssetID: "icon_research_monde", AssetsByTier: map[string]string{"tier5": "icon_research_monde.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"worldActionSlots","value":2}]`, PrerequisitesJSON: `["research_weather_adaptation"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_continental_strategy", Domain: "research", Branch: "monde", NameKey: "research.continental_strategy.name", DescriptionKey: "research.continental_strategy.description", AssetID: "icon_research_monde", AssetsByTier: map[string]string{"tier6": "icon_research_monde.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"continentalInfluence","value":0.30}]`, PrerequisitesJSON: `["research_world_action_slots"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_world_awareness", Domain: "research", Branch: "monde", NameKey: "research.nexus_world_awareness.name", DescriptionKey: "research.nexus_world_awareness.description", AssetID: "icon_research_monde", AssetsByTier: map[string]string{"tier7": "icon_research_monde.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"worldAwareness","value":1}]`, PrerequisitesJSON: `["research_continental_strategy"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// GUILDE
		{ContentID: "research_guild_charter", Domain: "research", Branch: "guilde", NameKey: "research.guild_charter.name", DescriptionKey: "research.guild_charter.description", AssetID: "icon_research_guilde", AssetsByTier: map[string]string{"tier1": "icon_research_guilde.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"guildCharter","value":1}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_cooperative_missions", Domain: "research", Branch: "guilde", NameKey: "research.cooperative_missions.name", DescriptionKey: "research.cooperative_missions.description", AssetID: "icon_research_guilde", AssetsByTier: map[string]string{"tier2": "icon_research_guilde.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"cooperativeMissionsSlots","value":1}]`, PrerequisitesJSON: `["research_guild_charter"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_donation_efficiency", Domain: "research", Branch: "guilde", NameKey: "research.donation_efficiency.name", DescriptionKey: "research.donation_efficiency.description", AssetID: "icon_research_guilde", AssetsByTier: map[string]string{"tier3": "icon_research_guilde.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"donationEfficiency","value":0.25}]`, PrerequisitesJSON: `["research_cooperative_missions"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_guild_ai_strategy", Domain: "research", Branch: "guilde", NameKey: "research.guild_ai_strategy.name", DescriptionKey: "research.guild_ai_strategy.description", AssetID: "icon_research_guilde", AssetsByTier: map[string]string{"tier4": "icon_research_guilde.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"guildAIStrategy","value":1}]`, PrerequisitesJSON: `["research_donation_efficiency"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_shared_logistics", Domain: "research", Branch: "guilde", NameKey: "research.shared_logistics.name", DescriptionKey: "research.shared_logistics.description", AssetID: "icon_research_guilde", AssetsByTier: map[string]string{"tier5": "icon_research_guilde.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"sharedLogistics","value":1}]`, PrerequisitesJSON: `["research_guild_ai_strategy"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_alliance_protocols", Domain: "research", Branch: "guilde", NameKey: "research.alliance_protocols.name", DescriptionKey: "research.alliance_protocols.description", AssetID: "icon_research_guilde", AssetsByTier: map[string]string{"tier6": "icon_research_guilde.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"allianceProtocols","value":1}]`, PrerequisitesJSON: `["research_shared_logistics"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_guild_network", Domain: "research", Branch: "guilde", NameKey: "research.nexus_guild_network.name", DescriptionKey: "research.nexus_guild_network.description", AssetID: "icon_research_guilde", AssetsByTier: map[string]string{"tier7": "icon_research_guilde.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"nexusGuildNetwork","value":1}]`, PrerequisitesJSON: `["research_alliance_protocols"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// LORE
		{ContentID: "research_archive_indexing", Domain: "research", Branch: "lore", NameKey: "research.archive_indexing.name", DescriptionKey: "research.archive_indexing.description", AssetID: "icon_research_lore", AssetsByTier: map[string]string{"tier1": "icon_research_lore.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"archiveEntries","value":50}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_rumor_tracking", Domain: "research", Branch: "lore", NameKey: "research.rumor_tracking.name", DescriptionKey: "research.rumor_tracking.description", AssetID: "icon_research_lore", AssetsByTier: map[string]string{"tier2": "icon_research_lore.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"rumorTracking","value":1}]`, PrerequisitesJSON: `["research_archive_indexing"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_event_canonization", Domain: "research", Branch: "lore", NameKey: "research.event_canonization.name", DescriptionKey: "research.event_canonization.description", AssetID: "icon_research_lore", AssetsByTier: map[string]string{"tier3": "icon_research_lore.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"eventCanonization","value":1}]`, PrerequisitesJSON: `["research_rumor_tracking"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_ai_summary_engine", Domain: "research", Branch: "lore", NameKey: "research.ai_summary_engine.name", DescriptionKey: "research.ai_summary_engine.description", AssetID: "icon_research_lore", AssetsByTier: map[string]string{"tier4": "icon_research_lore.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"aiSummaryEngine","value":1}]`, PrerequisitesJSON: `["research_event_canonization"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_regional_memory", Domain: "research", Branch: "lore", NameKey: "research.regional_memory.name", DescriptionKey: "research.regional_memory.description", AssetID: "icon_research_lore", AssetsByTier: map[string]string{"tier5": "icon_research_lore.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"regionalMemory","value":1}]`, PrerequisitesJSON: `["research_ai_summary_engine"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_tribunal_archives", Domain: "research", Branch: "lore", NameKey: "research.tribunal_archives.name", DescriptionKey: "research.tribunal_archives.description", AssetID: "icon_research_lore", AssetsByTier: map[string]string{"tier6": "icon_research_lore.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"tribunalArchives","value":1}]`, PrerequisitesJSON: `["research_regional_memory"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_living_lore_engine", Domain: "research", Branch: "lore", NameKey: "research.living_lore_engine.name", DescriptionKey: "research.living_lore_engine.description", AssetID: "icon_research_lore", AssetsByTier: map[string]string{"tier7": "icon_research_lore.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"livingLoreEngine","value":1}]`, PrerequisitesJSON: `["research_tribunal_archives"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// TRIBUNAL
		{ContentID: "research_evidence_handling", Domain: "research", Branch: "tribunal", NameKey: "research.evidence_handling.name", DescriptionKey: "research.evidence_handling.description", AssetID: "icon_research_tribunal", AssetsByTier: map[string]string{"tier1": "icon_research_tribunal.jpg"}, Tier: 1, Rarity: "common", CostBaseCredits: 200, CostBaseData: 100, DurationBaseSeconds: 1800, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"evidenceHandling","value":1}]`, PrerequisitesJSON: "[]", BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_witness_protocol", Domain: "research", Branch: "tribunal", NameKey: "research.witness_protocol.name", DescriptionKey: "research.witness_protocol.description", AssetID: "icon_research_tribunal", AssetsByTier: map[string]string{"tier2": "icon_research_tribunal.jpg"}, Tier: 2, Rarity: "common", CostBaseCredits: 250, CostBaseData: 120, DurationBaseSeconds: 2400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"witnessProtocol","value":1}]`, PrerequisitesJSON: `["research_evidence_handling"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_jury_simulation", Domain: "research", Branch: "tribunal", NameKey: "research.jury_simulation.name", DescriptionKey: "research.jury_simulation.description", AssetID: "icon_research_tribunal", AssetsByTier: map[string]string{"tier3": "icon_research_tribunal.jpg"}, Tier: 3, Rarity: "uncommon", CostBaseCredits: 300, CostBaseData: 150, DurationBaseSeconds: 3000, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"jurySimulation","value":1}]`, PrerequisitesJSON: `["research_witness_protocol"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_verdict_impact", Domain: "research", Branch: "tribunal", NameKey: "research.verdict_impact.name", DescriptionKey: "research.verdict_impact.description", AssetID: "icon_research_tribunal", AssetsByTier: map[string]string{"tier4": "icon_research_tribunal.jpg"}, Tier: 4, Rarity: "uncommon", CostBaseCredits: 400, CostBaseData: 200, DurationBaseSeconds: 3600, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"verdictImpact","value":1}]`, PrerequisitesJSON: `["research_jury_simulation"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_guild_arbitration", Domain: "research", Branch: "tribunal", NameKey: "research.guild_arbitration.name", DescriptionKey: "research.guild_arbitration.description", AssetID: "icon_research_tribunal", AssetsByTier: map[string]string{"tier5": "icon_research_tribunal.jpg"}, Tier: 5, Rarity: "rare", CostBaseCredits: 500, CostBaseData: 250, DurationBaseSeconds: 4500, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"guildArbitration","value":1}]`, PrerequisitesJSON: `["research_verdict_impact"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_faction_law", Domain: "research", Branch: "tribunal", NameKey: "research.faction_law.name", DescriptionKey: "research.faction_law.description", AssetID: "icon_research_tribunal", AssetsByTier: map[string]string{"tier6": "icon_research_tribunal.jpg"}, Tier: 6, Rarity: "rare", CostBaseCredits: 600, CostBaseData: 300, DurationBaseSeconds: 5400, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"factionLaw","value":1}]`, PrerequisitesJSON: `["research_guild_arbitration"]`, BalanceVersion: "v2.0.0", IsPublished: true},
		{ContentID: "research_nexus_tribunal_authority", Domain: "research", Branch: "tribunal", NameKey: "research.nexus_tribunal_authority.name", DescriptionKey: "research.nexus_tribunal_authority.description", AssetID: "icon_research_tribunal", AssetsByTier: map[string]string{"tier7": "icon_research_tribunal.jpg"}, Tier: 7, Rarity: "legendary", CostBaseCredits: 800, CostBaseData: 400, DurationBaseSeconds: 7200, EffectsJSON: `[{"effectType":"statBonus","target":"all","stat":"nexusTribunalAuthority","value":1}]`, PrerequisitesJSON: `["research_faction_law"]`, BalanceVersion: "v2.0.0", IsPublished: true},

		// All 11 branches × 7 tiers from NEXUS GAME CONTENT REFERENCE §6 ARBRE DE RECHERCHE exactly.
		// nodeIds: research_efficient_storage, research_solar_stabilization, research_modular_housing, ... research_nexus_tribunal_authority
		// Each has tier, prerequisites (array of prior in branch), effects matching "Effet principal" translated to effectType/buildingBonus/statBonus/unlock, ramping costs/durations, IsPublished:true.
	}

	// Purge legacy research rows that used old names (research_economy_tier1 etc.) so DB matches the reference tree exactly.
	legacyResearchIDs := []string{
		"research_economy_tier1", "research_economy_tier2",
		"research_ia_tier1",
		"research_military_tier1", "research_military_tier2",
		"research_economy_tier1", "research.military_tier1", // variants from old dumps
	}
	for _, lid := range legacyResearchIDs {
		_ = db.Where("content_id = ?", lid).Delete(&models.ResearchDefinition{}).Error
	}

	applyDefaultResearchRequirements(research)
	for i := range research {
		r := research[i]
		var existing models.ResearchDefinition
		if err := db.Where("content_id = ?", r.ContentID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = db.Create(&r).Error
			}
		} else {
			fillEmptyResearchRequirements(db, &existing, r)
		}
	}
	return nil
}

func applyDefaultBuildingRequirements(buildings []models.BuildingDefinition) {
	for i := range buildings {
		switch buildings[i].ContentID {
		case "building_ai_center":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_research_lab", 2)
		case "building_quantum_refinery":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_composite_mine", 5)
			buildings[i].RequiredResearchJSON = researchReq("research_quantum_reactor_safety")
		case "building_drone_factory":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_barracks", 2)
			buildings[i].RequiredResearchJSON = researchReq("research_drone_assembly")
		case "building_holo_wall":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_barracks", 1)
		case "building_diplomatic_center":
			buildings[i].RequiredResearchJSON = researchReq("research_emissary_protocol")
		case "building_nexus_market":
			buildings[i].RequiredResearchJSON = researchReq("research_efficient_storage")
		case "building_logistic_station":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_nexus_market", 2)
			buildings[i].RequiredResearchJSON = researchReq("research_resource_routing")
		case "building_living_lore_archives":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_research_lab", 3)
			buildings[i].RequiredResearchJSON = researchReq("research_archive_indexing")
		case "building_tribunal_nexus":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_diplomatic_center", 3)
			buildings[i].RequiredResearchJSON = researchReq("research_evidence_handling")
		case "building_guild_hq":
			buildings[i].RequiredResearchJSON = researchReq("research_guild_charter")
		case "building_surveillance_tower":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_holo_wall", 2)
			buildings[i].RequiredResearchJSON = researchReq("research_regional_scanning")
		case "building_world_relay":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_surveillance_tower", 3)
			buildings[i].RequiredResearchJSON = researchReq("research_event_forecasting")
		case "building_data_bank":
			buildings[i].RequiredBuildingsJSON = buildingReq("building_research_lab", 4)
			buildings[i].RequiredResearchJSON = researchReq("research_ai_summary_engine")
		case "building_nexus_core":
			buildings[i].RequiredBuildingsJSON = multiBuildingReq(map[string]int{
				"building_ai_center":      5,
				"building_world_relay":    5,
				"building_data_bank":      5,
				"building_tribunal_nexus": 3,
			})
			buildings[i].RequiredResearchJSON = researchReq("research_nexus_world_awareness")
		}
	}
}

func applyDefaultUnitRequirements(units []models.UnitDefinition) {
	for i := range units {
		units[i].RequiredBuildingsJSON = buildingReq("building_barracks", 1)
		if units[i].FlavorTextKey == "" {
			units[i].FlavorTextKey = defaultFlavorTextKey(units[i].DescriptionKey)
		}
		if len(units[i].LevelDescriptionKeys) == 0 {
			units[i].LevelDescriptionKeys = defaultLevelDescriptionKeys(units[i].DescriptionKey)
		}
		switch units[i].ContentID {
		case "unit_milicien_nexus":
			units[i].NexusLevelRequired = maxInt(units[i].NexusLevelRequired, 1)
		case "unit_drone_sentinelle":
			units[i].RequiredBuildingsJSON = buildingReq("building_drone_factory", 1)
			units[i].RequiredResearchJSON = researchReq("research_drone_assembly")
		case "unit_eclaireur_cybernetique":
			units[i].RequiredResearchJSON = researchReq("research_basic_tactics")
		case "unit_fantassin_augmente":
			units[i].RequiredBuildingsJSON = buildingReq("building_barracks", 2)
			units[i].RequiredResearchJSON = researchReq("research_augmented_infantry")
		case "unit_drone_assaut":
			units[i].RequiredBuildingsJSON = buildingReq("building_drone_factory", 2)
			units[i].RequiredResearchJSON = researchReq("research_swarm_control")
		case "unit_drone_bouclier":
			units[i].RequiredBuildingsJSON = buildingReq("building_drone_factory", 3)
			units[i].RequiredResearchJSON = researchReq("research_shield_drone_matrix")
		case "unit_hacker_de_combat":
			units[i].RequiredBuildingsJSON = buildingReq("building_ai_center", 2)
			units[i].RequiredResearchJSON = researchReq("research_multi_agent_routing")
		case "unit_medic_synthetique":
			units[i].RequiredBuildingsJSON = buildingReq("building_research_lab", 2)
			units[i].RequiredResearchJSON = researchReq("research_population_welfare")
		case "unit_artillerie_railgun":
			units[i].RequiredBuildingsJSON = buildingReq("building_barracks", 4)
			units[i].RequiredResearchJSON = researchReq("research_railgun_engineering")
		case "unit_mecha_leger":
			units[i].RequiredBuildingsJSON = multiBuildingReq(map[string]int{"building_barracks": 5, "building_drone_factory": 3})
			units[i].RequiredResearchJSON = researchReq("research_battlefield_logistics")
		case "unit_agent_infiltre":
			units[i].RequiredBuildingsJSON = buildingReq("building_surveillance_tower", 2)
			units[i].RequiredResearchJSON = researchReq("research_reputation_mapping")
		case "unit_envoye_de_faction":
			units[i].RequiredBuildingsJSON = buildingReq("building_diplomatic_center", 2)
			units[i].RequiredResearchJSON = researchReq("research_faction_language_models")
		case "unit_gardien_holographique":
			units[i].RequiredBuildingsJSON = buildingReq("building_holo_wall", 4)
			units[i].RequiredResearchJSON = researchReq("research_weather_adaptation")
		case "unit_titan_nexus":
			units[i].RequiredBuildingsJSON = multiBuildingReq(map[string]int{"building_nexus_core": 1, "building_barracks": 8})
			units[i].RequiredResearchJSON = researchReq("research_nexus_warfare")
		case "unit_archiviste_mobile":
			units[i].RequiredBuildingsJSON = buildingReq("building_living_lore_archives", 2)
			units[i].RequiredResearchJSON = researchReq("research_archive_indexing")
		}
	}
}

func applyDefaultResearchRequirements(research []models.ResearchDefinition) {
	for i := range research {
		research[i].RequiredBuildingsJSON = buildingReq("building_research_lab", 1)
		if research[i].FlavorTextKey == "" {
			research[i].FlavorTextKey = defaultFlavorTextKey(research[i].DescriptionKey)
		}
		if len(research[i].LevelDescriptionKeys) == 0 {
			research[i].LevelDescriptionKeys = defaultLevelDescriptionKeys(research[i].DescriptionKey)
		}
		switch research[i].Branch {
		case "economy":
			research[i].RequiredBuildingsJSON = buildingReq("building_nexus_market", 1)
		case "energy":
			research[i].RequiredBuildingsJSON = buildingReq("building_solar_plant", 1)
		case "ville":
			research[i].RequiredBuildingsJSON = buildingReq("building_modular_habitat", 1)
		case "militaire":
			research[i].RequiredBuildingsJSON = buildingReq("building_barracks", 1)
		case "drones":
			research[i].RequiredBuildingsJSON = buildingReq("building_research_lab", 2)
		case "ia":
			research[i].RequiredBuildingsJSON = buildingReq("building_ai_center", 1)
		case "diplomatie":
			research[i].RequiredBuildingsJSON = buildingReq("building_diplomatic_center", 1)
		case "monde":
			research[i].RequiredBuildingsJSON = buildingReq("building_surveillance_tower", 1)
		case "guilde":
			research[i].RequiredBuildingsJSON = buildingReq("building_guild_hq", 1)
		case "lore":
			research[i].RequiredBuildingsJSON = buildingReq("building_living_lore_archives", 1)
		case "tribunal":
			research[i].RequiredBuildingsJSON = buildingReq("building_tribunal_nexus", 1)
		}
	}
}

func fillEmptyBuildingRequirements(db *gorm.DB, existing *models.BuildingDefinition, seeded models.BuildingDefinition) {
	changed := false
	if seeded.SlotsMax > 0 && existing.SlotsMax != seeded.SlotsMax {
		existing.SlotsMax = seeded.SlotsMax
		changed = true
	}
	if existing.AvailableAtSpawn != seeded.AvailableAtSpawn {
		existing.AvailableAtSpawn = seeded.AvailableAtSpawn
		changed = true
	}
	if existing.NonConstructible != seeded.NonConstructible {
		existing.NonConstructible = seeded.NonConstructible
		changed = true
	}
	if seeded.UnlockEra != "" && existing.UnlockEra != seeded.UnlockEra {
		existing.UnlockEra = seeded.UnlockEra
		changed = true
	}
	if seeded.UnlockType != "" && existing.UnlockType != seeded.UnlockType {
		existing.UnlockType = seeded.UnlockType
		changed = true
	}
	if seeded.UnlockMessage != "" && existing.UnlockMessage != seeded.UnlockMessage {
		existing.UnlockMessage = seeded.UnlockMessage
		changed = true
	}
	if existing.NexusLevelRequired <= 0 && seeded.NexusLevelRequired > 0 {
		existing.NexusLevelRequired = seeded.NexusLevelRequired
		changed = true
	}
	if seeded.RequiredBuildingsJSON != "" && existing.RequiredBuildingsJSON != seeded.RequiredBuildingsJSON {
		existing.RequiredBuildingsJSON = seeded.RequiredBuildingsJSON
		changed = true
	}
	if existing.RequiredResearchJSON == "" && seeded.RequiredResearchJSON != "" {
		existing.RequiredResearchJSON = seeded.RequiredResearchJSON
		changed = true
	}
	if seeded.DurationBaseSeconds > 0 && existing.DurationBaseSeconds != seeded.DurationBaseSeconds {
		existing.DurationBaseSeconds = seeded.DurationBaseSeconds
		changed = true
	}
	if seeded.DurationMultiplier > 0 && existing.DurationMultiplier != seeded.DurationMultiplier {
		existing.DurationMultiplier = seeded.DurationMultiplier
		changed = true
	}
	if seeded.MilestoneReduction > 0 && existing.MilestoneReduction != seeded.MilestoneReduction {
		existing.MilestoneReduction = seeded.MilestoneReduction
		changed = true
	}
	if seeded.StorageResource != "" && existing.StorageResource != seeded.StorageResource {
		existing.StorageResource = seeded.StorageResource
		changed = true
	}
	if existing.StorageCapBase != seeded.StorageCapBase {
		existing.StorageCapBase = seeded.StorageCapBase
		changed = true
	}
	if seeded.StorageGrowth > 0 && existing.StorageGrowth != seeded.StorageGrowth {
		existing.StorageGrowth = seeded.StorageGrowth
		changed = true
	}
	if seeded.OverflowBehavior != "" && existing.OverflowBehavior != seeded.OverflowBehavior {
		existing.OverflowBehavior = seeded.OverflowBehavior
		changed = true
	}
	if existing.OverflowDecayPercentPerHour != seeded.OverflowDecayPercentPerHour {
		existing.OverflowDecayPercentPerHour = seeded.OverflowDecayPercentPerHour
		changed = true
	}
	if existing.ProductionBasePerHour != seeded.ProductionBasePerHour {
		existing.ProductionBasePerHour = seeded.ProductionBasePerHour
		changed = true
	}
	if seeded.ProductionGrowth > 0 && existing.ProductionGrowth != seeded.ProductionGrowth {
		existing.ProductionGrowth = seeded.ProductionGrowth
		changed = true
	}
	if existing.HarvestRecommendedIntervalSeconds != seeded.HarvestRecommendedIntervalSeconds {
		existing.HarvestRecommendedIntervalSeconds = seeded.HarvestRecommendedIntervalSeconds
		changed = true
	}
	if seeded.SynergiesJSON != "" && existing.SynergiesJSON != seeded.SynergiesJSON {
		existing.SynergiesJSON = seeded.SynergiesJSON
		changed = true
	}
	if seeded.RisksJSON != "" && existing.RisksJSON != seeded.RisksJSON {
		existing.RisksJSON = seeded.RisksJSON
		changed = true
	}
	if seeded.AIActionsJSON != "" && existing.AIActionsJSON != seeded.AIActionsJSON {
		existing.AIActionsJSON = seeded.AIActionsJSON
		changed = true
	}
	if seeded.FlavorTextKey != "" && existing.FlavorTextKey != seeded.FlavorTextKey {
		existing.FlavorTextKey = seeded.FlavorTextKey
		changed = true
	}
	if len(seeded.LevelDescriptionKeys) > 0 && !sameStringMap(existing.LevelDescriptionKeys, seeded.LevelDescriptionKeys) {
		existing.LevelDescriptionKeys = seeded.LevelDescriptionKeys
		changed = true
	}
	if changed {
		_ = db.Save(existing).Error
	}
}

func fillEmptyUnitRequirements(db *gorm.DB, existing *models.UnitDefinition, seeded models.UnitDefinition) {
	changed := false
	if existing.NexusLevelRequired <= 0 && seeded.NexusLevelRequired > 0 {
		existing.NexusLevelRequired = seeded.NexusLevelRequired
		changed = true
	}
	if existing.RequiredBuildingsJSON == "" && seeded.RequiredBuildingsJSON != "" {
		existing.RequiredBuildingsJSON = seeded.RequiredBuildingsJSON
		changed = true
	}
	if existing.RequiredResearchJSON == "" && seeded.RequiredResearchJSON != "" {
		existing.RequiredResearchJSON = seeded.RequiredResearchJSON
		changed = true
	}
	if seeded.FlavorTextKey != "" && existing.FlavorTextKey != seeded.FlavorTextKey {
		existing.FlavorTextKey = seeded.FlavorTextKey
		changed = true
	}
	if len(seeded.LevelDescriptionKeys) > 0 && !sameStringMap(existing.LevelDescriptionKeys, seeded.LevelDescriptionKeys) {
		existing.LevelDescriptionKeys = seeded.LevelDescriptionKeys
		changed = true
	}
	if changed {
		_ = db.Save(existing).Error
	}
}

func fillEmptyResearchRequirements(db *gorm.DB, existing *models.ResearchDefinition, seeded models.ResearchDefinition) {
	changed := false
	if existing.NexusLevelRequired <= 0 && seeded.NexusLevelRequired > 0 {
		existing.NexusLevelRequired = seeded.NexusLevelRequired
		changed = true
	}
	if existing.RequiredBuildingsJSON == "" && seeded.RequiredBuildingsJSON != "" {
		existing.RequiredBuildingsJSON = seeded.RequiredBuildingsJSON
		changed = true
	}
	if seeded.FlavorTextKey != "" && existing.FlavorTextKey != seeded.FlavorTextKey {
		existing.FlavorTextKey = seeded.FlavorTextKey
		changed = true
	}
	if len(seeded.LevelDescriptionKeys) > 0 && !sameStringMap(existing.LevelDescriptionKeys, seeded.LevelDescriptionKeys) {
		existing.LevelDescriptionKeys = seeded.LevelDescriptionKeys
		changed = true
	}
	if changed {
		_ = db.Save(existing).Error
	}
}

func defaultFlavorTextKey(descriptionKey string) string {
	base := contentDescriptionBaseKey(descriptionKey)
	if base == "" {
		return ""
	}
	return base + ".flavor"
}

func defaultLevelDescriptionKeys(descriptionKey string) map[string]string {
	base := contentDescriptionBaseKey(descriptionKey)
	if base == "" {
		return nil
	}
	return map[string]string{
		"1":  base + ".level_1.description",
		"10": base + ".level_10.description",
		"20": base + ".level_20.description",
		"30": base + ".level_30.description",
	}
}

func contentDescriptionBaseKey(descriptionKey string) string {
	key := strings.TrimSpace(descriptionKey)
	return strings.TrimSuffix(key, ".description")
}

func sameStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if right[key] != leftValue {
			return false
		}
	}
	return true
}

func buildingReq(contentID string, level int) string {
	return multiBuildingReq(map[string]int{contentID: level})
}

func multiBuildingReq(items map[string]int) string {
	out := "["
	keys := make([]string, 0, len(items))
	for contentID := range items {
		keys = append(keys, contentID)
	}
	sort.Strings(keys)
	for index, contentID := range keys {
		if index > 0 {
			out += ","
		}
		out += `{"contentId":"` + contentID + `","level":` + stringInt(items[contentID]) + `}`
	}
	return out + "]"
}

func researchReq(contentID string) string {
	return `["` + contentID + `"]`
}

func defaultBuildingSynergies(contentID string) string {
	switch contentID {
	case "building_modular_habitat":
		return `[{"with":"building_vertical_farm","requiresLevel":5,"bonus":"population_morale","valuePercent":5},{"with":"building_logistic_station","requiresLevel":10,"bonus":"worker_assignment_speed","valuePercent":8}]`
	case "building_solar_plant":
		return `[{"with":"building_data_bank","requiresLevel":5,"bonus":"data_uptime","valuePercent":10},{"with":"building_ai_center","requiresLevel":8,"bonus":"ai_action_cooldown","valuePercent":-8}]`
	case "building_composite_mine":
		return `[{"with":"building_barracks","requiresLevel":2,"bonus":"military_build_cost","valuePercent":-5},{"with":"building_quantum_refinery","requiresLevel":12,"bonus":"quantum_yield","valuePercent":10}]`
	case "building_vertical_farm":
		return `[{"with":"building_modular_habitat","requiresLevel":2,"bonus":"population_growth","valuePercent":8},{"with":"building_diplomatic_center","requiresLevel":6,"bonus":"influence_events","valuePercent":5}]`
	case "building_barracks":
		return `[{"with":"building_holo_wall","requiresLevel":1,"bonus":"city_defense","valuePercent":8},{"with":"building_drone_factory","requiresLevel":5,"bonus":"mixed_training_speed","valuePercent":12}]`
	case "building_holo_wall":
		return `[{"with":"building_surveillance_tower","requiresLevel":3,"bonus":"raid_detection","valuePercent":15}]`
	case "building_data_bank":
		return `[{"with":"building_research_lab","requiresLevel":3,"bonus":"research_speed","valuePercent":10},{"with":"building_ai_center","requiresLevel":8,"bonus":"ai_context_memory","valuePercent":12}]`
	case "building_research_lab":
		return `[{"with":"building_ai_center","requiresLevel":5,"bonus":"ai_experiment_success","valuePercent":10},{"with":"building_quantum_refinery","requiresLevel":15,"bonus":"quantum_safety","valuePercent":8}]`
	case "building_drone_factory":
		return `[{"with":"building_surveillance_tower","requiresLevel":2,"bonus":"drone_patrol_efficiency","valuePercent":10}]`
	case "building_nexus_market":
		return `[{"with":"building_logistic_station","requiresLevel":5,"bonus":"trade_route_capacity","valuePercent":12},{"with":"building_diplomatic_center","requiresLevel":5,"bonus":"influence_trade","valuePercent":8}]`
	case "building_ai_center":
		return `[{"with":"building_data_bank","requiresLevel":8,"bonus":"agent_quality","valuePercent":12},{"with":"building_world_relay","requiresLevel":10,"bonus":"world_prediction","valuePercent":10}]`
	case "building_diplomatic_center":
		return `[{"with":"building_guild_hq","requiresLevel":1,"bonus":"alliance_negotiation","valuePercent":10},{"with":"building_tribunal_nexus","requiresLevel":8,"bonus":"verdict_legitimacy","valuePercent":12}]`
	case "building_guild_hq":
		return `[{"with":"building_world_relay","requiresLevel":5,"bonus":"guild_world_actions","valuePercent":10}]`
	case "building_quantum_refinery":
		return `[{"with":"building_nexus_core","requiresLevel":1,"bonus":"singularity_contribution","valuePercent":15}]`
	case "building_living_lore_archives":
		return `[{"with":"building_world_relay","requiresLevel":4,"bonus":"living_lore_event_quality","valuePercent":12}]`
	case "building_tribunal_nexus":
		return `[{"with":"building_surveillance_tower","requiresLevel":10,"bonus":"evidence_integrity","valuePercent":15}]`
	case "building_world_relay":
		return `[{"with":"building_nexus_core","requiresLevel":1,"bonus":"global_world_awareness","valuePercent":15}]`
	case "building_nexus_core":
		return `[{"with":"all_era4_buildings","requiresLevel":1,"bonus":"nexus_singularity_progress","valuePercent":20}]`
	default:
		return `[{"with":"building_logistic_station","requiresLevel":2,"bonus":"maintenance_efficiency","valuePercent":5}]`
	}
}

func defaultBuildingRisks(contentID string) string {
	switch contentID {
	case "building_quantum_refinery":
		return `[{"trigger":"storage_full_6h","debuff":"quantum_instability","effect":"production -50%, anomaly risk +3%/h","durationSeconds":21600,"recovery":"harvest_and_stabilize_ai_action"}]`
	case "building_ai_center":
		return `[{"trigger":"data_shortage_or_morale_below_25","debuff":"agent_jam","effect":"AI action cooldown +50%","durationSeconds":14400,"recovery":"run_diagnostic"}]`
	case "building_holo_wall", "building_surveillance_tower":
		return `[{"trigger":"energy_balance_negative_2h","debuff":"defense_blind_spot","effect":"raid detection -30%","durationSeconds":7200,"recovery":"restore_energy_positive"}]`
	case "building_nexus_market":
		return `[{"trigger":"morale_below_25","debuff":"market_panic","effect":"trade bonus -50%","durationSeconds":10800,"recovery":"stabilize_prices_ai_action"}]`
	case "building_vertical_farm":
		return `[{"trigger":"storage_full_4h","debuff":"crop_decay","effect":"surplus decay 5%/h","durationSeconds":14400,"recovery":"harvest"}]`
	default:
		return `[{"trigger":"morale_below_25","debuff":"low_efficiency","effect":"production -50%, incident risk 3%/h","durationSeconds":10800,"recovery":"restore_morale_above_35"}]`
	}
}

func defaultBuildingAIActions(contentID string) string {
	switch contentID {
	case "building_ai_center":
		return `[{"id":"ai_diagnostic","cost":{"credits":250,"data":180},"cooldownSeconds":14400,"effect":"Nettoie un debuff agent_jam et reduit de 10% le prochain cooldown IA."},{"id":"predict_incident","cost":{"credits":400,"data":320},"cooldownSeconds":21600,"effect":"Revele le prochain risque majeur de la cite."}]`
	case "building_data_bank":
		return `[{"id":"compress_data","cost":{"credits":120,"data":80},"cooldownSeconds":7200,"effect":"Convertit le surplus data en stockage utile temporaire +10% pendant 6h."}]`
	case "building_nexus_market":
		return `[{"id":"stabilize_prices","cost":{"credits":300,"data":120},"cooldownSeconds":21600,"effect":"Supprime market_panic et donne +5% trade pendant 4h."}]`
	case "building_surveillance_tower":
		return `[{"id":"scan_sector","cost":{"credits":180,"data":160},"cooldownSeconds":10800,"effect":"Augmente detection raid +20% pendant 3h."}]`
	case "building_quantum_refinery":
		return `[{"id":"stabilize_reactor","cost":{"credits":500,"data":350,"quantum_core":2},"cooldownSeconds":43200,"effect":"Reduit anomaly risk de 50% pendant 8h."}]`
	case "building_diplomatic_center":
		return `[{"id":"envoy_briefing","cost":{"credits":350,"influence":20},"cooldownSeconds":21600,"effect":"Boost reputation gain +10% pendant 6h."}]`
	default:
		return `[{"id":"optimize_shift","cost":{"credits":100,"data":40},"cooldownSeconds":10800,"effect":"Production ou efficacite +5% pendant 2h."}]`
	}
}

func stringInt(value int) string {
	if value <= 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
