package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ResourceDefinition struct {
	Id          uint `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Key         string         `gorm:"size:120;uniqueIndex" json:"key"`
	Name        string         `gorm:"size:160;index" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Category    string         `gorm:"size:64;index" json:"category"`
	IsActive    bool           `gorm:"index" json:"isActive"`
	SortOrder   int            `gorm:"index" json:"sortOrder"`
}

type ResearchTreeDefinition struct {
	Id          uint `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt           `gorm:"index" json:"-"`
	Key         string                   `gorm:"size:120;uniqueIndex" json:"key"`
	Name        string                   `gorm:"size:160;index" json:"name"`
	Description string                   `gorm:"type:text" json:"description"`
	Domain      string                   `gorm:"size:120;index" json:"domain"`
	BuildingKey string                   `gorm:"size:120;index" json:"buildingKey"`
	IsActive    bool                     `gorm:"index" json:"isActive"`
	SortOrder   int                      `gorm:"index" json:"sortOrder"`
	Nodes       []ResearchNodeDefinition `gorm:"foreignKey:ResearchTreeDefinitionID" json:"nodes,omitempty"`
}

type ResearchNodeDefinition struct {
	Id                       uint `gorm:"primaryKey" json:"id"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                gorm.DeletedAt         `gorm:"index" json:"-"`
	ResearchTreeDefinitionID uint                   `gorm:"index" json:"researchTreeDefinitionId"`
	ResearchTreeDefinition   ResearchTreeDefinition `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Key                      string                 `gorm:"size:120;uniqueIndex" json:"key"`
	Name                     string                 `gorm:"size:160;index" json:"name"`
	Description              string                 `gorm:"type:text" json:"description"`
	Domain                   string                 `gorm:"size:120;index" json:"domain"`
	Branch                   string                 `gorm:"size:120;index" json:"branch"`
	ResourcesJSON            datatypes.JSON         `gorm:"type:json" json:"resourcesJson"`
	ParentKeysJSON           datatypes.JSON         `gorm:"type:json" json:"parentKeysJson"`
	RequirementsJSON         datatypes.JSON         `gorm:"type:json" json:"requirementsJson"`
	EffectsJSON              datatypes.JSON         `gorm:"type:json" json:"effectsJson"`
	LevelProgressionJSON     datatypes.JSON         `gorm:"type:json" json:"levelProgressionJson"`
	MaxLevel                 int                    `json:"maxLevel"`
	PositionX                int                    `json:"positionX"`
	PositionY                int                    `json:"positionY"`
	IsActive                 bool                   `gorm:"index" json:"isActive"`
	SortOrder                int                    `gorm:"index" json:"sortOrder"`
}
