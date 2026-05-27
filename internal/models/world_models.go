package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type World struct {
	Id                  uint `gorm:"primaryKey" json:"id"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
	Name                string         `gorm:"size:120;index" json:"name"`
	Status              string         `gorm:"size:32;index" json:"status"`
	Seed                string         `gorm:"size:80;uniqueIndex" json:"seed"`
	AIProvider          string         `gorm:"size:64;index" json:"aiProvider"`
	MaxPlayers          int            `json:"maxPlayers"`
	CurrentPlayers      int            `gorm:"index" json:"currentPlayers"`
	CurrentCycle        int            `json:"currentCycle"`
	GlobalTensionLevel  int            `gorm:"index" json:"globalTensionLevel"`
	GlobalWeatherRisk   int            `gorm:"index" json:"globalWeatherRisk"`
	GlobalEconomicState string         `gorm:"size:48;index" json:"globalEconomicState"`
	LastSimulationAt    *time.Time     `gorm:"index" json:"lastSimulationAt"`
	LastDailyMessageAt  *time.Time     `gorm:"index" json:"lastDailyMessageAt"`
	Continents          []Continent    `gorm:"foreignKey:WorldID" json:"continents,omitempty"`
}

type Continent struct {
	Id                uint `gorm:"primaryKey" json:"id"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	WorldID           uint           `gorm:"index:idx_world_continent,unique" json:"worldId"`
	World             World          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Name              string         `gorm:"size:120;index" json:"name"`
	Index             int            `gorm:"index:idx_world_continent,unique" json:"index"`
	Status            string         `gorm:"size:32;index" json:"status"`
	MaxPlayers        int            `json:"maxPlayers"`
	CurrentPlayers    int            `gorm:"index" json:"currentPlayers"`
	ClimateState      string         `gorm:"size:64;index" json:"climateState"`
	PoliticalState    string         `gorm:"size:64;index" json:"politicalState"`
	EconomicState     string         `gorm:"size:64;index" json:"economicState"`
	TensionLevel      int            `gorm:"index" json:"tensionLevel"`
	AIBehaviorProfile string         `gorm:"size:64;index" json:"aiBehaviorProfile"`
}

type PlayerSave struct {
	Id                    uint `gorm:"primaryKey" json:"id"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
	PlayerID              uint           `gorm:"uniqueIndex" json:"playerId"`
	Player                Users          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	WorldID               uint           `gorm:"index" json:"worldId"`
	World                 World          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	ContinentID           uint           `gorm:"index" json:"continentId"`
	Continent             Continent      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	CityName              string         `gorm:"size:120" json:"cityName"`
	CityLevel             int            `gorm:"index" json:"cityLevel"`
	XP                    int64          `json:"xp"`
	Population            int64          `json:"population"`
	Satisfaction          int            `json:"satisfaction"`
	Food                  int64          `json:"food"`
	Energy                int64          `json:"energy"`
	Credits               int64          `json:"credits"`
	Gems                  int64          `json:"gems"`
	BuildingsJSON         datatypes.JSON `gorm:"type:json" json:"buildingsJson"`
	ConstructionQueueJSON datatypes.JSON `gorm:"type:json" json:"constructionQueueJson"`
	ResearchJSON          datatypes.JSON `gorm:"type:json" json:"researchJson"`
	InventoryJSON         datatypes.JSON `gorm:"type:json" json:"inventoryJson"`
	ActiveEffectsJSON     datatypes.JSON `gorm:"type:json" json:"activeEffectsJson"`
	Version               int            `gorm:"index" json:"version"`
	LastSyncedAt          *time.Time     `gorm:"index" json:"lastSyncedAt"`
	LastClientVersion     int            `json:"lastClientVersion"`
}

type PlayerBuilding struct {
	Id              uint `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	PlayerID        uint           `gorm:"index" json:"playerId"`
	Player          Users          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	BuildingKey     string         `gorm:"size:120;index" json:"buildingKey"`
	Level           int            `json:"level"`
	PositionX       int            `json:"positionX"`
	PositionY       int            `json:"positionY"`
	State           string         `gorm:"size:32;index" json:"state"`
	StartedAt       *time.Time     `gorm:"index" json:"startedAt"`
	CompletedAt     *time.Time     `gorm:"index" json:"completedAt"`
	LastCollectedAt *time.Time     `gorm:"index" json:"lastCollectedAt"`
}

