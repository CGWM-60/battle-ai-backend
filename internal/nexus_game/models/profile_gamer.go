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
	// Continent assignment for world capacity and faction-based distribution.
	// Assigned at profile creation based on chosen Faction's continent.
	ContinentID uint      `json:"continent_id" gorm:"index"`
	WorldID     uint      `json:"world_id" gorm:"index"`

	// City power and level for the player (Nexus entry HUD).
	// Power = sum of building powers + research + army units.
	// Starts at 0 / Level 1 on profile creation.
	Level int `json:"level" gorm:"default:1"`
	Power int `json:"power" gorm:"default:0"`

	// Evolutionary base city stats (Population, Morale, Energy, Security).
	// These evolve on ticks, actions, events. Server is source of truth.
	// See detailed rules in AGENTS / dev notes for growth, decline, factors, priorities, relations.
	Population         int `json:"population" gorm:"default:0"`
	PopulationCapacity int `json:"population_capacity" gorm:"default:0"`
	Morale             int `json:"morale" gorm:"default:50"`
	EnergyProduction   int `json:"energy_production" gorm:"default:0"`
	EnergyConsumption  int `json:"energy_consumption" gorm:"default:0"`
	EnergyBalance      int `json:"energy_balance" gorm:"default:0"`
	EnergyStored       int `json:"energy_stored" gorm:"default:0"`
	Security           int `json:"security" gorm:"default:50"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
