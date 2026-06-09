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
