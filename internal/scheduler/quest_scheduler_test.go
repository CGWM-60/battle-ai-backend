package scheduler

import (
	"errors"
	"strings"
	"testing"
)

func TestProviderCallErrorIncludesProviderModelAndURL(t *testing.T) {
	cause := errors.New("context deadline exceeded")
	err := providerCallError{
		Provider: "mistral",
		Model:    "mistral-large-latest",
		URL:      "https://api.mistral.ai/v1/chat/completions",
		Err:      cause,
	}

	msg := err.Error()
	for _, expected := range []string{"provider=mistral", "model=mistral-large-latest", "api.mistral.ai", "context deadline exceeded"} {
		if !strings.Contains(msg, expected) {
			t.Fatalf("expected %q in error %q", expected, msg)
		}
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected providerCallError to unwrap original cause")
	}
}

func TestProviderAttemptsWithExclusionsSkipsTimedOutProvider(t *testing.T) {
	t.Setenv("AI_QUEST_PROVIDER_ROTATION", "openai,mistral")
	t.Setenv("OPEN_AI_KEY", "openai-key")
	t.Setenv("OPEN_AI_MODEL", "gpt-5-mini")
	t.Setenv("MISTRAL_AI_KEY", "mistral-key")
	t.Setenv("MISTRAL_AI_MODEL", "mistral-large-latest")

	primary, ok := providerConfigForName("openai")
	if !ok {
		t.Fatal("expected openai test config")
	}
	attempts := providerAttemptsWithExclusions(primary, map[string]bool{"mistral": true})

	if len(attempts) != 1 {
		t.Fatalf("expected only primary attempt, got %#v", attempts)
	}
	if attempts[0].Name != "openai" {
		t.Fatalf("expected openai attempt, got %#v", attempts[0])
	}
}

func TestDisableTimedOutProviders(t *testing.T) {
	disabled := map[string]bool{}
	err := providerAttemptsError{
		Primary: aiProviderConfig{Name: "openai", Model: "gpt-5-mini"},
		Failures: []providerAttemptFailure{
			{Provider: "openai", Model: "gpt-5-mini", TimedOut: false},
			{Provider: "mistral", Model: "mistral-large-latest", TimedOut: true},
		},
		LastErr: errors.New("timeout"),
	}

	disableTimedOutProviders(err, disabled, cronTrace{})

	if !disabled["mistral"] {
		t.Fatalf("expected mistral to be disabled after timeout")
	}
	if disabled["openai"] {
		t.Fatalf("did not expect openai to be disabled without timeout")
	}
}

func TestParseGeneratedTribunalCasePayloadAcceptsCasesWrapper(t *testing.T) {
	payload := `{"cases":[{"title":"Le reçu fantôme","synopsis":"Une preuve contredit le témoin.","level":3,"cast":[{"actorId":"w1"}],"evidence":[{"evidenceId":"e1"}]}]}`

	got, err := parseGeneratedTribunalCasePayload(payload, 3)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got.Title != "Le reçu fantôme" {
		t.Fatalf("title = %q", got.Title)
	}
	if got.Synopsis == "" {
		t.Fatal("expected synopsis to be preserved")
	}
}

func TestFallbackGeneratedTribunalCaseIsPlayableAndVaried(t *testing.T) {
	first := fallbackGeneratedTribunalCase(1, "seed-a")
	second := fallbackGeneratedTribunalCase(2, "seed-b")

	if first.Title == "" || second.Title == "" {
		t.Fatal("fallback titles must be present")
	}
	if first.Title == second.Title {
		t.Fatalf("fallback titles should vary, got %q", first.Title)
	}
	if len(first.Cast) == 0 || len(first.Evidence) == 0 || len(first.Scenes) == 0 || len(first.ProgressionRules) == 0 {
		t.Fatalf("fallback case is not playable enough: %#v", first)
	}
	if isGenericTribunalTitle(first.Title) {
		t.Fatalf("fallback title is generic: %q", first.Title)
	}
}
