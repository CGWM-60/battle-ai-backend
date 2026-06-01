package population

import "context"

type CityPopulation struct {
	Total      int     `json:"total"`
	Capacity   int     `json:"capacity"`
	Workers    int     `json:"workers"`
	Unemployed int     `json:"unemployed"`
	Happiness  int     `json:"happiness"`
	GrowthRate float64 `json:"growthRate"`
}

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) GetPopulation(ctx context.Context, playerID uint) (CityPopulation, error) {
	// TODO: full happiness calculation from services, food, security, tax, weather, policies
	return CityPopulation{
		Total:      1240,
		Capacity:   1500,
		Workers:    980,
		Unemployed: 260,
		Happiness:  72,
		GrowthRate: 4.2,
	}, nil
}

// HourlyUpdate applies the exact spec formulas for one player (called by world scheduler on continental/daily cycles):
// happiness = clamp(0,100, base + services + foodBonus + security - taxMalus - weatherMalus + policyBonus)
// growth = (Capacity - Total)*0.05 if happy > threshold else -Total*0.01
// Also triggers population loss on food deficit (cascaded from resources engine).
func (e *Engine) HourlyUpdate(ctx context.Context, playerID uint) error {
	// TODO(real): load CityPopulation + active policies + weather + tax rate + services from buildings
	// Apply exact happiness formula, compute growth, persist new Total/Workers/Unemployed/Happiness/GrowthRate
	// Cross-call resources for food deficit effects.
	_ = playerID
	return nil
}
