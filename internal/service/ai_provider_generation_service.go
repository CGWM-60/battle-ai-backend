package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
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
	apiKey := strings.TrimSpace(input.APIKey)
	prompt := strings.TrimSpace(input.Prompt)
	billingMode := strings.TrimSpace(input.BillingMode)

	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	providerName, modelName = resolveRolePlayProviderDefaults(billingMode, providerName, modelName)
	if providerName == "" || modelName == "" {
		return nil, fmt.Errorf("providerName and modelName are required")
	}
	if billingMode == "" && apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

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
