package service

import (
	"encoding/json"
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

func TestArmyUnitCatalogDefinitionsExposeFlutterContract(t *testing.T) {
	for _, unitType := range supportedArmyUnitTypes() {
		stats := baseStatsForUnit(unitType)
		item := ArmyUnitCatalogItem{
			UnitType:                unitType,
			Name:                    armyUnitDisplayName(unitType),
			Description:             armyUnitGameplayDescription(unitType),
			Role:                    armyUnitRole(unitType),
			RequiredBarracksLevel:   minBarracksLevelForUnit(unitType),
			TrainingDurationSeconds: stats.TrainingDurationSeconds,
			TrainingDurationMinutes: ceilMinutes(stats.TrainingDurationSeconds),
			TrainingCost: map[string]int64{
				"credits": stats.CreditCostTrain,
				"food":    stats.FoodCost,
				"energy":  stats.EnergyCost,
			},
			UpkeepPerHour: map[string]int64{
				"credits": stats.CreditCost,
				"food":    stats.FoodConsumption,
				"energy":  stats.EnergyConsumption,
			},
			Stats: map[string]int{
				"health": stats.Health,
				"attack": stats.Attack,
			},
			Constraints: map[string]any{
				"requiredBarracksLevel": minBarracksLevelForUnit(unitType),
			},
		}
		if item.Name == "" || item.Description == "" || item.Role == "" {
			t.Fatalf("%s missing playable catalog text", unitType)
		}
		if item.RequiredBarracksLevel <= 0 {
			t.Fatalf("%s missing barracks constraint", unitType)
		}
		if item.TrainingDurationSeconds <= 0 || item.TrainingDurationMinutes <= 0 {
			t.Fatalf("%s missing training duration", unitType)
		}
		if item.TrainingCost["credits"] != stats.CreditCostTrain {
			t.Fatalf("%s should expose training credit cost, not upkeep", unitType)
		}
		payload, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("catalog item should marshal for Flutter: %v", err)
		}
		for _, key := range []string{"unitType", "description", "requiredBarracksLevel", "trainingDurationSeconds", "trainingDurationMinutes", "trainingCost", "constraints"} {
			if !json.Valid(payload) || !containsJSONKey(payload, key) {
				t.Fatalf("%s missing json key %s in %s", unitType, key, string(payload))
			}
		}
	}
}

func containsJSONKey(payload []byte, key string) bool {
	var parsed map[string]any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return false
	}
	_, ok := parsed[key]
	return ok
}

func TestWorldAIProviderOrderDefaultsToMistralThenOpenAI(t *testing.T) {
	t.Setenv("WORLD_AI_PRIMARY_PROVIDER", "")
	t.Setenv("AI_WORLD_PRIMARY_PROVIDER", "")
	t.Setenv("WORLD_AI_FALLBACK_PROVIDER", "")
	t.Setenv("AI_WORLD_FALLBACK_PROVIDER", "")

	order := worldAIProviderOrder()
	if len(order) != 2 || order[0] != "mistral" || order[1] != "openai" {
		t.Fatalf("expected default provider order mistral then openai, got %#v", order)
	}
}

