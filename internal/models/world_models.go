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

type PlayerActionLog struct {
	Id           uint `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time
	PlayerID     uint           `gorm:"index" json:"playerId"`
	WorldID      *uint          `gorm:"index" json:"worldId"`
	ContinentID  *uint          `gorm:"index" json:"continentId"`
	Action       string         `gorm:"size:120;index" json:"action"`
	TargetType   string         `gorm:"size:80;index" json:"targetType"`
	TargetID     string         `gorm:"size:80;index" json:"targetId"`
	Status       string         `gorm:"size:32;index" json:"status"`
	Error        string         `gorm:"type:text" json:"error"`
	BeforeJSON   datatypes.JSON `gorm:"type:json" json:"beforeJson"`
	AfterJSON    datatypes.JSON `gorm:"type:json" json:"afterJson"`
	MetadataJSON datatypes.JSON `gorm:"type:json" json:"metadataJson"`
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
	PowerScore    int64          `json:"powerScore"`
	LevelScore    int64          `json:"levelScore"`
	MaxMembers    int            `json:"maxMembers"`
	MemberCapacity int           `json:"memberCapacity"`
	Visibility    string         `gorm:"size:32;index" json:"visibility"`
	JoinPolicy    string         `gorm:"size:32" json:"joinPolicy"`
	Language      string         `gorm:"size:32" json:"language"`
	Emblem        string         `gorm:"size:255" json:"emblem"`
	Banner        string         `gorm:"size:255" json:"banner"`
	Reputation    int            `json:"reputation"`
	Influence     int64          `json:"influence"`
	Status        string         `gorm:"size:32;index" json:"status"`
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

type GuildContribution struct {
	Id           uint `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time
	GuildID      uint           `gorm:"index" json:"guildId"`
	Guild        Guild          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PlayerID     uint           `gorm:"index" json:"playerId"`
	Contribution string         `gorm:"size:64;index" json:"contribution"`
	Amount       int64          `json:"amount"`
	Payload      datatypes.JSON `gorm:"type:json" json:"payload"`
}

// GuildTreasury - Coffre commun de la guilde (spec point 10)
type GuildTreasury struct {
	Id            uint `gorm:"primaryKey" json:"id"`
	GuildID       uint           `gorm:"index" json:"guildId"`
	Guild         Guild          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Credits       int64          `json:"credits"`
	Food          int64          `json:"food"`
	Energy        int64          `json:"energy"`
	Materials     int64          `json:"materials"`
	RareResources int64          `json:"rareResources"`
	Influence     int64          `json:"influence"`
	UpdatedAt     time.Time
}

// GuildTreasuryLog - Historique des mouvements du coffre
type GuildTreasuryLog struct {
	Id          uint `gorm:"primaryKey" json:"id"`
	GuildID     uint           `gorm:"index" json:"guildId"`
	PlayerID    *uint          `gorm:"index" json:"playerId"`
	Action      string         `gorm:"size:32;index" json:"action"` // donate, spend, reward, penalty, trade_income, war_reward, quest_reward
	ResourceType string        `gorm:"size:32" json:"resourceType"`
	Amount      int64          `json:"amount"`
	BeforeValue int64          `json:"beforeValue"`
	AfterValue  int64          `json:"afterValue"`
	Description string         `gorm:"type:text" json:"description"`
	CreatedAt   time.Time
}

// GuildXPLog - Historique des gains d'XP de guilde (spec point 5)
type GuildXPLog struct {
	Id          uint `gorm:"primaryKey" json:"id"`
	GuildID     uint  `gorm:"index" json:"guildId"`
	PlayerID    *uint `gorm:"index" json:"playerId"`
	SourceType  string `gorm:"size:32;index" json:"sourceType"` // donation, help, quest, war, trade, diplomacy, research, ai_event, world_event, daily_activity
	SourceID    *uint  `gorm:"index" json:"sourceId"`
	Amount      int64  `json:"amount"`
	Description string `gorm:"type:text" json:"description"`
	CreatedAt   time.Time
}

