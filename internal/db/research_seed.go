package db

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type researchSeedLevel struct {
	Level             int `json:"level"`
	DurationMinutes   int `json:"durationMinutes"`
	CumulativeMinutes int `json:"cumulativeMinutes"`
}

type researchSeedNode struct {
	DomainKey   string
	Domain      string
	BuildingKey string
	Key         string
	Name        string
	Description string
	Resources   []string
	SortOrder   int
	Progression []researchSeedLevel
}

func seedDefaultResearchSystem(db *gorm.DB) error {
	nodes, err := parseResearchSeedRows(defaultResearchSeedRows)
	if err != nil {
		return err
	}

	trees := map[string]models.ResearchTreeDefinition{}
	resourceSeen := map[string]string{}
	for _, node := range nodes {
		if _, ok := trees[node.DomainKey]; !ok {
			trees[node.DomainKey] = models.ResearchTreeDefinition{
				Key:         node.DomainKey,
				Name:        node.Domain,
				Description: "Arbre de recherche " + strings.ToLower(node.Domain) + ".",
				Domain:      node.Domain,
				BuildingKey: node.BuildingKey,
				IsActive:    true,
				SortOrder:   len(trees)*10 + 10,
			}
		}
		for _, resource := range node.Resources {
			resourceSeen[researchSeedSlug(resource)] = resource
		}
	}

	resourceIndex := 10
	for key, name := range resourceSeen {
		if err := seedResourceDefinition(db, models.ResourceDefinition{
			Key:       key,
			Name:      name,
			Category:  "research",
			IsActive:  true,
			SortOrder: resourceIndex,
		}); err != nil {
			return err
		}
		resourceIndex += 10
	}

	treeIDs := map[string]uint{}
	for _, seed := range trees {
		tree, err := seedResearchTree(db, seed)
		if err != nil {
			return err
		}
		treeIDs[seed.Key] = tree.Id
	}

	for index, seed := range nodes {
		treeID := treeIDs[seed.DomainKey]
		resourcesJSON, err := json.Marshal(seed.Resources)
		if err != nil {
			return err
		}
		progressionJSON, err := json.Marshal(seed.Progression)
		if err != nil {
			return err
		}
		node := models.ResearchNodeDefinition{
			ResearchTreeDefinitionID: treeID,
			Key:                      seed.Key,
			Name:                     seed.Name,
			Description:              seed.Description,
			Domain:                   seed.Domain,
			Branch:                   seed.Name,
			ResourcesJSON:            datatypes.JSON(resourcesJSON),
			ParentKeysJSON:           jsonValue(`[]`),
			RequirementsJSON:         jsonValue(`{}`),
			EffectsJSON:              jsonValue(`{}`),
			LevelProgressionJSON:     datatypes.JSON(progressionJSON),
			MaxLevel:                 len(seed.Progression),
			PositionX:                (index % 5) * 260,
			PositionY:                (index / 5) * 180,
			IsActive:                 true,
			SortOrder:                seed.SortOrder,
		}
		if err := seedResearchNode(db, node); err != nil {
			return err
		}
	}
	return nil
}

func seedResourceDefinition(db *gorm.DB, seed models.ResourceDefinition) error {
	var existing models.ResourceDefinition
	err := db.Unscoped().Where("`key` = ?", seed.Key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&seed).Error
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
	if len(updates) == 0 {
		return nil
	}
	return db.Unscoped().Model(&existing).Updates(updates).Error
}

func seedResearchTree(db *gorm.DB, seed models.ResearchTreeDefinition) (models.ResearchTreeDefinition, error) {
	var existing models.ResearchTreeDefinition
	err := db.Unscoped().Where("`key` = ?", seed.Key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&seed).Error; err != nil {
			return seed, err
		}
		return seed, nil
	}
	if err != nil {
		return existing, err
	}
	updates := map[string]any{}
	if !existing.IsActive {
		updates["is_active"] = true
	}
	if existing.BuildingKey == "" && seed.BuildingKey != "" {
		updates["building_key"] = seed.BuildingKey
	}
	if existing.DeletedAt.Valid {
		updates["deleted_at"] = nil
	}
	if len(updates) > 0 {
		if err := db.Unscoped().Model(&existing).Updates(updates).Error; err != nil {
			return existing, err
		}
	}
	return existing, nil
}

