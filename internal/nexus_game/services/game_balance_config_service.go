package services

import (
	"context"
	"math"
	"time"

	"cgwm/battle/internal/nexus_game/models"
	"gorm.io/gorm"
)

func DefaultGameBalanceConfig() models.GameBalanceConfig {
	return models.GameBalanceConfig{
		Key:                                        models.ActiveGameBalanceConfigKey,
		PopulationHabitatBase:                      500,
		PopulationHabitatPerLevel:                  250,
		FoodPerPopulationPerHour:                   0.08,
		EnergyBaseConsumptionPerHour:               5,
		PopulationPerEnergy:                        20,
		BuildingEnergySurchargeExponent:            1.25,
		BuildingEnergyHighLevelThreshold:           20,
		BuildingEnergyHighLevelMultiplier:          1.2,
		IndustryEnergyPerLevel:                     2,
		DataEnergyPerLevel:                         1,
		SolarEnergyReliefPerLevel:                  1,
		HabitatFarmCoverageDivisor:                 2,
		HabitatFarmEnergyPenaltyDivisor:            3,
		EnergyRatioCriticalThreshold:               0.75,
		EnergyRatioTensionThreshold:                1.0,
		EnergyRatioSurplusThreshold:                1.2,
		EnergyRatioDominanceThreshold:              1.5,
		ResourceMultiplierCritical:                 0.75,
		ResourceMultiplierTension:                  0.95,
		ResourceMultiplierBalanced:                 1.0,
		ResourceMultiplierSurplus:                  1.10,
		ResourceMultiplierDominance:                1.20,
		DefaultBuildingDurationMultiplier:          1.28,
		DefaultBuildingMilestoneReduction:          0.15,
		DefaultBuildingHighLevelMilestoneReduction: 0.08,
		UpdatedBy: "system-default",
	}
}

type GameBalanceConfigService struct {
	db *gorm.DB
}

func NewGameBalanceConfigService(db *gorm.DB) *GameBalanceConfigService {
	return &GameBalanceConfigService{db: db}
}

func (s *GameBalanceConfigService) GetActive(ctx context.Context) (models.GameBalanceConfig, error) {
	return LoadGameBalanceConfig(ctx, s.db)
}

func (s *GameBalanceConfigService) SaveActive(ctx context.Context, incoming models.GameBalanceConfig) (models.GameBalanceConfig, error) {
	if s.db == nil {
		cfg := sanitizeGameBalanceConfig(incoming)
		return cfg, nil
	}
	current, err := LoadGameBalanceConfig(ctx, s.db)
	if err != nil {
		return current, err
	}
	incoming.ID = current.ID
	incoming.Key = models.ActiveGameBalanceConfigKey
	incoming.CreatedAt = current.CreatedAt
	if incoming.UpdatedBy == "" {
		incoming.UpdatedBy = "admin"
	}
	cfg := sanitizeGameBalanceConfig(incoming)
	cfg.UpdatedAt = time.Now().UTC()
	return cfg, s.db.WithContext(ctx).Save(&cfg).Error
}

func LoadGameBalanceConfig(ctx context.Context, db *gorm.DB) (models.GameBalanceConfig, error) {
	cfg := sanitizeGameBalanceConfig(DefaultGameBalanceConfig())
	if db == nil {
		return cfg, nil
	}
	var existing models.GameBalanceConfig
	err := db.WithContext(ctx).Where("`key` = ?", models.ActiveGameBalanceConfigKey).First(&existing).Error
	if err == nil {
		return sanitizeGameBalanceConfig(existing), nil
	}
	if err != gorm.ErrRecordNotFound {
		return cfg, err
	}
	cfg.CreatedAt = time.Now().UTC()
	cfg.UpdatedAt = cfg.CreatedAt
	err = db.WithContext(ctx).Create(&cfg).Error
	return cfg, err
}

