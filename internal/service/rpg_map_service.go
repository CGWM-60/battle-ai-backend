package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RPGMapService struct {
	maps     *repository.RPGMapRepository
	roleplay *repository.RolePlayRepository
	quests   *repository.QuestRepository
	coop     *repository.CoopRepository
}

func NewRPGMapService(
	maps *repository.RPGMapRepository,
	roleplay *repository.RolePlayRepository,
	quests *repository.QuestRepository,
	coop *repository.CoopRepository,
) *RPGMapService {
	return &RPGMapService{maps: maps, roleplay: roleplay, quests: quests, coop: coop}
}

func (s *RPGMapService) GetOrCreateArcMapForSession(ctx context.Context, sessionID uint, ownerID uint) (*models.RPGArcMap, *models.RPGSessionMapState, error) {
	state, err := s.maps.GetSessionMapState(ctx, sessionID)
	if err == nil {
		arcMap, err := s.maps.GetArcMapByID(ctx, state.ArcMapID)
		if err != nil {
			return nil, nil, err
		}
		return arcMap, state, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, nil, err
	}
	return s.GenerateArcMapForSession(ctx, sessionID, ownerID)
}

func (s *RPGMapService) GenerateArcMapForSession(ctx context.Context, sessionID uint, ownerID uint) (*models.RPGArcMap, *models.RPGSessionMapState, error) {
	session, err := s.roleplay.GetSessionOwnedByID(ctx, sessionID, ownerID)
	if err != nil {
		return nil, nil, fmt.Errorf("session not found")
	}

	questID, arcID, chapters, arcTitle, err := s.resolveQuestContext(ctx, session)
	if err != nil {
		return nil, nil, err
	}

	var arcMap *models.RPGArcMap
	existing, err := s.maps.GetArcMapByQuestArc(ctx, questID, arcID)
	if err == nil {
		arcMap = existing
	} else if err == gorm.ErrRecordNotFound {
		arcMap, err = BuildLocalArcMapFallback(questID, arcID, arcTitle, chapters)
		if err != nil {
			return nil, nil, err
		}
		if err := s.maps.CreateArcMap(ctx, arcMap); err != nil {
			return nil, nil, err
		}
	} else {
		return nil, nil, err
	}

	nodes, _, objectives := parseArcMap(arcMap)
	entranceID := findEntranceNode(nodes)
	if entranceID == "" {
		entranceID = nodes[0].ID
	}

	activeChapterID := uint(0)
	activeObjectiveID := ""
	if len(objectives) > 0 {
		activeChapterID = objectives[0].ChapterID
		activeObjectiveID = objectives[0].ID
	}

	discovered, _ := json.Marshal([]string{entranceID})
	completed, _ := json.Marshal([]string{})
	locked, _ := json.Marshal(map[string]bool{})
	flags, _ := json.Marshal(map[string]any{})
	history, _ := json.Marshal([]map[string]any{})

	state := &models.RPGSessionMapState{
		SessionID:         sessionID,
		ArcMapID:          arcMap.ID,
		CurrentNodeID:     entranceID,
		ActiveChapterID:   activeChapterID,
		ActiveObjectiveID: activeObjectiveID,
		DiscoveredJSON:    datatypes.JSON(discovered),
		CompletedJSON:     datatypes.JSON(completed),
		LockedJSON:        datatypes.JSON(locked),
		FlagsJSON:         datatypes.JSON(flags),
		NPCStateJSON:      datatypes.JSON([]byte("[]")),
		EnemyStateJSON:    datatypes.JSON([]byte("[]")),
		LootStateJSON:     datatypes.JSON([]byte("[]")),
		TrapStateJSON:     datatypes.JSON([]byte("[]")),
		CombatStateJSON:   datatypes.JSON([]byte("null")),
		HistoryJSON:       datatypes.JSON(history),
	}

	if err := s.maps.CreateSessionMapState(ctx, state); err != nil {
		return nil, nil, err
	}
	return arcMap, state, nil
}

func (s *RPGMapService) EnterMap(ctx context.Context, sessionID uint, ownerID uint) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}
	return s.buildMapResponse(arcMap, state, "Tu descends sous les rails. La vapeur colle à ta peau, et l'obscurité semble respirer autour de toi.")
}

