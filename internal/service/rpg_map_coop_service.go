package service

import (
	"context"
	"encoding/json"
	"fmt"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *RPGMapService) GetOrCreateCoopMap(ctx context.Context, partyCode string) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, fmt.Errorf("party not found")
	}
	if party.RolePlaySessionID == nil {
		return nil, fmt.Errorf("party has no roleplay session")
	}

	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err == gorm.ErrRecordNotFound {
		arcMap, sessionState, err := s.GenerateArcMapForSession(ctx, *party.RolePlaySessionID, party.HostUserID)
		if err != nil {
			return nil, err
		}

		positions, _ := json.Marshal(map[string]string{"host": sessionState.CurrentNodeID})
		coopState := &models.RPGCoopMapState{
			CoopPartyID:       party.Id,
			ArcMapID:          arcMap.ID,
			CurrentNodeID:     sessionState.CurrentNodeID,
			ActiveChapterID:   sessionState.ActiveChapterID,
			ActiveObjectiveID: sessionState.ActiveObjectiveID,
			PlayerPositions:   datatypes.JSON(positions),
			DiscoveredJSON:    sessionState.DiscoveredJSON,
			CompletedJSON:     sessionState.CompletedJSON,
			LockedJSON:        sessionState.LockedJSON,
			FlagsJSON:         sessionState.FlagsJSON,
			NPCStateJSON:      sessionState.NPCStateJSON,
			EnemyStateJSON:    sessionState.EnemyStateJSON,
			LootStateJSON:     sessionState.LootStateJSON,
			TrapStateJSON:     sessionState.TrapStateJSON,
			CombatStateJSON:   sessionState.CombatStateJSON,
			HistoryJSON:       sessionState.HistoryJSON,
		}
		if err := s.maps.CreateCoopMapState(ctx, coopState); err != nil {
			return nil, err
		}
		state = coopState
	} else if err != nil {
		return nil, err
	}

	return s.buildCoopMapResponse(ctx, party, state, "")
}

func (s *RPGMapService) CoopRoll(ctx context.Context, partyCode string, actionID, attribute, skill string, difficulty int) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("party not found")
		}
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("coop map not initialized")
		}
		return nil, err
	}

	result := computeRollResult(actionID, attribute, skill, difficulty)
	resp, err := s.buildCoopMapResponse(ctx, party, state, "Le dé coop est lancé. L'équipe retient son souffle.")
	if err != nil {
		return nil, err
	}
	resp.RollResult = result
	return resp, nil
}

func (s *RPGMapService) CoopResolveAction(ctx context.Context, partyCode string, actionID string, rollSuccess bool) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("party not found")
		}
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("coop map not initialized")
		}
		return nil, err
	}
	arcMap, err := s.maps.GetArcMapByID(ctx, state.ArcMapID)
	if err != nil {
		return nil, err
	}

	nodes, _, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	flags := decodeFlags(state.FlagsJSON)
	flags["lastResolvedActionId"] = actionID
	flags["lastCoopRollSuccess"] = rollSuccess

	narration := ""
	switch actionID {
	case "disable_trap", "disarm_trap":
		if rollSuccess {
			sessionState := s.coopStateAsSession(state)
			narration = s.disarmTraps(sessionState, currentNode)
			state.TrapStateJSON = sessionState.TrapStateJSON
			flags["trap_disarmed"] = true
		} else {
			narration = "Le piège se déclenche ! L'équipe subit des dégâts."
		}
	case "collect_loot_pass", "loot_chest":
		if rollSuccess {
			flags["loot_chest_opened"] = true
			narration = "L'équipe ouvre le coffre avec succès."
		} else {
			narration = "Le coffre résiste aux efforts du groupe."
		}
	default:
		if rollSuccess {
			narration = fmt.Sprintf("L'équipe réussit l'action %s.", actionID)
		} else {
			narration = fmt.Sprintf("L'équipe échoue sur l'action %s.", actionID)
		}
	}

	state.FlagsJSON = rpgMustJSON(flags)
	sessionState := s.coopStateAsSession(state)
	s.checkObjectiveProgress(sessionState, arcMap, flags)
	s.syncSessionToCoop(state, sessionState)
	_ = s.maps.UpdateCoopMapState(ctx, state)
	_ = s.patchCoopSharedDungeonFlags(ctx, party.Id, party.SharedState, actionID, rollSuccess, flags)

	return s.buildCoopMapResponse(ctx, party, state, narration)
}

