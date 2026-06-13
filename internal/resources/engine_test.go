package resources

import (
	"testing"
	"time"

	"cgwm/battle/internal/models"
)

func TestCollectorProfileForBuildingReturnsKnownResources(t *testing.T) {
	tests := []struct {
		name      string
		building  string
		resource  string
		baseRate  float64
		wantFound bool
	}{
		{name: "solar park", building: "solar_park", resource: "energy", baseRate: 40, wantFound: true},
		{name: "vertical farm", building: "vertical_farm", resource: "food", baseRate: 35, wantFound: true},
		{name: "mine", building: "mine", resource: "materials", baseRate: 22, wantFound: true},
		{name: "market", building: "market", resource: "credits", baseRate: 55, wantFound: true},
		{name: "unknown", building: "bunker", wantFound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := collectorProfileForBuilding(tt.building)
			if ok != tt.wantFound {
				t.Fatalf("found=%v want %v", ok, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if got.Resource != tt.resource {
				t.Fatalf("resource=%q want %q", got.Resource, tt.resource)
			}
			if got.BaseRate != tt.baseRate {
				t.Fatalf("baseRate=%v want %v", got.BaseRate, tt.baseRate)
			}
		})
	}
}

func TestComputeCollectorHarvestCapsAtOneCycle(t *testing.T) {
	profile := collectorProfile{Resource: "food", BaseRate: 35}
	lastCollectedAt := time.Now().Add(-2 * collectorCycleDuration)
	amount, progress := computeCollectorHarvest(profile, 2, &lastCollectedAt, time.Now())

	if amount != 70 {
		t.Fatalf("amount=%v want 70", amount)
	}
	if progress != 100 {
		t.Fatalf("progress=%v want 100", progress)
	}
}

func TestComputeCollectorHarvestUsesElapsedFraction(t *testing.T) {
	profile := collectorProfile{Resource: "energy", BaseRate: 40}
	lastCollectedAt := time.Now().Add(-30 * time.Minute)
	amount, progress := computeCollectorHarvest(profile, 3, &lastCollectedAt, time.Now())

	if amount != 60 {
		t.Fatalf("amount=%v want 60", amount)
	}
	if progress != 50 {
		t.Fatalf("progress=%v want 50", progress)
	}
}

func TestFindCollectibleBuildingIndexMatchesKeyAndID(t *testing.T) {
	buildings := []map[string]any{
		{"id": 11, "buildingKey": "habitation"},
		{"id": 22, "buildingKey": "solar_park"},
	}

	if got := findCollectibleBuildingIndex(buildings, "solar_park"); got != 1 {
		t.Fatalf("building key lookup = %d want 1", got)
	}
	if got := findCollectibleBuildingIndex(buildings, "22"); got != 1 {
		t.Fatalf("numeric id lookup = %d want 1", got)
	}
	if got := findCollectibleBuildingIndex(buildings, "missing"); got != -1 {
		t.Fatalf("missing building lookup = %d want -1", got)
	}
}

func TestAddResourceToSaveUpdatesInventoryAndOfficialColumns(t *testing.T) {
	save := &models.PlayerSave{Food: 10, Energy: 20, Credits: 30}
	inventory := map[string]float64{}

	addResourceToSave(save, inventory, "food", 12.4)
	addResourceToSave(save, inventory, "energy", 5.6)
	addResourceToSave(save, inventory, "credits", 7.9)
	addResourceToSave(save, inventory, "materials", 3.2)

	if save.Food != 22 {
		t.Fatalf("food=%d want 22", save.Food)
	}
	if save.Energy != 26 {
		t.Fatalf("energy=%d want 26", save.Energy)
	}
	if save.Credits != 38 {
		t.Fatalf("credits=%d want 38", save.Credits)
	}
	if inventory["food"] != 12.4 {
		t.Fatalf("food inventory=%v want 12.4", inventory["food"])
	}
	if inventory["energy"] != 5.6 {
		t.Fatalf("energy inventory=%v want 5.6", inventory["energy"])
	}
	if inventory["credits"] != 7.9 {
		t.Fatalf("credits inventory=%v want 7.9", inventory["credits"])
	}
	if inventory["materials"] != 3.2 {
		t.Fatalf("materials inventory=%v want 3.2", inventory["materials"])
	}
}