func (s *RPGMapService) GetMap(ctx context.Context, sessionID uint, ownerID uint) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}
	return s.buildMapResponse(arcMap, state, "")
}

func (s *RPGMapService) MoveToNode(ctx context.Context, sessionID uint, ownerID uint, targetNodeID string) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	combat := decodeCombat(state.CombatStateJSON)
	if combat != nil && combat.Active {
		return nil, fmt.Errorf("cannot move during combat")
	}

	nodes, edges, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	targetNode := findNode(nodes, targetNodeID)
	if currentNode == nil || targetNode == nil {
		return nil, fmt.Errorf("invalid node")
	}

	discovered := decodeStringSlice(state.DiscoveredJSON)
	flags := decodeFlags(state.FlagsJSON)

	if !isAdjacent(edges, state.CurrentNodeID, targetNodeID, discovered, flags) {
		return nil, fmt.Errorf("node not reachable")
	}

	state.CurrentNodeID = targetNodeID
	discovered = appendUnique(discovered, targetNodeID)
	state.DiscoveredJSON = rpgMustJSON(discovered)

	s.checkObjectiveProgress(state, arcMap, flags)
	state.FlagsJSON = rpgMustJSON(flags)

	history := decodeHistory(state.HistoryJSON)
	history = append(history, map[string]any{
		"type": "move", "from": currentNode.ID, "to": targetNodeID, "at": time.Now().Format(time.RFC3339),
	})
	state.HistoryJSON = rpgMustJSON(history)

	if err := s.maps.UpdateSessionMapState(ctx, state); err != nil {
		return nil, err
	}

	narration := fmt.Sprintf("Tu avances vers %s. %s", targetNode.Name, targetNode.Description)
	resp, err := s.buildMapResponse(arcMap, state, narration)
	if err != nil {
		return nil, err
	}

	if len(targetNode.Enemies) > 0 && !allEnemiesDefeated(targetNode.Enemies) {
		combat = s.startCombat(targetNode)
		state.CombatStateJSON = rpgMustJSON(combat)
		_ = s.maps.UpdateSessionMapState(ctx, state)
		resp.InCombat = true
		resp.Combat = combat
		resp.Narration += " Des ennemis surgissent !"
	}

	if len(targetNode.Traps) > 0 {
		for _, trap := range targetNode.Traps {
			if !trap.Triggered && !trap.Disarmed {
				resp.Narration += fmt.Sprintf(" Attention : %s !", trap.Name)
			}
		}
	}

	return resp, nil
}

func (s *RPGMapService) RunAction(ctx context.Context, sessionID uint, ownerID uint, actionID, actionType, nodeID string) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	nodes, _, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	if currentNode == nil {
		return nil, fmt.Errorf("invalid current node")
	}
	if nodeID != "" && nodeID != state.CurrentNodeID {
		return nil, fmt.Errorf("action not available on this node")
	}

	flags := decodeFlags(state.FlagsJSON)
	narration := ""

	switch actionType {
	case "inspect":
		narration = fmt.Sprintf("Tu examines %s. %s", currentNode.Name, currentNode.Description)
		for _, trap := range currentNode.Traps {
			if !trap.Detected {
				narration += fmt.Sprintf(" Tu repères %s.", trap.Name)
			}
		}
	case "detect_trap":
		narration = s.detectTraps(state, currentNode)
	case "interact_lever":
		if currentNode.IsLever && currentNode.LeverTarget != "" {
			flags["door_unlocked"] = true
			flags["lever_"+currentNode.ID] = true
			narration = fmt.Sprintf("Tu actionnes le levier. Un mécanisme lointain grince — la porte vers %s se déverrouille.", currentNode.LeverTarget)
		}
	case "disable_node":
		flags["node_disabled"] = true
		narration = "Tu neutralises le nœud énergétique. Les lumières vacillent puis s'éteignent."
	default:
		narration = fmt.Sprintf("Action %s effectuée.", actionID)
	}

	state.FlagsJSON = rpgMustJSON(flags)
	s.checkObjectiveProgress(state, arcMap, flags)
	_ = s.maps.UpdateSessionMapState(ctx, state)

	return s.buildMapResponse(arcMap, state, narration)
}

func normalizeRollDifficulty(difficulty int) int {
	if difficulty < 5 {
		return 12
	}
	if difficulty > 30 {
		return 30
	}
	return difficulty
}

