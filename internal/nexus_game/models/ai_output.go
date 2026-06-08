package models

import "time"

// AIOutput stores server AI generations for history (persisted in DB + Redis).
// Used for admin visualization of what the server IA actually generated (events, lore, seeds, etc.).
type AIOutput struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Feature       string    `json:"feature" gorm:"size:100;not null"` // e.g. "event_generation", "quest_seed_generation"
	WorldID       uint      `json:"world_id" gorm:"index"`
	LinkedType    string    `json:"linked_type" gorm:"size:50"`
	LinkedID      uint      `json:"linked_id"`
	Output        string    `json:"output" gorm:"type:text"` // JSON of the generated content
	PromptVersion string    `json:"prompt_version" gorm:"size:50"`
	TokensIn      int       `json:"tokens_in"`
	TokensOut     int       `json:"tokens_out"`
	LatencyMs     int64     `json:"latency_ms"`
	Status        string    `json:"status" gorm:"size:20"`
	CreatedAt     time.Time `json:"created_at"`
}
