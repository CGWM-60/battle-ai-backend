package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"
)

type AIProviderGenerationInput struct {
	OwnerID         uint
	ProviderName    string
	ModelName       string
	APIKey          string
	SystemPrompt    string
	Prompt          string
	MaxChars        int
	BillingMode     string
	FallbackEnabled bool
	Feature         string
	Operation       string
}

type AIProviderGenerationResult struct {
	ProviderName string
	ModelName    string
	Response     string
	Latency      time.Duration
	Timeout      time.Duration
}

type AIProviderGenerationService struct {
	usage        *repository.AIUsageRepository
	orchestrator *AIOrchestrator
}

func NewAIProviderGenerationService(usage *repository.AIUsageRepository, orchestrator *AIOrchestrator) *AIProviderGenerationService {
	return &AIProviderGenerationService{
		usage:        usage,
		orchestrator: orchestrator,
	}
}

func (s *AIProviderGenerationService) Generate(ctx context.Context, input AIProviderGenerationInput) (*AIProviderGenerationResult, error) {
	providerName := strings.TrimSpace(input.ProviderName)
	modelName := strings.TrimSpace(input.ModelName)
	prompt := strings.TrimSpace(input.Prompt)
	billingMode := strings.TrimSpace(input.BillingMode)

	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	providerName, modelName = resolveRolePlayProviderDefaults(billingMode, providerName, modelName)
	if providerName == "" || modelName == "" {
		return nil, fmt.Errorf("providerName and modelName are required")
	}
	if billingMode == "" && strings.TrimSpace(input.APIKey) == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	attempts := aiProviderGenerationAttempts(billingMode, providerName, modelName, input.FallbackEnabled)
	var lastErr error
	for index, attempt := range attempts {
		attemptInput := input
		attemptInput.ProviderName = attempt.providerName
		attemptInput.ModelName = attempt.modelName
		result, err := s.generateOnce(ctx, attemptInput)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !shouldTryNextAIProviderGenerationAttempt(billingMode, input.FallbackEnabled, index, len(attempts), err) {
			return nil, err
		}
	}

	return nil, lastErr
}

type aiProviderGenerationAttempt struct {
	providerName string
	modelName    string
}

func aiProviderGenerationAttempts(billingMode, providerName, modelName string, fallbackEnabled bool) []aiProviderGenerationAttempt {
	attempts := []aiProviderGenerationAttempt{{
		providerName: providerName,
		modelName:    modelName,
	}}
	if normalizeBillingMode(billingMode) != models.BillingModePlatform || !fallbackEnabled {
		return attempts
	}

	seen := map[string]bool{normalizeProviderName(providerName): true}
	for _, fallbackProvider := range []string{"mistral", "openai"} {
		normalized := normalizeProviderName(fallbackProvider)
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		attempts = append(attempts, aiProviderGenerationAttempt{
			providerName: fallbackProvider,
			modelName:    platformBattleModel(fallbackProvider),
		})
	}
	return attempts
}

func shouldTryNextAIProviderGenerationAttempt(billingMode string, fallbackEnabled bool, index int, total int, err error) bool {
	if normalizeBillingMode(billingMode) != models.BillingModePlatform || !fallbackEnabled || index >= total-1 {
		return false
	}
	if billingErr, ok := AsBillingError(err); ok {
		if strings.Contains(strings.ToLower(billingErr.Message), "insufficient credits") {
			return false
		}
	}
	return true
}