func rollModifierForAttribute(attribute string) int {
	if attribute == "dexterity" {
		return 3
	}
	return 2
}

func computeRollResult(actionID, attribute, skill string, difficulty int) *models.RPGMapRollResult {
	difficulty = normalizeRollDifficulty(difficulty)
	modifier := rollModifierForAttribute(attribute)
	roll := rand.Intn(20) + 1
	total := roll + modifier
	success := total >= difficulty

	result := &models.RPGMapRollResult{
		ActionID: actionID, Attribute: attribute, Skill: skill,
		Roll: roll, Modifier: modifier, Total: total,
		Difficulty: difficulty, Success: success,
	}
	if success {
		result.Message = "Réussite"
	} else {
		result.Message = "Échec"
	}
	return result
}

func (s *RPGMapService) Roll(ctx context.Context, sessionID uint, ownerID uint, actionID, attribute, skill string, difficulty int) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	result := computeRollResult(actionID, attribute, skill, difficulty)

	resp, err := s.buildMapResponse(arcMap, state, "Le dé est lancé.")
	if err != nil {
		return nil, err
	}
	resp.RollResult = result
	return resp, nil
}

func (s *RPGMapService) ResolveAction(ctx context.Context, sessionID uint, ownerID uint, actionID string, rollSuccess bool) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	nodes, _, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	flags := decodeFlags(state.FlagsJSON)
	narration := ""

	switch actionID {
	case "disable_trap", "disarm_trap":
		if rollSuccess {
			narration = s.disarmTraps(state, currentNode)
		} else {
			narration = "Le piège se déclenche ! Tu subis des dégâts."
		}
	case "collect_loot_pass":
		if rollSuccess {
			flags["has_access_pass"] = true
			narration = "Tu récupères le passe d'accès."
			s.markLootCollected(state, "loot_pass")
		} else {
			narration = "Le coffre résiste à tes efforts."
		}
	default:
		if rollSuccess {
			narration = "Action réussie."
		} else {
			narration = "Action échouée."
		}
	}

	state.FlagsJSON = rpgMustJSON(flags)
	s.checkObjectiveProgress(state, arcMap, flags)
	_ = s.maps.UpdateSessionMapState(ctx, state)

	return s.buildMapResponse(arcMap, state, narration)
}

func (s *RPGMapService) InteractNPC(ctx context.Context, sessionID uint, ownerID uint, npcID string) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	nodes, _, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	var npc *models.RPGMapNPC
	for i := range currentNode.NPCs {
		if currentNode.NPCs[i].ID == npcID {
			npc = &currentNode.NPCs[i]
			break
		}
	}
	if npc == nil {
		return nil, fmt.Errorf("npc not found on current node")
	}

	resp, err := s.buildMapResponse(arcMap, state, "")
	if err != nil {
		return nil, err
	}
	resp.Dialogues = []models.RPGMapDialogueLine{
		{Speaker: npc.Name, Text: "Les bas-fonds ont des oreilles partout. Cherche la cache de maintenance — le passe s'y trouve.", Intent: npc.DialogueIntent},
		{Speaker: "MJ", Text: "L'informateur te fixe, attendant ta réaction.", Intent: "narration"},
	}
	return resp, nil
}

func (s *RPGMapService) CollectLoot(ctx context.Context, sessionID uint, ownerID uint, lootID string) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	nodes, _, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	var loot *models.RPGMapLoot
	for i := range currentNode.Loot {
		if currentNode.Loot[i].ID == lootID {
			loot = &currentNode.Loot[i]
			break
		}
	}
	if loot == nil {
		return nil, fmt.Errorf("loot not found")
	}
	if loot.Collected {
		return nil, fmt.Errorf("loot already collected")
	}

	flags := decodeFlags(state.FlagsJSON)
	narration := ""

	if loot.RequiredRoll {
		narration = "Ce butin nécessite un jet de dé. Utilisez l'action de ramassage avec un roll."
	} else {
		s.markLootCollected(state, lootID)
		if loot.Type == "key" {
			flags["has_access_pass"] = true
		}
		narration = fmt.Sprintf("Tu récupères %s.", loot.Name)
	}

	state.FlagsJSON = rpgMustJSON(flags)
	s.checkObjectiveProgress(state, arcMap, flags)
	_ = s.maps.UpdateSessionMapState(ctx, state)

	return s.buildMapResponse(arcMap, state, narration)
}

