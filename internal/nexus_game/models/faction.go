package models

// Faction represents a global faction in the Nexus world.
// Players can have reputation with them.
type Faction struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `json:"name" gorm:"size:100;not null"`
	Description string `json:"description" gorm:"type:text"`
	Color       string `json:"color" gorm:"size:20"` // e.g. "#FF0000" for UI
	Filename    string `json:"filename" gorm:"size:255"`
	URL         string `json:"url" gorm:"size:500"`
}