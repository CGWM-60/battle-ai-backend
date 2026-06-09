package models

import "time"

// BuildingDefinition is the static catalog entry from NEXUS GAME CONTENT REFERENCE v2.0.
// Player instances (PlayerBuilding) reference ContentID and hold current Level + queue state.
// All support levels 1-30.
// Images/assets served by Go server from /nexus-assets/content/buildings/{assetId}_tierX.ext after upload via CRUD.

type BuildingDefinition struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ContentID string `gorm:"uniqueIndex;size:128" json:"contentId"` // e.g. "building_modular_habitat"
	Domain    string `gorm:"size:32" json:"domain"`                 // "building"
	Type      string `gorm:"size:64" json:"type"`                   // "habitation", "energy"...

	NameKey        string `json:"nameKey"`
	DescriptionKey string `json:"descriptionKey"`
	FlavorTextKey  string `json:"flavorTextKey,omitempty"`
	// Maps level numbers ("1".."30") to i18n keys for level-specific descriptions.
	LevelDescriptionKeys map[string]string `gorm:"serializer:json" json:"levelDescriptionKeys"`

	AssetID      string            `json:"assetId"`
	AssetsByTier map[string]string `gorm:"serializer:json" json:"assetsByTier"` // tier1, tier2...

	MaxLevel int    `json:"maxLevel"`
	Rarity   string `json:"rarity"` // common, uncommon...

	// Unlock
	NexusLevelRequired int `json:"nexusLevelRequired"`
	// JSON arrays for requirements (simplified for MVP)
	RequiredBuildingsJSON string `gorm:"type:text" json:"requiredBuildings"` // [{"contentId":"..","level":2}]
	RequiredResearchJSON  string `gorm:"type:text" json:"requiredResearch"`

	// Base values for formulas (level 1)
	CostBaseCredits int `json:"costBaseCredits"`
	CostBaseMetal   int `json:"costBaseMetal"`
	CostBaseData    int `json:"costBaseData"`
	// add other resources as needed

	DurationBaseSeconds int `json:"durationBaseSeconds"`

	// Effects (simplified; full in effects JSON or separate table later)
	EffectsJSON string `gorm:"type:text" json:"effects"` // array of effect objects

	WorkersMin int `json:"workersMin"`
	WorkersMax int `json:"workersMax"`

	AIAgentSlots int      `json:"aiAgentSlots"`
	AIAgentTypes []string `gorm:"serializer:json" json:"aiAgentTypes"`

	BalanceVersion string    `json:"balanceVersion"`
	IsPublished    bool      `json:"isPublished"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// PlayerBuilding is the player-owned instance in their city.
type PlayerBuilding struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	ProfileGamerID        uint       `json:"profileGamerId" gorm:"index"`
	ContentID             string     `json:"contentId"`
	Level                 int        `json:"level" gorm:"default:1"`
	IsConstructing        bool       `json:"isConstructing"`
	ConstructionStartedAt *time.Time `json:"constructionStartedAt"`
	ConstructionEndsAt    *time.Time `json:"constructionEndsAt"`

	AssignedWorkers int `json:"assignedWorkers"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
