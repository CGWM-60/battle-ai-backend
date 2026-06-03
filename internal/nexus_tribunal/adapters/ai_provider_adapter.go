package adapters

import (
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/service"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type AIProviderAdapter struct {
	envAPIKey func(providerType string) string
	timeout   time.Duration
}

type GenerateRequest struct {
	ProviderType  string
	Model         string
	APIKey        string
	LocalEndpoint string
	SystemPrompt  string
	Prompt        string
}

type GenerateResponse struct {
	Text      string
	LatencyMs int64
}

func NewAIProviderAdapter(envAPIKey func(providerType string) string) AIProviderAdapter {
	return AIProviderAdapter{
		envAPIKey: envAPIKey,
		timeout:   45 * time.Second,
	}
}

func (a AIProviderAdapter) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	cfg, err := a.resolve(req)
	if err != nil {
		return GenerateResponse{}, err
	}
	prompt := defaultAdapterText(req.Prompt, "Reponds uniquement OK")
	startedAt := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	messages := make([]provider.ProviderMessage, 0, 2)
	if system := strings.TrimSpace(req.SystemPrompt); system != "" {
		messages = append(messages, provider.ProviderMessage{Role: "system", Content: system})
	}
	messages = append(messages, provider.ProviderMessage{Role: "user", Content: prompt})

	client := provider.NewsProvider(cfg.apiKey, cfg.chatURL, cfg.model)
	output, err := client.Chat(callCtx, messages)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("provider indisponible: %w", err)
	}
	return GenerateResponse{
		Text:      strings.TrimSpace(output),
		LatencyMs: time.Since(startedAt).Milliseconds(),
	}, nil
}

func (a AIProviderAdapter) Stream(ctx context.Context, req GenerateRequest, onChunk func(chunk string)) (GenerateResponse, error) {
	cfg, err := a.resolve(req)
	if err != nil {
		return GenerateResponse{}, err
	}
	prompt := defaultAdapterText(req.Prompt, "Reponds uniquement OK")
	startedAt := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	messages := make([]provider.ProviderMessage, 0, 2)
	if system := strings.TrimSpace(req.SystemPrompt); system != "" {
		messages = append(messages, provider.ProviderMessage{Role: "system", Content: system})
	}
	messages = append(messages, provider.ProviderMessage{Role: "user", Content: prompt})

	client := provider.NewsProvider(cfg.apiKey, cfg.chatURL, cfg.model)
	output, err := client.ChatStream(callCtx, messages, onChunk)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("provider stream indisponible: %w", err)
	}
	return GenerateResponse{
		Text:      strings.TrimSpace(output),
		LatencyMs: time.Since(startedAt).Milliseconds(),
	}, nil
}

type resolvedAIProvider struct {
	providerType string
	model        string
	apiKey       string
	chatURL      string
	isLocal      bool
}

func (a AIProviderAdapter) resolve(req GenerateRequest) (resolvedAIProvider, error) {
	providerType := NormalizeProviderType(req.ProviderType)
	model := defaultAdapterText(req.Model, DefaultModelForProvider(providerType))
	apiKey := strings.TrimSpace(req.APIKey)

	if IsLocalProvider(providerType) {
		chatURL, err := LocalChatCompletionsURL(req.LocalEndpoint)
		if err != nil {
			return resolvedAIProvider{}, err
		}
		if apiKey == "" {
			apiKey = "local-no-key"
		}
		return resolvedAIProvider{
			providerType: providerType,
			model:        model,
			apiKey:       apiKey,
			chatURL:      chatURL,
			isLocal:      true,
		}, nil
	}

	chatURL, err := service.ProviderURL(providerType)
	if err != nil {
		return resolvedAIProvider{}, fmt.Errorf("provider %s non supporte", providerType)
	}
	if apiKey == "" && a.envAPIKey != nil {
		apiKey = strings.TrimSpace(a.envAPIKey(providerType))
	}
	if apiKey == "" {
		return resolvedAIProvider{}, fmt.Errorf("aucune cle disponible pour %s", providerType)
	}
	return resolvedAIProvider{
		providerType: providerType,
		model:        model,
		apiKey:       apiKey,
		chatURL:      chatURL,
		isLocal:      false,
	}, nil
}

func NormalizeProviderType(providerType string) string {
	return strings.ToLower(strings.TrimSpace(providerType))
}

func IsLocalProvider(providerType string) bool {
	switch NormalizeProviderType(providerType) {
	case "ollama", "lmstudio", "lm_studio", "local", "custom":
		return true
	default:
		return false
	}
}

func LocalChatCompletionsURL(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("localEndpoint est obligatoire pour un provider local")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("localEndpoint invalide")
	}
	clean := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(clean, "/chat/completions") {
		return clean, nil
	}
	if strings.HasSuffix(clean, "/v1") {
		return clean + "/chat/completions", nil
	}
	return clean + "/v1/chat/completions", nil
}

func DefaultModelForProvider(providerType string) string {
	switch NormalizeProviderType(providerType) {
	case "mistral":
		return "mistral-large-latest"
	case "openrouter", "open_router":
		return "openai/gpt-4o-mini"
	case "xia", "xai", "x-ai":
		return "grok-3-mini"
	case "claude", "anthropic":
		return "claude-sonnet-4-20250514"
	case "gemini", "google", "google_ai", "google-ai":
		return "gemini-3.5-flash"
	case "ollama":
		return "llama3.2"
	default:
		return "gpt-4o-mini"
	}
}

func defaultAdapterText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