// GuildHelpRequest - Demandes d'aide entre membres (spec point 11)
type GuildHelpRequest struct {
	Id            uint `gorm:"primaryKey" json:"id"`
	GuildID       uint           `gorm:"index" json:"guildId"`
	RequesterID   uint           `gorm:"index" json:"requesterId"`
	TargetType    string         `gorm:"size:32" json:"targetType"` // construction, research, resource, military
	TargetID      *uint          `gorm:"index" json:"targetId"`
	Title         string         `gorm:"size:200" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	HelpType      string         `gorm:"size:32;index" json:"helpType"` // construction_speedup, research_speedup, resource_support, military_support
	MaxAssists    int            `json:"maxAssists"`
	CurrentAssists int           `json:"currentAssists"`
	EffectPerAssist string        `gorm:"type:text" json:"effectPerAssist"`
	MaxEffect     string         `gorm:"type:text" json:"maxEffect"`
	Status        string         `gorm:"size:32;index" json:"status"` // active, completed, expired, cancelled
	ExpiresAt     *time.Time     `gorm:"index" json:"expiresAt"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// GuildQuest - Quêtes collectives de guilde (spec point 12)
type GuildQuest struct {
	Id            uint   `gorm:"primaryKey" json:"id"`
	GuildID       uint   `gorm:"index" json:"guildId"`
	Guild         Guild  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Title         string `gorm:"size:200" json:"title"`
	Description   string `gorm:"type:text" json:"description"`
	Status        string `gorm:"size:32;index" json:"status"` // active, completed, failed, cancelled
	Progress      int    `json:"progress"`
	Target        int    `json:"target"`
	RewardCredits int64  `json:"rewardCredits"`
	RewardFood    int64  `json:"rewardFood"`
	RewardEnergy  int64  `json:"rewardEnergy"`
	RewardMaterials int64 `json:"rewardMaterials"`
	RewardXP      int64  `json:"rewardXP"`
	StartedByID   *uint  `gorm:"index" json:"startedById"`
	StartedBy     *Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	CompletedAt   *time.Time `json:"completedAt"`
	ExpiresAt     *time.Time `gorm:"index" json:"expiresAt"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// GuildWar - Guerres entre guildes (spec point 13)
type GuildWar struct {
	Id                uint   `gorm:"primaryKey" json:"id"`
	AttackerGuildID   uint   `gorm:"index" json:"attackerGuildId"`
	AttackerGuild     Guild  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	DefenderGuildID   uint   `gorm:"index" json:"defenderGuildId"`
	DefenderGuild     Guild  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Status            string `gorm:"size:32;index" json:"status"` // preparing, active, ended, cancelled
	Cause             string `gorm:"type:text" json:"cause"`
	StartAt           *time.Time `json:"startAt"`
	EndAt             *time.Time `json:"endAt"`
	ScoreAttacker     int64  `json:"scoreAttacker"`
	ScoreDefender     int64  `json:"scoreDefender"`
	WinnerGuildID     *uint  `gorm:"index" json:"winnerGuildId"`
	WinnerGuild       *Guild `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	AttackerPowerUsed int64  `json:"attackerPowerUsed"`
	DefenderPowerUsed int64  `json:"defenderPowerUsed"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// GuildWarContribution - Contributions des membres pendant une guerre
type GuildWarContribution struct {
	Id         uint   `gorm:"primaryKey" json:"id"`
	WarID      uint   `gorm:"index" json:"warId"`
	GuildWar   GuildWar `gorm:"foreignKey:WarID;references:Id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PlayerID   uint   `gorm:"index" json:"playerId"`
	Player     Users  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ContributionType string `gorm:"size:32" json:"contributionType"` // units_sent, resources, kills, defense
	Amount     int64  `json:"amount"`
	PowerDelta int64  `json:"powerDelta"`
	CreatedAt  time.Time
}

// GuildResearch - Recherches collectives de guilde (spec point 14)
type GuildResearch struct {
	Id            uint   `gorm:"primaryKey" json:"id"`
	GuildID       uint   `gorm:"index" json:"guildId"`
	Guild         Guild  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	TechKey       string `gorm:"size:120;index" json:"techKey"`
	Level         int    `json:"level"`
	Progress      int    `json:"progress"`
	Target        int    `json:"target"`
	Status        string `gorm:"size:32;index" json:"status"` // active, completed, paused
	CostCredits   int64  `json:"costCredits"`
	CostEnergy    int64  `json:"costEnergy"`
	CostMaterials int64  `json:"costMaterials"`
	EffectJSON    datatypes.JSON `gorm:"type:json" json:"effectJson"`
	StartedByID   *uint  `gorm:"index" json:"startedById"`
	CompletedAt   *time.Time `json:"completedAt"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// GuildDiplomacy - Traités et relations entre guildes
type GuildDiplomacy struct {
	Id              uint   `gorm:"primaryKey" json:"id"`
	GuildAID        uint   `gorm:"index" json:"guildAId"`
	GuildA          Guild  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	GuildBID        uint   `gorm:"index" json:"guildBId"`
	GuildB          Guild  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Type            string `gorm:"size:32;index" json:"type"` // non_aggression, alliance, trade_pact, vassal
	Status          string `gorm:"size:32;index" json:"status"` // proposed, active, broken, expired
	TermsJSON       datatypes.JSON `gorm:"type:json" json:"termsJson"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
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

// DailyTask - Feature: Tâches quotidiennes générées par l'IA méchante (20-40 par jour)
// Récompenses: ressources, XP, ou gems (petite quantité)
type DailyTask struct {
	Id              uint           `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PlayerID        uint           `gorm:"index" json:"playerId"`
	WorldID         uint           `gorm:"index" json:"worldId"`
	Title           string         `gorm:"size:200" json:"title"`
	Description     string         `gorm:"type:text" json:"description"`
	TaskType        string         `gorm:"size:32;index" json:"taskType"` // resource, xp, gem, military, etc.
	TargetValue     int            `json:"targetValue"`
	CurrentValue    int            `json:"currentValue"`
	RewardType      string         `gorm:"size:32" json:"rewardType"` // credits, food, energy, xp, gems
	RewardAmount    int64          `json:"rewardAmount"`
	DurationMinutes int            `json:"durationMinutes"` // temps de réalisation
	Progress        float64        `json:"progress"` // 0.0 - 1.0
	Status          string         `gorm:"size:32;index" json:"status"` // available, in_progress, completed, claimed, expired
	ExpiresAt       *time.Time     `gorm:"index" json:"expiresAt"`
	StartedAt       *time.Time     `json:"startedAt"`
	CompletedAt     *time.Time     `json:"completedAt"`
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
	IsActive           bool           `gorm:"index;default:true" json:"isActive"`
	Error              string         `gorm:"type:text" json:"error"`
}

type WorldRoutineSnapshot struct {
	Id             uint `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time
	WorldID        uint           `gorm:"index" json:"worldId"`
	World          World          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	RoutineVersion string         `gorm:"size:64;index" json:"routineVersion"`
	SnapshotJSON   datatypes.JSON `gorm:"type:json" json:"snapshotJson"`
	AIOutputJSON   datatypes.JSON `gorm:"type:json" json:"aiOutputJson"`
	MetricsJSON    datatypes.JSON `gorm:"type:json" json:"metricsJson"`
	Provider       string         `gorm:"size:64;index" json:"provider"`
	Model          string         `gorm:"size:160;index" json:"model"`
	Status         string         `gorm:"size:32;index" json:"status"`
	Error          string         `gorm:"type:text" json:"error"`
}

type PlayerWorldMetric struct {
	Id             uint `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	PlayerID       uint           `gorm:"uniqueIndex:idx_player_world_metric;index" json:"playerId"`
	WorldID        uint           `gorm:"uniqueIndex:idx_player_world_metric;index" json:"worldId"`
	ContinentID    uint           `gorm:"index" json:"continentId"`
	Popularity     int            `gorm:"index" json:"popularity"`
	Stability      int            `gorm:"index" json:"stability"`
	Sustainability int            `gorm:"index" json:"sustainability"`
	ConflictScore  int            `json:"conflictScore"`
	DiplomacyScore int            `json:"diplomacyScore"`
	CommerceScore  int            `json:"commerceScore"`
	WeatherScore   int            `json:"weatherScore"`
	InputJSON      datatypes.JSON `gorm:"type:json" json:"inputJson"`
	GeneratedAt    time.Time      `gorm:"index" json:"generatedAt"`
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
	ResearchTreeKey        string          `gorm:"size:120;index" json:"researchTreeKey"`
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
