package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
)

type CoopPartyInput struct {
	Mode              string
	BattleSaveID      *uint
	RolePlaySessionID *uint
	MaxMembers        int
	SharedState       map[string]any
}

type CoopService struct {
	coop *repository.CoopRepository
}

func NewCoopService(coop *repository.CoopRepository) *CoopService {
	return &CoopService{coop: coop}
}

func (s *CoopService) Create(ctx context.Context, hostUserID uint, input CoopPartyInput) (*models.CoopParty, error) {
	if input.BattleSaveID != nil && input.RolePlaySessionID != nil {
		return nil, fmt.Errorf("coop can target only one game source")
	}
	if input.MaxMembers <= 0 {
		input.MaxMembers = 4
	}
	state, _ := json.Marshal(input.SharedState)
	now := time.Now()
	party := &models.CoopParty{
		Code:              "coop-" + randomCode(),
		Mode:              defaultString(input.Mode, constants.ModeCoop),
		Status:            constants.CoopStatusWaiting,
		HostUserID:        hostUserID,
		BattleSaveID:      input.BattleSaveID,
		RolePlaySessionID: input.RolePlaySessionID,
		MaxMembers:        input.MaxMembers,
		SharedState:       datatypes.JSON(state),
		LastActivityAt:    &now,
	}
	if err := s.coop.CreateParty(ctx, party); err != nil {
		return nil, err
	}
	_ = s.coop.Join(ctx, party.Id, hostUserID, "host")
	return party, nil
}

func (s *CoopService) ListByHost(ctx context.Context, hostUserID uint, limit int) ([]models.CoopParty, error) {
	return s.coop.ListPartiesByHost(ctx, hostUserID, limit)
}

func (s *CoopService) Get(ctx context.Context, code string) (*models.CoopParty, error) {
	return s.coop.GetByCode(ctx, code)
}

func (s *CoopService) Join(ctx context.Context, code string, userID uint) (*models.CoopParty, error) {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if err := s.coop.Join(ctx, party.Id, userID, "player"); err != nil {
		return nil, err
	}
	if err := s.syncPartyStatus(ctx, party.Id, decodeCoopSharedState(party.SharedState)); err != nil {
		return nil, err
	}
	return s.coop.GetByCode(ctx, code)
}

func (s *CoopService) Leave(ctx context.Context, code string, userID uint) error {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := s.coop.Leave(ctx, party.Id, userID); err != nil {
		return err
	}
	return s.syncPartyStatus(ctx, party.Id, decodeCoopSharedState(party.SharedState))
}

func (s *CoopService) Ready(ctx context.Context, code string, userID uint) error {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	return s.coop.Ready(ctx, party.Id, userID)
}

func (s *CoopService) Members(ctx context.Context, code string, userID uint) ([]models.CoopPartyMember, error) {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	_ = s.coop.TouchMember(ctx, party.Id, userID)
	return s.coop.ListMembers(ctx, party.Id)
}

func (s *CoopService) UpdateState(ctx context.Context, code string, userID uint, state map[string]any) (*models.CoopParty, error) {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	members, err := s.coop.ListMembers(ctx, party.Id)
	if err != nil {
		return nil, err
	}
	if !memberCanMutateCoopState(userID, party.HostUserID, members) {
		return nil, fmt.Errorf("only active members can update coop state")
	}
	mergedState := mergeCoopSharedState(decodeCoopSharedState(party.SharedState), state)
	payload, err := json.Marshal(mergedState)
	if err != nil {
		return nil, err
	}
	if err := s.coop.UpdateSharedState(ctx, party.Id, datatypes.JSON(payload)); err != nil {
		return nil, err
	}
	_ = s.coop.TouchMember(ctx, party.Id, userID)
	if err := s.syncPartyStatus(ctx, party.Id, mergedState); err != nil {
		return nil, err
	}
	return s.coop.GetByCode(ctx, code)
}

func (s *CoopService) syncPartyStatus(ctx context.Context, partyID uint, state map[string]any) error {
	members, err := s.coop.ListMembers(ctx, partyID)
	if err != nil {
		return err
	}

	status := constants.CoopStatusWaiting
	if len(members) >= 2 {
		status = constants.CoopStatusRunning
	}

	storyStatus := fmt.Sprint(state["storyStatus"])
	if storyStatus == "completed" {
		status = constants.CoopStatusFinished
	}

	return s.coop.UpdatePartyStatus(ctx, partyID, status)
}

func memberCanMutateCoopState(userID uint, hostUserID uint, members []models.CoopPartyMember) bool {
	if userID == hostUserID {
		return true
	}

	for _, member := range members {
		if member.UserID == userID && member.Status != "left" {
			return true
		}
	}

	return false
}

func decodeCoopSharedState(raw datatypes.JSON) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil || state == nil {
		return map[string]any{}
	}

	return state
}

func mergeCoopSharedState(base map[string]any, patch map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	if patch == nil {
		return base
	}

	for key, value := range patch {
		switch typed := value.(type) {
		case nil:
			delete(base, key)
		case map[string]any:
			existing, _ := base[key].(map[string]any)
			base[key] = mergeCoopSharedState(existing, typed)
		case []any:
			if key == "quickMessages" {
				base[key] = mergeCoopQuickMessages(base[key], typed)
				continue
			}
			base[key] = typed
		default:
			base[key] = value
		}
	}

	return base
}

func mergeCoopQuickMessages(existing any, incoming []any) []any {
	merged := make([]any, 0)
	seen := make(map[string]struct{})

	appendUnique := func(values []any) {
		for _, value := range values {
			key := coopStateValueKey(value)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, value)
		}
	}

	if values, ok := existing.([]any); ok {
		appendUnique(values)
	}
	appendUnique(incoming)

	return merged
}

func coopStateValueKey(value any) string {
	if item, ok := value.(map[string]any); ok {
		return fmt.Sprintf("%v|%v|%v", item["createdAt"], item["author"], item["content"])
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}

	return string(payload)
}