func (s *AIProviderGenerationService) generateOnce(ctx context.Context, input AIProviderGenerationInput) (*AIProviderGenerationResult, error) {
	providerName := strings.TrimSpace(input.ProviderName)
	modelName := strings.TrimSpace(input.ModelName)
	apiKey := strings.TrimSpace(input.APIKey)
	billingMode := strings.TrimSpace(input.BillingMode)
	prompt := strings.TrimSpace(input.Prompt)

	url, err := ProviderURL(providerName)
	if err != nil {
		return nil, fmt.Errorf("providerName invalide")
	}

	plan := AIExecutionPlan{BillingSource: billingSourceClientKey, ProviderName: providerName, ModelName: modelName}
	if s.orchestrator != nil {
		plan = s.orchestrator.BuildExecutionPlan(billingMode, apiKey, providerName, modelName, 512, 512, defaultString(input.Operation, "ai_provider_generate"))
		if err := s.orchestrator.Authorize(ctx, input.OwnerID, plan); err != nil {
			return nil, MapBillingError(err)
		}
		resolvedKey, keyErr := s.orchestrator.ResolveAPIKey(plan, apiKey, providerName)
		if keyErr != nil {
			return nil, MapBillingError(keyErr)
		}
		apiKey = resolvedKey
	} else if normalizeBillingMode(billingMode) == "own_key" || billingMode == "" {
		if apiKey == "" {
			return nil, fmt.Errorf("apiKey is required")
		}
	} else {
		return nil, PaymentRequiredError("platform ai orchestration unavailable", nil)
	}

	messages := make([]provider.ProviderMessage, 0, 2)
	if systemPrompt := strings.TrimSpace(input.SystemPrompt); systemPrompt != "" {
		messages = append(messages, provider.ProviderMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	messages = append(messages, provider.ProviderMessage{
		Role:    "user",
		Content: prompt,
	})

	timeout := AIProviderGenerationTimeout()
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseProvider := provider.NewsProvider(apiKey, url, modelName)
	if s.orchestrator != nil {
		baseProvider = s.orchestrator.AttachProvider(plan, baseProvider)
	}
	feature := defaultString(input.Feature, constants.ModeRolePlayIA)
	ai := attachUsageRecorder(s.usage, usageSessionRef{
		OwnerID:        input.OwnerID,
		SessionMode:    feature,
		BillingSource:  plan.BillingSource,
		ProviderName:   providerName,
		ModelName:      modelName,
		Feature:        feature,
		SettlementPlan: &plan,
		Orchestrator:   s.orchestrator,
	}, baseProvider)
	if ai != nil {
		ai.WithUsageMetadata(provider.UsageMetadata{
			Mode:      feature,
			Operation: defaultString(input.Operation, "ai_provider_generate"),
			Phase:     "generate",
			Round:     1,
			ActorName: "Narrateur",
		})
	}

	startedAt := time.Now()
	response, err := ai.Chat(callCtx, messages)
	latency := time.Since(startedAt)
	if err != nil {
		return nil, err
	}

	generated := strings.TrimSpace(response)
	if generated == "" {
		return nil, fmt.Errorf("ai provider returned empty response")
	}
	if looksLikeAIMockOrPromptLeak(generated) {
		log.Printf(
			"[AI_PROVIDER_REJECTED] reason=mock_or_prompt_leak provider=%s model=%s feature=%s operation=%s mockMode=%v allowUserFacingMock=%v",
			providerName,
			modelName,
			feature,
			input.Operation,
			usesMockAIProvider(),
			allowMockForUserFacingRoutes(),
		)
		if !allowMockForUserFacingRoutes() {
			return nil, fmt.Errorf("ai provider returned mock or prompt leak response")
		}
	}
	if input.MaxChars > 0 {
		generated = truncateRunes(generated, input.MaxChars)
	}

	return &AIProviderGenerationResult{
		ProviderName: normalizeProviderName(providerName),
		ModelName:    modelName,
		Response:     generated,
		Latency:      latency,
		Timeout:      timeout,
	}, nil
}

func AIProviderGenerationTimeout() time.Duration {
	value, err := strconv.Atoi(envString("AI_PROVIDER_GENERATION_TIMEOUT_SECONDS", "", "60"))
	if err != nil || value <= 0 {
		return 60 * time.Second
	}

	return time.Duration(value) * time.Second
}

func looksLikeAIMockOrPromptLeak(value string) bool {
	text := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(text, "mock:") ||
		strings.Contains(text, "tu es le maître du jeu ia") ||
		strings.Contains(text, "réponds uniquement avec ce json") ||
		strings.Contains(text, "règles rpg:") ||
		strings.Contains(text, "prompt source:")
}

func truncateRunes(value string, maxLength int) string {
	if maxLength <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
}