func seedResearchNode(db *gorm.DB, seed models.ResearchNodeDefinition) error {
	var existing models.ResearchNodeDefinition
	err := db.Unscoped().Where("`key` = ?", seed.Key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&seed).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{}
	if existing.ResearchTreeDefinitionID == 0 {
		updates["research_tree_definition_id"] = seed.ResearchTreeDefinitionID
	}
	if len(existing.LevelProgressionJSON) == 0 {
		updates["level_progression_json"] = seed.LevelProgressionJSON
	}
	if !existing.IsActive {
		updates["is_active"] = true
	}
	if existing.DeletedAt.Valid {
		updates["deleted_at"] = nil
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Unscoped().Model(&existing).Updates(updates).Error
}

func parseResearchSeedRows(raw string) ([]researchSeedNode, error) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	nodes := make([]researchSeedNode, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 9 {
			return nil, errors.New("invalid research seed row")
		}
		sortOrder, err := strconv.Atoi(parts[7])
		if err != nil {
			return nil, err
		}
		progression, err := parseResearchSeedProgression(parts[8])
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, researchSeedNode{
			DomainKey:   parts[0],
			Domain:      parts[1],
			BuildingKey: parts[2],
			Key:         parts[3],
			Name:        parts[4],
			Description: parts[5],
			Resources:   strings.Split(parts[6], ","),
			SortOrder:   sortOrder,
			Progression: progression,
		})
	}
	return nodes, nil
}

func parseResearchSeedProgression(raw string) ([]researchSeedLevel, error) {
	parts := strings.Split(raw, ",")
	levels := make([]researchSeedLevel, 0, len(parts))
	for _, part := range parts {
		values := strings.Split(part, ":")
		if len(values) != 3 {
			return nil, errors.New("invalid research seed level")
		}
		level, err := strconv.Atoi(values[0])
		if err != nil {
			return nil, err
		}
		duration, err := strconv.Atoi(values[1])
		if err != nil {
			return nil, err
		}
		cumulative, err := strconv.Atoi(values[2])
		if err != nil {
			return nil, err
		}
		levels = append(levels, researchSeedLevel{
			Level:             level,
			DurationMinutes:   duration,
			CumulativeMinutes: cumulative,
		})
	}
	return levels, nil
}

func researchSeedSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(
		"à", "a", "â", "a", "ä", "a",
		"ç", "c",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"î", "i", "ï", "i",
		"ô", "o", "ö", "o",
		"ù", "u", "û", "u", "ü", "u",
		" ", "_", "'", "_", "&", "_",
	).Replace(value)
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return strings.Trim(value, "_")
}

