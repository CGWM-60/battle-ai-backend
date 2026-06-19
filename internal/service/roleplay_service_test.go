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