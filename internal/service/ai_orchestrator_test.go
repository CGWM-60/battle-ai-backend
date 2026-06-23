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

func TestBattleServiceResolveBattleProviderModeKeepsPlatformCreditsAuthoritative(t *testing.T) {
	service := &BattleService{}

	if mode := service.resolveBattleProviderMode("platform", "profile-key"); mode != AIOrchestratorModePlatform {
		t.Fatalf("mode=%s want platform when credits mode sends a stale profile key", mode)
	}
	if mode := service.resolveBattleProviderMode("platform", ""); mode != AIOrchestratorModePlatform {
		t.Fatalf("mode=%s want platform without profile key", mode)
	}
	if mode := service.resolveBattleProviderMode("", "profile-key"); mode != AIOrchestratorModeBYOK {
		t.Fatalf("mode=%s want byok for legacy payload without billing mode", mode)
	}
	if mode := service.resolveBattleProviderMode("own_key", "profile-key"); mode != AIOrchestratorModeBYOK {
		t.Fatalf("mode=%s want byok for explicit own_key", mode)
	}
}

func TestAIOrchestratorPlatformKeepsBillingWithMockProvider(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)

	if mode := orchestrator.ResolveMode("platform", ""); mode != AIOrchestratorModePlatform {
		t.Fatalf("mode=%s want platform billing", mode)
	}

	plan := orchestrator.BuildExecutionPlan("platform", "", "openai", "gpt-4o-mini", 4000, 4000, "test")
	if !plan.RequiresWallet {
		t.Fatalf("expected wallet requirement with platform billing: %+v", plan)
	}
	if !plan.UsesMockProvider {
		t.Fatalf("expected mock provider with AI_PLATFORM_MODE=mock: %+v", plan)
	}
	if plan.UsesPlatformKey {
		t.Fatalf("expected no platform key when mock provider is enabled: %+v", plan)
	}
}

func TestAIOrchestratorAuthorizePaymentRequired(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	wallets := NewWalletService(nil, NewAICreditEstimator())
	orchestrator := NewAIOrchestrator(wallets, NewAICreditEstimator(), nil)

	plan := orchestrator.BuildExecutionPlan("platform", "", "openai", "gpt-4o-mini", 4000, 4000, "test")
	err := orchestrator.Authorize(context.Background(), 1, plan)
	if err == nil {
		t.Fatalf("expected authorize error with nil wallet repo")
	}
}

func TestAIOrchestratorBYOKSkipsWallet(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	wallets := NewWalletService(nil, NewAICreditEstimator())
	orchestrator := NewAIOrchestrator(wallets, NewAICreditEstimator(), nil)

	plan := orchestrator.BuildExecutionPlan("own_key", "client-key", "openai", "gpt-4o-mini", 4000, 4000, "test")
	if err := orchestrator.Authorize(context.Background(), 1, plan); err != nil {
		t.Fatalf("byok should not require wallet: %v", err)
	}
	if plan.RequiresWallet {
		t.Fatalf("byok plan should not require wallet: %+v", plan)
	}
}
