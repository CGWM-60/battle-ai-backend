#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "[security_scan] FAIL: $1" >&2
  exit 1
}

scan_forbidden() {
  local pattern="$1"
  local message="$2"
  local hits
  hits=$(rg -n --glob '!scripts/security_scan.sh' --glob '!**/*_test.go' --glob '!**/testdata/**' "$pattern" . || true)
  if [ -n "$hits" ]; then
    echo "$hits"
    fail "$message"
  fi
}

# Default env must not enable mock in production code.
if rg -n 'AI_MOCK_ENABLED.*true|getEnv\("AI_MOCK_ENABLED", "true"\)' internal --glob '!**/*_test.go' >/dev/null 2>&1; then
  fail "AI_MOCK_ENABLED defaults to true in production code"
fi

if rg -n 'AI_PLATFORM_MODE.*mock|getEnv\("AI_PLATFORM_MODE", "mock"\)' internal --glob '!**/*_test.go' >/dev/null 2>&1; then
  fail "AI_PLATFORM_MODE defaults to mock in production code"
fi

scan_forbidden 'sk-[A-Za-z0-9]{10,}' 'hardcoded OpenAI-style API key detected'
scan_forbidden 'OPENAI_API_KEY\s*=\s*"[A-Za-z0-9]+"' 'hardcoded OPENAI_API_KEY detected'
scan_forbidden 'JWT_SECRET\s*=\s*"[A-Za-z0-9]+"' 'hardcoded JWT secret detected'
scan_forbidden 'Authorization: Bearer ey[A-Za-z0-9._-]+' 'hardcoded bearer token detected'

# User-facing mock strings outside tests and mock-provider internals.
mock_hits=$(rg -n 'mock:' internal \
  --glob '!**/*_test.go' \
  --glob '!**/testdata/**' \
  --glob '!internal/provider/providerIa.go' \
  --glob '!internal/service/ai_error_codes.go' \
  --glob '!internal/service/ai_provider_generation_service.go' || true)
if [ -n "$mock_hits" ]; then
  echo "$mock_hits"
  fail "user-facing mock prefix found outside tests"
fi

echo "[security_scan] OK"