type ChatMessage struct {
	Id           uint `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	WorldID      *uint          `gorm:"index" json:"worldId"`
	ContinentID  *uint          `gorm:"index" json:"continentId"`
	GuildID      *uint          `gorm:"index" json:"guildId"`
	PlayerID     uint           `gorm:"index" json:"playerId"`
	Player       Users          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ChannelType  string         `gorm:"size:32;index" json:"channelType"`
	Message      string         `gorm:"type:text" json:"message"`
	MetadataJSON datatypes.JSON `gorm:"type:json" json:"metadataJson"`
	ModeratedAt  *time.Time     `gorm:"index" json:"moderatedAt"`
}

type Guild struct {
	Id            uint `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	WorldID       uint           `gorm:"index:idx_guild_world_name,unique;index:idx_guild_world_tag,unique" json:"worldId"`
	World         World          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Name          string         `gorm:"size:120;index:idx_guild_world_name,unique" json:"name"`
	Tag           string         `gorm:"size:16;index:idx_guild_world_tag,unique" json:"tag"`
	Description   string         `gorm:"type:text" json:"description"`
	OwnerPlayerID uint           `gorm:"index" json:"ownerPlayerId"`
	Level         int            `json:"level"`
	XP            int64          `json:"xp"`
	MaxMembers    int            `json:"maxMembers"`
	Visibility    string         `gorm:"size:32;index" json:"visibility"`
	RequiredLevel int            `json:"requiredLevel"`
	Members       []GuildMember  `gorm:"foreignKey:GuildID" json:"members,omitempty"`
}

type GuildMember struct {
	Id           uint `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	GuildID      uint           `gorm:"index:idx_guild_member,unique" json:"guildId"`
	Guild        Guild          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PlayerID     uint           `gorm:"uniqueIndex;index:idx_guild_member,unique" json:"playerId"`
	Player       Users          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Role         string         `gorm:"size:32;index" json:"role"`
	JoinedAt     time.Time      `gorm:"index" json:"joinedAt"`
	LastActiveAt *time.Time     `gorm:"index" json:"lastActiveAt"`
}

type GuildInvite struct {
	Id              uint `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	GuildID         uint           `gorm:"index" json:"guildId"`
	Guild           Guild          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	InviterPlayerID uint           `gorm:"index" json:"inviterPlayerId"`
	InvitedPlayerID uint           `gorm:"index" json:"invitedPlayerId"`
	Status          string         `gorm:"size:32;index" json:"status"`
	ExpiresAt       time.Time      `gorm:"index" json:"expiresAt"`
}

type AIWorldFaction struct {
	Id                uint `gorm:"primaryKey" json:"id"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	WorldID           uint           `gorm:"index" json:"worldId"`
	ContinentID       *uint          `gorm:"index" json:"continentId"`
	Name              string         `gorm:"size:120;index" json:"name"`
	Type              string         `gorm:"size:64;index" json:"type"`
	Aggressiveness    int            `json:"aggressiveness"`
	Diplomacy         int            `json:"diplomacy"`
	Economy           int            `json:"economy"`
	MilitaryPower     int            `json:"militaryPower"`
	ClimateResistance int            `json:"climateResistance"`
	Status            string         `gorm:"size:32;index" json:"status"`
}

type Conflict struct {
	Id            uint `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	WorldID       uint           `gorm:"index" json:"worldId"`
	ContinentID   *uint          `gorm:"index" json:"continentId"`
	AttackerType  string         `gorm:"size:32;index" json:"attackerType"`
	AttackerID    uint           `gorm:"index" json:"attackerId"`
	DefenderType  string         `gorm:"size:32;index" json:"defenderType"`
	DefenderID    uint           `gorm:"index" json:"defenderId"`
	Title         string         `gorm:"size:180" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	Intensity     int            `gorm:"index" json:"intensity"`
	RiskLevel     string         `gorm:"size:32;index" json:"riskLevel"`
	Status        string         `gorm:"size:32;index" json:"status"`
	StartsAt      time.Time      `gorm:"index" json:"startsAt"`
	EndsAt        time.Time      `gorm:"index" json:"endsAt"`
	RewardsJSON   datatypes.JSON `gorm:"type:json" json:"rewardsJson"`
	PenaltiesJSON datatypes.JSON `gorm:"type:json" json:"penaltiesJson"`
	CreatedByAI   bool           `gorm:"index" json:"createdByAi"`
}

type ConflictAction struct {
	Id         uint `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time
	ConflictID uint           `gorm:"index" json:"conflictId"`
	PlayerID   uint           `gorm:"index" json:"playerId"`
	ActionType string         `gorm:"size:64;index" json:"actionType"`
	Payload    datatypes.JSON `gorm:"type:json" json:"payload"`
}