func (s *RPGMapService) patchCoopSharedDungeonFlags(
	ctx context.Context,
	partyID uint,
	rawState datatypes.JSON,
	actionID string,
	rollSuccess bool,
	flags map[string]any,
) error {
	shared := decodeCoopSharedState(rawState)
	dungeonState, _ := shared["dungeonState"].(map[string]any)
	if dungeonState == nil {
		dungeonState = map[string]any{}
	}
	dungeonFlags, _ := dungeonState["flags"].(map[string]any)
	if dungeonFlags == nil {
		dungeonFlags = map[string]any{}
	}
	dungeonFlags["lastResolvedActionId"] = actionID
	dungeonFlags["lastCoopRollSuccess"] = rollSuccess
	for k, v := range flags {
		if k == "trap_disarmed" || k == "loot_chest_opened" {
			dungeonFlags[k] = v
		}
	}
	dungeonState["flags"] = dungeonFlags
	shared["dungeonState"] = dungeonState
	payload, err := json.Marshal(shared)
	if err != nil {
		return err
	}
	return s.coop.UpdateSharedState(ctx, partyID, datatypes.JSON(payload))
}

func (s *RPGMapService) EnterCoopMap(ctx context.Context, partyCode string) (*models.RPGCoopMapResponse, error) {
	resp, err := s.GetOrCreateCoopMap(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	resp.Narration = "Vous pénétrez ensemble dans la zone. La carte est partagée entre les deux joueurs."
	return resp, nil
}

func (s *RPGMapService) CoopProposeMove(ctx context.Context, partyCode string, userID uint, targetNodeID string) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		return nil, err
	}

	vote := map[string]any{
		"type":         "move",
		"targetNodeId": targetNodeID,
		"proposedBy":   userID,
		"votes":        map[string]bool{fmt.Sprintf("%d", userID): true},
		"status":       "pending",
	}
	state.PendingVoteJSON = rpgMustJSON(vote)
	_ = s.maps.UpdateCoopMapState(ctx, state)

	return s.buildCoopMapResponse(ctx, party, state, fmt.Sprintf("Proposition de déplacement vers %s en attente de vote.", targetNodeID))
}