func (s *RPGMapService) DisarmTrap(ctx context.Context, sessionID uint, ownerID uint, trapID string) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	nodes, _, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)

	traps := decodeTraps(state.TrapStateJSON)
	found := false
	for i := range traps {
		if traps[i].ID == trapID {
			traps[i].Disarmed = true
			found = true
			break
		}
	}
	if !found {
		for _, t := range currentNode.Traps {
			if t.ID == trapID {
				traps = append(traps, models.RPGMapTrap{ID: trapID, Name: t.Name, Disarmed: true})
				found = true
				break
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("trap not found")
	}

	state.TrapStateJSON = rpgMustJSON(traps)
	_ = s.maps.UpdateSessionMapState(ctx, state)

	return s.buildMapResponse(arcMap, state, "Piège désamorcé avec succès.")
}

func (s *RPGMapService) CombatAction(ctx context.Context, sessionID uint, ownerID uint, combatID, actionID, targetID string) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	combat := decodeCombat(state.CombatStateJSON)
	if combat == nil || !combat.Active {
		return nil, fmt.Errorf("no active combat")
	}

	narration := ""
	switch actionID {
	case "attack":
		for i := range combat.Enemies {
			if combat.Enemies[i].ID == targetID && !combat.Enemies[i].Defeated {
				damage := rand.Intn(12) + 5
				combat.Enemies[i].Health -= damage
				narration = fmt.Sprintf("Tu frappes %s pour %d dégâts.", combat.Enemies[i].Name, damage)

				if combat.Enemies[i].Health <= 0 {
					if combat.Enemies[i].Boss && combat.Enemies[i].Phase < combat.Enemies[i].MaxPhase {
						combat.Enemies[i].Phase++
						combat.Enemies[i].Health = combat.Enemies[i].MaxHealth / 2
						combat.Phase = combat.Enemies[i].Phase
						narration += fmt.Sprintf(" %s entre en phase %d !", combat.Enemies[i].Name, combat.Enemies[i].Phase)
					} else {
						combat.Enemies[i].Defeated = true
						combat.Enemies[i].Health = 0
						narration += fmt.Sprintf(" %s est vaincu !", combat.Enemies[i].Name)
					}
				}
				break
			}
		}
	case "defend":
		narration = "Tu te mets en position défensive."
	case "skill":
		narration = "Tu déchaînes une compétence spéciale."
	default:
		narration = fmt.Sprintf("Action de combat : %s", actionID)
	}

	combat.Turn++
	combat.Log = append(combat.Log, narration)

	if allEnemiesDefeated(combat.Enemies) {
		combat.Active = false
		combat.Victory = true
		flags := decodeFlags(state.FlagsJSON)
		flags["boss_defeated"] = true
		state.FlagsJSON = rpgMustJSON(flags)
		s.checkObjectiveProgress(state, arcMap, flags)
		narration += " Victoire ! Tu reprends ton exploration."
	}

	state.CombatStateJSON = rpgMustJSON(combat)
	_ = s.maps.UpdateSessionMapState(ctx, state)

	resp, err := s.buildMapResponse(arcMap, state, narration)
	if err != nil {
		return nil, err
	}
	resp.InCombat = combat.Active
	resp.Combat = combat
	return resp, nil
}

func (s *RPGMapService) FleeCombat(ctx context.Context, sessionID uint, ownerID uint, combatID string) (*models.RPGMapResponse, error) {
	arcMap, state, err := s.GetOrCreateArcMapForSession(ctx, sessionID, ownerID)
	if err != nil {
		return nil, err
	}

	combat := decodeCombat(state.CombatStateJSON)
	if combat == nil || !combat.Active {
		return nil, fmt.Errorf("no active combat")
	}

	combat.Active = false
	combat.Fled = true
	state.CombatStateJSON = rpgMustJSON(combat)
	_ = s.maps.UpdateSessionMapState(ctx, state)

	return s.buildMapResponse(arcMap, state, "Tu prends la fuite et retournes à l'exploration.")
}

// --- helpers ---