type WeatherEvent struct {
	Id          uint `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	WorldID     uint           `gorm:"index" json:"worldId"`
	ContinentID uint           `gorm:"index" json:"continentId"`
	Type        string         `gorm:"size:64;index" json:"type"`
	Severity    int            `gorm:"index" json:"severity"`
	Title       string         `gorm:"size:180" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	StartsAt    time.Time      `gorm:"index" json:"startsAt"`
	EndsAt      time.Time      `gorm:"index" json:"endsAt"`
	EffectsJSON datatypes.JSON `gorm:"type:json" json:"effectsJson"`
	CreatedByAI bool           `gorm:"index" json:"createdByAi"`
}

type GameEvent struct {
	Id               uint `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	WorldID          uint           `gorm:"index" json:"worldId"`
	ContinentID      *uint          `gorm:"index" json:"continentId"`
	GuildID          *uint          `gorm:"index" json:"guildId"`
	PlayerID         *uint          `gorm:"index" json:"playerId"`
	Title            string         `gorm:"size:180" json:"title"`
	Description      string         `gorm:"type:text" json:"description"`
	Type             string         `gorm:"size:64;index" json:"type"`
	Difficulty       string         `gorm:"size:32;index" json:"difficulty"`
	Status           string         `gorm:"size:32;index" json:"status"`
	StartsAt         time.Time      `gorm:"index" json:"startsAt"`
	EndsAt           time.Time      `gorm:"index" json:"endsAt"`
	DurationMinutes  int            `json:"durationMinutes"`
	RewardsJSON      datatypes.JSON `gorm:"type:json" json:"rewardsJson"`
	RequirementsJSON datatypes.JSON `gorm:"type:json" json:"requirementsJson"`
	ConsequencesJSON datatypes.JSON `gorm:"type:json" json:"consequencesJson"`
	CreatedByAI      bool           `gorm:"index" json:"createdByAi"`
}

type GameEventParticipation struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	EventID   uint           `gorm:"index:idx_event_player,unique" json:"eventId"`
	PlayerID  uint           `gorm:"index:idx_event_player,unique" json:"playerId"`
	Status    string         `gorm:"size:32;index" json:"status"`
	Payload   datatypes.JSON `gorm:"type:json" json:"payload"`
}

type GameEventClaim struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	EventID   uint           `gorm:"index:idx_event_claim_player,unique" json:"eventId"`
	PlayerID  uint           `gorm:"index:idx_event_claim_player,unique" json:"playerId"`
	Reward    datatypes.JSON `gorm:"type:json" json:"reward"`
}

type DailyAIMessage struct {
	Id                   uint `gorm:"primaryKey" json:"id"`
	CreatedAt            time.Time
	WorldID              uint           `gorm:"index" json:"worldId"`
	ContinentID          uint           `gorm:"index" json:"continentId"`
	PlayerID             uint           `gorm:"index" json:"playerId"`
	Title                string         `gorm:"size:160" json:"title"`
	Message              string         `gorm:"type:text" json:"message"`
	Tone                 string         `gorm:"size:64;index" json:"tone"`
	RelatedEventsJSON    datatypes.JSON `gorm:"type:json" json:"relatedEventsJson"`
	RelatedConflictsJSON datatypes.JSON `gorm:"type:json" json:"relatedConflictsJson"`
	IsRead               bool           `gorm:"index" json:"isRead"`
}

type AIWorldDecision struct {
	Id                 uint `gorm:"primaryKey" json:"id"`
	CreatedAt          time.Time
	WorldID            uint           `gorm:"index" json:"worldId"`
	ContinentID        *uint          `gorm:"index" json:"continentId"`
	Type               string         `gorm:"size:64;index" json:"type"`
	InputSnapshotJSON  datatypes.JSON `gorm:"type:json" json:"inputSnapshotJson"`
	OutputDecisionJSON datatypes.JSON `gorm:"type:json" json:"outputDecisionJson"`
	AppliedChangesJSON datatypes.JSON `gorm:"type:json" json:"appliedChangesJson"`
	Provider           string         `gorm:"size:64;index" json:"provider"`
	Model              string         `gorm:"size:160;index" json:"model"`
	Status             string         `gorm:"size:32;index" json:"status"`
	Error              string         `gorm:"type:text" json:"error"`
}

type BuildingDefinition struct {
	Id                     uint `gorm:"primaryKey" json:"id"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
	DeletedAt              gorm.DeletedAt  `gorm:"index" json:"-"`
	Key                    string          `gorm:"size:120;uniqueIndex" json:"key"`
	Name                   string          `gorm:"size:160;index" json:"name"`
	Description            string          `gorm:"type:text" json:"description"`
	Category               string          `gorm:"size:64;index" json:"category"`
	MaxLevel               int             `json:"maxLevel"`
	BaseCostJSON           datatypes.JSON  `gorm:"type:json" json:"baseCostJson"`
	LevelCostFormulaJSON   datatypes.JSON  `gorm:"type:json" json:"levelCostFormulaJson"`
	EffectsJSON            datatypes.JSON  `gorm:"type:json" json:"effectsJson"`
	UnlockRequirementsJSON datatypes.JSON  `gorm:"type:json" json:"unlockRequirementsJson"`
	IsActive               bool            `gorm:"index" json:"isActive"`
	SortOrder              int             `gorm:"index" json:"sortOrder"`
	Assets                 []BuildingAsset `gorm:"foreignKey:BuildingDefinitionID" json:"assets,omitempty"`
}