func (s *RPGMapService) CoopVote(ctx context.Context, partyCode string, userID uint, accept bool) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		return nil, err
	}

	vote := decodeFlags(state.PendingVoteJSON)
	if vote == nil || vote["status"] != "pending" {
		return nil, fmt.Errorf("no pending vote")
	}

	votes, _ := vote["votes"].(map[string]any)
	if votes == nil {
		votes = map[string]any{}
	}
	votes[fmt.Sprintf("%d", userID)] = accept
	vote["votes"] = votes

	members, _ := s.coop.ListMembers(ctx, party.Id)
	allVoted := len(votes) >= len(members) && len(members) >= 2
	allAccepted := true
	for _, v := range votes {
		if b, ok := v.(bool); !ok || !b {
			allAccepted = false
			break
		}
	}

	narration := ""
	if allVoted && allAccepted {
		voteType, _ := vote["type"].(string)
		switch voteType {
		case "move":
			target, _ := vote["targetNodeId"].(string)
			arcMap, _ := s.maps.GetArcMapByID(ctx, state.ArcMapID)
			nodes, edges, _ := parseArcMap(arcMap)
			flags := decodeFlags(state.FlagsJSON)
			discovered := decodeStringSlice(state.DiscoveredJSON)

			if isAdjacent(edges, state.CurrentNodeID, target, discovered, flags) {
				state.CurrentNodeID = target
				discovered = appendUnique(discovered, target)
				state.DiscoveredJSON = rpgMustJSON(discovered)
				positions := decodePlayerPositions(state.PlayerPositions)
				for k := range positions {
					positions[k] = target
				}
				state.PlayerPositions = rpgMustJSON(positions)
				narration = fmt.Sprintf("Déplacement accepté ! Le groupe avance vers %s.", target)
				if n := findNode(nodes, target); n != nil && len(n.Enemies) > 0 {
					combat := s.startCombat(n)
					state.CombatStateJSON = rpgMustJSON(combat)
					narration += " Des ennemis apparaissent !"
				}
			}
		case "separate":
			positions := decodePlayerPositions(state.PlayerPositions)
			targetA, _ := vote["targetA"].(string)
			targetB, _ := vote["targetB"].(string)
			if targetA != "" {
				positions["host"] = targetA
			}
			if targetB != "" {
				positions["ally"] = targetB
			}
			state.PlayerPositions = rpgMustJSON(positions)
			flags := decodeFlags(state.FlagsJSON)
			flags["separated"] = true
			state.FlagsJSON = rpgMustJSON(flags)
			narration = "Le groupe se sépare pour accomplir des objectifs distincts."
		case "regroup":
			positions := decodePlayerPositions(state.PlayerPositions)
			for k := range positions {
				positions[k] = state.CurrentNodeID
			}
			state.PlayerPositions = rpgMustJSON(positions)
			flags := decodeFlags(state.FlagsJSON)
			flags["separated"] = false
			state.FlagsJSON = rpgMustJSON(flags)
			narration = "Le groupe se reforme."
		case "lever":
			flags := decodeFlags(state.FlagsJSON)
			flags["door_unlocked"] = true
			state.FlagsJSON = rpgMustJSON(flags)
			narration = "Le levier est actionné ! La porte se déverrouille pour l'autre joueur."
		}
		vote["status"] = "resolved"
	} else if allVoted {
		vote["status"] = "rejected"
		narration = "La proposition est rejetée."
	} else {
		narration = "Vote enregistré, en attente de l'autre joueur."
	}

	state.PendingVoteJSON = rpgMustJSON(vote)
	_ = s.maps.UpdateCoopMapState(ctx, state)

	return s.buildCoopMapResponse(ctx, party, state, narration)
}

func (s *RPGMapService) CoopProposeSeparate(ctx context.Context, partyCode string, userID uint, targetA, targetB string) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		return nil, err
	}

	vote := map[string]any{
		"type": "separate", "targetA": targetA, "targetB": targetB,
		"proposedBy": userID,
		"votes":      map[string]bool{fmt.Sprintf("%d", userID): true},
		"status":     "pending",
	}
	state.PendingVoteJSON = rpgMustJSON(vote)
	_ = s.maps.UpdateCoopMapState(ctx, state)

	return s.buildCoopMapResponse(ctx, party, state, "Proposition de séparation en attente de vote.")
}

func (s *RPGMapService) CoopProposeRegroup(ctx context.Context, partyCode string, userID uint) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		return nil, err
	}

	vote := map[string]any{
		"type": "regroup", "proposedBy": userID,
		"votes":  map[string]bool{fmt.Sprintf("%d", userID): true},
		"status": "pending",
	}
	state.PendingVoteJSON = rpgMustJSON(vote)
	_ = s.maps.UpdateCoopMapState(ctx, state)

	return s.buildCoopMapResponse(ctx, party, state, "Proposition de regroupement en attente de vote.")
}

func (s *RPGMapService) CoopProposeLever(ctx context.Context, partyCode string, userID uint) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		return nil, err
	}

	vote := map[string]any{
		"type": "lever", "proposedBy": userID,
		"votes":  map[string]bool{fmt.Sprintf("%d", userID): true},
		"status": "pending",
	}
	state.PendingVoteJSON = rpgMustJSON(vote)
	_ = s.maps.UpdateCoopMapState(ctx, state)

	return s.buildCoopMapResponse(ctx, party, state, "Proposition d'actionner le levier en attente de vote.")
}

