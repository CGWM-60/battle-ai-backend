package service

import (
	"os"
	"testing"
)

func TestResolveRolePlayProviderDefaultsUsesPlatformDefaults(t *testing.T) {
	t.Setenv("AI_DEFAULT_PROVIDER", "openai")
	t.Setenv("AI_DEFAULT_MODEL", "gpt-5-mini")

	provider, model := resolveRolePlayProviderDefaults("platform", "", "")
	if provider != "openai" {
		t.Fatalf("provider=%q want openai", provider)
	}
	if model != "gpt-5-mini" {
		t.Fatalf("model=%q want gpt-5-mini", model)
	}
}

func TestResolveRolePlayProviderDefaultsKeepsExplicitValues(t *testing.T) {
	provider, model := resolveRolePlayProviderDefaults("platform", "claude", "claude-sonnet")
	if provider != "claude" {
		t.Fatalf("provider=%q want claude", provider)
	}
	if model != "claude-sonnet" {
		t.Fatalf("model=%q want claude-sonnet", model)
	}
}

func TestSnapshotHasClientOpeningDetectsNarration(t *testing.T) {
	if !snapshotHasClientOpening(map[string]any{
		"openingNarration": "La brume recouvre l'entrée.",
	}) {
		t.Fatal("expected opening narration to be detected")
	}
}

func TestSnapshotHasClientOpeningDetectsSceneDialogues(t *testing.T) {
	if !snapshotHasClientOpening(map[string]any{
		"sceneDialogues": []any{
			map[string]any{"speakerName": "Lyra", "content": "Avance."},
		},
	}) {
		t.Fatal("expected scene dialogues to be detected")
	}
}

func TestSnapshotHasClientOpeningFalseWhenEmpty(t *testing.T) {
	if snapshotHasClientOpening(map[string]any{}) {
		t.Fatal("expected empty snapshot to skip appendInitialNarration")
	}
}

func TestDefaultRolePlayProviderFallback(t *testing.T) {
	_ = os.Unsetenv("AI_DEFAULT_PROVIDER")
	if defaultRolePlayProvider() != "openai" {
		t.Fatalf("unexpected default provider")
	}
}

func TestValidateRolePlaySnapshotQuestIDAllowsMatch(t *testing.T) {
	err := validateRolePlaySnapshotQuestID(map[string]any{"questId": uint(259)}, 259)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRolePlaySnapshotQuestIDRejectsMismatch(t *testing.T) {
	err := validateRolePlaySnapshotQuestID(map[string]any{"questId": uint(123)}, 259)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestEnsureRolePlaySnapshotQuestIDInjectsMissingValue(t *testing.T) {
	snapshot := map[string]any{}
	ensureRolePlaySnapshotQuestID(snapshot, 259)
	if uintFromSnapshot(snapshot, "questId") != 259 {
		t.Fatalf("questId=%v want 259", snapshot["questId"])
	}
}

func TestStampRolePlaySnapshotFromTemplate(t *testing.T) {
	snapshot := map[string]any{}
	stampRolePlaySnapshotFromTemplate(snapshot, 259, "Brume", "brume-cryptes")
	if uintFromSnapshot(snapshot, "questId") != 259 {
		t.Fatalf("questId=%v", snapshot["questId"])
	}
	if snapshot["serverQuestTitle"] != "Brume" {
		t.Fatalf("serverQuestTitle=%v", snapshot["serverQuestTitle"])
	}
	if snapshot["serverQuestSlug"] != "brume-cryptes" {
		t.Fatalf("serverQuestSlug=%v", snapshot["serverQuestSlug"])
	}
}

func TestResolveRolePlayProviderDefaultsOwnKeyKeepsEmptyWithoutPlatform(t *testing.T) {
	provider, model := resolveRolePlayProviderDefaults("own_key", "", "")
	if provider != "" || model != "" {
		t.Fatalf("provider=%q model=%q want empty values for BYOK without defaults", provider, model)
	}
}