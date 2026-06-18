package service

import (
	"context"
	"testing"
)

func TestAIOrchestratorResolveMode(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "live")
	t.Setenv("AI_MOCK_ENABLED", "false")
	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)

	if mode := orchestrator.ResolveMode("own_key", "client-key"); mode != AIOrchestratorModeBYOK {
		t.Fatalf("mode=%s want byok", mode)
	}
	if mode := orchestrator.ResolveMode("platform", ""); mode != AIOrchestratorModePlatform {
		t.Fatalf("mode=%s want platform", mode)
	}
	if mode := orchestrator.ResolveMode("", "client-key"); mode != AIOrchestratorModeBYOK {
		t.Fatalf("mode=%s want byok fallback", mode)
	}
}

func TestAIOrchestratorResolveModeMockDefault(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)

	if mode := orchestrator.ResolveMode("platform", ""); mode != AIOrchestratorModeMock {
		t.Fatalf("mode=%s want mock", mode)
	}
}

func TestAIOrchestratorAuthorizePaymentRequired(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "live")
	t.Setenv("AI_MOCK_ENABLED", "false")
	wallets := NewWalletService(nil, NewAICreditEstimator())
	orchestrator := NewAIOrchestrator(wallets, NewAICreditEstimator(), nil)

	plan := orchestrator.BuildExecutionPlan("platform", "", "openai", "gpt-4o-mini", 4000, 4000, "test")
	err := orchestrator.Authorize(context.Background(), 1, plan)
	if err == nil {
		t.Fatalf("expected authorize error with nil wallet repo")
	}
}