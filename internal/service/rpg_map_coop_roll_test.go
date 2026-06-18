package service

import (
	"testing"
)

func TestNormalizeRollDifficulty(t *testing.T) {
	if got := normalizeRollDifficulty(0); got != 12 {
		t.Fatalf("expected default 12, got %d", got)
	}
	if got := normalizeRollDifficulty(3); got != 12 {
		t.Fatalf("expected min clamp to 12, got %d", got)
	}
	if got := normalizeRollDifficulty(13); got != 13 {
		t.Fatalf("expected 13, got %d", got)
	}
	if got := normalizeRollDifficulty(40); got != 30 {
		t.Fatalf("expected max 30, got %d", got)
	}
}

func TestComputeRollResultBounds(t *testing.T) {
	for i := 0; i < 50; i++ {
		result := computeRollResult("disarm_trap", "dexterity", "Mécanique", 13)
		if result.Roll < 1 || result.Roll > 20 {
			t.Fatalf("roll out of bounds: %d", result.Roll)
		}
		if result.Total != result.Roll+result.Modifier {
			t.Fatalf("total mismatch: roll=%d modifier=%d total=%d", result.Roll, result.Modifier, result.Total)
		}
		if result.Success != (result.Total >= result.Difficulty) {
			t.Fatalf("success incoherent: total=%d difficulty=%d success=%v", result.Total, result.Difficulty, result.Success)
		}
		if result.ActionID != "disarm_trap" {
			t.Fatalf("unexpected action id: %s", result.ActionID)
		}
	}
}

func TestComputeRollResultDexterityModifier(t *testing.T) {
	result := computeRollResult("test", "dexterity", "", 12)
	if result.Modifier != 3 {
		t.Fatalf("expected dexterity modifier 3, got %d", result.Modifier)
	}
}