package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type LiveRepository struct {
	db *gorm.DB
}

func NewLiveRepository(db *gorm.DB) *LiveRepository {
	return &LiveRepository{db: db}
}

func (r *LiveRepository) CreateSession(ctx context.Context, session *models.LiveSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *LiveRepository) ListSessionsByOwner(ctx context.Context, ownerID uint, limit int) ([]models.LiveSession, error) {
	var sessions []models.LiveSession
	err := PreloadLiveRolePlayQuestVisuals(r.db.WithContext(ctx)).
		Where("owner_id = ?", ownerID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&sessions).Error
	return sessions, err
}

func (r *LiveRepository) GetSessionOwnedByID(ctx context.Context, id uint, ownerID uint) (*models.LiveSession, error) {
	var session models.LiveSession
	err := PreloadLiveRolePlayQuestVisuals(r.db.WithContext(ctx)).
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&session).Error
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *LiveRepository) GetSessionOwnedByChannel(ctx context.Context, channel string, ownerID uint) (*models.LiveSession, error) {
	var session models.LiveSession
	err := PreloadLiveRolePlayQuestVisuals(r.db.WithContext(ctx)).
		Where("channel_key = ? AND owner_id = ?", channel, ownerID).
		First(&session).Error
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *LiveRepository) AppendEvent(ctx context.Context, event *models.LiveEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *LiveRepository) ListEventsAfter(ctx context.Context, sessionID uint, after int, limit int) ([]models.LiveEvent, error) {
	var events []models.LiveEvent
	err := r.db.WithContext(ctx).
		Where("live_session_id = ? AND sequence > ?", sessionID, after).
		Order("sequence ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *LiveRepository) ListSessionsByBattle(ctx context.Context, battleID uint) ([]models.LiveSession, error) {
	var sessions []models.LiveSession
	err := r.db.WithContext(ctx).
		Where("battle_save_id = ? AND status <> ?", battleID, "ended").
		Find(&sessions).Error
	return sessions, err
}

func (r *LiveRepository) NextEventSequence(ctx context.Context, sessionID uint) (int, error) {
	var last models.LiveEvent
	err := r.db.WithContext(ctx).
		Where("live_session_id = ?", sessionID).
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

func (r *LiveRepository) UpdateSessionFields(ctx context.Context, sessionID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.LiveSession{}).Where("id = ?", sessionID).Updates(fields).Error
}
