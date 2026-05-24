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
	return party, s.coop.Join(ctx, party.Id, userID, "player")
}

func (s *CoopService) Leave(ctx context.Context, code string, userID uint) error {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	return s.coop.Leave(ctx, party.Id, userID)
}

func (s *CoopService) Ready(ctx context.Context, code string, userID uint) error {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	return s.coop.Ready(ctx, party.Id, userID)
}

func (s *CoopService) Members(ctx context.Context, code string) ([]models.CoopPartyMember, error) {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.coop.ListMembers(ctx, party.Id)
}

func (s *CoopService) UpdateState(ctx context.Context, code string, hostUserID uint, state map[string]any) error {
	party, err := s.coop.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(state)
	return s.coop.UpdateSharedState(ctx, party.Id, hostUserID, datatypes.JSON(payload))
}
