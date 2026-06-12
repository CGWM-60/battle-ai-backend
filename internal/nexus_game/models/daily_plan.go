package models

import "time"

// DailyPlanContext is the safe, official context sent to the player's AI provider (or fallback).
// It contains current city stats, resources, queues, available action types, and server rules.
// Player's AI (provider > local > governor agent > server fallback) generates recommendations.
// Server NEVER trusts recommendations blindly: all must go through /actions/validate then /actions/resolve.
type DailyPlanContext struct {
	ProfileGamerID   uint                   `json:"profile_gamer_id"`
	PlayerStyle      string                 `json:"player_style"` // e.g. "defensive", "aggressive", "balanced" (from profile or default)
	City             map[string]interface{} `json:"city"`         // population, capacity, morale, security, energyBalance, foodBalance etc.
	Resources        map[string]interface{} `json:"resources"`    // credits, metal, energy, food...
	ActiveQueues     map[string]interface{} `json:"active_queues"`
	AvailableActions []string               `json:"available_actions"`
	ServerRules      []string               `json:"server_rules"`
	GeneratedAt      time.Time              `json:"generated_at"`
}

// DailyPlanRecommendation is one item from player's AI.
type DailyPlanRecommendation struct {
	Priority                 int                    `json:"priority"`
	Type                     string                 `json:"type"`        // "build", "upgrade", "train_unit", "start_research", "collect"...
	TargetType               string                 `json:"target_type"` // "building", "unit", "research"...
	TargetID                 string                 `json:"target_id"`   // e.g. "solar_plant", "milicien_nexus"
	Title                    string                 `json:"title"`
	Reason                   string                 `json:"reason"`
	ExpectedImpact           map[string]interface{} `json:"expected_impact"` // e.g. {"energyBalance": "+25", "morale": "+2"}
	Risk                     string                 `json:"risk"`            // "low", "medium", "high"
	ServerValidationRequired bool                   `json:"server_validation_required"`
}

// DailyPlan is the full plan (context + AI recommendations).
// Stored for audit, acceptance tracking.
type DailyPlan struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	ProfileGamerID   uint       `json:"profile_gamer_id" gorm:"index"`
	Context          string     `json:"context" gorm:"type:text"`         // JSON of DailyPlanContext
	Recommendations  string     `json:"recommendations" gorm:"type:text"` // JSON array of DailyPlanRecommendation
	Summary          string     `json:"summary"`
	GlobalRisk       string     `json:"global_risk"`
	GeneratedBy      string     `json:"generated_by"` // "player_provider", "local_model", "governor_agent", "server_fallback", "algorithmic"
	GeneratedAt      time.Time  `json:"generated_at"`
	AcceptedActionID string     `json:"accepted_action_id"` // if player accepted one
	AppliedAt        *time.Time `json:"applied_at"`
	AppliedIndices   []int      `gorm:"serializer:json" json:"applied_indices"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
