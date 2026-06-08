package models

import "time"

// Avatar represents a player's avatar in Nexus MMO.
// Stored persistently via Docker volume to survive rebuilds.
// Image is always converted to WebP on upload.
type Avatar struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PlayerID  uint      `json:"player_id" gorm:"index"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Filename  string    `json:"filename" gorm:"size:255;not null"` // e.g. "550e8400-e29b-41d4-a716-446655440000.webp"
	URL       string    `json:"url" gorm:"size:500"`               // full accessible URL e.g. https://.../nexus_game/assets/avatar/xxx.webp
	CreatedAt time.Time `json:"created_at"`
}