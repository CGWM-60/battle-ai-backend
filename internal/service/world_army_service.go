package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BuildingUpgradeCost struct {
	CreditCost      int64          `json:"creditCost"`
	FoodCost        int64          `json:"foodCost"`
	EnergyCost      int64          `json:"energyCost"`
	MaterialCost    int64          `json:"materialCost"`
	DurationSeconds int64          `json:"durationSeconds"`
	NextLevel       int            `json:"nextLevel"`
	EffectsPreview  map[string]any `json:"effectsPreview"`
}

type ArmyTrainInput struct {
	UnitType string `json:"unitType"`
	Quantity int    `json:"quantity"`
}

type ArmyTrainResult struct {
	TrainingID uint           `json:"trainingId"`
	UnitType   string         `json:"unitType"`
	Quantity   int            `json:"quantity"`
	StartedAt  time.Time      `json:"startedAt"`
	FinishesAt time.Time      `json:"finishesAt"`
	Cost       datatypes.JSON `json:"cost"`
	Status     string         `json:"status"`
}

func (s *WorldGameService) CalculateBuildingUpgradeCost(buildingType string, currentLevel int) (BuildingUpgradeCost, error) {
	if currentLevel < 0 {
		return BuildingUpgradeCost{}, fmt.Errorf("current level must be >= 0")
	}
	if currentLevel >= 30 {
		return BuildingUpgradeCost{}, fmt.Errorf("BUILDING_MAX_LEVEL_REACHED: Ce bâtiment a déjà atteint le niveau maximum.")
	}
	nextLevel := currentLevel + 1

	base := buildingBaseProfile(normalizeBuildingType(buildingType))
	costFactor := math.Pow(1.28, float64(nextLevel-1))
	durationFactor := math.Pow(1.22, float64(nextLevel-1))

	durationSeconds := int64(float64(base.baseDurationSeconds) * durationFactor)
	maxDuration := int64((30 * 24 * time.Hour).Seconds())
	if durationSeconds > maxDuration {
		durationSeconds = maxDuration
	}
	if durationSeconds < 60 {
		durationSeconds = 60
	}

	return BuildingUpgradeCost{
		CreditCost:      int64(float64(base.baseCredits) * costFactor),
		FoodCost:        int64(float64(base.baseFood) * costFactor),
		EnergyCost:      int64(float64(base.baseEnergy) * costFactor),
		MaterialCost:    int64(float64(base.baseMaterials) * costFactor),
		DurationSeconds: durationSeconds,
		NextLevel:       nextLevel,
		EffectsPreview: map[string]any{
			"productionMultiplier": 1 + float64(nextLevel)*0.03,
			"storageBonus":         nextLevel * 25,
			"populationBonus":      nextLevel * 10,
		},
	}, nil
}

