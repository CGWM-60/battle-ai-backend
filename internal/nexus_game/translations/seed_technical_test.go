package translations

import (
	"strings"
	"testing"
)

func TestInitialSeedDoesNotContainTechnicalStorageKeys(t *testing.T) {
	rows, err := loadInitialSeedRows()
	if err != nil {
		t.Fatalf("load initial seed rows: %v", err)
	}

	forbiddenFragments := []string{
		"_box",
		"_id",
		"_v2",
		"_v3",
		"assets/",
		"secure_auth_jwt",
		"token",
		"api_key",
	}

	for _, row := range rows {
		lowerKey := strings.ToLower(row.Key)
		lowerValue := strings.ToLower(row.Value)
		for _, fragment := range forbiddenFragments {
			if strings.Contains(lowerKey, fragment) || strings.Contains(lowerValue, fragment) {
				t.Errorf("technical storage key in seed: %s = %q", row.Key, row.Value)
			}
		}
		if strings.HasPrefix(lowerKey, "anima.const.string.") ||
			strings.HasPrefix(lowerKey, "common.const.string.") {
			t.Errorf("const string key must not be seeded: %s", row.Key)
		}
	}
}

func TestInitialSeedContainsAnimaBusinessKeys(t *testing.T) {
	rows, err := loadInitialSeedRows()
	if err != nil {
		t.Fatalf("load initial seed rows: %v", err)
	}

	keys := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		keys[row.Key] = struct{}{}
	}

	required := []string{
		"common.format.percent",
		"anima.status.age_interactions",
		"anima.gauge.hunger.label",
		"anima.stage.larva",
		"anima.mood.flourishing",
		"anima.action.feed",
	}
	for _, key := range required {
		if _, ok := keys[key]; !ok {
			t.Errorf("missing anima business key in seed: %s", key)
		}
	}
}