package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type RPGMapRepository struct {
	db *gorm.DB
}

func NewRPGMapRepository(db *gorm.DB) *RPGMapRepository {
	return &RPGMapRepository{db: db}
}

func (r *RPGMapRepository) GetArcMapByQuestArc(ctx context.Context, questID, arcID uint) (*models.RPGArcMap, error) {
	var arcMap models.RPGArcMap
	err := r.db.WithContext(ctx).
		Where("quest_id = ? AND arc_id = ?", questID, arcID).
		First(&arcMap).Error
	if err != nil {
		return nil, err
	}
	return &arcMap, nil
}

func (r *RPGMapRepository) CreateArcMap(ctx context.Context, arcMap *models.RPGArcMap) error {
	return r.db.WithContext(ctx).Create(arcMap).Error
}

func (r *RPGMapRepository) GetSessionMapState(ctx context.Context, sessionID uint) (*models.RPGSessionMapState, error) {
	var state models.RPGSessionMapState
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *RPGMapRepository) CreateSessionMapState(ctx context.Context, state *models.RPGSessionMapState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

func (r *RPGMapRepository) UpdateSessionMapState(ctx context.Context, state *models.RPGSessionMapState) error {
	return r.db.WithContext(ctx).Save(state).Error
}

func (r *RPGMapRepository) GetArcMapByID(ctx context.Context, id uint) (*models.RPGArcMap, error) {
	var arcMap models.RPGArcMap
	err := r.db.WithContext(ctx).First(&arcMap, id).Error
	if err != nil {
		return nil, err
	}
	return &arcMap, nil
}

func (r *RPGMapRepository) GetCoopMapState(ctx context.Context, partyID uint) (*models.RPGCoopMapState, error) {
	var state models.RPGCoopMapState
	err := r.db.WithContext(ctx).
		Where("coop_party_id = ?", partyID).
		First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *RPGMapRepository) CreateCoopMapState(ctx context.Context, state *models.RPGCoopMapState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

func (r *RPGMapRepository) UpdateCoopMapState(ctx context.Context, state *models.RPGCoopMapState) error {
	return r.db.WithContext(ctx).Save(state).Error
}