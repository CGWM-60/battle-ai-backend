package repositories

import (
	"context"
	"errors"

	"cgwm/battle/internal/nexus_game/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResourceCatalogRepository struct {
	db *gorm.DB
}

func NewResourceCatalogRepository(db *gorm.DB) *ResourceCatalogRepository {
	return &ResourceCatalogRepository{db: db}
}

func (r *ResourceCatalogRepository) List(ctx context.Context, activeOnly bool) ([]models.ResourceCatalog, error) {
	var resources []models.ResourceCatalog
	q := r.db.WithContext(ctx).Order("sort_order ASC, id ASC")
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	if err := q.Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

func (r *ResourceCatalogRepository) Upsert(ctx context.Context, resource *models.ResourceCatalog) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"category",
			"is_consumable",
			"is_rare",
			"is_storage_limited",
			"base_storage",
			"is_active",
			"sort_order",
			"updated_at",
		}),
	}).Create(resource).Error
}

type PlayerResourceRepository struct {
	db *gorm.DB
}

func NewPlayerResourceRepository(db *gorm.DB) *PlayerResourceRepository {
	return &PlayerResourceRepository{db: db}
}

func (r *PlayerResourceRepository) List(ctx context.Context, profileID uint) ([]models.PlayerResource, error) {
	var resources []models.PlayerResource
	if err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Order("resource_code ASC").Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

func (r *PlayerResourceRepository) Get(ctx context.Context, profileID uint, code string) (*models.PlayerResource, error) {
	var resource models.PlayerResource
	if err := r.db.WithContext(ctx).Where("profile_gamer_id = ? AND resource_code = ?", profileID, code).First(&resource).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *PlayerResourceRepository) Create(ctx context.Context, resource *models.PlayerResource) error {
	return r.db.WithContext(ctx).Create(resource).Error
}

func (r *PlayerResourceRepository) Save(ctx context.Context, resource *models.PlayerResource) error {
	return r.db.WithContext(ctx).Save(resource).Error
}

func (r *PlayerResourceRepository) Add(ctx context.Context, profileID uint, code string, delta int64, capacity int64, limited bool) (*models.PlayerResource, error) {
	resource, err := r.Get(ctx, profileID, code)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		resource = &models.PlayerResource{
			ProfileGamerID: profileID,
			ResourceCode:   code,
			Capacity:       capacity,
		}
	}
	resource.Amount += delta
	if resource.Amount < 0 {
		resource.Amount = 0
	}
	if limited && resource.Capacity > 0 && resource.Amount > resource.Capacity {
		resource.Amount = resource.Capacity
	}
	if resource.ID == 0 {
		err = r.Create(ctx, resource)
	} else {
		err = r.Save(ctx, resource)
	}
	if err != nil {
		return nil, err
	}
	return resource, nil
}

type PlayerCityStatsRepository struct {
	db *gorm.DB
}

func NewPlayerCityStatsRepository(db *gorm.DB) *PlayerCityStatsRepository {
	return &PlayerCityStatsRepository{db: db}
}

func (r *PlayerCityStatsRepository) GetOrCreate(ctx context.Context, profileID uint) (*models.PlayerCityStats, error) {
	var stats models.PlayerCityStats
	err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).First(&stats).Error
	if err == nil {
		return &stats, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	stats = models.PlayerCityStats{
		ProfileGamerID:  profileID,
		StorageCapacity: 1000,
	}
	if err := r.db.WithContext(ctx).Create(&stats).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *PlayerCityStatsRepository) Save(ctx context.Context, stats *models.PlayerCityStats) error {
	return r.db.WithContext(ctx).Save(stats).Error
}

type ResourceTransactionRepository struct {
	db *gorm.DB
}

func NewResourceTransactionRepository(db *gorm.DB) *ResourceTransactionRepository {
	return &ResourceTransactionRepository{db: db}
}

func (r *ResourceTransactionRepository) Create(ctx context.Context, tx *models.ResourceTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *ResourceTransactionRepository) List(ctx context.Context, profileID uint, limit int) ([]models.ResourceTransaction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var transactions []models.ResourceTransaction
	if err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Order("created_at DESC, id DESC").Limit(limit).Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}

type DailyGrantClaimRepository struct {
	db *gorm.DB
}

func NewDailyGrantClaimRepository(db *gorm.DB) *DailyGrantClaimRepository {
	return &DailyGrantClaimRepository{db: db}
}

func (r *DailyGrantClaimRepository) FindByDate(ctx context.Context, profileID uint, date string) (*models.DailyGrantClaim, error) {
	var claim models.DailyGrantClaim
	if err := r.db.WithContext(ctx).Where("profile_gamer_id = ? AND claimed_date = ?", profileID, date).First(&claim).Error; err != nil {
		return nil, err
	}
	return &claim, nil
}

func (r *DailyGrantClaimRepository) Last(ctx context.Context, profileID uint) (*models.DailyGrantClaim, error) {
	var claim models.DailyGrantClaim
	if err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Order("claimed_date DESC, id DESC").First(&claim).Error; err != nil {
		return nil, err
	}
	return &claim, nil
}

func (r *DailyGrantClaimRepository) Create(ctx context.Context, claim *models.DailyGrantClaim) error {
	return r.db.WithContext(ctx).Create(claim).Error
}

func (r *DailyGrantClaimRepository) List(ctx context.Context, profileID uint, limit int) ([]models.DailyGrantClaim, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var claims []models.DailyGrantClaim
	if err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Order("claimed_date DESC, id DESC").Limit(limit).Find(&claims).Error; err != nil {
		return nil, err
	}
	return claims, nil
}

type DailyGrantConfigRepository struct {
	db *gorm.DB
}

func NewDailyGrantConfigRepository(db *gorm.DB) *DailyGrantConfigRepository {
	return &DailyGrantConfigRepository{db: db}
}

func (r *DailyGrantConfigRepository) List(ctx context.Context, enabledOnly bool) ([]models.DailyGrantConfig, error) {
	var configs []models.DailyGrantConfig
	q := r.db.WithContext(ctx).Order("resource_code ASC")
	if enabledOnly {
		q = q.Where("is_enabled = ?", true)
	}
	if err := q.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *DailyGrantConfigRepository) Upsert(ctx context.Context, config *models.DailyGrantConfig) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "resource_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"base_amount",
			"is_enabled",
			"updated_at",
		}),
	}).Create(config).Error
}
