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

	// MMO city-builder constraints and unlock tree.
	SlotsMax         int    `json:"slotsMax"` // max buildings of this type per city
	AvailableAtSpawn bool   `json:"availableAtSpawn"`
	UnlockEra        string `gorm:"size:32" json:"unlockEra"`  // 0,1,2,3,4,nexus
	UnlockType       string `gorm:"size:16" json:"unlockType"` // AND, OR
	UnlockMessage    string `gorm:"type:text" json:"unlockMessage"`
	NonConstructible bool   `json:"nonConstructible"`

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

	DurationBaseSeconds int     `json:"durationBaseSeconds"`
	DurationMultiplier  float64 `json:"durationMultiplier"`
	MilestoneReduction  float64 `json:"milestoneReduction"`

	// Production storage and MMO behavior.
	StorageResource                   string  `gorm:"size:64" json:"storageResource"`
	StorageCapBase                    int     `json:"storageCapBase"`
	StorageGrowth                     float64 `json:"storageGrowth"`
	OverflowBehavior                  string  `gorm:"size:32" json:"overflowBehavior"` // suspend, loop, decay, realtime
	OverflowDecayPercentPerHour       float64 `json:"overflowDecayPercentPerHour"`
	ProductionBasePerHour             int     `json:"productionBasePerHour"`
	ProductionGrowth                  float64 `json:"productionGrowth"`
	HarvestRecommendedIntervalSeconds int     `json:"harvestRecommendedIntervalSeconds"`

	// Effects (simplified; full in effects JSON or separate table later)
	EffectsJSON   string `gorm:"type:text" json:"effects"` // array of effect objects
	SynergiesJSON string `gorm:"type:text" json:"synergies"`
	RisksJSON     string `gorm:"type:text" json:"risks"`
	AIActionsJSON string `gorm:"type:text" json:"aiActions"`

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
