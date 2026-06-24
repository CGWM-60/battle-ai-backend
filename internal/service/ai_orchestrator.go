package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
)

// AIOrchestratorMode = source de facturation / execution IA.
type AIOrchestratorMode string

const (
	AIOrchestratorModeBYOK     AIOrchestratorMode = "byok"
	AIOrchestratorModePlatform AIOrchestratorMode = "platform"
	AIOrchestratorModeMock     AIOrchestratorMode = "mock"
)

// AIExecutionPlan = decision d'orchestration avant un appel IA.
type AIExecutionPlan struct {
	Mode             AIOrchestratorMode `json:"mode"`
	BillingSource    string             `json:"billingSource"`
	RequiresWallet   bool               `json:"requiresWallet"`
	UsesClientKey    bool               `json:"usesClientKey"`
	UsesPlatformKey  bool               `json:"usesPlatformKey"`
	UsesMockProvider bool               `json:"usesMockProvider"`
	EstimatedCredits int64              `json:"estimatedCredits"`
	EstimatedTokens  int                `json:"estimatedTokens"`
	IdempotencyKey   string             `json:"idempotencyKey"`
	ProviderName     string             `json:"providerName"`
	ModelName        string             `json:"modelName"`
}

// AIOrchestrator route les appels IA selon BYOK / platform / mock.
type AIOrchestrator struct {
	wallets      *WalletService
	estimator    *AICreditEstimator
	entitlements *EntitlementService
}

func NewAIOrchestrator(wallets *WalletService, estimator *AICreditEstimator, entitlements *EntitlementService) *AIOrchestrator {
	if estimator == nil {
		estimator = NewAICreditEstimator()
	}
	return &AIOrchestrator{
		wallets:      wallets,
		estimator:    estimator,
		entitlements: entitlements,
	}
}

func usesMockAIProvider() bool {
	return AIMockEnabled() && strings.EqualFold(AIPlatformMode(), "mock")
}

func (o *AIOrchestrator) ResolveMode(requestedMode string, clientAPIKey string) AIOrchestratorMode {
	if strings.EqualFold(strings.TrimSpace(requestedMode), string(AIOrchestratorModeMock)) {
		return AIOrchestratorModeMock
	}
	if strings.TrimSpace(clientAPIKey) != "" {
		normalized := normalizeBillingMode(requestedMode)
		if normalized == models.BillingModeOwnKey || strings.TrimSpace(requestedMode) == "" {
			return AIOrchestratorModeBYOK
		}
	}
	normalized := normalizeBillingMode(requestedMode)
	if normalized == models.BillingModeOwnKey {
		return AIOrchestratorModeBYOK
	}
	return AIOrchestratorModePlatform
}

func (o *AIOrchestrator) BuildExecutionPlan(
	requestedMode string,
	clientAPIKey string,
	providerName string,
	modelName string,
	estimatedPromptTokens int,
	estimatedCompletionTokens int,
	idempotencyKey string,
) AIExecutionPlan {
	mode := o.ResolveMode(requestedMode, clientAPIKey)
	estimate := o.estimator.EstimateFromTokens(estimatedPromptTokens, estimatedCompletionTokens)

	plan := AIExecutionPlan{
		Mode:             mode,
		EstimatedCredits: estimate.NexusCoins,
		EstimatedTokens:  estimate.TotalTokens,
		IdempotencyKey:   idempotencyKey,
		ProviderName:     providerName,
		ModelName:        modelName,
	}

	switch mode {
	case AIOrchestratorModeBYOK:
		plan.BillingSource = billingSourceClientKey
		plan.UsesClientKey = true
	case AIOrchestratorModeMock:
		plan.BillingSource = "mock"
		plan.UsesMockProvider = AllowUserFacingMockAI()
	default:
		plan.BillingSource = billingSourcePlatformKey
		plan.RequiresWallet = estimate.NexusCoins > 0
		plan.UsesPlatformKey = true
		plan.UsesMockProvider = false
	}
	return plan
}

