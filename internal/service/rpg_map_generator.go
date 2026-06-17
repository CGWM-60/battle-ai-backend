package service

import (
	"encoding/json"
	"fmt"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
)

func BuildLocalArcMapFallback(questID uint, arcID uint, arcTitle string, chapters []models.RolePlayQuestChapter) (*models.RPGArcMap, error) {
	nodes, edges, objectives := generateMapGraph(arcTitle, chapters)

	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)
	loreJSON, _ := json.Marshal(map[string]any{
		"objectives": objectives,
		"theme":      "dungeon",
		"arcTitle":   arcTitle,
	})

	return &models.RPGArcMap{
		QuestID:   questID,
		ArcID:     arcID,
		Name:      arcTitle,
		Theme:     "dungeon",
		Width:     800,
		Height:    600,
		NodesJSON: datatypes.JSON(nodesJSON),
		EdgesJSON: datatypes.JSON(edgesJSON),
		LoreJSON:  datatypes.JSON(loreJSON),
	}, nil
}

func generateMapGraph(arcTitle string, chapters []models.RolePlayQuestChapter) ([]models.RPGMapNode, []models.RPGMapEdge, []models.RPGMapObjective) {
	type nodeSpec struct {
		id, name, nodeType, desc, danger string
		x, y                             int
		isEntrance, isExit, isBoss, isSafe, isLever bool
		leverTarget                      string
	}

	specs := []nodeSpec{
		{"entrance", "Entrée des bas-fonds", "entrance", "Une porte rouillée s'ouvre sur l'obscurité.", "low", 100, 300, true, false, false, true, false, ""},
		{"black_market", "Marché noir", "room", "Des marchands chuchotent entre les étales.", "medium", 250, 200, false, false, false, false, false, ""},
		{"maintenance_cache", "Cache de maintenance", "room", "Des outils et des plans traînent partout.", "medium", 250, 400, false, false, false, false, false, ""},
		{"steam_corridor", "Conduits de vapeur", "corridor", "La vapeur brûlante obscurcit la vision.", "high", 400, 300, false, false, false, false, false, ""},
		{"locked_door", "Porte verrouillée", "door", "Une porte massive bloque le passage.", "medium", 550, 250, false, false, false, false, false, ""},
		{"automaton_workshop", "Atelier des automates", "room", "Des engrenages grincent dans l'ombre.", "high", 550, 400, false, false, false, false, false, ""},
		{"lever_room", "Salle du levier", "lever", "Un levier rouillé domine la pièce.", "low", 400, 150, false, false, false, false, true, "locked_door"},
		{"energy_node", "Salle du nœud énergétique", "room", "Un cœur mécanique pulse faiblement.", "high", 700, 300, false, false, false, false, false, ""},
		{"boss_room", "Salle du Gardien", "boss", "Le Gardien de la Machine vous attend.", "extreme", 700, 450, false, false, true, false, false, ""},
		{"hidden_passage", "Passage secret", "secret", "Un couloir discret mène ailleurs.", "low", 400, 500, false, false, false, true, false, ""},
		{"treasure_cache", "Coffre oublié", "loot", "Un coffre scellé repose dans l'angle.", "low", 300, 500, false, false, false, false, false, ""},
		{"npc_alley", "Ruelle du fixeur", "npc", "Un informateur observe discrètement.", "low", 150, 450, false, false, false, false, false, ""},
		{"trap_hall", "Couloir piégé", "trap", "Des mécanismes suspects tapissent les murs.", "high", 600, 150, false, false, false, false, false, ""},
		{"exit_gate", "Sortie vers l'arc suivant", "exit", "La lumière du monde extérieur filtre.", "low", 850, 300, false, true, false, false, false, ""},
	}

	nodes := make([]models.RPGMapNode, 0, len(specs))
	for _, s := range specs {
		node := models.RPGMapNode{
			ID: s.id, Name: s.name, Type: s.nodeType,
			X: s.x, Y: s.y, Description: s.desc, DangerLevel: s.danger,
			IsEntrance: s.isEntrance, IsExit: s.isExit, IsBossRoom: s.isBoss,
			IsSafeRoom: s.isSafe, IsLever: s.isLever, LeverTarget: s.leverTarget,
			Flags: map[string]any{},
		}
		nodes = append(nodes, enrichNode(node, s.id))
	}

	edges := []models.RPGMapEdge{
		{From: "entrance", To: "black_market", Description: "Vers le marché"},
		{From: "entrance", To: "npc_alley", Description: "Vers la ruelle"},
		{From: "entrance", To: "steam_corridor", Description: "Vers les conduits"},
		{From: "black_market", To: "maintenance_cache", Description: "Vers la cache"},
		{From: "black_market", To: "steam_corridor", Description: "Raccourci vers les conduits"},
		{From: "maintenance_cache", To: "treasure_cache", Description: "Vers le coffre"},
		{From: "maintenance_cache", To: "hidden_passage", Description: "Passage discret", Hidden: true},
		{From: "steam_corridor", To: "locked_door", Description: "Vers la porte verrouillée"},
		{From: "steam_corridor", To: "automaton_workshop", Description: "Vers l'atelier"},
		{From: "steam_corridor", To: "lever_room", Description: "Vers le levier"},
		{From: "locked_door", To: "energy_node", Description: "Au-delà de la porte", Locked: true, LockType: "key", RequiredKey: "access_pass", RequiredFlag: "door_unlocked"},
		{From: "automaton_workshop", To: "boss_room", Description: "Vers le gardien"},
		{From: "automaton_workshop", To: "trap_hall", Description: "Couloir piégé"},
		{From: "hidden_passage", To: "energy_node", Description: "Raccourci secret", Hidden: true},
		{From: "energy_node", To: "boss_room", Description: "Affronter le gardien"},
		{From: "boss_room", To: "exit_gate", Description: "Sortir de la zone", RequiredFlag: "boss_defeated"},
		{From: "npc_alley", To: "treasure_cache", Description: "Indice vers le coffre"},
		{From: "trap_hall", To: "energy_node", Description: "Contourner par le haut"},
	}

	objectives := buildObjectives(chapters, nodes)

	return nodes, edges, objectives
}

