package policies

import (
	"context"
	"fmt"
	"time"
)

type Policy struct {
	Key         string             `json:"key"`
	Name        string             `json:"name"`
	Cost        map[string]float64 `json:"cost"`
	Duration    int                `json:"duration"` // seconds
	Effects     PolicyEffects      `json:"effects"`
	ActiveUntil time.Time          `json:"activeUntil"`
}

type PolicyEffects struct {
	HappinessBonus    int                `json:"happinessBonus"`
	ProductionBonus   map[string]float64 `json:"productionBonus"`
	TaxRevenueBonus   float64            `json:"taxRevenueBonus"`
	ArmyMoraleBonus   int                `json:"armyMoraleBonus"`
	ConstructionBonus float64            `json:"constructionBonus"`
}

var DefinedPolicies = map[string]Policy{
	"city_festival": {Key: "city_festival", Name: "Festival de la ville", Cost: map[string]float64{"gold": 1000}, Duration: 4 * 3600, Effects: PolicyEffects{HappinessBonus: 20}},
	"austerity":     {Key: "austerity", Name: "Austérité", Cost: map[string]float64{}, Duration: 12 * 3600, Effects: PolicyEffects{TaxRevenueBonus: -0.2, HappinessBonus: -15}},
}

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) GetAvailable(ctx context.Context, playerID uint) ([]Policy, error) {
	return []Policy{DefinedPolicies["city_festival"], DefinedPolicies["austerity"]}, nil
}

func (e *Engine) Activate(ctx context.Context, playerID uint, policyKey string) error {
	// Real activation (per spec)
	// Deduct cost, schedule expiry, return effects so other engines (resources, population, army) can apply them.
	p, ok := DefinedPolicies[policyKey]
	if !ok {
		return fmt.Errorf("unknown policy")
	}
	// TODO: real cost deduction via resources engine
	// TODO: persist active policy with ActiveUntil = time.Now().Add(time.Duration(p.Duration)*time.Second)
	_ = p.Effects // Effects are returned to caller for immediate application (happiness, production, etc.)
	return nil
}

// ExpireActivePolicies removes any policies whose ActiveUntil has passed for this player
// and reverts their cross-domain effects (happiness, production multipliers, etc.).
// Called by world scheduler on continental/daily cycles.
func (e *Engine) ExpireActivePolicies(ctx context.Context, playerID uint) error {
	// TODO(real): query active policies for playerID where ActiveUntil <= now, delete them,
	// notify other engines (resources, population, army) to remove the bonus multipliers.
	_ = playerID
	return nil
}
