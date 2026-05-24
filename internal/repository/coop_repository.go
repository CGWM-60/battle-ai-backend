package repository

import (
	"context"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type CoopRepository struct {
	db *gorm.DB
}

func NewCoopRepository(db *gorm.DB) *CoopRepository {
	return &CoopRepository{db: db}
}

func (r *CoopRepository) ListPartiesByHost(ctx context.Context, hostUserID uint, limit int) ([]models.CoopParty, error) {
	var parties []models.CoopParty
	err := r.db.WithContext(ctx).
		Where("host_user_id = ?", hostUserID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&parties).Error
	return parties, err
}

func (r *CoopRepository) CreateParty(ctx context.Context, party *models.CoopParty) error {
	return r.db.WithContext(ctx).Create(party).Error
}

func (r *CoopRepository) GetByCode(ctx context.Context, code string) (*models.CoopParty, error) {
	var party models.CoopParty
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&party).Error
	if err != nil {
		return nil, err
	}
	return &party, nil
}

func (r *CoopRepository) ListMembers(ctx context.Context, partyID uint) ([]models.CoopPartyMember, error) {
	var members []models.CoopPartyMember
	err := r.db.WithContext(ctx).
		Where("coop_party_id = ?", partyID).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *CoopRepository) Join(ctx context.Context, partyID uint, userID uint, role string) error {
	now := time.Now()
	member := models.CoopPartyMember{
		CoopPartyID: partyID,
		UserID:      userID,
		Role:        role,
		Status:      "joined",
		JoinedAt:    &now,
		LastSeenAt:  &now,
	}
	return r.db.WithContext(ctx).Where(models.CoopPartyMember{
		CoopPartyID: partyID,
		UserID:      userID,
	}).Assign(member).FirstOrCreate(&member).Error
}

func (r *CoopRepository) Leave(ctx context.Context, partyID uint, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.CoopPartyMember{}).
		Where("coop_party_id = ? AND user_id = ?", partyID, userID).
		Updates(map[string]any{"status": "left", "last_seen_at": &now}).Error
}

func (r *CoopRepository) Ready(ctx context.Context, partyID uint, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.CoopPartyMember{}).
		Where("coop_party_id = ? AND user_id = ?", partyID, userID).
		Updates(map[string]any{"status": "ready", "last_seen_at": &now}).Error
}

func (r *CoopRepository) UpdateSharedState(ctx context.Context, partyID uint, hostUserID uint, state any) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.CoopParty{}).
		Where("id = ? AND host_user_id = ?", partyID, hostUserID).
		Updates(map[string]any{"shared_state": state, "last_activity_at": &now}).Error
}
