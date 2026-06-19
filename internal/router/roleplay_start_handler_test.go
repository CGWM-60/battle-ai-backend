package router

import (
	"testing"

	"cgwm/battle/internal/service"
)

func TestRolePlayStartSnapshotQuestIDMismatch(t *testing.T) {
	snapshot := map[string]any{"questId": uint(123)}
	snapshotQuestID := service.UintFromSnapshot(snapshot, "questId")
	pathQuestID := uint(259)

	if snapshotQuestID == 0 || snapshotQuestID == pathQuestID {
		t.Fatal("expected mismatch setup")
	}
	if snapshotQuestID == pathQuestID {
		t.Fatal("snapshot should not match path quest")
	}
}

func TestRolePlayStartSnapshotQuestIDMatch(t *testing.T) {
	snapshot := map[string]any{"questId": uint(259)}
	snapshotQuestID := service.UintFromSnapshot(snapshot, "questId")
	if snapshotQuestID != 259 {
		t.Fatalf("snapshotQuestID=%d want 259", snapshotQuestID)
	}
}

func TestRolePlayStartSnapshotQuestIDInjection(t *testing.T) {
	snapshot := map[string]any{}
	service.EnsureRolePlaySnapshotQuestID(snapshot, 259)
	if service.UintFromSnapshot(snapshot, "questId") != 259 {
		t.Fatalf("questId=%v want 259", snapshot["questId"])
	}
}

func TestRolePlayStartProviderDefaultsForPlatformCredits(t *testing.T) {
	t.Setenv("AI_DEFAULT_PROVIDER", "openai")
	t.Setenv("AI_DEFAULT_MODEL", "gpt-5-mini")

	provider, model := service.ResolveRolePlayProviderDefaults("platformCredits", "", "")
	if provider != "openai" {
		t.Fatalf("provider=%q want openai", provider)
	}
	if model != "gpt-5-mini" {
		t.Fatalf("model=%q want gpt-5-mini", model)
	}
}