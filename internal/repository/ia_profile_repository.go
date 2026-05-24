package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type IAProfileRepository struct {
	db *gorm.DB
}

func NewIAProfileRepository(db *gorm.DB) *IAProfileRepository {
	return &IAProfileRepository{db: db}
}

func (r *IAProfileRepository) GetOwnedByID(ctx context.Context, id uint, ownerID uint) (*models.IAProfile, error) {
	var profile models.IAProfile
	err := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&profile).Error
	if err != nil {
		return nil, err
	}

	return &profile, nil
}
