package services

import (
	"testing"

	"cgwm/battle/internal/nexus_game/models"
)

func TestArmyRulesCapacityCostByUnitType(t *testing.T) {
	tests := []struct {
		name string
		def  models.UnitDefinition
		want int
	}{
		{name: "infantry", def: models.UnitDefinition{Type: "infantry"}, want: 1},
		{name: "drone", def: models.UnitDefinition{Type: "drone"}, want: 2},
		{name: "support", def: models.UnitDefinition{Type: "support"}, want: 3},
		{name: "mecha", def: models.UnitDefinition{Type: "mecha"}, want: 8},
		{name: "titan", def: models.UnitDefinition{Type: "mecha", ContentID: "unit_titan_nexus"}, want: 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := capacityCostForUnit(&tt.def); got != tt.want {
				t.Fatalf("capacityCostForUnit = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestArmyRulesTrainingCostUsesExpectedResources(t *testing.T) {
	drone := trainingCostForUnit(&models.UnitDefinition{Type: "drone", UpkeepBase: 2}, 3)
	for _, key := range []string{"components", "energy", "data", "metal"} {
		if drone[key] <= 0 {
			t.Fatalf("drone training cost missing %s: %#v", key, drone)
		}
	}

	infantry := trainingCostForUnit(&models.UnitDefinition{Type: "infantry", UpkeepBase: 1}, 2)
	for _, key := range []string{"food", "energy", "metal"} {
		if infantry[key] <= 0 {
			t.Fatalf("infantry training cost missing %s: %#v", key, infantry)
		}
	}
	if infantry["data"] != 0 {
		t.Fatalf("infantry should not require data by default: %#v", infantry)
	}
}

func TestArmyRulesSlotCompatibility(t *testing.T) {
	if !slotAllowsUnit("drone_slot", &models.UnitDefinition{Type: "drone", ContentID: "unit_drone_sentinelle"}) {
		t.Fatal("drone_slot should accept drone units")
	}
	if slotAllowsUnit("infantry_slot", &models.UnitDefinition{Type: "mecha", ContentID: "unit_mecha_leger"}) {
		t.Fatal("infantry_slot should reject mecha units")
	}
	if !slotAllowsUnit("scout_slot", &models.UnitDefinition{Type: "infantry", ContentID: "unit_eclaireur_cybernetique"}) {
		t.Fatal("scout_slot should accept scout infantry")
	}
}

func TestArmyRulesDefaultFormationSlots(t *testing.T) {
	defense := defaultSlotsForFormation("defense")
	if len(defense) != 3 {
		t.Fatalf("defense slots = %d, want 3", len(defense))
	}
	if defense[0].Type != "infantry_slot" || defense[1].Type != "drone_slot" || defense[2].Type != "support_slot" {
		t.Fatalf("unexpected defense slots: %#v", defense)
	}

	guild := defaultSlotsForFormation("guild_raid")
	if len(guild) != 4 {
		t.Fatalf("guild slots = %d, want 4", len(guild))
	}
	if guild[3].Type != "heavy_slot" {
		t.Fatalf("guild raid should expose heavy slot: %#v", guild)
	}
}