func (s *RPGMapService) resolveQuestContext(ctx context.Context, session *models.RolePlaySession) (uint, uint, []models.RolePlayQuestChapter, string, error) {
	var questID, arcID uint
	var chapters []models.RolePlayQuestChapter
	arcTitle := "Zone inconnue"

	run, runErr := s.roleplay.GetQuestRunBySession(ctx, session.Id)
	if runErr == nil && run != nil {
		if run.TemplateID != nil {
			questID = *run.TemplateID
			template, err := s.quests.GetRolePlayQuestByID(ctx, questID)
			if err == nil && len(template.Arcs) > 0 {
				arc := template.Arcs[0]
				if run.CurrentArcID != nil {
					for _, a := range template.Arcs {
						if a.Id == *run.CurrentArcID {
							arc = a
							break
						}
					}
				}
				arcID = arc.Id
				arcTitle = arc.Title
				chapters = arc.Chapters
			}
		}
	}

	if questID == 0 {
		questID = 1
		arcID = 1
		arcTitle = "Les bas-fonds huilés"
	}

	return questID, arcID, chapters, arcTitle, nil
}

func (s *RPGMapService) buildMapResponse(arcMap *models.RPGArcMap, state *models.RPGSessionMapState, narration string) (*models.RPGMapResponse, error) {
	nodes, edges, objectives := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	if currentNode == nil && len(nodes) > 0 {
		currentNode = &nodes[0]
	}

	discovered := decodeStringSlice(state.DiscoveredJSON)
	flags := decodeFlags(state.FlagsJSON)
	combat := decodeCombat(state.CombatStateJSON)

	var activeObjective *models.RPGMapObjective
	for i := range objectives {
		if objectives[i].ID == state.ActiveObjectiveID {
			activeObjective = &objectives[i]
			break
		}
	}
	if activeObjective == nil && len(objectives) > 0 {
		activeObjective = &objectives[0]
	}

	actions := s.buildAvailableActions(currentNode, edges, state.CurrentNodeID, discovered, flags, combat)
	adjacent := s.getAdjacentNodes(nodes, edges, state.CurrentNodeID, discovered, flags)

	activeChapter := map[string]any{}
	if state.ActiveChapterID > 0 {
		activeChapter["id"] = state.ActiveChapterID
		if activeObjective != nil {
			activeChapter["title"] = activeObjective.Title
			activeChapter["objective"] = activeObjective.Description
		}
	}

	return &models.RPGMapResponse{
		ArcMap:           arcMap,
		MapState:         state,
		CurrentNode:      currentNode,
		ActiveChapter:    activeChapter,
		ActiveObjective:  activeObjective,
		Objectives:       objectives,
		Narration:        narration,
		Dialogues:        []models.RPGMapDialogueLine{},
		AvailableActions: actions,
		InCombat:         combat != nil && combat.Active,
		Combat:           combat,
		DiscoveredNodes:  discovered,
		AdjacentNodes:    adjacent,
	}, nil
}

