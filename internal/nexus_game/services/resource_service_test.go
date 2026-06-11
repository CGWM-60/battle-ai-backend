package services

import (
	"math"
	"testing"

	"cgwm/battle/internal/nexus_game/models"
)

func TestEnergyProductionKeepsHourlyDisplayAndPerSecondResourceTick(t *testing.T) {
	acc := newProductionAccumulator()
	applyBuildingStorageAndProduction(&acc, models.BuildingDefinition{
		StorageResource:       "energy",
		ProductionBasePerHour: 80,
		ProductionGrowth:      1,
	}, 1)

	if acc.EnergyProduction != 80 {
		t.Fatalf("energy display production = %d, want 80 per hour", acc.EnergyProduction)
	}
	if got, want := acc.ResourceProduction["energy"], 80.0/3600.0; math.Abs(got-want) > 0.000001 {
		t.Fatalf("energy resource production per tick = %f, want %f", got, want)
	}

	setEnergyProductionPerHour(&acc, 120)
	if acc.EnergyProduction != 120 {
		t.Fatalf("energy display production after bonus = %d, want 120 per hour", acc.EnergyProduction)
	}
	if got, want := acc.ResourceProduction["energy"], 120.0/3600.0; math.Abs(got-want) > 0.000001 {
		t.Fatalf("energy resource production after bonus = %f, want %f", got, want)
	}
}

func TestLevel1StarterAllocationCoversFourStarterBuildings(t *testing.T) {
	starter := Level1StarterAllocation()

	required := map[string]int64{
		"credits": 100 + 150 + 120 + 200,
		"metal":   200 + 300 + 250 + 350,
		"data":    50,
	}
	for code, wantMin := range required {
		if got := starter[code]; got < wantMin {
			t.Fatalf("starter %s = %d, want at least %d for habitat+solar+farm+mine", code, got, wantMin)
		}
	}
	if starter["metal"] > DefaultStorageCapacity {
		t.Fatalf("starter metal = %d exceeds default storage capacity %d", starter["metal"], DefaultStorageCapacity)
	}
}

func TestStarterAllocationUpgradeOnlyAddsMissingDelta(t *testing.T) {
	previous := map[string]int64{
		"credits": 450,
		"metal":   800,
		"data":    100,
	}
	target := map[string]int64{
		"credits": 800,
		"metal":   1400,
		"data":    200,
	}

	upgrade := starterAllocationUpgrade(previous, target)
	if got, want := upgrade["credits"], int64(350); got != want {
		t.Fatalf("credits upgrade = %d, want %d", got, want)
	}
	if got, want := upgrade["metal"], int64(600); got != want {
		t.Fatalf("metal upgrade = %d, want %d", got, want)
	}
	if got, want := upgrade["data"], int64(100); got != want {
		t.Fatalf("data upgrade = %d, want %d", got, want)
	}

	if second := starterAllocationUpgrade(target, target); len(second) != 0 {
		t.Fatalf("second upgrade = %#v, want none", second)
	}
}
