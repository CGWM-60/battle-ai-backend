package service

import (
	"context"
	"os"
	"testing"
)

func TestPlatformCreditsAuthorizeWithSufficientWallet(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	t.Setenv("AI_DEFAULT_PROVIDER", "openai")
	t.Setenv("AI_DEFAULT_MODEL", "gpt-5-mini")

	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)
	plan := orchestrator.BuildExecutionPlan("platformCredits", "", "openai", "gpt-5-mini", 4000, 4000, "roleplay:test")
	if plan.Mode != AIOrchestratorModePlatform {
		t.Fatalf("mode=%s want platform", plan.Mode)
	}
	if !plan.RequiresWallet {
		t.Fatalf("platform plan should require wallet settlement")
	}
}

func TestOwnKeyRequiresClientApiKey(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)
	_, err := orchestrator.ResolveAPIKey(
		orchestrator.BuildExecutionPlan("own_key", "", "openai", "gpt-5-mini", 512, 512, "test"),
		"",
		"openai",
	)
	if err == nil {
		t.Fatal("expected own_key without apiKey to fail")
	}
}

func TestOwnKeyWithApiKeySkipsWalletRequirement(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)
	plan := orchestrator.BuildExecutionPlan("own_key", "sk-test", "openai", "gpt-5-mini", 512, 512, "test")
	if err := orchestrator.Authorize(context.Background(), 1, plan); err != nil {
		t.Fatalf("byok authorize should not require wallet: %v", err)
	}
	if plan.RequiresWallet {
		t.Fatalf("byok should not require wallet: %+v", plan)
	}
}

func TestResolveRolePlayProviderDefaultsForPlatformCredits(t *testing.T) {
	t.Setenv("AI_DEFAULT_PROVIDER", "openai")
	t.Setenv("AI_DEFAULT_MODEL", "gpt-5-mini")
	provider, model := resolveRolePlayProviderDefaults("platformCredits", "", "")
	if provider != "openai" || model != "gpt-5-mini" {
		t.Fatalf("provider=%q model=%q", provider, model)
	}
}

func TestSnapshotHasClientOpeningSkipsDoubleGeneration(t *testing.T) {
	if !snapshotHasClientOpening(map[string]any{
		"openingNarration": "La brume recouvre l'entrée.",
		"questId":          uint(259),
	}) {
		t.Fatal("expected client opening to skip appendInitialNarration")
	}
}

func TestOwnKeyKeepsEmptyProviderWithoutPlatformDefaults(t *testing.T) {
	_ = os.Unsetenv("AI_DEFAULT_PROVIDER")
	provider, model := resolveRolePlayProviderDefaults("own_key", "", "")
	if provider != "" || model != "" {
		t.Fatalf("provider=%q model=%q want empty for BYOK without explicit values", provider, model)
	}
}