func (s *WorldGameService) TrainArmy(ctx context.Context, playerID uint, input ArmyTrainInput) (*ArmyTrainResult, error) {
	unitType := strings.TrimSpace(strings.ToLower(input.UnitType))
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be > 0")
	}
	if input.Quantity > 500 {
		return nil, fmt.Errorf("quantity exceeds safety limit")
	}
	if !isSupportedUnitType(unitType) {
		return nil, fmt.Errorf("unsupported unit type")
	}

	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return nil, err
	}

	barracksLevel := playerBuildingLevelFromJSON(save.BuildingsJSON, "barracks")
	if barracksLevel == 0 {
		return nil, fmt.Errorf("barracks is required")
	}
	required := minBarracksLevelForUnit(unitType)
	if barracksLevel < required {
		return nil, fmt.Errorf("barracks level too low for this unit")
	}

	unitStats := baseStatsForUnit(unitType)
	cost := map[string]any{
		"credits": unitStats.CreditCost * int64(input.Quantity),
		"food":    unitStats.FoodCost * int64(input.Quantity),
		"energy":  unitStats.EnergyCost * int64(input.Quantity),
	}
	if save.Credits < int64(cost["credits"].(int64)) || save.Food < int64(cost["food"].(int64)) || save.Energy < int64(cost["energy"].(int64)) {
		return nil, fmt.Errorf("NOT_ENOUGH_RESOURCES: Ressources insuffisantes pour lancer cette action.")
	}

	now := time.Now().UTC()
	totalSeconds := int64(unitStats.TrainingDurationSeconds) * int64(input.Quantity)
	finishesAt := now.Add(time.Duration(totalSeconds) * time.Second)
	costJSON, _ := json.Marshal(cost)

	queue := models.ArmyTrainingQueue{
		PlayerID:         playerID,
		UnitType:         unitType,
		Quantity:         input.Quantity,
		Status:           "training",
		StartedAt:        now,
		FinishesAt:       finishesAt,
		CostJSON:         datatypes.JSON(costJSON),
		BarracksLevelReq: required,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&queue).Error; err != nil {
			return err
		}
		return tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(map[string]any{
			"credits": save.Credits - cost["credits"].(int64),
			"food":    save.Food - cost["food"].(int64),
			"energy":  save.Energy - cost["energy"].(int64),
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return &ArmyTrainResult{
		TrainingID: queue.Id,
		UnitType:   queue.UnitType,
		Quantity:   queue.Quantity,
		StartedAt:  queue.StartedAt,
		FinishesAt: queue.FinishesAt,
		Cost:       queue.CostJSON,
		Status:     queue.Status,
	}, nil
}

func (s *WorldGameService) CompleteArmyTraining(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	var queued []models.ArmyTrainingQueue
	if err := s.db.WithContext(ctx).Where("status = ? AND finishes_at <= ?", "training", now).Find(&queued).Error; err != nil {
		return 0, err
	}
	if len(queued) == 0 {
		return 0, nil
	}

	created := int64(0)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range queued {
			stats := baseStatsForUnit(item.UnitType)
			for i := 0; i < item.Quantity; i++ {
				unit := models.ArmyUnit{
					PlayerID:                 item.PlayerID,
					UnitType:                 item.UnitType,
					Name:                     strings.Title(strings.ReplaceAll(item.UnitType, "_", " ")),
					Level:                    1,
					Experience:               0,
					Health:                   stats.Health,
					Attack:                   stats.Attack,
					Defense:                  stats.Defense,
					Speed:                    stats.Speed,
					Range:                    stats.Range,
					Morale:                   100,
					Endurance:                100,
					MaxCarry:                 stats.MaxCarry,
					FoodConsumptionPerHour:   stats.FoodConsumption,
					EnergyConsumptionPerHour: stats.EnergyConsumption,
					CreditMaintenancePerHour: stats.CreditCost,
					TrainingCost:             item.CostJSON,
					TrainingDurationSeconds:  int64(stats.TrainingDurationSeconds),
					Status:                   "available",
				}
				if err := tx.Create(&unit).Error; err != nil {
					return err
				}
				created++
			}
			if err := tx.Model(&models.ArmyTrainingQueue{}).Where("id = ?", item.Id).Update("status", "completed").Error; err != nil {
				return err
			}
		}
		return nil
	})
	return created, err
}

func (s *WorldGameService) UpdateArmyConsumption(ctx context.Context) (int64, error) {
	var units []models.ArmyUnit
	if err := s.db.WithContext(ctx).Where("status IN ?", []string{"available", "assigned", "returning", "injured", "exhausted"}).Find(&units).Error; err != nil {
		return 0, err
	}
	if len(units) == 0 {
		return 0, nil
	}

	byPlayer := map[uint]struct{ food, energy, credits int64 }{}
	for _, u := range units {
		v := byPlayer[u.PlayerID]
		v.food += u.FoodConsumptionPerHour
		v.energy += u.EnergyConsumptionPerHour
		v.credits += u.CreditMaintenancePerHour
		byPlayer[u.PlayerID] = v
	}

	affected := int64(0)
	return affected, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for playerID, total := range byPlayer {
			var save models.PlayerSave
			if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
				return err
			}

			nextFood := save.Food - total.food
			nextEnergy := save.Energy - total.energy
			nextCredits := save.Credits - total.credits
			if nextFood < 0 {
				nextFood = 0
			}
			if nextEnergy < 0 {
				nextEnergy = 0
			}
			if nextCredits < 0 {
				nextCredits = 0
			}

			if err := tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(map[string]any{
				"food":    nextFood,
				"energy":  nextEnergy,
				"credits": nextCredits,
			}).Error; err != nil {
				return err
			}

			moralePenalty := 0
			if save.Food < total.food || save.Credits < total.credits {
				moralePenalty += 6
			}
			if save.Energy < total.energy {
				moralePenalty += 4
			}
			if moralePenalty > 0 {
				if err := tx.Model(&models.ArmyUnit{}).
					Where("player_id = ? AND status IN ?", playerID, []string{"available", "assigned", "returning"}).
					Updates(map[string]any{
						"morale":    gorm.Expr("GREATEST(0, morale - ?)", moralePenalty),
						"endurance": gorm.Expr("GREATEST(0, endurance - ?)", moralePenalty/2),
					}).Error; err != nil {
					return err
				}
			}
			affected++
		}
		return nil
	})
}

