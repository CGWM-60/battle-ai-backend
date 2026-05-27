package service

import (
	"testing"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
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

func TestValidateSaveSyncRejectsStaleClientSave(t *testing.T) {
	lastSync := time.Now()
	clientSavedAt := lastSync.Add(-10 * time.Minute)
	save := &models.PlayerSave{
		Version:      4,
		CityLevel:    3,
		XP:           100,
		Population:   1000,
		Satisfaction: 80,
		Food:         1000,
		Energy:       1000,
		Credits:      1000,
		Gems:         5,
		LastSyncedAt: &lastSync,
	}
	input := PlayerSaveSyncInput{
		Version:       4,
		CityLevel:     3,
		XP:            100,
		Population:    1000,
		Satisfaction:  80,
		Food:          1000,
		Energy:        1000,
		Credits:       1000,
		Gems:          5,
		ClientSavedAt: &clientSavedAt,
	}
	if err := validateSaveSync(save, input, lastSync); err == nil {
		t.Fatalf("expected stale client save to be rejected")
	}
}

func TestApplyRewardToSaveUpdatesOfficialResources(t *testing.T) {
	save := &models.PlayerSave{XP: 10, Food: 20, Energy: 30, Credits: 40, Gems: 1, Population: 100, Satisfaction: 95}
	applyRewardToSave(save, EventReward{XP: 5, Food: 6, Energy: 7, Credits: 8, Gems: 2, Population: 10, Satisfaction: 20})
	if save.XP != 15 || save.Food != 26 || save.Energy != 37 || save.Credits != 48 || save.Gems != 3 || save.Population != 110 {
		t.Fatalf("reward not applied: %+v", save)
	}
	if save.Satisfaction != 100 {
		t.Fatalf("satisfaction should be clamped to 100, got %d", save.Satisfaction)
	}
}

func TestValidateEventRequirementsRejectsLowLevelPlayer(t *testing.T) {
	save := &models.PlayerSave{CityLevel: 2, XP: 50, Population: 1000}
	requirements := datatypes.JSON([]byte(`{"minCityLevel":3,"minXp":25,"minPopulation":500}`))
	if err := validateEventRequirements(save, requirements); err == nil {
		t.Fatalf("expected low level player to be rejected")
	}
}
