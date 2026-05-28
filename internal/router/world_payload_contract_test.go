package router

import (
	"context"
	"testing"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
)

func TestFormatEventItemsProvidesContractKeys(t *testing.T) {
	events := []models.GameEvent{{
		Id:               42,
		Title:            "Alerte",
		Description:      "Description",
		Type:             "world",
		Difficulty:       "medium",
		Status:           "active",
		StartsAt:         time.Now().Add(-time.Hour),
		EndsAt:           time.Now().Add(time.Hour),
		RewardsJSON:      datatypes.JSON([]byte(`{"credits":100}`)),
		RequirementsJSON: datatypes.JSON([]byte(`{"minCityLevel":1}`)),
		ConsequencesJSON: datatypes.JSON([]byte(`{"tension":5}`)),
	}}

	items := formatEventItems(nil, context.Background(), 1, events)
	if len(items) != 1 {
		t.Fatalf("expected one event item, got %d", len(items))
	}
	item := items[0]
	for _, key := range []string{"id", "title", "description", "type", "status", "hasParticipated", "canClaim", "claimed"} {
		if _, ok := item[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
	if item["hasParticipated"] != false {
		t.Fatalf("expected hasParticipated to default false")
	}
}

func TestFormatConflictItemsProvidesContractKeys(t *testing.T) {
	items := formatConflictItems(nil, context.Background(), []models.Conflict{{
		Id:           7,
		Title:        "Conflit",
		Description:  "Desc",
		AttackerType: "continent",
		DefenderType: "guild",
		Intensity:    75,
		Status:       "active",
		StartsAt:     time.Now().Add(-time.Hour),
		EndsAt:       time.Now().Add(time.Hour),
	}})

	if len(items) != 1 {
		t.Fatalf("expected one conflict item, got %d", len(items))
	}
	item := items[0]
	for _, key := range []string{"id", "title", "description", "attackerType", "attackerName", "defenderType", "defenderName", "intensity", "risk", "status", "availableActions"} {
		if _, ok := item[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
}

func TestFormatWeatherItemsProvidesContractKeys(t *testing.T) {
	items := formatWeatherItems([]models.WeatherEvent{{
		Id:          3,
		Type:        "storm",
		Severity:    80,
		Title:       "Tempête",
		Description: "Forte",
		StartsAt:    time.Now().Add(-time.Hour),
		EndsAt:      time.Now().Add(time.Hour),
		EffectsJSON: datatypes.JSON([]byte(`{"energy":-10}`)),
	}})
	if len(items) != 1 {
		t.Fatalf("expected one weather item, got %d", len(items))
	}
	item := items[0]
	for _, key := range []string{"id", "type", "severity", "title", "description", "effects"} {
		if _, ok := item[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
}
