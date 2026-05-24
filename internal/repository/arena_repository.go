package repository

import (
	"context"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type ArenaRepository struct {
	db *gorm.DB
}

func NewArenaRepository(db *gorm.DB) *ArenaRepository {
	return &ArenaRepository{db: db}
}

func (r *ArenaRepository) List(ctx context.Context, limit int) ([]models.BattleArena, error) {
	var arenas []models.BattleArena
	err := r.db.WithContext(ctx).
		Order("updated_at DESC").
		Limit(limit).
		Find(&arenas).Error
	return arenas, err
}

func (r *ArenaRepository) GetByCode(ctx context.Context, code string) (*models.BattleArena, error) {
	var arena models.BattleArena
	err := r.db.WithContext(ctx).
		Where("code = ?", code).
		First(&arena).Error
	if err != nil {
		return nil, err
	}

	return &arena, nil
}

func (r *ArenaRepository) Create(ctx context.Context, arena *models.BattleArena) error {
	return r.db.WithContext(ctx).Create(arena).Error
}

func (r *ArenaRepository) ListMembers(ctx context.Context, arenaID uint) ([]models.BattleArenaMember, error) {
	var members []models.BattleArenaMember
	err := r.db.WithContext(ctx).
		Where("arena_id = ?", arenaID).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *ArenaRepository) Join(ctx context.Context, arenaID uint, userID uint, role string) error {
	now := time.Now()
	member := models.BattleArenaMember{
		ArenaID:    arenaID,
		UserID:     userID,
		Role:       role,
		Status:     "joined",
		JoinedAt:   &now,
		LastSeenAt: &now,
	}
	return r.db.WithContext(ctx).Where(models.BattleArenaMember{
		ArenaID: arenaID,
		UserID:  userID,
	}).Assign(member).FirstOrCreate(&member).Error
}

func (r *ArenaRepository) Leave(ctx context.Context, arenaID uint, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.BattleArenaMember{}).
		Where("arena_id = ? AND user_id = ?", arenaID, userID).
		Updates(map[string]any{"status": "left", "last_seen_at": &now}).Error
}

func (r *ArenaRepository) Close(ctx context.Context, arenaID uint, hostUserID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.BattleArena{}).
		Where("id = ? AND host_user_id = ?", arenaID, hostUserID).
		Update("status", constants.ArenaStatusClosed).Error
}
