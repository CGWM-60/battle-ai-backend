package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type StoreTransactionRepository struct {
	db *gorm.DB
}

func NewStoreTransactionRepository(db *gorm.DB) *StoreTransactionRepository {
	return &StoreTransactionRepository{db: db}
}

func (r *StoreTransactionRepository) Create(ctx context.Context, txModel *models.StoreTransaction) error {
	return r.db.WithContext(ctx).Create(txModel).Error
}

func (r *StoreTransactionRepository) GetByID(ctx context.Context, id uint) (*models.StoreTransaction, error) {
	var txModel models.StoreTransaction
	err := r.db.WithContext(ctx).Preload("StoreProduct").Where("id = ?", id).First(&txModel).Error
	if err != nil {
		return nil, err
	}
	return &txModel, nil
}

func (r *StoreTransactionRepository) GetByProviderRef(ctx context.Context, provider string, providerRef string) (*models.StoreTransaction, error) {
	var txModel models.StoreTransaction
	err := r.db.WithContext(ctx).
		Preload("StoreProduct").
		Where("provider = ? AND provider_ref = ?", provider, providerRef).
		First(&txModel).Error
	if err != nil {
		return nil, err
	}
	return &txModel, nil
}

func (r *StoreTransactionRepository) ListByUserID(ctx context.Context, userID uint, limit int) ([]models.StoreTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	var transactions []models.StoreTransaction
	err := r.db.WithContext(ctx).
		Preload("StoreProduct").
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&transactions).Error
	return transactions, err
}