package translations

import (
	"testing"
	"time"
)

func TestForcedContentDescriptionEntriesCoverThreePillars(t *testing.T) {
	entries := forcedContentDescriptionEntries(time.Unix(0, 0))
	byKey := map[string]TranslationEntry{}
	for _, entry := range entries {
		byKey[entry.Key] = entry
	}

	required := []string{
		"building.modular_habitat.description",
		"building.modular_habitat.flavor",
		"building.modular_habitat.level_30.description",
		"unit.milicien_nexus.description",
		"unit.milicien_nexus.flavor",
		"unit.milicien_nexus.level_30.description",
		"research.efficient_storage.description",
		"research.efficient_storage.flavor",
		"research.efficient_storage.level_30.description",
	}
	for _, key := range required {
		entry, ok := byKey[key]
		if !ok {
			t.Fatalf("missing forced translation key %s", key)
		}
		if entry.Locale != "fr" || entry.Domain == "" || entry.Value == "" {
			t.Fatalf("invalid forced translation entry for %s: %#v", key, entry)
		}
	}
}