func enrichNode(node models.RPGMapNode, id string) models.RPGMapNode {
	switch id {
	case "npc_alley":
		node.NPCs = []models.RPGMapNPC{{
			ID: "fixer_01", Name: "Le Fixeur", Role: "informateur",
			Mood: "méfiant", CurrentNode: id,
			DialogueIntent: "propose un indice contre une faveur",
		}}
	case "treasure_cache":
		node.Loot = []models.RPGMapLoot{{
			ID: "loot_pass", Name: "Passe d'accès", Type: "key",
			Rarity: "rare", RequiredRoll: true,
		}, {
			ID: "loot_scrap", Name: "Ferraille précieuse", Type: "material",
			Rarity: "common",
		}}
	case "trap_hall":
		node.Traps = []models.RPGMapTrap{{
			ID: "trap_spikes", Name: "Pointes rétractables",
			Type: "mechanical", Difficulty: 14, Damage: 8,
		}}
	case "steam_corridor":
		node.Traps = []models.RPGMapTrap{{
			ID: "trap_steam", Name: "Jet de vapeur",
			Type: "environmental", Difficulty: 12, Damage: 5,
		}}
	case "boss_room":
		node.Enemies = []models.RPGMapEnemy{{
			ID: "boss_guardian", Name: "Gardien de la Machine",
			Type: "boss", Level: 5, Health: 80, MaxHealth: 80,
			Phase: 1, MaxPhase: 2, Boss: true,
		}}
	case "automaton_workshop":
		node.Enemies = []models.RPGMapEnemy{{
			ID: "enemy_automaton", Name: "Automate défectueux",
			Type: "construct", Level: 3, Health: 30, MaxHealth: 30,
			Phase: 1, MaxPhase: 1,
		}}
	case "energy_node":
		node.Enemies = []models.RPGMapEnemy{{
			ID: "enemy_sentinel", Name: "Sentinelle énergétique",
			Type: "guardian", Level: 4, Health: 40, MaxHealth: 40,
			Phase: 1, MaxPhase: 1,
		}}
	}
	return node
}

func buildObjectives(chapters []models.RolePlayQuestChapter, nodes []models.RPGMapNode) []models.RPGMapObjective {
	targets := []string{"steam_corridor", "maintenance_cache", "locked_door", "energy_node", "boss_room"}
	objectives := make([]models.RPGMapObjective, 0)

	for i, ch := range chapters {
		targetIdx := i
		if targetIdx >= len(targets) {
			targetIdx = len(targets) - 1
		}
		obj := models.RPGMapObjective{
			ID:           fmt.Sprintf("obj_%d", ch.Id),
			ChapterID:    ch.Id,
			Title:        ch.Title,
			Description:  ch.Objective,
			TargetNodeID: targets[targetIdx],
		}
		if ch.IsBoss {
			obj.TargetNodeID = "boss_room"
			obj.RequiredFlag = "boss_defeated"
		}
		objectives = append(objectives, obj)
	}

	if len(objectives) == 0 {
		objectives = []models.RPGMapObjective{
			{ID: "obj_default_1", ChapterID: 0, Title: "Atteindre les conduits", Description: "Explorer les conduits de vapeur", TargetNodeID: "steam_corridor"},
			{ID: "obj_default_2", ChapterID: 0, Title: "Trouver le passe", Description: "Récupérer le passe d'accès", TargetNodeID: "maintenance_cache", RequiredFlag: "has_access_pass"},
			{ID: "obj_default_3", ChapterID: 0, Title: "Ouvrir la porte", Description: "Déverrouiller la porte", TargetNodeID: "locked_door", RequiredFlag: "door_unlocked"},
			{ID: "obj_default_4", ChapterID: 0, Title: "Désactiver le nœud", Description: "Neutraliser le nœud énergétique", TargetNodeID: "energy_node", RequiredFlag: "node_disabled"},
			{ID: "obj_default_5", ChapterID: 0, Title: "Vaincre le Gardien", Description: "Terrasser le boss et sortir", TargetNodeID: "boss_room", RequiredFlag: "boss_defeated"},
		}
	}

	_ = nodes
	return objectives
}