package service

import "strings"

const (
	AIErrorProviderEmptyResponse       = "ai.error.provider_empty_response"
	AIErrorProviderMockResponse        = "ai.error.provider_mock_response"
	AIErrorProviderPromptLeak          = "ai.error.provider_prompt_leak"
	AIErrorProviderInvalidModel        = "ai.error.provider_invalid_model"
	AIErrorProviderAuthFailed          = "ai.error.provider_auth_failed"
	AIErrorProviderRateLimited         = "ai.error.provider_rate_limited"
	AIErrorProviderTimeout             = "ai.error.provider_timeout"
	AIErrorProviderIncompatibleParams  = "ai.error.provider_incompatible_params"
	AIErrorNoUsableProvider            = "ai.error.no_usable_provider"
)

type AIProviderErrorDetail struct {
	Code       string
	Message    string
	Provider   string
	Model      string
	Retryable  bool
	StatusCode int
}

func classifyAIProviderError(err error, providerName, modelName string) AIProviderErrorDetail {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	lower := strings.ToLower(message)

	detail := AIProviderErrorDetail{
		Message:    message,
		Provider:   strings.TrimSpace(providerName),
		Model:      strings.TrimSpace(modelName),
		Retryable:  false,
		StatusCode: 502,
	}

	switch {
	case strings.Contains(lower, "mock or prompt leak") ||
		strings.HasPrefix(lower, "mock:") ||
		strings.Contains(lower, "tu es le maître du jeu ia"):
		detail.Code = AIErrorProviderMockResponse
		detail.StatusCode = 422
	case strings.Contains(lower, "prompt leak"):
		detail.Code = AIErrorProviderPromptLeak
		detail.StatusCode = 422
	case strings.Contains(lower, "empty response"),
		strings.Contains(lower, "sans contenu"),
		strings.Contains(lower, "no content"):
		detail.Code = AIErrorProviderEmptyResponse
		detail.Retryable = true
		detail.StatusCode = 502
	case strings.Contains(lower, "timeout"),
		strings.Contains(lower, "deadline exceeded"):
		detail.Code = AIErrorProviderTimeout
		detail.Retryable = true
		detail.StatusCode = 504
	case strings.Contains(lower, "rate limit"),
		strings.Contains(lower, "429"):
		detail.Code = AIErrorProviderRateLimited
		detail.Retryable = true
		detail.StatusCode = 429
	case strings.Contains(lower, "401"),
		strings.Contains(lower, "403"),
		strings.Contains(lower, "unauthorized"),
		strings.Contains(lower, "invalid api key"),
		strings.Contains(lower, "authentication"):
		detail.Code = AIErrorProviderAuthFailed
		detail.StatusCode = 401
	case strings.Contains(lower, "max_tokens"),
		strings.Contains(lower, "max_completion_tokens"),
		strings.Contains(lower, "unsupported parameter"),
		strings.Contains(lower, "incompatible"):
		detail.Code = AIErrorProviderIncompatibleParams
		detail.StatusCode = 400
	case strings.Contains(lower, "invalid model"),
		strings.Contains(lower, "model not found"):
		detail.Code = AIErrorProviderInvalidModel
		detail.StatusCode = 400
	case strings.Contains(lower, "required"),
		strings.Contains(lower, "invalide"):
		detail.Code = AIErrorNoUsableProvider
		detail.StatusCode = 400
	default:
		detail.Code = AIErrorProviderEmptyResponse
		detail.Retryable = true
	}

	if detail.Message == "" {
		detail.Message = detail.Code
	}
	return detail
}

func allowMockForUserFacingRoutes() bool {
	return strings.EqualFold(envString("AI_ALLOW_USER_FACING_MOCK", "", "false"), "true")
}

func ClassifyAIProviderErrorForHandler(err error, providerName, modelName string) AIProviderErrorDetail {
	return classifyAIProviderError(err, providerName, modelName)
}