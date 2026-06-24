package translations

import "testing"

func TestParseFlutterScanBytes_RawArray(t *testing.T) {
	entries, err := ParseFlutterScanBytes([]byte(`[
		{
			"id":"abc",
			"text":"Supprimer",
			"sourceFile":"lib/test.dart",
			"sourceLine":10,
			"suggestedKey":"common.button.delete",
			"tags":["common","button"]
		}
	]`))
	if err != nil {
		t.Fatalf("ParseFlutterScanBytes: %v", err)
	}
	if len(entries) != 1 || entries[0].SuggestedKey != "common.button.delete" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestParseFlutterScanBytes_ObjectEntries(t *testing.T) {
	entries, err := ParseFlutterScanBytes([]byte(`{
		"generatedAt":"2026-06-24",
		"entries":[
			{
				"id":"x1",
				"fullKey":"common.button.save",
				"file":"lib/a.dart",
				"line":12,
				"originalText":"Sauvegarder",
				"defaultText":"Sauvegarder"
			}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseFlutterScanBytes object: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	n := entries[0].Normalized()
	if n.FullKey != "common.button.save" || n.SourceFile != "lib/a.dart" || n.SourceLine != 12 {
		t.Fatalf("normalized mismatch: %+v", n)
	}
}

func TestFlutterScanEntry_Normalized_FlutterFormat(t *testing.T) {
	entry := FlutterScanEntry{
		FullKey:      "prompts.sandbox.system",
		File:         "lib/sandbox.dart",
		Line:         40,
		OriginalText: "Tu es Sandbox Nexus...",
		DefaultText:  "Tu es Sandbox Nexus...",
		Module:       "sandbox",
		Kind:         "prompt",
		Tags:         []string{"prompt"},
	}.Normalized()

	if entry.FullKey != "prompts.sandbox.system" {
		t.Fatalf("fullKey=%s", entry.FullKey)
	}
	if entry.Text != "Tu es Sandbox Nexus..." {
		t.Fatalf("text=%s", entry.Text)
	}
	if entry.Namespace != "prompts.sandbox" {
		t.Fatalf("namespace=%s", entry.Namespace)
	}
}

func TestIsTranslatableFlutterEntry_RejectsTechnical(t *testing.T) {
	cases := []struct {
		key    string
		text   string
		reject bool
	}{
		{"anima.const.string.anima_active_id", "anima_active_id", true},
		{"common.ui.text.route", "/profile", true},
		{"common.ui.text.asset", "assets/foo.png", true},
		{"common.button.delete", "Supprimer", false},
		{"prompts.sandbox.system", "Tu es Sandbox Nexus, une IA directe.", false},
	}
	for _, tc := range cases {
		ok, _ := isTranslatableFlutterEntry(NormalizedFlutterScanEntry{
			FullKey:     tc.key,
			DefaultText: tc.text,
			Text:        tc.text,
		})
		if ok == tc.reject {
			t.Fatalf("key=%s text=%q ok=%v want reject=%v", tc.key, tc.text, ok, tc.reject)
		}
	}
}

func TestPreviewFlutterScan_SkipsTechnical(t *testing.T) {
	report, err := PreviewFlutterScan([]FlutterScanEntry{
		{SuggestedKey: "common.button.delete", Text: "Supprimer", DefaultText: "Supprimer"},
		{FullKey: "anima.const.string.anima_active_id", OriginalText: "anima_active_id"},
	})
	if err != nil {
		t.Fatalf("PreviewFlutterScan: %v", err)
	}
	if len(report.CreatedKeys) != 1 {
		t.Fatalf("created=%v skipped=%v", report.CreatedKeys, report.SkippedStorageKeys)
	}
	if len(report.SkippedEntries) != 1 {
		t.Fatalf("expected 1 skipped entry, got skipped=%v technical=%v storage=%v",
			report.SkippedEntries, report.SkippedTechnical, report.SkippedStorageKeys)
	}
}