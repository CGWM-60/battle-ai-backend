package router

import (
	"testing"
	"time"
)

func TestAIProviderGenerationTimeoutDefaultsToSixtySeconds(t *testing.T) {
	t.Setenv("AI_PROVIDER_GENERATION_TIMEOUT_SECONDS", "")

	if got := aiProviderGenerationTimeout(); got != 60*time.Second {
		t.Fatalf("expected 60s default timeout, got %s", got)
	}
}

func TestAIProviderGenerationTimeoutUsesValidEnvironmentOverride(t *testing.T) {
	t.Setenv("AI_PROVIDER_GENERATION_TIMEOUT_SECONDS", "68")

	if got := aiProviderGenerationTimeout(); got != 68*time.Second {
		t.Fatalf("expected 68s timeout override, got %s", got)
	}
}

func TestAIProviderGenerationTimeoutRejectsInvalidOverride(t *testing.T) {
	t.Setenv("AI_PROVIDER_GENERATION_TIMEOUT_SECONDS", "invalid")

	if got := aiProviderGenerationTimeout(); got != 60*time.Second {
		t.Fatalf("expected 60s fallback timeout, got %s", got)
	}
}
