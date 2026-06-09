package cache

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"testing"
)

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

func TestReadRESPBulkStringReadsFullPayload(t *testing.T) {
	want := strings.Repeat("payload-", 1024)
	resp := fmt.Sprintf("$%d\r\n%s\r\n", len(want), want)

	got, err := readRESP(bufio.NewReaderSize(&chunkedReader{data: resp, size: 3}, 16))
	if err != nil {
		t.Fatalf("readRESP returned error: %v", err)
	}
	if got != want {
		t.Fatalf("readRESP returned partial or corrupted payload: got length %d want %d", len(got), len(want))
	}
}

type chunkedReader struct {
	data string
	size int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if r.data == "" {
		return 0, io.EOF
	}
	n := r.size
	if n <= 0 || n > len(r.data) {
		n = len(r.data)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}
