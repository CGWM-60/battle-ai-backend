package models

import "time"

// Prompt represents a versioned prompt for Server AI (IA DU SERVEUR).
// Stored in DB, modifiable via world management endpoints (CRUD).
// Optimized for cost (short inputs), speed (structured outputs), constructive/detailed/enriching.
// Evolves automatically based on world state, day, universe (versioned, can have dynamic params).
type Prompt struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PromptID      string    `json:"prompt_id" gorm:"uniqueIndex;not null"` // e.g. "quest_seed_generation"
	Version       string    `json:"version" gorm:"size:50;not null"`       // e.g. "v1.2-optimized-2026"
	Domain        string    `json:"domain" gorm:"size:100"`                // world_tick_summary, quest_seed, etc.
	Purpose       string    `json:"purpose" gorm:"type:text"`
	SystemPrompt  string    `json:"system_prompt" gorm:"type:text"` // the actual optimized prompt text
	InputSchema   string    `json:"input_schema" gorm:"type:text"`  // JSON schema description
	OutputSchema  string    `json:"output_schema" gorm:"type:text"`
	SafetyRules   string    `json:"safety_rules" gorm:"type:text"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