func (s *RPGMapService) CoopMapAction(ctx context.Context, partyCode string, userID uint, actionID, actionType, targetID string) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
	if err != nil {
		return nil, err
	}
	arcMap, err := s.maps.GetArcMapByID(ctx, state.ArcMapID)
	if err != nil {
		return nil, err
	}

	nodes, _, _ := parseArcMap(arcMap)
	currentNode := findNode(nodes, state.CurrentNodeID)
	if currentNode == nil {
		return nil, fmt.Errorf("invalid current node")
	}

	flags := decodeFlags(state.FlagsJSON)
	narration := ""

	switch actionType {
	case "inspect":
		narration = fmt.Sprintf("Le groupe examine %s. %s", currentNode.Name, currentNode.Description)
	case "detect_trap":
		sessionState := s.coopStateAsSession(state)
		narration = s.detectTraps(sessionState, currentNode)
		state.FlagsJSON = sessionState.FlagsJSON
	case "interact_npc":
		var npc *models.RPGMapNPC
		for i := range currentNode.NPCs {
			if currentNode.NPCs[i].ID == targetID {
				npc = &currentNode.NPCs[i]
				break
			}
		}
		if npc == nil {
			return nil, fmt.Errorf("npc not found on current node")
		}
		narration = fmt.Sprintf("%s : %s", npc.Name, npc.DialogueIntent)
	case "loot":
		var loot *models.RPGMapLoot
		for i := range currentNode.Loot {
			if currentNode.Loot[i].ID == targetID {
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
		sessionState := s.coopStateAsSession(state)
		s.markLootCollected(sessionState, targetID)
		if loot.Type == "key" {
			flags["has_access_pass"] = true
		}
		state.LootStateJSON = sessionState.LootStateJSON
		narration = fmt.Sprintf("Le groupe récupère %s.", loot.Name)
	case "disarm_trap":
		sessionState := s.coopStateAsSession(state)
		narration = s.disarmTraps(sessionState, currentNode)
		state.TrapStateJSON = sessionState.TrapStateJSON
	default:
		narration = fmt.Sprintf("Action %s effectuée.", actionID)
	}

	state.FlagsJSON = rpgMustJSON(flags)
	sessionState := s.coopStateAsSession(state)
	s.checkObjectiveProgress(sessionState, arcMap, flags)
	s.syncSessionToCoop(state, sessionState)
	_ = s.maps.UpdateCoopMapState(ctx, state)

	return s.buildCoopMapResponse(ctx, party, state, narration)
}

func (s *RPGMapService) coopStateAsSession(state *models.RPGCoopMapState) *models.RPGSessionMapState {
	return &models.RPGSessionMapState{
		SessionID:         0,
		ArcMapID:          state.ArcMapID,
		CurrentNodeID:     state.CurrentNodeID,
		ActiveChapterID:   state.ActiveChapterID,
		ActiveObjectiveID: state.ActiveObjectiveID,
		DiscoveredJSON:    state.DiscoveredJSON,
		CompletedJSON:     state.CompletedJSON,
		LockedJSON:        state.LockedJSON,
		FlagsJSON:         state.FlagsJSON,
		NPCStateJSON:      state.NPCStateJSON,
		EnemyStateJSON:    state.EnemyStateJSON,
		LootStateJSON:     state.LootStateJSON,
		TrapStateJSON:     state.TrapStateJSON,
		CombatStateJSON:   state.CombatStateJSON,
		HistoryJSON:       state.HistoryJSON,
	}
}

func (s *RPGMapService) syncSessionToCoop(coop *models.RPGCoopMapState, session *models.RPGSessionMapState) {
	coop.FlagsJSON = session.FlagsJSON
	coop.CompletedJSON = session.CompletedJSON
	coop.LootStateJSON = session.LootStateJSON
	coop.TrapStateJSON = session.TrapStateJSON
	coop.ActiveObjectiveID = session.ActiveObjectiveID
	coop.ActiveChapterID = session.ActiveChapterID
}

func (s *RPGMapService) CoopCombatAction(ctx context.Context, partyCode string, actionID, targetID string) (*models.RPGCoopMapResponse, error) {
	party, err := s.coop.GetByCode(ctx, partyCode)
	if err != nil {
		return nil, err
	}
	state, err := s.maps.GetCoopMapState(ctx, party.Id)
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
				combat.Enemies[i].Health -= 10
				narration = fmt.Sprintf("Le groupe frappe %s.", combat.Enemies[i].Name)
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
		if narration == "" && len(combat.Enemies) > 0 {
			for i := range combat.Enemies {
				if !combat.Enemies[i].Defeated {
					combat.Enemies[i].Health -= 10
					narration = fmt.Sprintf("Le groupe frappe %s.", combat.Enemies[i].Name)
					break
				}
			}
		}
	case "defend":
		narration = "Le groupe se met en position défensive."
	case "flee":
		combat.Active = false
		combat.Fled = true
		narration = "Le groupe prend la fuite."
	default:
		narration = fmt.Sprintf("Action de combat : %s", actionID)
	}
	combat.Turn++
	if narration != "" {
		combat.Log = append(combat.Log, narration)
	}

	if allEnemiesDefeated(combat.Enemies) {
		combat.Active = false
		combat.Victory = true
		flags := decodeFlags(state.FlagsJSON)
		flags["boss_defeated"] = true
		state.FlagsJSON = rpgMustJSON(flags)
	}

	state.CombatStateJSON = rpgMustJSON(combat)
	_ = s.maps.UpdateCoopMapState(ctx, state)

	if narration == "" {
		narration = "Action de combat coop exécutée."
	}
	return s.buildCoopMapResponse(ctx, party, state, narration)
}

