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

func (r *CoopRepository) coopPartyQuery(ctx context.Context) *gorm.DB {
	return PreloadRolePlayQuestVisuals(r.db.WithContext(ctx), "RolePlaySession").
		Preload("Members", func(db *gorm.DB) *gorm.DB {
			return db.
				Select("id", "created_at", "updated_at", "coop_party_id", "user_id", "role", "status", "joined_at", "last_seen_at", "character_id").
				Where("status <> ?", "left").
				Order("created_at ASC")
		}).
		Preload("Members.User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "pseudo", "email", "avatar", "xp", "coin")
		}).
		Preload("Members.Character")
}

func (r *CoopRepository) ListPartiesByHost(ctx context.Context, hostUserID uint, limit int) ([]models.CoopParty, error) {
	var parties []models.CoopParty
	err := r.coopPartyQuery(ctx).
		Where("host_user_id = ?", hostUserID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&parties).Error
	if err == nil {
		for index := range parties {
			ApplyCoopPartyImageCompatibility(&parties[index])
		}
	}
	return parties, err
}

func (r *CoopRepository) CreateParty(ctx context.Context, party *models.CoopParty) error {
	return r.db.WithContext(ctx).Create(party).Error
}

func (r *CoopRepository) GetByCode(ctx context.Context, code string) (*models.CoopParty, error) {
	var party models.CoopParty
	err := r.coopPartyQuery(ctx).Where("code = ?", code).First(&party).Error
	if err != nil {
		return nil, err
	}
	ApplyCoopPartyImageCompatibility(&party)
	return &party, nil
}

func (r *CoopRepository) ListMembers(ctx context.Context, partyID uint) ([]models.CoopPartyMember, error) {
	var members []models.CoopPartyMember
	err := r.db.WithContext(ctx).
		Select("id", "created_at", "updated_at", "coop_party_id", "user_id", "role", "status", "joined_at", "last_seen_at", "character_id").
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "pseudo", "email", "avatar", "xp", "coin")
		}).
		Preload("Character").
		Where("coop_party_id = ? AND status <> ?", partyID, "left").
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

func (r *CoopRepository) TouchMember(ctx context.Context, partyID uint, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.CoopPartyMember{}).
		Where("coop_party_id = ? AND user_id = ? AND status <> ?", partyID, userID, "left").
		Updates(map[string]any{"last_seen_at": &now}).Error
}

func (r *CoopRepository) UpdatePartyStatus(ctx context.Context, partyID uint, status string) error {
	if status == "" {
		return nil
	}

	return r.db.WithContext(ctx).
		Model(&models.CoopParty{}).
		Where("id = ?", partyID).
		Update("status", status).Error
}

func (r *CoopRepository) UpdateSharedState(ctx context.Context, partyID uint, state any) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.CoopParty{}).
		Where("id = ?", partyID).
		Updates(map[string]any{"shared_state": state, "last_activity_at": &now}).Error
}
