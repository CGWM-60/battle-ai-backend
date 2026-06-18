package repository

import (
	"testing"

	"cgwm/battle/internal/app/constants"
)

func TestResolveBattleQuestStatusFilterPublishedUsesStatusOnly(t *testing.T) {
	plan := resolveBattleQuestStatusFilter(constants.QuestStatusPublished, true, false)
	if !plan.ApplyStatus || plan.StatusValue != constants.QuestStatusPublished {
		t.Fatalf("expected status=published filter, got %+v", plan)
	}
	if plan.ApplyIsPublished {
		t.Fatalf("must not use is_published when column is absent: %+v", plan)
	}
}

func TestResolveBattleQuestStatusFilterPublishedUsesIsPublishedFallback(t *testing.T) {
	plan := resolveBattleQuestStatusFilter(constants.QuestStatusPublished, false, true)
	if plan.ApplyStatus || !plan.ApplyIsPublished {
		t.Fatalf("expected is_published fallback, got %+v", plan)
	}
}

func TestResolveBattleQuestStatusFilterPublishedSkipsWhenNoColumns(t *testing.T) {
	plan := resolveBattleQuestStatusFilter(constants.QuestStatusPublished, false, false)
	if plan.ApplyStatus || plan.ApplyIsPublished || plan.WarnMessage == "" {
		t.Fatalf("expected skipped filter with warning, got %+v", plan)
	}
}

func TestResolveBattleQuestStatusFilterDraftUsesStatus(t *testing.T) {
	plan := resolveBattleQuestStatusFilter("draft", true, false)
	if !plan.ApplyStatus || plan.StatusValue != "draft" {
		t.Fatalf("expected draft status filter, got %+v", plan)
	}
}

func TestResolveBattleQuestStatusFilterAllSkips(t *testing.T) {
	plan := resolveBattleQuestStatusFilter("all", true, true)
	if plan.ApplyStatus || plan.ApplyIsPublished || plan.WarnMessage != "" {
		t.Fatalf("expected no filter for all, got %+v", plan)
	}
}

func TestHasQuestBattleColumnNilDB(t *testing.T) {
	if hasQuestBattleColumn(nil, "status") {
		t.Fatal("nil db must not report columns")
	}
}
