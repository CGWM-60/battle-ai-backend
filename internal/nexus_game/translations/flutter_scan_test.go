package translations

import "testing"

func TestParseFlutterScanBytes(t *testing.T) {
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