type BuildingAsset struct {
	Id                   uint `gorm:"primaryKey" json:"id"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt     `gorm:"index" json:"-"`
	BuildingDefinitionID uint               `gorm:"index" json:"buildingDefinitionId"`
	BuildingDefinition   BuildingDefinition `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Level                int                `gorm:"index" json:"level"`
	Variant              string             `gorm:"size:64;index" json:"variant"`
	ImageURL             string             `gorm:"size:500" json:"imageUrl"`
	ImageHash            string             `gorm:"size:128;index" json:"imageHash"`
	ImageSize            int64              `json:"imageSize"`
	Version              int                `gorm:"index" json:"version"`
	IsActive             bool               `gorm:"index" json:"isActive"`
}

type BuildingCatalogVersion struct {
	Id          uint `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int            `gorm:"uniqueIndex" json:"version"`
	PublishedAt *time.Time     `gorm:"index" json:"publishedAt"`
	Changelog   datatypes.JSON `gorm:"type:json" json:"changelog"`
}

type AdminAuditLog struct {
	Id         uint `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time
	AdminID    string         `gorm:"size:120;index" json:"adminId"`
	Action     string         `gorm:"size:120;index" json:"action"`
	TargetType string         `gorm:"size:80;index" json:"targetType"`
	TargetID   string         `gorm:"size:80;index" json:"targetId"`
	BeforeJSON datatypes.JSON `gorm:"type:json" json:"beforeJson"`
	AfterJSON  datatypes.JSON `gorm:"type:json" json:"afterJson"`
	IPAddress  string         `gorm:"size:80" json:"ipAddress"`
	UserAgent  string         `gorm:"size:255" json:"userAgent"`
}