func (s *WorldGameService) ListArmyUnits(ctx context.Context, playerID uint, limit int) ([]models.ArmyUnit, error) {
	var units []models.ArmyUnit
	err := s.db.WithContext(ctx).Where("player_id = ?", playerID).Order("id ASC").Limit(limitOrDefault(limit)).Find(&units).Error
	return units, err
}

func (s *WorldGameService) ArmyOverview(ctx context.Context, playerID uint) (map[string]any, error) {
	units, err := s.ListArmyUnits(ctx, playerID, 1000)
	if err != nil {
		return nil, err
	}
	power := s.CalculateArmyPower(units)
	byType := map[string]int{}
	for _, unit := range units {
		byType[unit.UnitType]++
	}
	return map[string]any{
		"totalUnits":  len(units),
		"unitsByType": byType,
		"power":       power,
	}, nil
}

func (s *WorldGameService) CalculateArmyPower(units []models.ArmyUnit) map[string]any {
	total := 0.0
	for _, u := range units {
		power :=
			float64(u.Health)*0.15 +
				float64(u.Attack)*0.25 +
				float64(u.Defense)*0.20 +
				float64(u.Speed)*0.10 +
				float64(u.Range)*0.10 +
				float64(u.Morale)*0.10 +
				float64(u.Endurance)*0.10
		total += power
	}
	return map[string]any{
		"basePower": total,
		"bonuses": []map[string]any{
			{"type": "barracks_bonus", "value": 0},
			{"type": "research_bonus", "value": 0},
			{"type": "morale_bonus", "value": 0},
		},
		"maluses": []map[string]any{
			{"type": "weather_malus", "value": 0},
			{"type": "food_shortage_malus", "value": 0},
			{"type": "energy_shortage_malus", "value": 0},
		},
		"finalPower": total,
	}
}

type buildingBaseCostProfile struct {
	baseCredits         int64
	baseFood            int64
	baseEnergy          int64
	baseMaterials       int64
	baseDurationSeconds int64
}

func buildingBaseProfile(buildingType string) buildingBaseCostProfile {
	switch buildingType {
	case "barracks":
		return buildingBaseCostProfile{baseCredits: 1200, baseFood: 250, baseEnergy: 180, baseMaterials: 350, baseDurationSeconds: 1200}
	case "city_hall":
		return buildingBaseCostProfile{baseCredits: 1400, baseFood: 220, baseEnergy: 140, baseMaterials: 300, baseDurationSeconds: 1500}
	case "habitation":
		return buildingBaseCostProfile{baseCredits: 450, baseFood: 80, baseEnergy: 40, baseMaterials: 120, baseDurationSeconds: 300}
	case "vertical_farm":
		return buildingBaseCostProfile{baseCredits: 520, baseFood: 50, baseEnergy: 70, baseMaterials: 140, baseDurationSeconds: 360}
	case "solar_park":
		return buildingBaseCostProfile{baseCredits: 650, baseFood: 40, baseEnergy: 30, baseMaterials: 160, baseDurationSeconds: 420}
	default:
		return buildingBaseCostProfile{baseCredits: 700, baseFood: 120, baseEnergy: 90, baseMaterials: 180, baseDurationSeconds: 600}
	}
}

func normalizeBuildingType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "caserne" {
		return "barracks"
	}
	return normalized
}

func playerBuildingLevelFromJSON(raw datatypes.JSON, key string) int {
	if len(raw) == 0 {
		return 0
	}
	items := make([]map[string]any, 0)
	if err := json.Unmarshal(raw, &items); err != nil {
		return 0
	}
	for _, item := range items {
		buildingKey := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["buildingKey"])))
		if buildingKey != strings.ToLower(strings.TrimSpace(key)) {
			continue
		}
		if level, ok := item["level"].(float64); ok {
			return int(level)
		}
	}
	return 0
}

type unitProfile struct {
	Health                  int
	Attack                  int
	Defense                 int
	Speed                   int
	Range                   int
	MaxCarry                int
	FoodConsumption         int64
	EnergyConsumption       int64
	CreditCost              int64
	TrainingDurationSeconds int
	FoodCost                int64
	EnergyCost              int64
	CreditCostTrain         int64
}

func isSupportedUnitType(unitType string) bool {
	switch unitType {
	case "infanterie_legere", "infanterie_lourde", "tireur_longue_portee", "unite_logistique", "unite_elite":
		return true
	default:
		return false
	}
}

