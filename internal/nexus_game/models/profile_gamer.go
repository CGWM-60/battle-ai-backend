package models

import "time"

// ProfileGamer is the player profile for Nexus Games MMO.
// Linked to the main app user table via UserID (the player's main id).
// Can be empty (no row or zero IDs) -> triggers creation flow in Flutter.
// Server is the source of truth. Flutter only displays and proposes.
type ProfileGamer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	AvatarID      uint      `json:"avatar_id" gorm:"index"`
	FactionID     uint      `json:"faction_id" gorm:"index"`
	IACompanionID uint      `json:"ia_companion_id" gorm:"index"`
	Pseudo        string    `json:"pseudo" gorm:"size:100"`
	CityName      string    `json:"city_name" gorm:"size:100"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
