package translations

import (
	"encoding/json"
	"testing"

	"cgwm/battle/internal/models"
)

func TestNormalizeTagsJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "[]"},
		{name: "whitespace", input: " ", want: "[]"},
		{name: "valid empty array", input: "[]", want: "[]"},
		{name: "valid tags", input: `["launch","button"]`, want: `["launch","button"]`},
		{name: "invalid json", input: "not json", want: "[]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeTagsJSON(tc.input); got != tc.want {
				t.Fatalf("normalizeTagsJSON(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestTranslationKeyDefaultsTagsJSON(t *testing.T) {
	key := &models.TranslationKey{Key: "launch.button.skip"}
	if err := key.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if key.TagsJSON != "[]" {
		t.Fatalf("BeforeCreate TagsJSON = %q, want []", key.TagsJSON)
	}

	blank := &models.TranslationKey{Key: "launch.button.enter_nexus", TagsJSON: " "}
	if err := blank.BeforeSave(nil); err != nil {
		t.Fatalf("BeforeSave returned error: %v", err)
	}
	if blank.TagsJSON != "[]" {
		t.Fatalf("BeforeSave TagsJSON = %q, want []", blank.TagsJSON)
	}

	withTags := &models.TranslationKey{
		Key:      "launch.boot.net",
		TagsJSON: `["launch","boot"]`,
	}
	if err := withTags.BeforeUpdate(nil); err != nil {
		t.Fatalf("BeforeUpdate returned error: %v", err)
	}
	if withTags.TagsJSON != `["launch","boot"]` {
		t.Fatalf("BeforeUpdate TagsJSON = %q, want preserved tags", withTags.TagsJSON)
	}

	if got := normalizeTagsJSON(""); got != "[]" {
		t.Fatalf("upsert default tags_json = %q, want []", got)
	}
}

func TestTagsJSONNeverEmpty(t *testing.T) {
	inputs := []string{"", " ", "[]", `["launch"]`, "not-json", `{"bad":"shape"}`}
	for _, input := range inputs {
		got := normalizeTagsJSON(input)
		if got == "" {
			t.Fatalf("normalizeTagsJSON(%q) returned empty string", input)
		}
		if !json.Valid([]byte(got)) {
			t.Fatalf("normalizeTagsJSON(%q) = invalid json %q", input, got)
		}
	}

	rows, err := loadInitialSeedRows()
	if err != nil {
		t.Fatalf("load initial seed rows: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("initial seed must not be empty")
	}

	key := &models.TranslationKey{Key: "launch.button.skip"}
	if err := key.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if key.TagsJSON == "" {
		t.Fatal("new translation key must never persist empty tags_json")
	}
}