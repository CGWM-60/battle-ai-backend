package service

import (
	"testing"

	"cgwm/battle/internal/models"
)

func TestWorldAIProviderStatusesExposePrimaryAndFallbackWithoutSecrets(t *testing.T) {
	t.Setenv("WORLD_AI_PRIMARY_PROVIDER", "mistral")
	t.Setenv("WORLD_AI_FALLBACK_PROVIDER", "openai")
	t.Setenv("MISTRAL_AI_KEY", "mistral-secret-value")
	t.Setenv("MISTRAL_AI_MODEL", "mistral-large")
	t.Setenv("OPEN_AI_KEY", "openai-secret-value")
	t.Setenv("OPEN_AI_MODEL", "gpt-test")

	statuses := NewWorldGameService(nil).AIProviderStatuses()
	if len(statuses) == 0 {
		t.Fatalf("expected provider statuses")
	}

	var primaryFound bool
	var fallbackFound bool
	for _, status := range statuses {
		if status.Name == "mistral" {
			primaryFound = status.Primary && status.Configured && status.SecretPreview != "mistral-secret-value"
		}
		if status.Name == "openai" {
			fallbackFound = status.Fallback && status.Configured && status.SecretPreview != "openai-secret-value"
		}
	}
	if !primaryFound {
		t.Fatalf("expected configured masked mistral primary provider")
	}
	if !fallbackFound {
		t.Fatalf("expected configured masked openai fallback provider")
	}
}

func TestBuildingAssetHashChangesWithVersion(t *testing.T) {
	service := NewWorldGameService(nil)
	first := service.CreateBuildingAssetHash("https://cdn/building.png", 1, 1)
	second := service.CreateBuildingAssetHash("https://cdn/building.png", 1, 2)
	if first == "" || second == "" {
		t.Fatalf("expected non-empty hashes")
	}
	if first == second {
		t.Fatalf("expected hash to change when asset version changes")
	}
}

func TestDeterministicNexusDecisionHasPlayableDurationsInputs(t *testing.T) {
	decision := deterministicNexusDecision(models.World{Name: "Test", GlobalTensionLevel: 10, GlobalWeatherRisk: 5})
	if len(decision.Events) == 0 || len(decision.Weather) == 0 || len(decision.Conflicts) == 0 {
		t.Fatalf("fallback decision should include event, weather and conflict")
	}
	if decision.Message.Message == "" {
		t.Fatalf("fallback decision should include daily message")
	}
}
