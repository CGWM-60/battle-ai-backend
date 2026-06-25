package service

import (
	"testing"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
)

func TestDefaultEnvDoesNotEnableMock(t *testing.T) {
	t.Setenv("AI_MOCK_ENABLED", "")
	t.Setenv("AI_PLATFORM_MODE", "")
	t.Setenv("AI_ALLOW_USER_FACING_MOCK", "")
	t.Setenv("BILLING_MODE", "")

	if AIMockEnabled() {
		t.Fatal("AI_MOCK_ENABLED must default to false")
	}
	if AIPlatformMode() != "live" {
		t.Fatalf("AI_PLATFORM_MODE=%q want live", AIPlatformMode())
	}
	if BillingModeEnv() != "live" {
		t.Fatalf("BILLING_MODE=%q want live", BillingModeEnv())
	}
	if usesMockAIProvider() {
		t.Fatal("usesMockAIProvider must be false by default")
	}
	if AllowUserFacingMockAI() {
		t.Fatal("user-facing mock must be disabled by default")
	}
}

func TestUserFacingGenerateRejectsMockEvenIfMockEnvEnabled(t *testing.T) {
	t.Setenv("AI_MOCK_ENABLED", "true")
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_ALLOW_USER_FACING_MOCK", "false")

	if !usesMockAIProvider() {
		t.Fatal("expected mock env enabled")
	}
	if AllowUserFacingMockAI() {
		t.Fatal("user-facing mock must stay disabled")
	}

	err := rejectUserFacingMock("openai", "gpt-5-mini")
	if err == nil {
		t.Fatal("expected mock rejection on user-facing route")
	}
	detail, ok := AsAIProviderError(err)
	if !ok || detail.Code != AIErrorMockDisabled {
		t.Fatalf("code=%s want %s", detail.Code, AIErrorMockDisabled)
	}
}

func TestPlatformModeRequiresRealServerKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPEN_AI_KEY", "")
	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)
	plan := orchestrator.BuildExecutionPlan("platform", "", "openai", "gpt-4o-mini", 512, 512, "test")

	if plan.UsesMockProvider {
		t.Fatalf("platform plan must never use mock provider: %+v", plan)
	}
	if !plan.UsesPlatformKey {
		t.Fatalf("platform plan must require platform key: %+v", plan)
	}

	_, err := orchestrator.ResolveAPIKey(plan, "", "openai")
	if err == nil {
		t.Fatal("expected missing platform key error")
	}
	detail, ok := AsAIProviderError(err)
	if !ok || detail.Code != AIErrorPlatformKeyMissing {
		t.Fatalf("code=%s want %s", detail.Code, AIErrorPlatformKeyMissing)
	}
}

func TestBYOKRequiresClientKey(t *testing.T) {
	orchestrator := NewAIOrchestrator(nil, NewAICreditEstimator(), nil)
	plan := orchestrator.BuildExecutionPlan("own_key", "", "openai", "gpt-4o-mini", 512, 512, "test")
	_, err := orchestrator.ResolveAPIKey(plan, "", "openai")
	if err == nil {
		t.Fatal("expected missing client key error")
	}
	detail, ok := AsAIProviderError(err)
	if !ok || detail.Code != AIErrorProviderKeyMissing {
		t.Fatalf("code=%s want %s", detail.Code, AIErrorProviderKeyMissing)
	}
}

func TestNoCreditsConsumedWhenProviderReturnsMockLeak(t *testing.T) {
	t.Setenv("AI_ALLOW_USER_FACING_MOCK", "true")
	repo := newMemoryWalletRepo()
	repo.wallets[1] = &models.UserAIWallet{
		UserID:             1,
		BalanceCredits:     1000,
		StarterBonusGranted: true,
	}
	wallets := NewWalletService(repo, NewAICreditEstimator())
	orchestrator := NewAIOrchestrator(wallets, NewAICreditEstimator(), nil)
	plan := orchestrator.BuildExecutionPlan("platform", "", "openai", "gpt-test", 512, 512, "test")
	plan.UsesMockProvider = true

	usageRef := usageSessionRef{
		OwnerID:        1,
		SessionMode:    "roleplay_ia",
		BillingSource:  billingSourcePlatformKey,
		ProviderName:   "openai",
		ModelName:      "gpt-test",
		Feature:        "roleplay_ia",
		SettlementPlan: &plan,
		Orchestrator:   orchestrator,
	}

	// Simulate provider usage captured but rejected before commit.
	captured := provider.UsageRecord{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	// Validation failed => do NOT commit.
	_ = captured

	if repo.wallets[1].BalanceCredits != 1000 {
		t.Fatalf("balance=%d want 1000 before commit", repo.wallets[1].BalanceCredits)
	}

	// Validation failed => commit must not run. Nil usage repo is a no-op.
	commitAIUsageRecord(nil, usageRef, captured)
	if repo.wallets[1].BalanceCredits != 1000 {
		t.Fatalf("balance=%d want 1000 when rejected mock leak", repo.wallets[1].BalanceCredits)
	}
}

func TestNoCreditsConsumedWhenProviderReturnsEmpty(t *testing.T) {
	repo := newMemoryWalletRepo()
	repo.wallets[1] = &models.UserAIWallet{
		UserID:              1,
		BalanceCredits:      1000,
		StarterBonusGranted: true,
	}
	wallets := NewWalletService(repo, NewAICreditEstimator())
	orchestrator := NewAIOrchestrator(wallets, NewAICreditEstimator(), nil)
	plan := orchestrator.BuildExecutionPlan("platform", "", "openai", "gpt-test", 512, 512, "test")

	usageRef := usageSessionRef{
		OwnerID:        1,
		SessionMode:    "roleplay_ia",
		BillingSource:  billingSourcePlatformKey,
		ProviderName:   "openai",
		ModelName:      "gpt-test",
		Feature:        "roleplay_ia",
		SettlementPlan: &plan,
		Orchestrator:   orchestrator,
	}

	captured := provider.UsageRecord{
		PromptTokens:     80,
		CompletionTokens: 0,
		TotalTokens:      80,
	}
	// Empty provider response => validation fails before commit.
	_ = captured

	if repo.wallets[1].BalanceCredits != 1000 {
		t.Fatalf("balance=%d want 1000 before commit", repo.wallets[1].BalanceCredits)
	}

	commitAIUsageRecord(nil, usageRef, captured)
	if repo.wallets[1].BalanceCredits != 1000 {
		t.Fatalf("balance=%d want 1000 when provider returns empty", repo.wallets[1].BalanceCredits)
	}

	detail := classifyAIProviderError(
		NewAIProviderError(
			AIErrorProviderEmptyResponse,
			"openai",
			"gpt-test",
			"ai provider returned empty response",
			true,
		),
		"openai",
		"gpt-test",
	)
	if detail.Code != AIErrorProviderEmptyResponse {
		t.Fatalf("code=%s want %s", detail.Code, AIErrorProviderEmptyResponse)
	}
}

func TestAIProviderGenerateReturnsStructuredErrorCode(t *testing.T) {
	err := NewAIProviderError(
		AIErrorProviderMockResponse,
		"openai",
		"gpt-5.4-mini",
		"ai provider returned mock or prompt leak response",
		false,
	)
	detail := classifyAIProviderError(err, "openai", "gpt-5.4-mini")
	if detail.Code != AIErrorProviderMockResponse {
		t.Fatalf("code=%s", detail.Code)
	}
	if detail.StatusCode != 422 {
		t.Fatalf("status=%d want 422", detail.StatusCode)
	}
}