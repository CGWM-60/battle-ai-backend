package repository

import (
	"context"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type EntitlementRepository struct {
	db *gorm.DB
}

func NewEntitlementRepository(db *gorm.DB) *EntitlementRepository {
	return &EntitlementRepository{db: db}
}

func (r *EntitlementRepository) Create(ctx context.Context, entitlement *models.UserEntitlement) error {
	return r.db.WithContext(ctx).Create(entitlement).Error
}

func (r *EntitlementRepository) Save(ctx context.Context, entitlement *models.UserEntitlement) error {
	return r.db.WithContext(ctx).Save(entitlement).Error
}

func (r *EntitlementRepository) ListByUserID(ctx context.Context, userID uint) ([]models.UserEntitlement, error) {
	var entitlements []models.UserEntitlement
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Find(&entitlements).Error
	return entitlements, err
}

func (r *EntitlementRepository) HasActive(ctx context.Context, userID uint, key string, at time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UserEntitlement{}).
		Where(
			"user_id = ? AND `key` = ? AND active = ? AND status = ? AND starts_at <= ? AND (expires_at IS NULL OR expires_at > ?)",
			userID, key, true, constants.BillingEntitlementStatusActive, at, at,
		).
		Count(&count).Error
	return count > 0, err
}

func (r *EntitlementRepository) GetActiveByKey(ctx context.Context, userID uint, key string, at time.Time) (*models.UserEntitlement, error) {
	var entitlement models.UserEntitlement
	err := r.db.WithContext(ctx).
		Where(
			"user_id = ? AND `key` = ? AND active = ? AND status = ? AND starts_at <= ? AND (expires_at IS NULL OR expires_at > ?)",
			userID, key, true, constants.BillingEntitlementStatusActive, at, at,
		).
		Order("expires_at DESC, id DESC").
		First(&entitlement).Error
	if err != nil {
		return nil, err
	}
	return &entitlement, nil
}