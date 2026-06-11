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
