package translations

import "testing"

func TestParseImportPayloadStructured(t *testing.T) {
	payload, err := ParseImportPayloadBytes([]byte(`{
		"language":{"code":"fr","name":"French","native_name":"Français","default":true},
		"locale":"fr",
		"file_name":"initial.fr.json",
		"rows":[{"domain":"common","key":"common.app.title","language":"fr","value":"NEXUS GAMES"}]
	}`))
	if err != nil {
		t.Fatalf("ParseImportPayloadBytes returned error: %v", err)
	}
	if payload.Language.Code != "fr" || payload.Locale != "fr" || payload.FileName != "initial.fr.json" {
		t.Fatalf("unexpected payload metadata: %+v", payload)
	}
	if len(payload.Rows) != 1 || payload.Rows[0].Locale != "fr" {
		t.Fatalf("unexpected normalized rows: %+v", payload.Rows)
	}
}

func TestParseImportPayloadArray(t *testing.T) {
	payload, err := ParseImportPayloadBytes([]byte(`[
		{"domain":"common","key":"common.button.confirm","locale":"fr","value":"CONFIRMER"}
	]`))
	if err != nil {
		t.Fatalf("ParseImportPayloadBytes returned error: %v", err)
	}
	if payload.Locale != "fr" || payload.Language.Code != "fr" {
		t.Fatalf("unexpected inferred locale: %+v", payload)
	}
}

func TestParseImportPayloadUppercaseRows(t *testing.T) {
	payload, err := ParseImportPayloadBytes([]byte(`{
		"rows":[{"Domain":"nexus_game","Key":"nexus_game.ai.ask","Locale":"fr","Value":"Demander à mon IA"}]
	}`))
	if err != nil {
		t.Fatalf("ParseImportPayloadBytes returned error: %v", err)
	}
	if len(payload.Rows) != 1 {
		t.Fatalf("expected one row, got %d", len(payload.Rows))
	}
	row := payload.Rows[0]
	if row.Domain != "nexus_game" || row.Key != "nexus_game.ai.ask" || row.Locale != "fr" || row.Value == "" {
		t.Fatalf("unexpected row: %+v", row)
	}
}
