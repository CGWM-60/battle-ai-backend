package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type BattleRepository struct {
	db *gorm.DB
}

func NewBattleRepository(db *gorm.DB) *BattleRepository {
	return &BattleRepository{db: db}
}

func (r *BattleRepository) Create(ctx context.Context, battle *models.BattleSave) error {
	return r.db.WithContext(ctx).Create(battle).Error
}

func (r *BattleRepository) GetOwnedByID(ctx context.Context, id uint, ownerID uint) (*models.BattleSave, error) {
	var battle models.BattleSave
	err := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&battle).Error
	if err != nil {
		return nil, err
	}

	return &battle, nil
}

func (r *BattleRepository) ListByOwner(ctx context.Context, ownerID uint, limit int) ([]models.BattleSave, error) {
	var battles []models.BattleSave
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&battles).Error
	return battles, err
}

func (r *BattleRepository) ListTurns(ctx context.Context, battleID uint) ([]models.BattleSaveTurn, error) {
	var turns []models.BattleSaveTurn
	err := r.db.WithContext(ctx).
		Where("battle_save_id = ?", battleID).
		Order("sequence ASC").
		Find(&turns).Error
	return turns, err
}

func (r *BattleRepository) AppendTurn(ctx context.Context, turn *models.BattleSaveTurn) error {
	return r.db.WithContext(ctx).Create(turn).Error
}

func (r *BattleRepository) NextTurnSequence(ctx context.Context, battleID uint) (int, error) {
	var last models.BattleSaveTurn
	err := r.db.WithContext(ctx).
		Where("battle_save_id = ?", battleID).
		Order("sequence DESC").
		First(&last).Error
	if err == gorm.ErrRecordNotFound {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}

	return last.Sequence + 1, nil
}

func (r *BattleRepository) UpdateFields(ctx context.Context, battleID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&models.BattleSave{}).
		Where("id = ?", battleID).
		Updates(fields).Error
}
