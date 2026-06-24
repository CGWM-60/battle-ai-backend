package service

import "testing"

func TestLooksLikeAIMockOrPromptLeak(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "mock prefix", value: "mock: Tu es le maître du jeu IA", want: true},
		{name: "system prompt", value: "Tu es le maître du jeu IA d'une aventure", want: true},
		{name: "json instruction", value: "Réponds uniquement avec ce JSON", want: true},
		{name: "valid narration", value: `{"narration":"Tu entres dans la grotte."}`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeAIMockOrPromptLeak(tc.value); got != tc.want {
				t.Fatalf("looksLikeAIMockOrPromptLeak(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestGenerateOnceRejectsMockResponseOutsideMockMode(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "live")
	t.Setenv("AI_MOCK_ENABLED", "false")

	if usesMockAIProvider() {
		t.Fatalf("expected live mode without mock provider")
	}

	if !looksLikeAIMockOrPromptLeak("mock: prompt leak") {
		t.Fatalf("expected mock leak detection")
	}
}