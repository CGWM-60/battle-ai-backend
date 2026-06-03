package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

// TribunalGeneratedCase stores a pre-generated playable case template produced by AI cron.
// Complex arrays (witnesses, evidence, statements, contradictions) are stored as JSON.
type TribunalGeneratedCase struct {
	gorm.Model
	GenerationBatchID uint           `gorm:"index" json:"generationBatchId"`
	CaseID            *uint          `gorm:"index" json:"caseId"` // set after load
	Title             string         `gorm:"size:180;not null" json:"title"`
	Summary           string         `gorm:"type:text" json:"summary"`
	CaseType          string         `gorm:"size:80" json:"caseType"`
	Level             int            `gorm:"index" json:"level"` // 1..10
	Difficulty        string         `gorm:"size:40" json:"difficulty"`
	EstimatedDurationMinutes int     `json:"estimatedDurationMinutes"`
	Mode              string         `gorm:"size:40" json:"mode"`
	Tone              string         `gorm:"size:80" json:"tone"`
	PlayerRoleSuggestion string      `gorm:"size:40" json:"playerRoleSuggestion"`
	AccusationPosition string        `gorm:"type:text" json:"accusationPosition"`
	DefensePosition   string         `gorm:"type:text" json:"defensePosition"`
	TagsJSON          datatypes.JSON `gorm:"type:json" json:"tags"`
	WitnessesJSON     datatypes.JSON `gorm:"type:json" json:"witnesses"`
	EvidenceJSON      datatypes.JSON `gorm:"type:json" json:"evidence"`
	TestimonyJSON     datatypes.JSON `gorm:"type:json" json:"testimonyStatements"`
	ExpectedContradictionsJSON datatypes.JSON `gorm:"type:json" json:"expectedContradictions"`
	Status            string         `gorm:"size:40;index" json:"status"` // draft,ready,published,rejected,failed,archived
	IsPlayable        bool           `gorm:"index" json:"isPlayable"`
	IsPublished       bool           `gorm:"index" json:"isPublished"`
	GeneratedByCron   bool           `json:"generatedByCron"`
	ProviderType      string         `gorm:"size:80" json:"providerType"`
	ProviderModel     string         `gorm:"size:160" json:"model"`
	ErrorMessage      string         `gorm:"type:text" json:"errorMessage"`
	MetadataJSON      datatypes.JSON `gorm:"type:json" json:"metadata"`
}

// TribunalCaseGenerationBatch tracks a cron (or manual) run that produces N generated cases.
type TribunalCaseGenerationBatch struct {
	gorm.Model
	StartedAt      time.Time      `json:"startedAt"`
	FinishedAt     *time.Time     `json:"finishedAt"`
	Source         string         `gorm:"size:40" json:"source"` // cron, admin, manual
	TriggerType    string         `gorm:"size:40" json:"triggerType"`
	Status         string         `gorm:"size:40;index" json:"status"` // pending,running,success,partial_success,failed,cancelled
	RequestedCount int            `json:"requestedCount"`
	GeneratedCount int            `json:"generatedCount"`
	PublishedCount int            `json:"publishedCount"`
	FailedCount    int            `json:"failedCount"`
	ProviderType   string         `gorm:"size:80" json:"providerType"`
	ProviderModel  string         `gorm:"size:160" json:"model"`
	CronSchedule   string         `gorm:"size:40" json:"cronSchedule"`
	DurationMs     int64          `json:"durationMs"`
	ErrorMessage   string         `gorm:"type:text" json:"errorMessage"`
	LogsJSON       datatypes.JSON `gorm:"type:json" json:"logs"`
	MetadataJSON   datatypes.JSON `gorm:"type:json" json:"metadata"`
}