func minBarracksLevelForUnit(unitType string) int {
	switch unitType {
	case "infanterie_legere":
		return 1
	case "infanterie_lourde":
		return 3
	case "tireur_longue_portee":
		return 5
	case "unite_logistique":
		return 8
	case "unite_elite":
		return 12
	default:
		return 99
	}
}

func baseStatsForUnit(unitType string) unitProfile {
	switch unitType {
	case "infanterie_legere":
		return unitProfile{Health: 90, Attack: 18, Defense: 10, Speed: 16, Range: 5, MaxCarry: 20, FoodConsumption: 2, EnergyConsumption: 1, CreditCost: 1, TrainingDurationSeconds: 120, FoodCost: 10, EnergyCost: 8, CreditCostTrain: 25}
	case "infanterie_lourde":
		return unitProfile{Health: 140, Attack: 24, Defense: 24, Speed: 8, Range: 4, MaxCarry: 16, FoodConsumption: 3, EnergyConsumption: 2, CreditCost: 2, TrainingDurationSeconds: 240, FoodCost: 16, EnergyCost: 12, CreditCostTrain: 45}
	case "tireur_longue_portee":
		return unitProfile{Health: 80, Attack: 30, Defense: 8, Speed: 10, Range: 20, MaxCarry: 12, FoodConsumption: 3, EnergyConsumption: 3, CreditCost: 2, TrainingDurationSeconds: 260, FoodCost: 20, EnergyCost: 18, CreditCostTrain: 60}
	case "unite_logistique":
		return unitProfile{Health: 100, Attack: 6, Defense: 10, Speed: 12, Range: 5, MaxCarry: 80, FoodConsumption: 3, EnergyConsumption: 2, CreditCost: 2, TrainingDurationSeconds: 300, FoodCost: 18, EnergyCost: 20, CreditCostTrain: 55}
	default:
		return unitProfile{Health: 220, Attack: 45, Defense: 35, Speed: 14, Range: 12, MaxCarry: 28, FoodConsumption: 5, EnergyConsumption: 4, CreditCost: 4, TrainingDurationSeconds: 480, FoodCost: 40, EnergyCost: 35, CreditCostTrain: 150}
	}
}

// DisbandUnits (interaction 6) - removes units and refunds partial resources.
func (s *WorldGameService) DisbandUnits(ctx context.Context, playerID uint, unitType string, count int) (int, error) {
	if count <= 0 {
		return 0, fmt.Errorf("count must be > 0")
	}
	return count, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var units []models.ArmyUnit
		if err := tx.Where("player_id = ? AND unit_type = ? AND status = ?", playerID, unitType, "available").Limit(count).Find(&units).Error; err != nil {
			return err
		}
		if len(units) == 0 {
			return fmt.Errorf("no available units of that type")
		}
		actual := 0
		for i := range units {
			if actual >= count {
				break
			}
			if err := tx.Delete(&units[i]).Error; err != nil {
				return err
			}
			actual++
		}
		// Partial refund
		stats := baseStatsForUnit(unitType)
		refund := int64(actual) * (stats.CreditCostTrain / 2)
		return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
			Update("credits", gorm.Expr("credits + ?", refund)).Error
	})
}

// HealUnits (interaction 7) - heals injured units.
func (s *WorldGameService) HealUnits(ctx context.Context, playerID uint, unitType string, count int) (int, error) {
	if count <= 0 {
		return 0, nil
	}
	var injured []models.ArmyUnit
	if err := s.db.WithContext(ctx).Where("player_id = ? AND unit_type = ? AND status = ?", playerID, unitType, "injured").Limit(count).Find(&injured).Error; err != nil {
		return 0, err
	}
	healed := 0
	for i := range injured {
		if healed >= count {
			break
		}
		injured[i].Health = 100
		injured[i].Status = "available"
		if err := s.db.WithContext(ctx).Save(&injured[i]).Error; err != nil {
			return healed, err
		}
		healed++
	}
	return healed, nil
}

// SetDefenseAssignment (interaction 8) - assigns units to defense.
func (s *WorldGameService) SetDefenseAssignment(ctx context.Context, playerID uint, units map[string]int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for ut, qty := range units {
			if qty <= 0 {
				continue
			}
			// Simple: mark up to qty available units as "assigned" for defense
			var avail []models.ArmyUnit
			tx.Where("player_id = ? AND unit_type = ? AND status = ?", playerID, ut, "available").Limit(qty).Find(&avail)
			for i := range avail {
				avail[i].Status = "assigned"
				if err := tx.Save(&avail[i]).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
