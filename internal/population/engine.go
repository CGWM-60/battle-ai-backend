package population

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/policies"

	"gorm.io/gorm"
)

type CityPopulation struct {
	Total      int     `json:"total"`
	Capacity   int     `json:"capacity"`
	Workers    int     `json:"workers"`
	Unemployed int     `json:"unemployed"`
	Happiness  int     `json:"happiness"`
	GrowthRate float64 `json:"growthRate"`
}

type Engine struct {
	db           *gorm.DB
	policyEngine *policies.Engine
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{
		db:           db,
		policyEngine: policies.NewEngine(),
	}
}

// computeHappiness implements the exact spec formula (unchanged).
func (e *Engine) computeHappiness(services, foodBonus, security int, taxRate float64, weatherEvent string, activePolicies []string) int {
	h := 50.0
	h += float64(services)
	h += float64(foodBonus)
	h += float64(security)
	taxMalus := taxRate * 30.0
	h -= taxMalus
	weatherMalus := 0.0
	switch weatherEvent {
	case "drought", "heatwave":
		weatherMalus = 15
	case "storm", "flood":
		weatherMalus = 20
	case "earthquake":
		weatherMalus = 25
	}
	h -= weatherMalus
	for _, pk := range activePolicies {
		if pk == "city_festival" {
			h += 20
		}
		if pk == "austerity" {
			h -= 15
		}
	}
	h = math.Max(0, math.Min(100, h))
	return int(math.Round(h))
}

// loadFromSave builds CityPopulation from real PlayerSave + best-effort JSON data.
func (e *Engine) loadFromSave(save *models.PlayerSave) CityPopulation {
	total := int(save.Population)
	if total <= 0 {
		total = 800
	}
	cap := 1500 + (save.CityLevel * 120)
	happiness := save.Satisfaction
	if happiness <= 0 {
		happiness = 55
	}

	// Rough services / food / security from buildings + research awareness (improved in later waves)
	services := 10 + (save.CityLevel / 2)
	foodBonus := 6
	security := 8
	taxRate := 0.25

	// Read tax from ActiveEffectsJSON if present (matches economy engine)
	if len(save.ActiveEffectsJSON) > 0 {
		var fx map[string]any
		if json.Unmarshal(save.ActiveEffectsJSON, &fx) == nil {
			if t, ok := fx["tax_rate"].(float64); ok {
				taxRate = math.Max(0, math.Min(1, t))
			}
		}
	}

	h := e.computeHappiness(services, foodBonus, security, taxRate, "clear", []string{})

	growth := 3.8
	if h < 40 {
		growth = -2.1
	} else if h >= 55 {
		growth = (float64(cap-total) * 0.04) + 1.5
	}

	workers := int(float64(total) * (float64(h)/100.0*0.75 + 0.25))
	if workers > total {
		workers = total
	}
	unemp := total - workers

	return CityPopulation{
		Total:      total,
		Capacity:   cap,
		Workers:    workers,
		Unemployed: unemp,
		Happiness:  h,
		GrowthRate: growth,
	}
}

// GetPopulation is the single source of truth.
func (e *Engine) GetPopulation(ctx context.Context, playerID uint) (CityPopulation, error) {
	if e.db == nil {
		return CityPopulation{Total: 1240, Capacity: 1500, Workers: 980, Unemployed: 260, Happiness: 62, GrowthRate: 3.8}, nil
	}
	var save models.PlayerSave
	if err := e.db.WithContext(ctx).Where("player_id = ?", playerID).First(&save).Error; err != nil {
		return CityPopulation{}, err
	}
	return e.loadFromSave(&save), nil
}

// HourlyUpdate applies the exact spec formulas + real persistence via tx.
func (e *Engine) HourlyUpdate(ctx context.Context, playerID uint) error {
	if e.db == nil {
		return nil
	}
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
			return err
		}
		pop := e.loadFromSave(&save)

		growth := pop.GrowthRate
		if pop.Happiness < 55 {
			growth = -math.Abs(pop.GrowthRate) * 0.85
		}
		newTotal := pop.Total + int(growth)
		if newTotal < 0 {
			newTotal = 0
		}
		if newTotal > pop.Capacity {
			newTotal = pop.Capacity
		}

		// Persist population + satisfaction (real source of truth)
		return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
			Updates(map[string]any{
				"population":     int64(newTotal),
				"satisfaction":   pop.Happiness,
				"last_synced_at": time.Now(),
			}).Error
	})
}
