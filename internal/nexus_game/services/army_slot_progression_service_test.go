package services

import (
	"testing"

	"cgwm/battle/internal/nexus_game/models"
)

func TestArmySlotProgressionDefenseStartsWithThreeBaseSlots(t *testing.T) {
	rules := DefaultArmyProgressionRules()
	base := make([]models.ArmyFormationProgressionRule, 0)
	for _, rule := range rules {
		if rule.FormationType == "defense" && rule.RuleType == armyRuleSlotUnlock && rule.UnlockBuildingCode == "" && rule.UnlockResearchCode == "" {
			base = append(base, rule)
		}
	}
	if len(base) != 3 {
		t.Fatalf("defense base slots = %d, want 3: %#v", len(base), base)
	}
	if base[0].SlotType != "infantry_slot" || base[1].SlotType != "drone_slot" || base[2].SlotType != "support_slot" {
		t.Fatalf("unexpected defense base slots: %#v", base)
	}
}

func TestArmySlotProgressionDefenseHeavyRequiresWallLevel3(t *testing.T) {
	rule := findProgressionRule(t, "defense_wall_heavy_6")
	if rule.FormationType != "defense" || rule.SlotType != "heavy_slot" {
		t.Fatalf("unexpected heavy defense rule: %#v", rule)
	}
	if rule.UnlockBuildingCode != "building_holo_wall" || rule.UnlockBuildingLevel != 3 {
		t.Fatalf("defense heavy unlock = %s/%d, want building_holo_wall/3", rule.UnlockBuildingCode, rule.UnlockBuildingLevel)
	}
}

func TestArmySlotProgressionCapacityUpgradeRulesExist(t *testing.T) {
	rule := findProgressionRule(t, "defense_urban_defense_1_bonus")
	if rule.RuleType != armyRuleCapacityBonus || rule.TargetSlotType != "all" || rule.CapacityBonus != 5 {
		t.Fatalf("unexpected urban defense I bonus: %#v", rule)
	}
	rule = findProgressionRule(t, "attack_mechanized_assault_1_bonus")
	if rule.RuleType != armyRuleCapacityBonus || rule.TargetSlotType != "heavy_slot" || rule.CapacityBonus != 10 {
		t.Fatalf("unexpected mechanized assault bonus: %#v", rule)
	}
}

func TestArmySlotProgressionRulesDoNotDuplicateSourceCodes(t *testing.T) {
	seen := map[string]bool{}
	for _, rule := range DefaultArmyProgressionRules() {
		if rule.SourceCode == "" {
			t.Fatalf("rule without source code: %#v", rule)
		}
		if seen[rule.SourceCode] {
			t.Fatalf("duplicate source code: %s", rule.SourceCode)
		}
		seen[rule.SourceCode] = true
	}
}

func TestArmySlotProgressionFormationDoesNotExceedMaxSlots(t *testing.T) {
	countByFormation := map[string]int{}
	for _, rule := range DefaultArmyProgressionRules() {
		if rule.RuleType == armyRuleSlotUnlock {
			countByFormation[rule.FormationType]++
		}
	}
	for formation, count := range countByFormation {
		if count > 12 {
			t.Fatalf("%s has %d slot rules, want <= 12", formation, count)
		}
	}
}

func TestArmySlotEffectiveCapacityUsesCurrentThenLegacy(t *testing.T) {
	if got := effectiveSlotCapacity(models.ArmyFormationSlot{CurrentCapacity: 12, Capacity: 5, BaseCapacity: 3}); got != 12 {
		t.Fatalf("effective current capacity = %d, want 12", got)
	}
	if got := effectiveSlotCapacity(models.ArmyFormationSlot{Capacity: 5, BaseCapacity: 3}); got != 5 {
		t.Fatalf("effective legacy capacity = %d, want 5", got)
	}
	if got := effectiveSlotCapacity(models.ArmyFormationSlot{BaseCapacity: 3}); got != 3 {
		t.Fatalf("effective base capacity = %d, want 3", got)
	}
}

func TestArmySlotAllCapacityBonusDoesNotApplyToCommander(t *testing.T) {
	rule := models.ArmyFormationProgressionRule{TargetSlotType: "all", CapacityBonus: 10}
	if capacityBonusApplies(rule, "commander_slot") {
		t.Fatal("all capacity bonus should not apply to commander_slot")
	}
	if !capacityBonusApplies(rule, "drone_slot") {
		t.Fatal("all capacity bonus should apply to normal slots")
	}
}

func TestArmySlotAntiSabotageCompatibility(t *testing.T) {
	if !slotAllowsUnit("anti_sabotage_slot", &models.UnitDefinition{Type: "drone"}) {
		t.Fatal("anti_sabotage_slot should accept drones")
	}
	if !slotAllowsUnit("anti_sabotage_slot", &models.UnitDefinition{Type: "special"}) {
		t.Fatal("anti_sabotage_slot should accept special units")
	}
	if slotAllowsUnit("anti_sabotage_slot", &models.UnitDefinition{Type: "mecha"}) {
		t.Fatal("anti_sabotage_slot should reject mecha")
	}
}

func findProgressionRule(t *testing.T, sourceCode string) models.ArmyFormationProgressionRule {
	t.Helper()
	for _, rule := range DefaultArmyProgressionRules() {
		if rule.SourceCode == sourceCode {
			return rule
		}
	}
	t.Fatalf("progression rule %s not found", sourceCode)
	return models.ArmyFormationProgressionRule{}
}
