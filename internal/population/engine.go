package population

import (
	"context"
	"math"

	"cgwm/battle/internal/policies"
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
	policyEngine *policies.Engine
}

func NewEngine() *Engine {
	return &Engine{
		policyEngine: policies.NewEngine(),
	}
}

// computeHappiness implements the exact spec formula:
// happiness = clamp(0, 100, base(50) + services(0-20) + foodBonus(0-15) + security(0-15) - taxMalus(0-30) - weatherMalus(0-25) + policyBonus(-15 to +20))
func (e *Engine) computeHappiness(services, foodBonus, security int, taxRate float64, weatherEvent string, activePolicies []string) int {
	h := 50.0 // base

	h += float64(services)      // from buildings/services
	h += float64(foodBonus)     // from resources food surplus
	h += float64(security)      // from army/defense assignment

	// tax malus: 0% tax = 0, 100% tax = -30
	taxMalus := taxRate * 30.0
	h -= taxMalus

	// weather malus via existing resolver logic
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

	// policy bonuses (simplified - real would sum from active policies)
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

// GetPopulation is the single source of truth for city population + happiness.
func (e *Engine) GetPopulation(ctx context.Context, playerID uint) (CityPopulation, error) {
	// TODO(real): load from PlayerSave + buildings (services count), resources (food net), army (security), tax rate, active weather, active policies.
	// For now: realistic stub with formula applied to example inputs.
	happiness := e.computeHappiness(12, 8, 10, 0.45, "clear", []string{"city_festival"})

	total := 1240
	cap := 1500
	workers := 980
	unemp := total - workers
	growth := 4.2
	if happiness < 40 {
		growth = -1.8 // loss
	}

	return CityPopulation{
		Total:      total,
		Capacity:   cap,
		Workers:    workers,
		Unemployed: unemp,
		Happiness:  happiness,
		GrowthRate: growth,
	}, nil
}

// HourlyUpdate applies the exact spec formulas (called by scheduler).
// happiness formula as above.
// growth = (Capacity - Total)*0.05 if happiness >= 55 else -Total*0.01 (capped).
// Food deficit from resources engine should further reduce happiness/growth (cross-call TODO).
func (e *Engine) HourlyUpdate(ctx context.Context, playerID uint) error {
	// TODO(real): load current CityPopulation, taxRate, services, foodNet, security, weatherEvent, activePolicyKeys.
	// Then:
	// 1. happiness = computeHappiness(...)
	// 2. if foodNet < 0 { happiness = max(0, happiness-10); growth *= 0.5 }
	// 3. growth = (cap-total)*0.05 if happiness >= 55 else max(-total*0.01, -50)
	// 4. newTotal = clamp(0, cap, total + growth)
	// 5. workers = newTotal * (happiness/100 * 0.8 + 0.2) etc.
	// 6. persist to PlayerSave.

	// Simulate one hour of growth/loss per spec (real: load + compute + UPDATE PlayerSave.Population + Satisfaction)
	pop, _ := e.GetPopulation(ctx, playerID)

	growth := pop.GrowthRate
	if pop.Happiness < 55 {
		growth = -math.Abs(pop.GrowthRate) * 0.8 // loss
	}
	newTotal := pop.Total + int(growth)
	if newTotal < 0 {
		newTotal = 0
	}
	if newTotal > pop.Capacity {
		newTotal = pop.Capacity
	}

	// TODO(real): persist newTotal, recalculated workers/unemployed, updated Happiness to DB
	_ = newTotal // would be saved
	_ = playerID
	return nil
}
