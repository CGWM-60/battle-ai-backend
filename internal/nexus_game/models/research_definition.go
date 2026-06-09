package models

import "time"

// ResearchDefinition - research nodes per the 11 branches x 7 tiers in the reference.
// contentId like "research_economy_tier1" or specific names.
type ResearchDefinition struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ContentID string `gorm:"uniqueIndex;size:128" json:"contentId"`
	Domain    string `gorm:"size:32" json:"domain"` // "research"
	Branch    string `gorm:"size:64" json:"branch"` // "economy", "military"...

	NameKey        string `json:"nameKey"`
	DescriptionKey string `json:"descriptionKey"`
	AssetID        string `json:"assetId"`
	AssetsByTier   map[string]string `gorm:"serializer:json" json:"assetsByTier"`

	Tier     int    `json:"tier"` // 1-7 per branch
	Rarity   string `json:"rarity"`

	CostBaseCredits int `json:"costBaseCredits"`
	CostBaseData    int `json:"costBaseData"`
	DurationBaseSeconds int `json:"durationBaseSeconds"`

	EffectsJSON string `gorm:"type:text" json:"effects"` // unlocks, bonuses

	// Dependencies as JSON for simplicity (list of contentIds)
	PrerequisitesJSON string `gorm:"type:text" json:"prerequisites"`

	BalanceVersion string    `json:"balanceVersion"`
	IsPublished    bool      `json:"isPublished"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// PlayerResearch - completed researches for the profile.
type PlayerResearch struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ProfileGamerID uint      `json:"profileGamerId"`
	ContentID      string    `json:"contentId"`
	CompletedAt    time.Time `json:"completedAt"`
}
