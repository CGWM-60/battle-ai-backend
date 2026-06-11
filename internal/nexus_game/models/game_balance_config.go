package models

import "time"

const ActiveGameBalanceConfigKey = "active"

// GameBalanceConfig stores the live server-side formulas used by Nexus Game.
// Admin can edit the active row without redeploying the backend.
type GameBalanceConfig struct {
	ID  uint   `gorm:"primaryKey" json:"id"`
	Key string `gorm:"uniqueIndex;size:64;not null" json:"key"`

	PopulationHabitatBase        int     `json:"populationHabitatBase"`
	PopulationHabitatPerLevel    int     `json:"populationHabitatPerLevel"`
	FoodPerPopulationPerHour     float64 `json:"foodPerPopulationPerHour"`
	EnergyBaseConsumptionPerHour int     `json:"energyBaseConsumptionPerHour"`
	PopulationPerEnergy          int     `json:"populationPerEnergy"`

	BuildingEnergySurchargeExponent            float64 `json:"buildingEnergySurchargeExponent"`
	BuildingEnergyHighLevelThreshold           int     `json:"buildingEnergyHighLevelThreshold"`
	BuildingEnergyHighLevelMultiplier          float64 `json:"buildingEnergyHighLevelMultiplier"`
	IndustryEnergyPerLevel                     int     `json:"industryEnergyPerLevel"`
	DataEnergyPerLevel                         int     `json:"dataEnergyPerLevel"`
	SolarEnergyReliefPerLevel                  int     `json:"solarEnergyReliefPerLevel"`
	HabitatFarmCoverageDivisor                 int     `json:"habitatFarmCoverageDivisor"`
	HabitatFarmEnergyPenaltyDivisor            int     `json:"habitatFarmEnergyPenaltyDivisor"`
	EnergyRatioCriticalThreshold               float64 `json:"energyRatioCriticalThreshold"`
	EnergyRatioTensionThreshold                float64 `json:"energyRatioTensionThreshold"`
	EnergyRatioSurplusThreshold                float64 `json:"energyRatioSurplusThreshold"`
	EnergyRatioDominanceThreshold              float64 `json:"energyRatioDominanceThreshold"`
	ResourceMultiplierCritical                 float64 `json:"resourceMultiplierCritical"`
	ResourceMultiplierTension                  float64 `json:"resourceMultiplierTension"`
	ResourceMultiplierBalanced                 float64 `json:"resourceMultiplierBalanced"`
	ResourceMultiplierSurplus                  float64 `json:"resourceMultiplierSurplus"`
	ResourceMultiplierDominance                float64 `json:"resourceMultiplierDominance"`
	DefaultBuildingDurationMultiplier          float64 `json:"defaultBuildingDurationMultiplier"`
	DefaultBuildingMilestoneReduction          float64 `json:"defaultBuildingMilestoneReduction"`
	DefaultBuildingHighLevelMilestoneReduction float64 `json:"defaultBuildingHighLevelMilestoneReduction"`

	UpdatedBy string    `gorm:"size:120" json:"updatedBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
