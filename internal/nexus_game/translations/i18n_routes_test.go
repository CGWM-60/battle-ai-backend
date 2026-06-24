package translations

import "testing"

func TestLanguageFromLocale(t *testing.T) {
	if got := languageFromLocale("fr-FR"); got != "fr" {
		t.Fatalf("expected fr, got %s", got)
	}
	if got := languageFromLocale("en-US"); got != "en" {
		t.Fatalf("expected en, got %s", got)
	}
}

func TestCountryFromLocale(t *testing.T) {
	if got := countryFromLocale("en-GB"); got != "GB" {
		t.Fatalf("expected GB, got %s", got)
	}
}