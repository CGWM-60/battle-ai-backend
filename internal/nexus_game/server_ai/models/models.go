package models

import "time"

type ServerAICity struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	WorldID         uint       `gorm:"uniqueIndex:idx_server_ai_city_continent" json:"worldId"`
	ContinentID     uint       `gorm:"uniqueIndex:idx_server_ai_city_continent" json:"continentId"`
	RegionID        uint       `gorm:"index" json:"regionId"`
	Name            string     `gorm:"size:120;not null" json:"name"`
	Level           int        `gorm:"default:1" json:"level"`
	Power           int        `gorm:"default:100" json:"power"`
	Population      int        `gorm:"default:0" json:"population"`
	PopulationCap   int        `gorm:"default:0" json:"populationCap"`
	Morale          int        `gorm:"default:50" json:"morale"`
	Energy          int        `gorm:"default:0" json:"energy"`
	Security        int        `gorm:"default:50" json:"security"`
	ResourcesJSON   string     `gorm:"type:text" json:"resourcesJson"`
	BuildingsJSON   string     `gorm:"type:text" json:"buildingsJson"`
	UnitsJSON       string     `gorm:"type:text" json:"unitsJson"`
	ResearchJSON    string     `gorm:"type:text" json:"researchJson"`
	StrategyJSON    string     `gorm:"type:text" json:"strategyJson"`
	Status          string     `gorm:"size:32;index;default:active" json:"status"`
	ProductionBonus float64    `gorm:"default:0.05" json:"productionBonus"`
	TrainingBonus   float64    `gorm:"default:0.02" json:"trainingBonus"`
	ResearchBonus   float64    `gorm:"default:0.02" json:"researchBonus"`
	LastTickAt      *time.Time `json:"lastTickAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type ServerAIStrategy struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	WorldID        uint      `gorm:"index" json:"worldId"`
	ContinentID    uint      `gorm:"index" json:"continentId"`
	ServerAICityID uint      `gorm:"index" json:"serverAiCityId"`
	Priority       string    `gorm:"size:64" json:"priority"`
	Difficulty     string    `gorm:"size:32;default:normal" json:"difficulty"`
	StrategyJSON   string    `gorm:"type:text" json:"strategyJson"`
	IsActive       bool      `gorm:"default:true" json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ServerAIMemory struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	WorldID             uint      `gorm:"uniqueIndex" json:"worldId"`
	AIVictories         int       `json:"aiVictories"`
	PlayerVictories     int       `json:"playerVictories"`
	Draws               int       `json:"draws"`
	GlobalDanger        int       `gorm:"default:30" json:"globalDanger"`
	GlobalStability     int       `gorm:"default:60" json:"globalStability"`
	BastionsJSON        string    `gorm:"type:text" json:"bastionsJson"`
	ContinentsJSON      string    `gorm:"type:text" json:"continentsJson"`
	TrendsJSON          string    `gorm:"type:text" json:"trendsJson"`
	EffectiveStrategies string    `gorm:"type:text" json:"effectiveStrategies"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type ServerAIPlayerMemory struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	WorldID             uint       `gorm:"index" json:"worldId"`
	UserID              uint       `gorm:"index" json:"userId"`
	ContinentID         uint       `gorm:"index" json:"continentId"`
	PlayerPower         int        `json:"playerPower"`
	PlayerLevel         int        `json:"playerLevel"`
	PlayerStyle         string     `gorm:"size:32;default:unknown" json:"playerStyle"`
	ThreatScore         int        `gorm:"index" json:"threatScore"`
	RivalryLevel        int        `json:"rivalryLevel"`
	VictoriesAgainstAI  int        `json:"victoriesAgainstAi"`
	DefeatsAgainstAI    int        `json:"defeatsAgainstAi"`
	DrawsAgainstAI      int        `json:"drawsAgainstAi"`
	KnownWeaknessesJSON string     `gorm:"type:text" json:"knownWeaknessesJson"`
	KnownStrengthsJSON  string     `gorm:"type:text" json:"knownStrengthsJson"`
	LastMessageJSON     string     `gorm:"type:text" json:"lastMessageJson"`
	LastAttackAt        *time.Time `json:"lastAttackAt"`
	LastEspionageAt     *time.Time `json:"lastEspionageAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type ServerAIAttack struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	WorldID        uint       `gorm:"index" json:"worldId"`
	ContinentID    uint       `gorm:"index" json:"continentId"`
	ServerAICityID uint       `gorm:"index" json:"serverAiCityId"`
	TargetUserID   uint       `gorm:"index" json:"targetUserId"`
	TargetCityID   uint       `gorm:"index" json:"targetCityId"`
	Status         string     `gorm:"size:32;index;default:scheduled" json:"status"`
	AttackType     string     `gorm:"size:32;index" json:"attackType"`
	AttackPower    int        `json:"attackPower"`
	DefensePower   int        `json:"defensePower"`
	ScalingFactor  float64    `json:"scalingFactor"`
	Result         string     `gorm:"size:32;index" json:"result"`
	RewardsJSON    string     `gorm:"type:text" json:"rewardsJson"`
	LossesJSON     string     `gorm:"type:text" json:"lossesJson"`
	ReportJSON     string     `gorm:"type:text" json:"reportJson"`
	ScheduledAt    time.Time  `gorm:"index" json:"scheduledAt"`
	WarningAt      *time.Time `json:"warningAt"`
	ResolvedAt     *time.Time `json:"resolvedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type ServerAISabotage struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	WorldID      uint       `gorm:"index" json:"worldId"`
	ContinentID  uint       `gorm:"index" json:"continentId"`
	TargetUserID uint       `gorm:"index" json:"targetUserId"`
	Status       string     `gorm:"size:32;index;default:proposed" json:"status"`
	SabotageType string     `gorm:"size:64" json:"sabotageType"`
	MaxImpact    string     `gorm:"size:120" json:"maxImpact"`
	Counterplay  string     `gorm:"type:text" json:"counterplay"`
	ReportJSON   string     `gorm:"type:text" json:"reportJson"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CancelledAt  *time.Time `json:"cancelledAt"`
}

