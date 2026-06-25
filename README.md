# Battle IA Backend

![Backend CI](https://github.com/CGWM-60/battle-ai-backend/actions/workflows/backend-ci.yml/badge.svg)

API Go pour Battle IA / Nexus Game (billing, traductions, IA, roleplay, tribunal).

## CI locale avant push

```bash
gofmt -w .
go vet ./...
go test ./internal/nexus_game/translations/... -count=1 -v
go test ./internal/service/... -count=1 -v
go test ./internal/router/... -count=1 -v
go test ./... -count=1 -v
go build ./...
bash scripts/security_scan.sh
```