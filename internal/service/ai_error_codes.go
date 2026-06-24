package service

import (
	"errors"
	"strings"
)

const (
	AIErrorMockDisabled                = "ai.error.mock_disabled"
	AIErrorPlatformKeyMissing          = "ai.error.platform_key_missing"
	AIErrorProviderKeyMissing          = "ai.error.provider_key_missing"
	AIErrorProviderModelMissing        = "ai.error.provider_model_missing"
	AIErrorProviderEmptyResponse       = "ai.error.provider_empty_response"
	AIErrorProviderMockResponse        = "ai.error.provider_mock_response"
	AIErrorProviderPromptLeak          = "ai.error.provider_prompt_leak"
	AIErrorProviderInvalidModel        = "ai.error.provider_invalid_model"
	AIErrorProviderAuthFailed          = "ai.error.provider_auth_failed"
	AIErrorProviderRateLimited         = "ai.error.provider_rate_limited"
	AIErrorProviderTimeout             = "ai.error.provider_timeout"
	AIErrorProviderIncompatibleParams  = "ai.error.provider_incompatible_params"
	AIErrorProviderUnavailable         = "ai.error.provider_unavailable"
	AIErrorNoUsableProvider            = "ai.error.no_usable_provider"
)

// AIProviderError is the canonical structured error for user-facing AI routes.
type AIProviderError struct {
	Code      string
	Provider  string
	Model     string
	Retryable bool
	Message   string
}

func (e AIProviderError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return e.Code
}

func NewAIProviderError(code, provider, model, message string, retryable bool) AIProviderError {
	return AIProviderError{
		Code:      code,
		Provider:  strings.TrimSpace(provider),
		Model:     strings.TrimSpace(model),
		Retryable: retryable,
		Message:   strings.TrimSpace(message),
	}
}

func AsAIProviderError(err error) (AIProviderError, bool) {
	if err == nil {
		return AIProviderError{}, false
	}
	var detail AIProviderError
	if errors.As(err, &detail) {
		return detail, true
	}
	return AIProviderError{}, false
}

type AIProviderErrorDetail struct {
	Code       string
	Message    string
	Provider   string
	Model      string
	Retryable  bool
	StatusCode int
}

func classifyAIProviderError(err error, providerName, modelName string) AIProviderErrorDetail {
	if detail, ok := AsAIProviderError(err); ok {
		return aiProviderErrorDetailFromStruct(detail)
	}

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
	case strings.Contains(lower, "mock ai is disabled"),
		strings.Contains(lower, AIErrorMockDisabled):
		detail.Code = AIErrorMockDisabled
		detail.StatusCode = 403
	case strings.Contains(lower, "platform provider key"),
		strings.Contains(lower, AIErrorPlatformKeyMissing),
		strings.Contains(lower, "platform api key unavailable"):
		detail.Code = AIErrorPlatformKeyMissing
		detail.StatusCode = 503
	case strings.Contains(lower, "provider api key is required"),
		strings.Contains(lower, "client api key required"),
		strings.Contains(lower, AIErrorProviderKeyMissing):
		detail.Code = AIErrorProviderKeyMissing
		detail.StatusCode = 400
	case strings.Contains(lower, "provider model is required"),
		strings.Contains(lower, AIErrorProviderModelMissing):
		detail.Code = AIErrorProviderModelMissing
		detail.StatusCode = 400
	case strings.Contains(lower, "provider url is required"):
		detail.Code = AIErrorProviderUnavailable
		detail.StatusCode = 503
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
		strings.Contains(lower, "invalide"),
		strings.Contains(lower, "unavailable"):
		detail.Code = AIErrorProviderUnavailable
		detail.StatusCode = 503
	default:
		detail.Code = AIErrorProviderEmptyResponse
		detail.Retryable = true
	}

	if detail.Message == "" {
		detail.Message = detail.Code
	}
	return detail
}

func aiProviderErrorDetailFromStruct(err AIProviderError) AIProviderErrorDetail {
	detail := AIProviderErrorDetail{
		Code:       err.Code,
		Message:    err.Error(),
		Provider:   err.Provider,
		Model:      err.Model,
		Retryable:  err.Retryable,
		StatusCode: 502,
	}
	switch err.Code {
	case AIErrorMockDisabled:
		detail.StatusCode = 403
	case AIErrorPlatformKeyMissing, AIErrorProviderUnavailable:
		detail.StatusCode = 503
	case AIErrorProviderKeyMissing, AIErrorProviderModelMissing, AIErrorProviderInvalidModel, AIErrorProviderIncompatibleParams:
		detail.StatusCode = 400
	case AIErrorProviderAuthFailed:
		detail.StatusCode = 401
	case AIErrorProviderRateLimited:
		detail.StatusCode = 429
		detail.Retryable = true
	case AIErrorProviderTimeout:
		detail.StatusCode = 504
		detail.Retryable = true
	case AIErrorProviderMockResponse, AIErrorProviderPromptLeak:
		detail.StatusCode = 422
	case AIErrorProviderEmptyResponse:
		detail.Retryable = true
	}
	return detail
}

func AllowUserFacingMockAI() bool {
	return strings.EqualFold(envString("AI_ALLOW_USER_FACING_MOCK", "", "false"), "true")
}

func allowMockForUserFacingRoutes() bool {
	return AllowUserFacingMockAI()
}

func rejectUserFacingMock(providerName, modelName string) error {
	if usesMockAIProvider() && !AllowUserFacingMockAI() {
		return NewAIProviderError(
			AIErrorMockDisabled,
			providerName,
			modelName,
			"mock ai is disabled on user-facing routes",
			false,
		)
	}
	return nil
}

func ClassifyAIProviderErrorForHandler(err error, providerName, modelName string) AIProviderErrorDetail {
	return classifyAIProviderError(err, providerName, modelName)
}