type ServerAIEspionage struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	WorldID      uint       `gorm:"index" json:"worldId"`
	ContinentID  uint       `gorm:"index" json:"continentId"`
	TargetUserID uint       `gorm:"index" json:"targetUserId"`
	Result       string     `gorm:"size:32;index" json:"result"`
	Summary      string     `gorm:"type:text" json:"summary"`
	FindingsJSON string     `gorm:"type:text" json:"findingsJson"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `gorm:"index" json:"deletedAt"`
}

type ServerAIPressure struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorldID     uint      `gorm:"uniqueIndex:idx_ai_pressure_continent" json:"worldId"`
	ContinentID uint      `gorm:"uniqueIndex:idx_ai_pressure_continent" json:"continentId"`
	Level       int       `gorm:"default:20" json:"level"`
	Status      string    `gorm:"size:32;default:calme" json:"status"`
	ReasonsJSON string    `gorm:"type:text" json:"reasonsJson"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ServerAIBroadcast struct {
	ID                     uint       `gorm:"primaryKey" json:"id"`
	WorldID                uint       `gorm:"index" json:"worldId"`
	Date                   string     `gorm:"size:10;index" json:"date"`
	Title                  string     `gorm:"size:180" json:"title"`
	ThreatLevel            string     `gorm:"size:32;index" json:"threatLevel"`
	Message                string     `gorm:"type:text" json:"message"`
	PastSummaryJSON        string     `gorm:"type:text" json:"pastSummaryJson"`
	UpcomingThreatsJSON    string     `gorm:"type:text" json:"upcomingThreatsJson"`
	SeasonalEventsJSON     string     `gorm:"type:text" json:"seasonalEventsJson"`
	RecommendedPreparation string     `gorm:"type:text" json:"recommendedPreparation"`
	Status                 string     `gorm:"size:32;index;default:draft" json:"status"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
	PublishedAt            *time.Time `json:"publishedAt"`
}

type ServerAISeasonalEvent struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	WorldID          uint       `gorm:"index" json:"worldId"`
	Title            string     `gorm:"size:180" json:"title"`
	Summary          string     `gorm:"type:text" json:"summary"`
	EventType        string     `gorm:"size:64;index" json:"eventType"`
	Status           string     `gorm:"size:32;index;default:proposed" json:"status"`
	ProposedBy       string     `gorm:"size:32;default:server_ai" json:"proposedBy"`
	ProposalJSON     string     `gorm:"type:text" json:"proposalJson"`
	RulesJSON        string     `gorm:"type:text" json:"rulesJson"`
	RewardsJSON      string     `gorm:"type:text" json:"rewardsJson"`
	RisksJSON        string     `gorm:"type:text" json:"risksJson"`
	AffectedJSON     string     `gorm:"type:text" json:"affectedJson"`
	AdminNote        string     `gorm:"type:text" json:"adminNote"`
	RejectionReason  string     `gorm:"type:text" json:"rejectionReason"`
	StartsAt         *time.Time `json:"startsAt"`
	EndsAt           *time.Time `json:"endsAt"`
	ApprovedByUserID *uint      `json:"approvedByUserId"`
	ApprovedAt       *time.Time `json:"approvedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type ServerAICallLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RequestID     string    `gorm:"size:80;index" json:"requestId"`
	Feature       string    `gorm:"size:100;index" json:"feature"`
	Provider      string    `gorm:"size:64;index" json:"provider"`
	Model         string    `gorm:"size:120" json:"model"`
	PromptKey     string    `gorm:"size:128;index" json:"promptKey"`
	PromptVersion int       `json:"promptVersion"`
	InputHash     string    `gorm:"size:128;index" json:"inputHash"`
	InputSummary  string    `gorm:"type:text" json:"inputSummary"`
	OutputSummary string    `gorm:"type:text" json:"outputSummary"`
	TokensIn      int       `json:"tokensIn"`
	TokensOut     int       `json:"tokensOut"`
	CostEstimate  float64   `json:"costEstimate"`
	LatencyMs     int64     `json:"latencyMs"`
	Status        string    `gorm:"size:32;index" json:"status"`
	ErrorMessage  string    `gorm:"type:text" json:"errorMessage"`
	LinkedType    string    `gorm:"size:64;index" json:"linkedType"`
	LinkedID      uint      `gorm:"index" json:"linkedId"`
	CreatedAt     time.Time `gorm:"index" json:"createdAt"`
}

type ServerAIAdminAction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"userId"`
	Action      string    `gorm:"size:80;index" json:"action"`
	TargetType  string    `gorm:"size:80;index" json:"targetType"`
	TargetID    uint      `gorm:"index" json:"targetId"`
	PayloadJSON string    `gorm:"type:text" json:"payloadJson"`
	CreatedAt   time.Time `json:"createdAt"`
}
