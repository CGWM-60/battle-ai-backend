package services

import "testing"

func TestOfficialResourceDefinitionsIncludeRequiredStarts(t *testing.T) {
	defs := OfficialResourceDefinitions()
	byCode := map[string]ResourceDefinition{}
	for _, def := range defs {
		byCode[def.Code] = def
	}

	if got := len(defs); got != 16 {
		t.Fatalf("expected 16 official resources, got %d", got)
	}
	if byCode["population"].InitialAmount != 0 {
		t.Fatalf("population must start at 0, got %d", byCode["population"].InitialAmount)
	}
	if byCode["food"].InitialAmount != 500 {
		t.Fatalf("food initial amount mismatch: %d", byCode["food"].InitialAmount)
	}
	if byCode["tokens"].DailyGrantAmount != 25 {
		t.Fatalf("tokens daily grant mismatch: %d", byCode["tokens"].DailyGrantAmount)
	}
	if byCode["quantum_core"].DailyGrantAmount != 0 {
		t.Fatalf("quantum_core must not be granted daily by default")
	}
}

func TestApplyStreakMultiplier(t *testing.T) {
	cases := []struct {
		day  int
		base int64
		want int64
	}{
		{day: 1, base: 100, want: 100},
		{day: 2, base: 100, want: 105},
		{day: 3, base: 100, want: 110},
		{day: 4, base: 100, want: 115},
		{day: 5, base: 100, want: 120},
		{day: 6, base: 100, want: 125},
		{day: 7, base: 100, want: 100},
	}

	for _, tc := range cases {
		if got := applyStreakMultiplier(tc.base, tc.day); got != tc.want {
			t.Fatalf("day %d multiplier: got %d want %d", tc.day, got, tc.want)
		}
	}
}
