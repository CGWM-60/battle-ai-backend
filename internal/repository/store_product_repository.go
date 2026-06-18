package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type StoreProductRepository struct {
	db *gorm.DB
}

func NewStoreProductRepository(db *gorm.DB) *StoreProductRepository {
	return &StoreProductRepository{db: db}
}

func (r *StoreProductRepository) Create(ctx context.Context, product *models.StoreProduct) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *StoreProductRepository) Save(ctx context.Context, product *models.StoreProduct) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *StoreProductRepository) GetBySlug(ctx context.Context, slug string) (*models.StoreProduct, error) {
	return r.GetBySKU(ctx, slug)
}

func (r *StoreProductRepository) GetBySKU(ctx context.Context, sku string) (*models.StoreProduct, error) {
	var product models.StoreProduct
	err := r.db.WithContext(ctx).
		Where("slug = ? OR store_product_id = ?", sku, sku).
		First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *StoreProductRepository) ListActive(ctx context.Context, productType string) ([]models.StoreProduct, error) {
	var products []models.StoreProduct
	query := r.db.WithContext(ctx).Where("status = ?", models.StoreProductStatusActive)
	if productType != "" {
		query = query.Where("product_type = ?", productType)
	}
	err := query.Order("position ASC, id ASC").Find(&products).Error
	return products, err
}

func (r *StoreProductRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.StoreProduct{}).Count(&count).Error
	return count, err
}