func (s *RPGMapService) buildAvailableActions(node *models.RPGMapNode, edges []models.RPGMapEdge, currentID string, discovered []string, flags map[string]any, combat *models.RPGCombatState) []models.RPGMapAction {
	if combat != nil && combat.Active {
		actions := []models.RPGMapAction{
			{ID: "combat_attack", Label: "Attaquer", Type: "combat", RequiresRoll: false},
			{ID: "combat_defend", Label: "Défendre", Type: "combat", RequiresRoll: false},
			{ID: "combat_flee", Label: "Fuir", Type: "combat_flee", RequiresRoll: false},
		}
		for _, e := range combat.Enemies {
			if !e.Defeated {
				actions[0].TargetID = e.ID
				break
			}
		}
		return actions
	}

	if node == nil {
		return nil
	}

	actions := []models.RPGMapAction{
		{ID: "inspect_" + node.ID, Label: "Inspecter " + node.Name, Type: "inspect"},
	}

	for _, edge := range edges {
		if edge.From == currentID {
			if edge.Hidden && !containsStr(discovered, edge.To) {
				continue
			}
			if edge.Locked {
				if edge.RequiredFlag != "" {
					if v, ok := flags[edge.RequiredFlag]; !ok || v != true {
						actions = append(actions, models.RPGMapAction{
							ID: "locked_" + edge.To, Label: "Passage verrouillé vers " + edge.To, Type: "locked",
						})
						continue
					}
				}
			}
			actions = append(actions, models.RPGMapAction{
				ID: "move_" + edge.To, Label: "Avancer vers " + edge.To, Type: "move", TargetNodeID: edge.To,
			})
		}
	}

	for _, npc := range node.NPCs {
		actions = append(actions, models.RPGMapAction{
			ID: "npc_" + npc.ID, Label: "Parler à " + npc.Name, Type: "interact_npc", TargetID: npc.ID,
		})
	}
	for _, loot := range node.Loot {
		if !loot.Collected {
			actions = append(actions, models.RPGMapAction{
				ID: "loot_" + loot.ID, Label: "Ramasser " + loot.Name, Type: "loot", TargetID: loot.ID,
				RequiresRoll: loot.RequiredRoll, Difficulty: 12,
			})
		}
	}
	for _, trap := range node.Traps {
		if !trap.Disarmed {
			actions = append(actions, models.RPGMapAction{
				ID: "trap_detect_" + trap.ID, Label: "Détecter piège", Type: "detect_trap", TargetID: trap.ID,
			})
			actions = append(actions, models.RPGMapAction{
				ID: "trap_disarm_" + trap.ID, Label: "Désamorcer " + trap.Name, Type: "disarm_trap",
				TargetID: trap.ID, RequiresRoll: true, Difficulty: trap.Difficulty, Attribute: "dexterity", Skill: "mechanics",
			})
		}
	}
	if node.IsLever {
		actions = append(actions, models.RPGMapAction{
			ID: "lever_" + node.ID, Label: "Actionner le levier", Type: "interact_lever",
		})
	}
	if node.ID == "energy_node" {
		actions = append(actions, models.RPGMapAction{
			ID: "disable_node", Label: "Désactiver le nœud", Type: "disable_node", RequiresRoll: true, Difficulty: 15,
		})
	}

	return actions
}

func (s *RPGMapService) getAdjacentNodes(nodes []models.RPGMapNode, edges []models.RPGMapEdge, currentID string, discovered []string, flags map[string]any) []models.RPGMapNode {
	result := []models.RPGMapNode{}
	for _, edge := range edges {
		if edge.From != currentID {
			continue
		}
		if edge.Hidden && !containsStr(discovered, edge.To) {
			continue
		}
		if edge.Locked && edge.RequiredFlag != "" {
			if v, ok := flags[edge.RequiredFlag]; !ok || v != true {
				continue
			}
		}
		if n := findNode(nodes, edge.To); n != nil {
			result = append(result, *n)
		}
	}
	return result
}

func (s *RPGMapService) startCombat(node *models.RPGMapNode) *models.RPGCombatState {
	enemies := make([]models.RPGMapEnemy, len(node.Enemies))
	copy(enemies, node.Enemies)
	maxPhase := 1
	for _, e := range enemies {
		if e.MaxPhase > maxPhase {
			maxPhase = e.MaxPhase
		}
	}
	return &models.RPGCombatState{
		ID: "combat_" + node.ID, Active: true, NodeID: node.ID,
		Turn: 1, CurrentActor: "player", Enemies: enemies,
		Log: []string{"Le combat commence !"}, Phase: 1, MaxPhase: maxPhase,
	}
}

func (s *RPGMapService) checkObjectiveProgress(state *models.RPGSessionMapState, arcMap *models.RPGArcMap, flags map[string]any) {
	_, _, objectives := parseArcMap(arcMap)
	completed := decodeStringSlice(state.CompletedJSON)

	for i, obj := range objectives {
		if containsStr(completed, obj.ID) {
			continue
		}
		done := false
		if obj.RequiredFlag != "" {
			if v, ok := flags[obj.RequiredFlag]; ok && v == true {
				done = true
			}
		} else if state.CurrentNodeID == obj.TargetNodeID {
			done = true
		}
		if done {
			completed = appendUnique(completed, obj.ID)
			if i+1 < len(objectives) {
				state.ActiveObjectiveID = objectives[i+1].ID
				state.ActiveChapterID = objectives[i+1].ChapterID
			}
		}
	}
	state.CompletedJSON = rpgMustJSON(completed)
}

