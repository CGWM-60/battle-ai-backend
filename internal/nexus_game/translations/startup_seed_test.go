package translations

import (
	"os"
	"testing"
)

var requiredStartupKeys = []string{
	"launch.button.skip",
	"launch.button.enter_nexus",
	"launch.status.preparing",
	"launch.status.portal_ready",
	"launch.status.audio_locked",
	"launch.status.audio_unavailable",
	"launch.boot.net",
	"launch.boot.sig",
	"launch.boot.bia",
	"launch.boot.qst",
	"launch.boot.rpg",
	"launch.boot.coop",
	"launch.boot.nxs",
	"launch.boot.sys",
	"home.ui.text.connexion",
	"home.ui.text.creer_un_compte",
	"home.ui.text.le_reseau_vous_attend",
	"auth.ui.text.creer_un_profil",
	"auth.ui.text.initialiser_la_connexion",
}

func TestInitialSeedContainsStartupKeys(t *testing.T) {
	raw, err := os.ReadFile("imports/NEXUS_TRANSLATIONS_INITIAL_IMPORT.fr.json")
	if err != nil {
		t.Fatalf("read seed file: %v", err)
	}

	payload, err := ParseImportPayloadBytes(raw)
	if err != nil {
		t.Fatalf("parse seed file: %v", err)
	}

	keys := make(map[string]struct{}, len(payload.Rows))
	for _, row := range payload.Rows {
		keys[row.Key] = struct{}{}
	}

	for _, required := range requiredStartupKeys {
		if _, ok := keys[required]; !ok {
			t.Errorf("startup key missing from seed: %s", required)
		}
	}
}