func sanitizeGameBalanceConfig(cfg models.GameBalanceConfig) models.GameBalanceConfig {
	defaults := DefaultGameBalanceConfig()
	cfg.Key = models.ActiveGameBalanceConfigKey
	if cfg.PopulationHabitatBase <= 0 {
		cfg.PopulationHabitatBase = defaults.PopulationHabitatBase
	}
	if cfg.PopulationHabitatPerLevel < 0 {
		cfg.PopulationHabitatPerLevel = defaults.PopulationHabitatPerLevel
	}
	if cfg.FoodPerPopulationPerHour <= 0 {
		cfg.FoodPerPopulationPerHour = defaults.FoodPerPopulationPerHour
	}
	if cfg.EnergyBaseConsumptionPerHour < 0 {
		cfg.EnergyBaseConsumptionPerHour = defaults.EnergyBaseConsumptionPerHour
	}
	if cfg.PopulationPerEnergy <= 0 {
		cfg.PopulationPerEnergy = defaults.PopulationPerEnergy
	}
	if cfg.BuildingEnergySurchargeExponent <= 0 {
		cfg.BuildingEnergySurchargeExponent = defaults.BuildingEnergySurchargeExponent
	}
	if cfg.BuildingEnergyHighLevelThreshold <= 0 {
		cfg.BuildingEnergyHighLevelThreshold = defaults.BuildingEnergyHighLevelThreshold
	}
	if cfg.BuildingEnergyHighLevelMultiplier <= 0 {
		cfg.BuildingEnergyHighLevelMultiplier = defaults.BuildingEnergyHighLevelMultiplier
	}
	if cfg.IndustryEnergyPerLevel < 0 {
		cfg.IndustryEnergyPerLevel = defaults.IndustryEnergyPerLevel
	}
	if cfg.DataEnergyPerLevel < 0 {
		cfg.DataEnergyPerLevel = defaults.DataEnergyPerLevel
	}
	if cfg.SolarEnergyReliefPerLevel < 0 {
		cfg.SolarEnergyReliefPerLevel = defaults.SolarEnergyReliefPerLevel
	}
	if cfg.HabitatFarmCoverageDivisor <= 0 {
		cfg.HabitatFarmCoverageDivisor = defaults.HabitatFarmCoverageDivisor
	}
	if cfg.HabitatFarmEnergyPenaltyDivisor <= 0 {
		cfg.HabitatFarmEnergyPenaltyDivisor = defaults.HabitatFarmEnergyPenaltyDivisor
	}
	if cfg.EnergyRatioCriticalThreshold <= 0 {
		cfg.EnergyRatioCriticalThreshold = defaults.EnergyRatioCriticalThreshold
	}
	if cfg.EnergyRatioTensionThreshold <= cfg.EnergyRatioCriticalThreshold {
		cfg.EnergyRatioTensionThreshold = defaults.EnergyRatioTensionThreshold
	}
	if cfg.EnergyRatioSurplusThreshold <= cfg.EnergyRatioTensionThreshold {
		cfg.EnergyRatioSurplusThreshold = defaults.EnergyRatioSurplusThreshold
	}
	if cfg.EnergyRatioDominanceThreshold <= cfg.EnergyRatioSurplusThreshold {
		cfg.EnergyRatioDominanceThreshold = defaults.EnergyRatioDominanceThreshold
	}
	if cfg.ResourceMultiplierCritical <= 0 {
		cfg.ResourceMultiplierCritical = defaults.ResourceMultiplierCritical
	}
	if cfg.ResourceMultiplierTension <= 0 {
		cfg.ResourceMultiplierTension = defaults.ResourceMultiplierTension
	}
	if cfg.ResourceMultiplierBalanced <= 0 {
		cfg.ResourceMultiplierBalanced = defaults.ResourceMultiplierBalanced
	}
	if cfg.ResourceMultiplierSurplus <= 0 {
		cfg.ResourceMultiplierSurplus = defaults.ResourceMultiplierSurplus
	}
	if cfg.ResourceMultiplierDominance <= 0 {
		cfg.ResourceMultiplierDominance = defaults.ResourceMultiplierDominance
	}
	if cfg.DefaultBuildingDurationMultiplier <= 0 {
		cfg.DefaultBuildingDurationMultiplier = defaults.DefaultBuildingDurationMultiplier
	}
	if cfg.DefaultBuildingMilestoneReduction <= 0 {
		cfg.DefaultBuildingMilestoneReduction = defaults.DefaultBuildingMilestoneReduction
	}
	if cfg.DefaultBuildingHighLevelMilestoneReduction <= 0 {
		cfg.DefaultBuildingHighLevelMilestoneReduction = defaults.DefaultBuildingHighLevelMilestoneReduction
	}
	return cfg
}

func PopulationCapacityForHabitatLevel(cfg models.GameBalanceConfig, level int) int {
	if level < 1 {
		level = 1
	}
	return cfg.PopulationHabitatBase + cfg.PopulationHabitatPerLevel*(level-1)
}

func BuildingEnergySurcharge(cfg models.GameBalanceConfig, level int) int {
	if level <= 0 {
		return 0
	}
	surcharge := int(math.Floor(math.Pow(float64(level), cfg.BuildingEnergySurchargeExponent)))
	if level > cfg.BuildingEnergyHighLevelThreshold {
		surcharge = int(math.Floor(float64(surcharge) * cfg.BuildingEnergyHighLevelMultiplier))
	}
	return surcharge
}

func ResourceMultiplierForEnergyRatio(cfg models.GameBalanceConfig, ratio float64) float64 {
	switch {
	case ratio < cfg.EnergyRatioCriticalThreshold:
		return cfg.ResourceMultiplierCritical
	case ratio < cfg.EnergyRatioTensionThreshold:
		return cfg.ResourceMultiplierTension
	case ratio < cfg.EnergyRatioSurplusThreshold:
		return cfg.ResourceMultiplierBalanced
	case ratio < cfg.EnergyRatioDominanceThreshold:
		return cfg.ResourceMultiplierSurplus
	default:
		return cfg.ResourceMultiplierDominance
	}
}
