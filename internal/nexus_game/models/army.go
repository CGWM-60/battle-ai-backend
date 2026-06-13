package models

import (
	"time"

	"gorm.io/datatypes"
)

type UnitTrainingQueue struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	ProfileGamerID   uint           `gorm:"index;not null" json:"profileGamerId"`
	WorldID          uint           `gorm:"index" json:"worldId"`
	BuildingID       uint           `gorm:"index" json:"buildingId"`
	UnitCode         string         `gorm:"size:128;index;not null" json:"unitCode"`
	Quantity         int            `json:"quantity"`
	Status           string         `gorm:"size:32;index;not null" json:"status"`
	StartedAt        time.Time      `json:"startedAt"`
	CompletedAt      time.Time      `gorm:"index" json:"completedAt"`
	ClaimedAt        *time.Time     `json:"claimedAt,omitempty"`
	CancelledAt      *time.Time     `json:"cancelledAt,omitempty"`
	CostJSON         datatypes.JSON `gorm:"type:json" json:"costJson"`
	RefundJSON       datatypes.JSON `gorm:"type:json" json:"refundJson"`
	ReservedCapacity int            `json:"reservedCapacity"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type ArmyFormation struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ProfileGamerID    uint           `gorm:"uniqueIndex:idx_army_formation_profile_type;not null" json:"profileGamerId"`
	WorldID           uint           `gorm:"index" json:"worldId"`
	Type              string         `gorm:"size:48;uniqueIndex:idx_army_formation_profile_type;not null" json:"type"`
	Name              string         `gorm:"size:120" json:"name"`
	Status            string         `gorm:"size:32;index;default:active" json:"status"`
	TotalPower        int            `json:"totalPower"`
	AttackPower       int            `json:"attackPower"`
	DefensePower      int            `json:"defensePower"`
	ScoutingPower     int            `json:"scoutingPower"`
	AntiSabotagePower int            `json:"antiSabotagePower"`
	SupportPower      int            `json:"supportPower"`
	UpkeepJSON        datatypes.JSON `gorm:"type:json" json:"upkeepJson"`
	IsAutoManaged     bool           `json:"isAutoManaged"`
	Doctrine          string         `gorm:"size:80" json:"doctrine"`
	LastCalculatedAt  *time.Time     `json:"lastCalculatedAt,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type ArmyFormationSlot struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	FormationID           uint           `gorm:"index;not null" json:"formationId"`
	SlotIndex             int            `json:"slotIndex"`
	SlotType              string         `gorm:"size:48;index;not null" json:"slotType"`
	Capacity              int            `json:"capacity"`
	AllowedUnitTypesJSON  datatypes.JSON `gorm:"type:json" json:"allowedUnitTypesJson"`
	IsLocked              bool           `json:"isLocked"`
	UnlockRequirementJSON datatypes.JSON `gorm:"type:json" json:"unlockRequirementJson"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
}

type ArmySlotAssignment struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	FormationID  uint      `gorm:"index;not null" json:"formationId"`
	SlotID       uint      `gorm:"index;not null" json:"slotId"`
	UnitCode     string    `gorm:"size:128;index;not null" json:"unitCode"`
	Quantity     int       `json:"quantity"`
	CapacityUsed int       `json:"capacityUsed"`
	Power        int       `json:"power"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ArmyAutomationSettings struct {
	ID                        uint      `gorm:"primaryKey" json:"id"`
	ProfileGamerID            uint      `gorm:"uniqueIndex;not null" json:"profileGamerId"`
	WorldID                   uint      `gorm:"index" json:"worldId"`
	AutoDefenseEnabled        bool      `gorm:"default:true" json:"autoDefenseEnabled"`
	AutoRepairEnabled         bool      `json:"autoRepairEnabled"`
	AutoHealEnabled           bool      `json:"autoHealEnabled"`
	AutoTrainEnabled          bool      `json:"autoTrainEnabled"`
	AutoComposeDefenseEnabled bool      `json:"autoComposeDefenseEnabled"`
	MaxAutoSpendPercent       int       `gorm:"default:20" json:"maxAutoSpendPercent"`
	MinFoodKeep               int64     `gorm:"default:200" json:"minFoodKeep"`
	MinEnergyKeep             int64     `gorm:"default:150" json:"minEnergyKeep"`
	MinDefensePower           int       `gorm:"default:300" json:"minDefensePower"`
	MaxUnitsOnMissionPercent  int       `gorm:"default:40" json:"maxUnitsOnMissionPercent"`
	AllowRareResourceSpend    bool      `json:"allowRareResourceSpend"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

type ArmyCombatReport struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	WorldID        uint           `gorm:"index" json:"worldId"`
	AttackerUserID uint           `gorm:"index" json:"attackerUserId"`
	DefenderUserID uint           `gorm:"index" json:"defenderUserId"`
	AttackerType   string         `gorm:"size:48" json:"attackerType"`
	DefenderType   string         `gorm:"size:48" json:"defenderType"`
	FormationID    uint           `gorm:"index" json:"formationId"`
	Result         string         `gorm:"size:48;index" json:"result"`
	AttackPower    int            `json:"attackPower"`
	DefensePower   int            `json:"defensePower"`
	LossesJSON     datatypes.JSON `gorm:"type:json" json:"lossesJson"`
	RewardsJSON    datatypes.JSON `gorm:"type:json" json:"rewardsJson"`
	ModifiersJSON  datatypes.JSON `gorm:"type:json" json:"modifiersJson"`
	Summary        string         `gorm:"type:text" json:"summary"`
	CreatedAt      time.Time      `gorm:"index" json:"createdAt"`
}

type ArmyTransactionLog struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ProfileGamerID uint           `gorm:"index;not null" json:"profileGamerId"`
	WorldID        uint           `gorm:"index" json:"worldId"`
	ActionType     string         `gorm:"size:64;index;not null" json:"actionType"`
	UnitCode       string         `gorm:"size:128;index" json:"unitCode"`
	Quantity       int            `json:"quantity"`
	BeforeJSON     datatypes.JSON `gorm:"type:json" json:"beforeJson"`
	AfterJSON      datatypes.JSON `gorm:"type:json" json:"afterJson"`
	Reason         string         `gorm:"size:255" json:"reason"`
	LinkedType     string         `gorm:"size:64;index" json:"linkedType"`
	LinkedID       uint           `gorm:"index" json:"linkedId"`
	CreatedAt      time.Time      `gorm:"index" json:"createdAt"`
}
