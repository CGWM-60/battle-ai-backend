package service

import (
	"context"
	"fmt"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"
)

type ArenaInput struct {
	Name            string
	BattleSaveID    uint
	MaxPlayers      int
	AllowSpectators bool
}

type ArenaService struct {
	arenas  *repository.ArenaRepository
	battles *repository.BattleRepository
}

func NewArenaService(arenas *repository.ArenaRepository, battles *repository.BattleRepository) *ArenaService {
	return &ArenaService{arenas: arenas, battles: battles}
}

func (s *ArenaService) Create(ctx context.Context, hostUserID uint, input ArenaInput) (*models.BattleArena, error) {
	if input.BattleSaveID == 0 {
		return nil, fmt.Errorf("battleSaveId is required")
	}
	if _, err := s.battles.GetOwnedByID(ctx, input.BattleSaveID, hostUserID); err != nil {
		return nil, fmt.Errorf("battle not found for host")
	}
	if input.MaxPlayers <= 0 {
		input.MaxPlayers = 2
	}
	now := time.Now()
	arena := &models.BattleArena{
		Code:            "arena-" + randomCode(),
		Name:            defaultString(input.Name, "Battle Arena"),
		Status:          constants.ArenaStatusWaiting,
		HostUserID:      hostUserID,
		BattleSaveID:    input.BattleSaveID,
		MaxPlayers:      input.MaxPlayers,
		AllowSpectators: input.AllowSpectators,
		LastHeartbeatAt: &now,
	}
	if err := s.arenas.Create(ctx, arena); err != nil {
		return nil, err
	}
	_ = s.arenas.Join(ctx, arena.Id, hostUserID, "host")
	return arena, nil
}

func (s *ArenaService) List(ctx context.Context, limit int) ([]models.BattleArena, error) {
	return s.arenas.List(ctx, limit)
}

func (s *ArenaService) Get(ctx context.Context, code string) (*models.BattleArena, error) {
	return s.arenas.GetByCode(ctx, code)
}

func (s *ArenaService) Join(ctx context.Context, code string, userID uint, spectator bool) (*models.BattleArena, error) {
	arena, err := s.arenas.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	role := "challenger"
	if spectator {
		if !arena.AllowSpectators {
			return nil, fmt.Errorf("spectators are not allowed")
		}
		role = "spectator"
	}
	return arena, s.arenas.Join(ctx, arena.Id, userID, role)
}

func (s *ArenaService) Leave(ctx context.Context, code string, userID uint) error {
	arena, err := s.arenas.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	return s.arenas.Leave(ctx, arena.Id, userID)
}

func (s *ArenaService) Members(ctx context.Context, code string) ([]models.BattleArenaMember, error) {
	arena, err := s.arenas.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.arenas.ListMembers(ctx, arena.Id)
}