func (o *AIOrchestrator) Authorize(ctx context.Context, userID uint, plan AIExecutionPlan) error {
	if o == nil {
		return fmt.Errorf("ai orchestrator unavailable")
	}
	if userID == 0 {
		return fmt.Errorf("user id is required")
	}
	switch plan.Mode {
	case AIOrchestratorModeBYOK, AIOrchestratorModeMock:
		return nil
	default:
		if o.wallets == nil {
			return fmt.Errorf("wallet service unavailable")
		}
		if plan.EstimatedCredits <= 0 {
			return nil
		}
		snapshot, err := o.wallets.GetOrCreateWithStarterBonus(ctx, userID)
		if err != nil {
			return err
		}
		if snapshot.BalanceCredits < plan.EstimatedCredits {
			return PaymentRequiredError("insufficient credits", map[string]any{
				"required": plan.EstimatedCredits,
				"balance":  snapshot.BalanceCredits,
			})
		}
		return nil
	}
}

func (o *AIOrchestrator) SettleUsage(
	ctx context.Context,
	userID uint,
	plan AIExecutionPlan,
	promptTokens int,
	completionTokens int,
	feature string,
	referenceID string,
) (AICreditEstimate, error) {
	estimate := o.estimator.EstimateFromTokens(promptTokens, completionTokens)
	if o == nil || o.wallets == nil || plan.Mode != AIOrchestratorModePlatform || estimate.NexusCoins <= 0 {
		return estimate, nil
	}
	idempotencyKey := plan.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("ai_usage:%d:%d", userID, estimate.TotalTokens)
	}
	if referenceID != "" {
		idempotencyKey = referenceID
	}
	_, settledEstimate, err := o.wallets.ConsumeForTokens(
		ctx,
		userID,
		promptTokens,
		completionTokens,
		idempotencyKey,
		feature,
		"ai usage settlement",
	)
	if err != nil {
		return estimate, MapBillingError(err)
	}
	return settledEstimate, nil
}

func (o *AIOrchestrator) ResolveAPIKey(plan AIExecutionPlan, clientAPIKey string, providerName string) (string, error) {
	switch plan.Mode {
	case AIOrchestratorModeBYOK:
		key := strings.TrimSpace(clientAPIKey)
		if key == "" {
			return "", NewAIProviderError(
				AIErrorProviderKeyMissing,
				providerName,
				plan.ModelName,
				"client api key required for own_key billing mode",
				false,
			)
		}
		return key, nil
	case AIOrchestratorModeMock:
		if !AllowUserFacingMockAI() {
			return "", NewAIProviderError(
				AIErrorMockDisabled,
				providerName,
				plan.ModelName,
				"mock ai is disabled on user-facing routes",
				false,
			)
		}
		return "", nil
	default:
		if plan.UsesMockProvider {
			return "", NewAIProviderError(
				AIErrorMockDisabled,
				providerName,
				plan.ModelName,
				"mock ai is disabled on user-facing routes",
				false,
			)
		}
		key := strings.TrimSpace(platformAPIKey(providerName))
		if key == "" {
			return "", NewAIProviderError(
				AIErrorPlatformKeyMissing,
				providerName,
				plan.ModelName,
				"platform provider key is not configured",
				false,
			)
		}
		return key, nil
	}
}

func (o *AIOrchestrator) AttachProvider(plan AIExecutionPlan, base *provider.Provider) *provider.Provider {
	if base == nil {
		return nil
	}
	if (plan.Mode == AIOrchestratorModeMock || plan.UsesMockProvider) && AllowUserFacingMockAI() {
		return provider.NewMockProvider(
			defaultString(plan.ProviderName, "mock"),
			defaultString(plan.ModelName, "mock-model"),
		)
	}
	return base
}

func platformAPIKey(providerName string) string {
	switch normalizeProviderName(providerName) {
	case "mistral":
		return firstNonEmptyEnv("MISTRAL_AI_KEY", "MISTRAL_API_KEY")
	case "openai", "openapi", "open_api":
		return firstNonEmptyEnv("OPEN_AI_KEY", "OPENAI_API_KEY")
	case "openrouter", "open_router":
		return firstNonEmptyEnv("OPENROUTER_API_KEY", "OPEN_ROUTER_API_KEY")
	case "xia", "xai", "x-ai":
		return firstNonEmptyEnv("XAI_API_KEY", "X_AI_KEY")
	case "claude", "anthropic":
		return firstNonEmptyEnv("ANTHROPIC_API_KEY", "CLAUDE_API_KEY")
	case "gemini", "google", "google_ai", "google-ai":
		return firstNonEmptyEnv("GEMINI_API_KEY", "GOOGLE_AI_API_KEY")
	default:
		return firstNonEmptyEnv("OPEN_AI_KEY", "OPENAI_API_KEY")
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
