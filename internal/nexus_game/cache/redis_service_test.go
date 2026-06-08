package cache

import "testing"

func TestRedisDisabledDoesNotErrorForFallbackOperations(t *testing.T) {
	t.Setenv(envRedisURL, "")
	service := NewRedisServiceFromEnv()

	if service.Status(t.Context()).Enabled {
		t.Fatal("redis should be disabled without REDIS_URL")
	}
	if err := service.SetString(t.Context(), "k", "v", 0); err != nil {
		t.Fatalf("disabled SetString should not error: %v", err)
	}
	if value, ok, err := service.GetString(t.Context(), "k"); err != nil || ok || value != "" {
		t.Fatalf("disabled GetString value=%q ok=%v err=%v", value, ok, err)
	}
	if locked, err := service.AcquireLock(t.Context(), "lock", 0); err != nil || locked {
		t.Fatalf("disabled AcquireLock locked=%v err=%v", locked, err)
	}
}

func TestRedactRedisURL(t *testing.T) {
	got := redactRedisURL("redis://:secret@localhost:6379/0")
	if got != "redis://redis:redacted@localhost:6379/0" {
		t.Fatalf("redacted url=%q", got)
	}
}