func (s *RPGMapService) detectTraps(state *models.RPGSessionMapState, node *models.RPGMapNode) string {
	traps := decodeTraps(state.TrapStateJSON)
	narration := ""
	for _, t := range node.Traps {
		found := false
		for i := range traps {
			if traps[i].ID == t.ID {
				traps[i].Detected = true
				found = true
			}
		}
		if !found {
			traps = append(traps, models.RPGMapTrap{ID: t.ID, Name: t.Name, Detected: true})
		}
		narration += fmt.Sprintf(" Tu détectes %s.", t.Name)
	}
	state.TrapStateJSON = rpgMustJSON(traps)
	return narration
}

func (s *RPGMapService) disarmTraps(state *models.RPGSessionMapState, node *models.RPGMapNode) string {
	traps := decodeTraps(state.TrapStateJSON)
	for i := range traps {
		traps[i].Disarmed = true
	}
	for _, t := range node.Traps {
		found := false
		for i := range traps {
			if traps[i].ID == t.ID {
				traps[i].Disarmed = true
				found = true
			}
		}
		if !found {
			traps = append(traps, models.RPGMapTrap{ID: t.ID, Name: t.Name, Disarmed: true})
		}
	}
	state.TrapStateJSON = rpgMustJSON(traps)
	return "Piège désamorcé."
}

func (s *RPGMapService) markLootCollected(state *models.RPGSessionMapState, lootID string) {
	collected := decodeStringSlice(state.LootStateJSON)
	collected = appendUnique(collected, lootID)
	state.LootStateJSON = rpgMustJSON(collected)
}

func parseArcMap(arcMap *models.RPGArcMap) ([]models.RPGMapNode, []models.RPGMapEdge, []models.RPGMapObjective) {
	var nodes []models.RPGMapNode
	var edges []models.RPGMapEdge
	var objectives []models.RPGMapObjective
	_ = json.Unmarshal(arcMap.NodesJSON, &nodes)
	_ = json.Unmarshal(arcMap.EdgesJSON, &edges)
	var lore map[string]any
	_ = json.Unmarshal(arcMap.LoreJSON, &lore)
	if lore != nil {
		if raw, ok := lore["objectives"]; ok {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &objectives)
		}
	}
	return nodes, edges, objectives
}

func findNode(nodes []models.RPGMapNode, id string) *models.RPGMapNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func findEntranceNode(nodes []models.RPGMapNode) string {
	for _, n := range nodes {
		if n.IsEntrance {
			return n.ID
		}
	}
	return ""
}

func isAdjacent(edges []models.RPGMapEdge, from, to string, discovered []string, flags map[string]any) bool {
	for _, e := range edges {
		if e.From == from && e.To == to {
			if e.Hidden && !containsStr(discovered, to) {
				return false
			}
			if e.Locked && e.RequiredFlag != "" {
				if v, ok := flags[e.RequiredFlag]; !ok || v != true {
					return false
				}
			}
			return true
		}
	}
	return false
}

func allEnemiesDefeated(enemies []models.RPGMapEnemy) bool {
	if len(enemies) == 0 {
		return true
	}
	for _, e := range enemies {
		if !e.Defeated {
			return false
		}
	}
	return true
}

func decodeStringSlice(data datatypes.JSON) []string {
	var result []string
	_ = json.Unmarshal(data, &result)
	return result
}

func decodeFlags(data datatypes.JSON) map[string]any {
	result := map[string]any{}
	_ = json.Unmarshal(data, &result)
	return result
}

func decodeCombat(data datatypes.JSON) *models.RPGCombatState {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var combat models.RPGCombatState
	if err := json.Unmarshal(data, &combat); err != nil {
		return nil
	}
	return &combat
}

func decodeTraps(data datatypes.JSON) []models.RPGMapTrap {
	var traps []models.RPGMapTrap
	_ = json.Unmarshal(data, &traps)
	return traps
}

func decodeHistory(data datatypes.JSON) []map[string]any {
	var history []map[string]any
	_ = json.Unmarshal(data, &history)
	return history
}

func rpgMustJSON(v any) datatypes.JSON {
	b, _ := json.Marshal(v)
	return datatypes.JSON(b)
}

func appendUnique(slice []string, item string) []string {
	if containsStr(slice, item) {
		return slice
	}
	return append(slice, item)
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}