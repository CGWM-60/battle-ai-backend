package models

import "time"

// World represents a persistent game world with 5 continents.
// New worlds are created when all continents in existing worlds are full (player or faction capacity).
type World struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
}

// Continent is one of the 5 fixed continents in a world.
// Max 500 players per continent, max 3 factions per continent.
// Player counts and faction assignments are managed primarily in Redis for speed and locking.
type Continent struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	WorldID       uint   `json:"world_id" gorm:"index"`
	Name          string `json:"name" gorm:"size:100;not null"` // e.g. "Amérique du Nord"
	MaxPlayers    int    `json:"max_players" gorm:"default:500"`
	MaxFactions   int    `json:"max_factions" gorm:"default:3"`
	// CurrentPlayers and AssignedFactions managed in Redis (e.g. HINCRBY, sets)
	// to allow fast proportional distribution and locking.
}

// ContinentNames are the 5 canonical continents.
var ContinentNames = []string{
	"Amérique du Nord",
	"Amérique du Sud",
	"Europe",
	"Afrique",
	"Eurasia",
}
