package models

import "time"

// MmoIAAgent represents an AI agent or companion attached to a ProfileGamer in Nexus Games.
// Multiple agents allowed (for different roles: armée, ville, etc.).
// Only one companion IA allowed per profile (is_companion = true).
// Avatar possible for agents (linked to existing Avatar assets).
// The companion can be linked to the one chosen during initial profile creation (via ProfileGamer.IACompanionID).
type MmoIAAgent struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ProfileGamerID uint      `json:"profile_gamer_id" gorm:"index;not null"`
	Name           string    `json:"name" gorm:"size:100;not null"`
	Role           string    `json:"role" gorm:"size:50"` // e.g. "armee", "ville", "diplomatie", "espionnage", "commerce", "recherche", or "companion"
	Personality    string    `json:"personality" gorm:"type:text"`
	Provider       string    `json:"provider" gorm:"size:50"`
	Model          string    `json:"model" gorm:"size:100"`
	AvatarID       uint      `json:"avatar_id" gorm:"index"` // optional, for custom avatar on agent (0 = none)
	IsCompanion    bool      `json:"is_companion" gorm:"default:false;index"` // only one per profile allowed
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
