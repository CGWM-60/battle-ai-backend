package resources

import (
	"context"

	"cgwm/battle/internal/policies"
	"cgwm/battle/internal/research"
	"cgwm/battle/internal/weather"

	"gorm.io/gorm"
)

// ResourceBalance is the source of truth struct returned by the resources engine.
type ResourceBalance struct {
	Current     map[string]float64 `json:"current"`
	Capacity    map[string]float64 `json:"capacity"`
	Production  map[string]float64 `json:"production"`  // per hour
	Consumption map[string]float64 `json:"consumption"`
	Net         map[string]float64 `json:"net"`
}

// Engine is the authoritative resources calculation engine.
// All production, consumption, ticks, and deficit effects must go through here.
type Engine struct {
	db *gorm.DB
	// Resolvers (use whatever is exported today)
	researchResolver *research.Resolver
	policyEngine     *policies.Engine
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{
		db:               db,
		researchResolver: research.NewResolver(),
		policyEngine:     policies.NewEngine(),
	}
}

// GetBalance returns the current resource state for a player.
// This is the single source of truth. No local calculation in Flutter should contradict this.
func (e *Engine) GetBalance(ctx context.Context, playerID uint) (ResourceBalance, error) {
	// TODO(real): load full state from PlayerSave + buildings + population + army upkeep.

	// Base numbers (will be replaced by real load + formulas)
	balance := ResourceBalance{
		Current:     map[string]float64{"gold": 12500, "energy": 3400, "food": 2890, "water": 2100, "materials": 980, "research_points": 145},
		Capacity:    map[string]float64{"gold": 50000, "energy": 8000, "food": 12000, "water": 6000, "materials": 5000, "research_points": 500},
		Production:  map[string]float64{"gold": 420, "energy": 180, "food": 95, "water": 40, "materials": 32, "research_points": 8},
		Consumption: map[string]float64{"gold": 180, "energy": 210, "food": 120, "water": 55, "materials": 18, "research_points": 0},
	}

	// === Apply cross-domain bonuses (Go = single source of truth) ===
	// Research bonuses (product of unlocked multipliers)
	researchMulti := 1.0
	if e.researchResolver != nil {
		// TODO(real): load the player's actually unlocked node keys
		bonuses := e.researchResolver.Compute([]string{})
		for _, m := range bonuses.ProductionMultipliers {
			researchMulti *= m
		}
	}

	// Weather (uses the existing package-level function)
	weatherMulti := 1.0
	_ = weather.ApplyWeatherModifiers(map[string]float64{}, "clear") // placeholder until we have active event

	// Policy (stub for now)
	policyMulti := 1.0
	if e.policyEngine != nil {
		// TODO: load active policies for player and multiply their productionBonus
	}

	finalProdMulti := researchMulti * weatherMulti * policyMulti

	for r := range balance.Production {
		balance.Production[r] *= finalProdMulti
	}

	for r, prod := range balance.Production {
		cons := balance.Consumption[r]
		balance.Net[r] = prod - cons
	}

	return balance, nil
}

// Tick applies the 10-minute resource tick (the authoritative one — Go source of truth).
// Applies Net * (minutes/60), clamps Current to [0, Capacity], and applies deficit cascading
// (energy <0 → production * 0.3, food deficit → happiness malus, etc.).
func (e *Engine) Tick(ctx context.Context, playerID uint, minutes float64) error {
	// In full impl: load full PlayerSave + buildings + research + weather + policies + army upkeep.
	// For now: demonstrate the exact formulas on the minimal balance returned by GetBalance.
	balance, err := e.GetBalance(ctx, playerID)
	if err != nil {
		return err
	}

	fraction := minutes / 60.0
	for r, net := range balance.Net {
		delta := net * fraction
		newCur := balance.Current[r] + delta
		if newCur < 0 {
			newCur = 0
		}
		if cap, ok := balance.Capacity[r]; ok && newCur > cap {
			newCur = cap
		}
		balance.Current[r] = newCur
	}

	// Deficit side effects (applied to the balance for next Get)
	energyNet := 0.0
	if v, ok := balance.Net["energy"]; ok {
		energyNet = v
	}
	if energyNet < 0 {
		// Example: energy deficit slows other production
		for r := range balance.Production {
			if r != "energy" {
				balance.Production[r] *= 0.3
			}
		}
	}

	foodNet := 0.0
	if v, ok := balance.Net["food"]; ok {
		foodNet = v
	}
	if foodNet < 0 {
		// Food deficit would reduce happiness in population engine
		_ = "food deficit marker for population/happiness cross-effect"
	}

	// TODO(real): persist the updated Current back to PlayerSave / resources table (or dedicated resources snapshot table)
	// Bonuses (research/weather/policy) are already applied in GetBalance before Tick uses the Net.

	return nil
}

// ManualCollect implements POST /api/v1/buildings/:id/collect for collector buildings.
func (e *Engine) ManualCollect(ctx context.Context, playerID uint, buildingID string) (map[string]float64, error) {
	// TODO: find the building, transfer its accumulator into Current, reset accumulator.
	return map[string]float64{"food": 124.5, "materials": 18}, nil
}
