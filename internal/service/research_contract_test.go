package service

import (
	"encoding/json"
	"testing"
	"time"

	"gorm.io/datatypes"
)

func TestResearchDurationUsesLevelProgressionForFlutterTimers(t *testing.T) {
	raw := datatypes.JSON([]byte(`[{"level":1,"durationMinutes":30,"cumulativeMinutes":30},{"level":2,"durationMinutes":90,"cumulativeMinutes":120}]`))
	duration := researchDuration(raw, 2)
	if duration != 90*time.Minute {
		t.Fatalf("expected level progression duration, got %s", duration)
	}

	active := ActiveResearchProgress{
		NodeKey:         "intelligence_artificielle",
		TargetLevel:     2,
		StartedAt:       time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC),
		FinishAt:        time.Date(2026, 6, 10, 11, 30, 0, 0, time.UTC),
		DurationSeconds: int64(duration.Seconds()),
		DurationMinutes: int64(duration / time.Minute),
	}
	payload, err := json.Marshal(active)
	if err != nil {
		t.Fatalf("active research should marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		t.Fatalf("active research payload should unmarshal: %v", err)
	}
	if parsed["durationSeconds"].(float64) != 5400 {
		t.Fatalf("expected durationSeconds 5400, got %v", parsed["durationSeconds"])
	}
	if parsed["durationMinutes"].(float64) != 90 {
		t.Fatalf("expected durationMinutes 90, got %v", parsed["durationMinutes"])
	}
}
