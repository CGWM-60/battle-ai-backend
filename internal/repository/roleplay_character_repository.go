package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type RolePlayCharacterRepository struct {
	db *gorm.DB
}

func NewRolePlayCharacterRepository(db *gorm.DB) *RolePlayCharacterRepository {
	return &RolePlayCharacterRepository{db: db}
}

func (r *RolePlayCharacterRepository) ListByUser(ctx context.Context, userID uint, limit int) ([]models.RolePlayCharacter, error) {
	var characters []models.RolePlayCharacter
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&characters).Error
	return characters, err
}

func (r *RolePlayCharacterRepository) GetOwnedByID(ctx context.Context, id uint, userID uint) (*models.RolePlayCharacter, error) {
	var character models.RolePlayCharacter
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&character).Error
	if err != nil {
		return nil, err
	}
	return &character, nil
}

func (r *RolePlayCharacterRepository) Create(ctx context.Context, character *models.RolePlayCharacter) error {
	return r.db.WithContext(ctx).Create(character).Error
}

func (r *RolePlayCharacterRepository) Update(ctx context.Context, id uint, userID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&models.RolePlayCharacter{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(fields).Error
}

func (r *RolePlayCharacterRepository) Delete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.RolePlayCharacter{}).Error
}

func (r *RolePlayCharacterRepository) UpdateLinks(ctx context.Context, id uint, userID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&models.RolePlayCharacter{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(fields).Error
}

func (r *RolePlayCharacterRepository) AttachToCoopMember(ctx context.Context, partyID uint, userID uint, characterID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.CoopPartyMember{}).
		Where("coop_party_id = ? AND user_id = ? AND status <> ?", partyID, userID, "left").
		Update("character_id", characterID).Error
}
