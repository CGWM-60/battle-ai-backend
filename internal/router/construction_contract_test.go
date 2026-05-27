package router

import (
	"testing"
	"time"
)

func TestConstructionDurationMatchesOfficialPreviewTable(t *testing.T) {
	tests := []struct {
		name         string
		buildingKey  string
		targetLevel  int
		satisfaction int
		want         time.Duration
	}{
		{name: "habitation level 1", buildingKey: "habitation", targetLevel: 1, satisfaction: 80, want: 4 * time.Minute},
		{name: "habitation level 30", buildingKey: "habitation", targetLevel: 30, satisfaction: 80, want: 25*24*time.Hour + 12*time.Hour},
		{name: "solar level 30", buildingKey: "solar_park", targetLevel: 30, satisfaction: 80, want: 30 * 24 * time.Hour},
		{name: "vertical farm capped", buildingKey: "vertical_farm", targetLevel: 30, satisfaction: 80, want: 30 * 24 * time.Hour},
		{name: "research center level 5", buildingKey: "research_center", targetLevel: 5, satisfaction: 80, want: 2*time.Hour + 24*time.Minute},
		{name: "ai center level 5", buildingKey: "ai_center", targetLevel: 5, satisfaction: 80, want: 2*time.Hour + 42*time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateConstructionDuration(tt.buildingKey, tt.targetLevel, tt.satisfaction, nil)
			if got != tt.want {
				t.Fatalf("duration mismatch: got %s want %s", got, tt.want)
			}
		})
	}
}

func TestConstructionDurationAppliesSatisfactionAndEffects(t *testing.T) {
	normal := calculateConstructionDuration("solar_park", 10, 80, nil)
	if normal != 12*time.Hour {
		t.Fatalf("normal level 10 duration mismatch: got %s", normal)
	}

	boosted := calculateConstructionDuration("solar_park", 10, 95, map[string]any{
		"constructionSpeedMultiplier": 2,
	})
	if boosted != 5*time.Hour+42*time.Minute {
		t.Fatalf("boosted duration mismatch: got %s", boosted)
	}

	penalized := calculateConstructionDuration("solar_park", 10, 50, map[string]any{
		"constructionDurationMalusPercent": 25,
	})
	if penalized != 17*time.Hour+42*time.Minute {
		t.Fatalf("penalized duration mismatch: got %s", penalized)
	}
}

func TestBaseConstructionMinutesByLevel(t *testing.T) {
	expected := map[int]int{
		1:  5,
		2:  15,
		3:  30,
		4:  60,
		5:  120,
		6:  240,
		10: 720,
		11: 720,
		15: 1440,
		20: 2880,
		25: 5760,
		26: 5760,
		30: 43200,
	}
	for level, want := range expected {
		if got := baseConstructionMinutesForLevel(level); got != want {
			t.Fatalf("level %d: got %d want %d", level, got, want)
		}
	}
}
