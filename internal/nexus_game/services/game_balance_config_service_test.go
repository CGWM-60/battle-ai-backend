package services

import "testing"

func TestDefaultGameBalanceConfigUsesMMOPopulationScale(t *testing.T) {
	cfg := DefaultGameBalanceConfig()

	cases := map[int]int{
		1:  500,
		2:  750,
		10: 2750,
		20: 5250,
		30: 7750,
	}
	for level, want := range cases {
		if got := PopulationCapacityForHabitatLevel(cfg, level); got != want {
			t.Fatalf("habitat level %d capacity = %d, want %d", level, got, want)
		}
	}
}

func TestResourceMultiplierForEnergyRatioUsesConfigurableThresholds(t *testing.T) {
	cfg := DefaultGameBalanceConfig()

	if got := ResourceMultiplierForEnergyRatio(cfg, 0.70); got != cfg.ResourceMultiplierCritical {
		t.Fatalf("critical multiplier = %f, want %f", got, cfg.ResourceMultiplierCritical)
	}
	if got := ResourceMultiplierForEnergyRatio(cfg, 1.30); got != cfg.ResourceMultiplierSurplus {
		t.Fatalf("surplus multiplier = %f, want %f", got, cfg.ResourceMultiplierSurplus)
	}
	if got := ResourceMultiplierForEnergyRatio(cfg, 1.60); got != cfg.ResourceMultiplierDominance {
		t.Fatalf("dominance multiplier = %f, want %f", got, cfg.ResourceMultiplierDominance)
	}
}
