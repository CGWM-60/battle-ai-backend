package repository

import (
	"context"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, subscription *models.UserSubscription) error {
	return r.db.WithContext(ctx).Create(subscription).Error
}

func (r *SubscriptionRepository) Save(ctx context.Context, subscription *models.UserSubscription) error {
	return r.db.WithContext(ctx).Save(subscription).Error
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id uint) (*models.UserSubscription, error) {
	var subscription models.UserSubscription
	err := r.db.WithContext(ctx).Preload("StoreProduct").Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepository) GetByProviderRef(ctx context.Context, provider string, providerRef string) (*models.UserSubscription, error) {
	var subscription models.UserSubscription
	err := r.db.WithContext(ctx).
		Preload("StoreProduct").
		Where("provider = ? AND provider_subscription_id = ?", provider, providerRef).
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepository) GetActiveByUserID(ctx context.Context, userID uint) (*models.UserSubscription, error) {
	now := time.Now()
	var subscription models.UserSubscription
	err := r.db.WithContext(ctx).
		Preload("StoreProduct").
		Where(
			"user_id = ? AND status = ? AND (current_period_end IS NULL OR current_period_end >= ?)",
			userID, models.SubscriptionStatusActive, now,
		).
		Order("current_period_end DESC").
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}