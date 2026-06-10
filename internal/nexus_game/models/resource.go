package models

import "time"

// ResourceCatalog is the Nexus Games resource catalog.
// It is seeded idempotently from the official resource list and edited via API.
type ResourceCatalog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Code             string    `gorm:"uniqueIndex;size:64;not null" json:"code"`
	Name             string    `gorm:"size:100;not null" json:"name"`
	Category         string    `gorm:"size:32;index" json:"category"`
	IsConsumable     bool      `json:"isConsumable"`
	IsRare           bool      `json:"isRare"`
	IsStorageLimited bool      `json:"isStorageLimited"`
	BaseStorage      int64     `json:"baseStorage"`
	IsActive         bool      `gorm:"default:true" json:"isActive"`
	SortOrder        int       `gorm:"index" json:"sortOrder"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// PlayerResource stores the current stock for one profile and one resource.
type PlayerResource struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	ProfileGamerID     uint      `gorm:"uniqueIndex:idx_player_resource;not null" json:"profileGamerId"`
	ResourceCode       string    `gorm:"uniqueIndex:idx_player_resource;size:64;not null" json:"resourceCode"`
	Amount             int64     `json:"amount"`
	Capacity           int64     `json:"capacity"`
	ProductionPerTick  float64   `json:"productionPerTick"`
	ConsumptionPerTick float64   `json:"consumptionPerTick"`
	BalancePerTick     float64   `json:"balancePerTick"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// PlayerCityStats stores Nexus resource-adjacent city stats not already owned by ProfileGamer.
// ProfileGamer remains the source for population, capacity, morale, energy, and security.
type PlayerCityStats struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ProfileGamerID  uint      `gorm:"uniqueIndex;not null" json:"profileGamerId"`
	StorageCapacity int64     `gorm:"default:1000" json:"storageCapacity"`
	FoodProduction  float64   `json:"foodProduction"`
	FoodConsumption float64   `json:"foodConsumption"`
	FoodBalance     float64   `json:"foodBalance"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ResourceTransaction is the immutable resource audit trail.
type ResourceTransaction struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ProfileGamerID  uint      `gorm:"index;not null" json:"profileGamerId"`
	ResourceCode    string    `gorm:"size:64;index;not null" json:"resourceCode"`
	AmountDelta     int64     `json:"amountDelta"`
	BalanceAfter    int64     `json:"balanceAfter"`
	TransactionType string    `gorm:"size:64;index;not null" json:"transactionType"`
	Source          string    `gorm:"size:128" json:"source"`
	MetadataJSON    string    `gorm:"type:text" json:"metadataJson"`
	CreatedAt       time.Time `gorm:"index" json:"createdAt"`
}

// DailyGrantClaim is one resource grant claim for one server day.
type DailyGrantClaim struct {
	ID              uint             `gorm:"primaryKey" json:"id"`
	ProfileGamerID  uint             `gorm:"uniqueIndex:idx_daily_grant_profile_date;not null" json:"profileGamerId"`
	ClaimedDate     string           `gorm:"uniqueIndex:idx_daily_grant_profile_date;size:10;not null" json:"claimedDate"`
	StreakDay       int              `json:"streakDay"`
	StreakCycleDay  int              `json:"streakCycleDay"`
	RewardResources map[string]int64 `gorm:"serializer:json" json:"rewardResources"`
	CreatedAt       time.Time        `gorm:"index" json:"createdAt"`
}

// DailyGrantConfig stores configurable daily grant defaults per resource.
type DailyGrantConfig struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ResourceCode string    `gorm:"uniqueIndex;size:64;not null" json:"resourceCode"`
	BaseAmount   int64     `json:"baseAmount"`
	IsEnabled    bool      `gorm:"default:true" json:"isEnabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// InitialAllocationLog prevents duplicate first-player allocations.
type InitialAllocationLog struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	ProfileGamerID uint             `gorm:"uniqueIndex;not null" json:"profileGamerId"`
	Resources      map[string]int64 `gorm:"serializer:json" json:"resources"`
	CreatedAt      time.Time        `json:"createdAt"`
}
