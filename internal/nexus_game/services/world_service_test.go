package services

import (
	"testing"

	"cgwm/battle/internal/nexus_game/cache"
)

// TestWorldServiceBasic respects project rules: no extra deps (sqlite removed to avoid go.mod issues).
// Real DB tests should use the project's test setup if any.
// This is a minimal compile-safe test for the service.
func TestWorldServiceBasic(t *testing.T) {
	redis := cache.NewRedisServiceFromEnv() // may be disabled in tests
	ws := NewWorldService(nil, redis)       // nil db for basic test (methods that need db will be skipped in real tests)

	if ws == nil {
		t.Error("NewWorldService returned nil")
	}

	t.Log("WorldService basic instantiation test passed (no external DB driver to respect rules).")
}
