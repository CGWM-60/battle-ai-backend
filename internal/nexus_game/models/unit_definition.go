package models

import "time"

// UnitDefinition - see NEXUS GAME CONTENT REFERENCE for full 15 units + stats per level 1-30.
// Similar structure to BuildingDefinition for consistency (CRUD, assets, effects).
type UnitDefinition struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ContentID string `gorm:"uniqueIndex;size:128" json:"contentId"`
	Domain    string `gorm:"size:32" json:"domain"` // "unit"
	Type      string `gorm:"size:64" json:"type"`   // "infantry", "drone"...

	NameKey        string `json:"nameKey"`
	DescriptionKey string `json:"descriptionKey"`
	FlavorTextKey  string `json:"flavorTextKey,omitempty"`
	// Maps level numbers ("1".."30") to i18n keys for level-specific descriptions.
	LevelDescriptionKeys map[string]string `gorm:"serializer:json" json:"levelDescriptionKeys"`
	AssetID              string            `json:"assetId"`
	AssetsByTier         map[string]string `gorm:"serializer:json" json:"assetsByTier"`

	MaxLevel int    `json:"maxLevel"`
	Rarity   string `json:"rarity"`

	// Base for formulas
	HealthBase              int `json:"healthBase"`
	AttackBase              int `json:"attackBase"`
	DefenseBase             int `json:"defenseBase"`
	SpeedBase               int `json:"speedBase"`
	TrainingTimeBaseSeconds int `json:"trainingTimeBaseSeconds"`
	UpkeepBase              int `json:"upkeepBase"`

	EffectsJSON string `gorm:"type:text" json:"effects"`

	BalanceVersion string    `json:"balanceVersion"`
	IsPublished    bool      `json:"isPublished"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// PlayerUnit - owned units (for army, training queues etc.)
type PlayerUnit struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	ProfileGamerID uint   `json:"profileGamerId"`
	ContentID      string `json:"contentId"`
	Count          int    `json:"count" gorm:"default:0"`
	// For training queue etc.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
