package service

import (
	"fmt"
	"testing"
)

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

func TestGenerateRejectsMockEvenWhenMockModeForUserFacingRoutes(t *testing.T) {
	t.Setenv("AI_MOCK_ENABLED", "true")
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_ALLOW_USER_FACING_MOCK", "false")

	if !usesMockAIProvider() {
		t.Fatal("expected mock mode")
	}

	if AllowUserFacingMockAI() {
		t.Fatal("user-facing mock must be disabled by default")
	}

	detail := classifyAIProviderError(
		fmt.Errorf("ai provider returned mock or prompt leak response"),
		"openai",
		"gpt-5-mini",
	)
	if detail.Code != AIErrorProviderMockResponse {
		t.Fatalf("expected mock code, got %s", detail.Code)
	}
}