func (s *RPGMapService) buildCoopMapResponse(ctx context.Context, party *models.CoopParty, state *models.RPGCoopMapState, narration string) (*models.RPGCoopMapResponse, error) {
	arcMap, err := s.maps.GetArcMapByID(ctx, state.ArcMapID)
	if err != nil {
		return nil, err
	}

	sessionState := &models.RPGSessionMapState{
		SessionID:         0,
		ArcMapID:          state.ArcMapID,
		CurrentNodeID:     state.CurrentNodeID,
		ActiveChapterID:   state.ActiveChapterID,
		ActiveObjectiveID: state.ActiveObjectiveID,
		DiscoveredJSON:    state.DiscoveredJSON,
		CompletedJSON:     state.CompletedJSON,
		LockedJSON:        state.LockedJSON,
		FlagsJSON:         state.FlagsJSON,
		NPCStateJSON:      state.NPCStateJSON,
		EnemyStateJSON:    state.EnemyStateJSON,
		LootStateJSON:     state.LootStateJSON,
		TrapStateJSON:     state.TrapStateJSON,
		CombatStateJSON:   state.CombatStateJSON,
		HistoryJSON:       state.HistoryJSON,
	}

	base, err := s.buildMapResponse(arcMap, sessionState, narration)
	if err != nil {
		return nil, err
	}

	positions := decodePlayerPositions(state.PlayerPositions)
	pendingVote := decodeFlags(state.PendingVoteJSON)
	members, _ := s.coop.ListMembers(ctx, party.Id)
	memberList := make([]map[string]any, 0, len(members))
	for _, m := range members {
		memberList = append(memberList, map[string]any{
			"userId": m.UserID, "role": m.Role, "status": m.Status,
		})
	}

	return &models.RPGCoopMapResponse{
		RPGMapResponse:  *base,
		CoopMapState:    state,
		PlayerPositions: positions,
		PendingVote:     pendingVote,
		Members:         memberList,
	}, nil
}

func decodePlayerPositions(data datatypes.JSON) map[string]string {
	result := map[string]string{}
	_ = json.Unmarshal(data, &result)
	return result
}