func TestWorldAIProviderOrderDeduplicatesFallback(t *testing.T) {
	t.Setenv("WORLD_AI_PRIMARY_PROVIDER", "mistral")
	t.Setenv("WORLD_AI_FALLBACK_PROVIDER", "mistral")

	order := worldAIProviderOrder()
	if len(order) != 1 || order[0] != "mistral" {
		t.Fatalf("expected duplicate fallback to be removed, got %#v", order)
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

func TestApplyPenaltyToSaveClampsResources(t *testing.T) {
	save := &models.PlayerSave{XP: 10, Food: 20, Energy: 30, Credits: 40, Gems: 1, Population: 100, Satisfaction: 5}
	applyPenaltyToSave(save, EventReward{XP: 50, Food: 50, Energy: 50, Credits: 50, Gems: 5, Population: 200, Satisfaction: 20})
	if save.XP != 0 || save.Food != 0 || save.Energy != 0 || save.Credits != 0 || save.Gems != 0 || save.Population != 0 {
		t.Fatalf("penalties should clamp resources to zero: %+v", save)
	}
	if save.Satisfaction != 0 {
		t.Fatalf("satisfaction should clamp to zero, got %d", save.Satisfaction)
	}
}

func TestBeginnerProtectionCoversNewLowLevelPlayer(t *testing.T) {
	save := models.PlayerSave{CityLevel: 2, CreatedAt: time.Now().Add(-24 * time.Hour)}
	if !isBeginnerProtected(save, time.Now()) {
		t.Fatalf("expected low-level recent player to be protected")
	}
	save.CityLevel = 5
	save.CreatedAt = time.Now().Add(-96 * time.Hour)
	if isBeginnerProtected(save, time.Now()) {
		t.Fatalf("expected established player not to be protected")
	}
}

func TestSimulationCycleNormalization(t *testing.T) {
	cases := map[string]string{
		"":              SimulationCycleManual,
		"15m":           SimulationCycleLight,
		"hourly":        SimulationCycleHourly,
		"continent":     SimulationCycleHourly,
		"daily":         SimulationCycleDaily,
		"unknown":       SimulationCycleManual,
		" continental ": SimulationCycleHourly,
	}
	for input, expected := range cases {
		if got := normalizeSimulationCycle(input); got != expected {
			t.Fatalf("cycle %q: expected %q, got %q", input, expected, got)
		}
	}
}

func TestSimulationProfilesProduceDifferentRiskDeltas(t *testing.T) {
	militaryTension, militaryWeather := simulationRiskDeltas("militaire", SimulationCycleHourly)
	ecoTension, ecoWeather := simulationRiskDeltas("ecologique", SimulationCycleHourly)
	if militaryTension <= ecoTension {
		t.Fatalf("military profile should raise tension more than ecological: %d <= %d", militaryTension, ecoTension)
	}
	if ecoWeather <= militaryWeather {
		t.Fatalf("ecological profile should raise weather risk more than military: %d <= %d", ecoWeather, militaryWeather)
	}
	lightTension, _ := simulationRiskDeltas("instable", SimulationCycleLight)
	if lightTension > 2 {
		t.Fatalf("light cycle should clamp tension delta, got %d", lightTension)
	}
}

func TestPickSimulationContinentUsesCycleStrategy(t *testing.T) {
	world := models.World{
		CurrentCycle: 1,
		Continents: []models.Continent{
			{Id: 1, CurrentPlayers: 100, TensionLevel: 10},
			{Id: 2, CurrentPlayers: 20, TensionLevel: 40},
			{Id: 3, CurrentPlayers: 5, TensionLevel: 90},
		},
	}
	if got := pickSimulationContinent(world, SimulationCycleLight); got.Id != 2 {
		t.Fatalf("light cycle should rotate by current cycle, got continent %d", got.Id)
	}
	if got := pickSimulationContinent(world, SimulationCycleHourly); got.Id != 3 {
		t.Fatalf("hourly cycle should target least populated continent, got continent %d", got.Id)
	}
	if got := pickSimulationContinent(world, SimulationCycleDaily); got.Id != 3 {
		t.Fatalf("daily cycle should target highest tension continent, got continent %d", got.Id)
	}
}

func TestDefaultWeatherEffectsVaryByProfile(t *testing.T) {
	commercial := defaultWeatherEffects("commercial")
	technical := defaultWeatherEffects("technologique")
	if commercial["credits"] == nil {
		t.Fatalf("commercial weather should affect credits")
	}
	if technical["research"] == nil {
		t.Fatalf("technological weather should affect research")
	}
}

func TestParseEventRewardRejectsNegativeReward(t *testing.T) {
	_, err := parseEventReward(datatypes.JSON([]byte(`{"credits":-1}`)))
	if err == nil {
		t.Fatalf("expected negative reward to be rejected")
	}
}

func TestValidateRequirementsPayloadRejectsInvalidJSON(t *testing.T) {
	if err := validateRequirementsPayload(datatypes.JSON([]byte(`{"minCityLevel":"bad"}`))); err == nil {
		t.Fatalf("expected invalid requirements payload to be rejected")
	}
}

func TestShouldReturnGlobalResearchCatalog(t *testing.T) {
	trueCases := []string{
		"",
		"global",
		"all",
		"themes",
		"recherche",
		"research",
		"researchcenter",
		"research_center",
		"centre_recherche",
		"centre_de_recherche",
		"lab",
		"laboratory",
	}
	for _, input := range trueCases {
		if !shouldReturnGlobalResearchCatalog(input) {
			t.Fatalf("expected %q to return global catalog", input)
		}
	}

	falseCases := []string{"city_hall", "solar_park", "trade_hub", "defense_grid"}
	for _, input := range falseCases {
		if shouldReturnGlobalResearchCatalog(input) {
			t.Fatalf("expected %q to stay building-scoped", input)
		}
	}
}

func TestDeduplicateDailyTasksByNormalizedTypeAndTitle(t *testing.T) {
	input := []models.DailyTask{
		{Title: "Pillage du Secteur 7", TaskType: "resource"},
		{Title: "  pillage   du   secteur 7 ", TaskType: "RESOURCE"},
		{Title: "Pillage du Secteur 7", TaskType: "military"}, // different type => keep
		{Title: "Négociation Nomades", TaskType: "diplomacy"},
		{Title: "", TaskType: "resource"}, // invalid title => removed
	}

	got := deduplicateDailyTasks(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique tasks, got %d", len(got))
	}

	if got[0].Title != "Pillage du Secteur 7" || got[0].TaskType != "resource" {
		t.Fatalf("first unique task should keep first occurrence, got %+v", got[0])
	}

	if got[1].TaskType != "military" {
		t.Fatalf("expected same title with different type to be kept, got %+v", got[1])
	}
}

func TestGenerateDeterministicDailyTasksHasVariedUniqueTitles(t *testing.T) {
	svc := NewWorldGameService(nil)
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)

	tasks := svc.generateDeterministicDailyTasks(42, 3, now, expires)
	if len(tasks) < 22 {
		t.Fatalf("expected at least 22 tasks, got %d", len(tasks))
	}

	seen := map[string]bool{}
	for _, task := range tasks {
		if task.Title == "" {
			t.Fatalf("task title should not be empty")
		}
		if seen[task.Title] {
			t.Fatalf("duplicate title found in deterministic tasks: %q", task.Title)
		}
		seen[task.Title] = true
	}
}