const defaultResearchSeedRows = `
stabilite_civile|STABILITÉ CIVILE|city_hall|securite_publique|Sécurité Publique|Contrôle des forces de l'ordre et maintien de l'ordre public|Population,Autorité|10|1:30:30,2:60:90,3:90:180,4:120:300,5:180:480,6:240:720,7:300:1020,8:420:1440,9:540:2016,10:720:2736,11:960:3600,12:1200:4896,13:1440:6336,14:1872:8208,15:2448:10656,16:3024:13680,17:3888:17568,18:5040:22608
stabilite_civile|STABILITÉ CIVILE|city_hall|gouvernance|Gouvernance|Systèmes politiques et administration de la cité|Population,Diplomatie|20|1:60:60,2:90:150,3:120:270,4:180:450,5:240:690,6:300:990,7:420:1410,8:600:2016,9:780:2736,10:960:3744,11:1200:4896,12:1584:6480,13:1872:8496,14:2448:10800,15:3024:13824,16:3888:17712,17:4752:22464,18:6048:28512,19:7776:36288,20:9648:47520
stabilite_civile|STABILITÉ CIVILE|city_hall|services_publics|Services Publics|Santé, éducation, services essentiels|Population,Infrastructure|30|1:45:45,2:78:120,3:108:228,4:150:378,5:210:588,6:300:888,7:360:1248,8:480:1728,9:600:2304,10:780:3168,11:1020:4176,12:1320:5472,13:1728:7056,14:2160:9216,15:2736:12096,16:3600:15696
stabilite_civile|STABILITÉ CIVILE|city_hall|infrastructure_urbaine|Infrastructure Urbaine|Routes, transports, réseaux de communication|Ressources,Ingénierie|40|1:60:60,2:90:150,3:120:270,4:180:450,5:240:690,6:300:990,7:420:1410,8:540:2016,9:720:2736,10:900:3600,11:1200:4752,12:1584:6336,13:2016:8352,14:2592:10944,15:3456:14544
prosperite_economique|PROSPÉRITÉ ÉCONOMIQUE|trade_hub|commerce_local|Commerce Local|Échanges et commerce intérieurs|Richesse,Population|50|1:30:30,2:45:78,3:60:138,4:90:228,5:120:348,6:180:528,7:240:768,8:300:1068,9:420:1440,10:540:2016,11:660:2736,12:900:3600,13:1200:4752,14:1584:6336,15:2016:8352,16:2736:11088
prosperite_economique|PROSPÉRITÉ ÉCONOMIQUE|trade_hub|commerce_international|Commerce International|Routes commerciales internationales et échanges mondiaux|Richesse,Diplomatie,Routes|60|1:90:90,2:120:210,3:180:390,4:240:630,5:300:930,6:420:1350,7:540:1872,8:720:2592,9:960:3600,10:1200:4752,11:1584:6336,12:1872:8208,13:2448:10656,14:3168:13824,15:3888:17712,16:4752:22464,17:6048:28512,18:7488:36000
prosperite_economique|PROSPÉRITÉ ÉCONOMIQUE|trade_hub|industries_manufacturieres|Industries Manufacturières|Production industrielle et manufactures|Richesse,Ingénierie,Ressources|70|1:60:60,2:90:150,3:120:270,4:180:450,5:240:690,6:300:990,7:420:1410,8:540:2016,9:660:2592,10:840:3456,11:1080:4464,12:1380:5904,13:1728:7632,14:2160:9936,15:2736:12672,16:3456:16128,17:4320:20448
prosperite_economique|PROSPÉRITÉ ÉCONOMIQUE|trade_hub|finance_avancee|Finance Avancée|Systèmes bancaires, investissements, économie|Richesse,Mathématiques|80|1:90:90,2:150:240,3:210:450,4:300:750,5:420:1170,6:600:1728,7:780:2592,8:1020:3600,9:1260:4896,10:1584:6480,11:2016:8496,12:2592:11088,13:3312:14256,14:4032:18432,15:5184:23616
durabilite_energie|DURABILITÉ & ÉNERGIE|solar_park|energies_renouvelables|Énergies Renouvelables|Panneaux solaires, éoliennes, hydroélectricité|Énergie,Ingénierie,Ressources|90|1:60:60,2:90:150,3:120:270,4:180:450,5:240:690,6:360:1050,7:480:1584,8:600:2160,9:780:2880,10:960:3888,11:1200:5040,12:1584:6624,13:2016:8640,14:2592:11088,15:3168:14256,16:3888:18144,17:4752:22752,18:5760:28512
durabilite_energie|DURABILITÉ & ÉNERGIE|solar_park|gestion_de_l_environnement|Gestion de l'Environnement|Protection écologique et gestion des déchets|Population,Ressources|100|1:45:45,2:78:120,3:108:228,4:150:378,5:210:588,6:300:888,7:390:1278,8:480:1728,9:600:2304,10:780:3168,11:960:4032,12:1200:5328,13:1584:6912,14:2016:8784,15:2592:11376,16:3312:14544
durabilite_energie|DURABILITÉ & ÉNERGIE|solar_park|agriculture_durable|Agriculture Durable|Agricultures bio et techniques durables|Ressources,Population|110|1:45:45,2:60:108,3:90:198,4:120:318,5:180:498,6:240:738,7:330:1068,8:420:1440,9:540:2016,10:660:2736,11:840:3456,12:1080:4608,13:1380:6048,14:1872:7776,15:2304:10080
durabilite_energie|DURABILITÉ & ÉNERGIE|solar_park|recuperation_d_energie|Récupération d'Énergie|Récupération et stockage d'énergie|Énergie,Science,Ingénierie|120|1:90:90,2:120:210,3:180:390,4:270:660,5:360:1020,6:480:1440,7:630:2160,8:780:2880,9:960:3888,10:1200:5040,11:1440:6624,12:1872:8496,13:2304:10656,14:2880:13536,15:3456:16992,16:4320:21312
diplomatie_influence|DIPLOMATIE & INFLUENCE|diplomacy_center|relations_diplomatiques|Relations Diplomatiques|Traités, alliances, négociations|Population,Diplomatie|130|1:60:60,2:90:150,3:120:270,4:180:450,5:240:690,6:330:1020,7:420:1440,8:540:2016,9:660:2592,10:840:3456,11:1080:4608,12:1380:5904,13:1728:7632,14:2160:9936,15:2880:12672,16:3600:16272
diplomatie_influence|DIPLOMATIE & INFLUENCE|diplomacy_center|renseignement_espionnage|Renseignement & Espionnage|Réseau d'espionnage et renseignement militaire|Diplomatie,Argent Secret|140|1:90:90,2:150:240,3:210:450,4:300:750,5:390:1140,6:510:1584,7:660:2304,8:840:3168,9:1080:4176,10:1320:5616,11:1584:7200,12:2016:9216,13:2592:11664,14:3168:14832,15:3888:18720
diplomatie_influence|DIPLOMATIE & INFLUENCE|diplomacy_center|soft_power|Soft Power|Culture, propagande, influence médiatique|Population,Culture|150|1:60:60,2:90:150,3:120:270,4:180:450,5:240:690,6:330:1020,7:420:1440,8:540:2016,9:690:2736,10:900:3600,11:1140:4752,12:1440:6192,13:1872:8064,14:2448:10368
diplomatie_influence|DIPLOMATIE & INFLUENCE|diplomacy_center|coalitions_internationales|Coalitions Internationales|Formation d'alliances stratégiques et blocs|Diplomatie,Richesse,Autorité|160|1:120:120,2:180:300,3:240:540,4:360:900,5:480:1380,6:600:2016,7:780:2736,8:960:3744,9:1200:4896,10:1440:6480,11:1872:8352,12:2304:10512,13:2880:13392,14:3456:16848,15:4320:21168,16:5184:26352,17:6336:32688
defense_militaire|DÉFENSE & MILITAIRE|defense_grid|forces_militaires|Forces Militaires|Armée, entraînement, casernes|Population,Richesse,Minerai|170|1:90:90,2:150:240,3:210:450,4:300:750,5:420:1170,6:540:1728,7:720:2448,8:900:3312,9:1140:4464,10:1440:5904,11:1872:7776,12:2160:9936,13:2736:12672,14:3456:16128,15:4176:20304,16:5184:25488,17:6336:31824,18:7632:39456
defense_militaire|DÉFENSE & MILITAIRE|defense_grid|technologie_militaire|Technologie Militaire|Armes avancées, blindage, artillerie|Science,Richesse,Minerai|180|1:120:120,2:180:300,3:270:570,4:360:930,5:480:1410,6:660:2016,7:840:2880,8:1080:4032,9:1380:5328,10:1728:7056,11:2160:9216,12:2736:11952,13:3312:15264,14:4176:19584,15:5184:24624,16:6336:30960
defense_militaire|DÉFENSE & MILITAIRE|defense_grid|defense_aerienne|Défense Aérienne|Escadrilles aériennes et défense anti-aérienne|Science,Richesse,Tech|190|1:120:120,2:180:300,3:270:570,4:390:960,5:510:1440,6:660:2160,7:840:3024,8:1080:4032,9:1380:5472,10:1728:7200,11:2160:9360,12:2880:12240,13:3600:15840,14:4608:20304,15:5760:26064
defense_militaire|DÉFENSE & MILITAIRE|defense_grid|cyber_securite|Cyber Sécurité|Cyberdéfense, attaques numériques|Science,Tech,Population|200|1:90:90,2:150:240,3:210:450,4:300:750,5:420:1170,6:540:1728,7:720:2448,8:900:3312,9:1140:4464,10:1440:5904,11:1872:7776,12:2304:9936,13:2880:12816,14:3600:16560,15:4608:21168
construction_genie_civil|CONSTRUCTION & GÉNIE CIVIL|engineering_office|architecture_residentielle|Architecture Résidentielle|Habitations, immeubles, villas|Richesse,Ingénierie|210|1:15:15,2:30:45,3:45:90,4:60:150,5:90:240,6:120:360,7:150:510,8:210:720,9:270:990,10:360:1350,11:480:1872,12:660:2448,13:840:3312,14:1140:4464
construction_genie_civil|CONSTRUCTION & GÉNIE CIVIL|engineering_office|genie_civil_avance|Génie Civil Avancé|Ponts, tunnels, structures massives|Ingénierie,Minerai,Richesse|220|1:60:60,2:90:150,3:120:270,4:180:450,5:240:690,6:330:1020,7:420:1440,8:540:2016,9:690:2736,10:900:3600,11:1140:4752,12:1440:6192,13:1872:7920,14:2304:10224,15:2880:13104,16:3744:16848
construction_genie_civil|CONSTRUCTION & GÉNIE CIVIL|engineering_office|usines_manufactures|Usines & Manufactures|Constructions industrielles|Ingénierie,Richesse,Minerai|230|1:90:90,2:120:210,3:180:390,4:240:630,5:330:960,6:420:1380,7:540:1872,8:690:2592,9:870:3456,10:1110:4608,11:1380:5904,12:1728:7776,13:2160:9936,14:2880:12816,15:3600:16416
construction_genie_civil|CONSTRUCTION & GÉNIE CIVIL|engineering_office|megastructures|Mégastructures|Pyramides, tours massives, structures monumentales|Ingénierie,Richesse,Minerai,Ouvriers|240|1:180:180,2:300:480,3:450:930,4:600:1584,5:840:2304,6:1080:3456,7:1380:4896,8:1872:6624,9:2448:9072,10:3168:12096,11:4032:16272,12:5328:21456
technologie_science|TECHNOLOGIE & SCIENCE|research_center|intelligence_artificielle|Intelligence Artificielle|IA, machine learning, robotique|Science,Tech,Math|250|1:180:180,2:300:480,3:420:900,4:600:1440,5:840:2304,6:1080:3456,7:1380:4752,8:1728:6480,9:2160:8784,10:2880:11520,11:3600:15120,12:4608:19728,13:5760:25488,14:7200:32688,15:9072:41760,16:11088:51840
technologie_science|TECHNOLOGIE & SCIENCE|research_center|biotechnologie|Biotechnologie|Génie génétique, médecine avancée|Science,Bio,Ressources|260|1:120:120,2:210:330,3:300:630,4:420:1050,5:600:1584,6:780:2448,7:1020:3456,8:1320:4752,9:1728:6480,10:2160:8640,11:2736:11376,12:3600:14976,13:4464:19440,14:5760:25056,15:7200:32256
technologie_science|TECHNOLOGIE & SCIENCE|research_center|nanotechnologie|Nanotechnologie|Nano-matériaux, construction moléculaire|Science,Tech,Minerai|270|1:180:180,2:270:450,3:390:840,4:540:1380,5:720:2160,6:960:3024,7:1200:4320,8:1584:5760,9:2016:7776,10:2592:10368,11:3168:13536,12:4032:17568,13:5040:22608,14:6336:28800,15:7776:36720
technologie_science|TECHNOLOGIE & SCIENCE|research_center|physique_quantique|Physique Quantique|Énergie quantique, téléportation|Science,Math,Énergie|280|1:240:240,2:360:600,3:540:1140,4:720:1872,5:960:2880,6:1260:4032,7:1584:5760,8:2160:7776,9:2736:10512,10:3456:13968,11:4464:18432,12:5616:24048,13:7056:31104,14:8928:40032
technologie_science|TECHNOLOGIE & SCIENCE|research_center|exploration_spatiale|Exploration Spatiale|Fusées, satellites, colonisation spatiale|Science,Richesse,Tech|290|1:240:240,2:360:600,3:540:1140,4:780:1872,5:1020:2880,6:1320:4320,7:1728:5904,8:2160:8064,9:2736:10800,10:3600:14400,11:4464:18864,12:5760:24624,13:7200:31824,14:9072:40752,15:11088:51840,16:13824:64800
`
