package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ArmyUnit struct {
	Id                       uint `gorm:"primaryKey" json:"id"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`
	PlayerID                 uint           `gorm:"index" json:"playerId"`
	UnitType                 string         `gorm:"size:64;index" json:"unitType"`
	Name                     string         `gorm:"size:120" json:"name"`
	Level                    int            `gorm:"index" json:"level"`
	Experience               int64          `json:"experience"`
	Health                   int            `json:"health"`
	Attack                   int            `json:"attack"`
	Defense                  int            `json:"defense"`
	Speed                    int            `json:"speed"`
	Range                    int            `json:"range"`
	Morale                   int            `json:"morale"`
	Power                    int            `json:"power"` // Feature: Puissance - impacts world conflicts interventions and city power display
	Endurance                int            `json:"endurance"`
	MaxCarry                 int            `json:"maxCarry"`
	FoodConsumptionPerHour   int64          `json:"foodConsumptionPerHour"`
	EnergyConsumptionPerHour int64          `json:"energyConsumptionPerHour"`
	CreditMaintenancePerHour int64          `json:"creditMaintenancePerHour"`
	TrainingCost             datatypes.JSON `gorm:"type:json" json:"trainingCost"`
	TrainingDurationSeconds  int64          `json:"trainingDurationSeconds"`
	Status                   string         `gorm:"size:32;index" json:"status"`
	AssignedConflictID       *uint          `gorm:"index" json:"assignedConflictId"`
}

type ArmyTrainingQueue struct {
	Id               uint `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	PlayerID         uint           `gorm:"index" json:"playerId"`
	UnitType         string         `gorm:"size:64;index" json:"unitType"`
	Quantity         int            `json:"quantity"`
	Status           string         `gorm:"size:32;index" json:"status"`
	StartedAt        time.Time      `gorm:"index" json:"startedAt"`
	FinishesAt       time.Time      `gorm:"index" json:"finishesAt"`
	CostJSON         datatypes.JSON `gorm:"type:json" json:"cost"`
	BarracksLevelReq int            `json:"barracksLevelReq"`
}
