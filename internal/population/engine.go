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
