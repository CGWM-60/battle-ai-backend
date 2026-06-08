package models

import "time"

// IACompanion represents a player's AI companion/agent in Nexus MMO.
// Similar to avatars, these are assigned to roles (gouverneur, stratège, etc.).
type IACompanion struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PlayerID  uint      `json:"player_id" gorm:"index"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Role      string    `json:"role" gorm:"size:50"` // e.g. "Gouverneur", "Stratège", "Commandant"
	Level     int       `json:"level"`
	Filename  string    `json:"filename" gorm:"size:255"`
	URL       string    `json:"url" gorm:"size:500"`
	CreatedAt time.Time `json:"created_at"`
}