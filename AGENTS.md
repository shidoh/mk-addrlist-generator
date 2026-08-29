# mk-addrlist-generator — Agent Instructions

## Stack
- Go 1.26, module `mk-addrlist-generator`, no vendor dir
- CLI: `github.com/spf13/cobra`, HTTP: `github.com/gin-gonic/gin`
- Observability: Prometheus metrics via `prometheus/client_golang`
- Config: YAML via `gopkg.in/yaml.v3`

## Directory layout
```
cmd/          — cobra commands (root, serve, generate, validate, version)
pkg/
  api/        — gin HTTP server + routes + middleware
  config/     — YAML loading, validation, custom duration parser
  generator/  — address list generation, output templates (mikrotik/plain/json/nftables)
  netutil/    — IP/CIDR parsing, deduplication, CIDR aggregation
main.go       — entry point, sets version ldflags
helm/         — Helm chart
config/       — runtime config (config.yaml), config.example.yaml at root
```

## Developer commands

```bash
# Run all tests (race detector enabled, matching CI)
go test -v -race ./...

# Lint
golangci-lint run --timeout=5m

# Build locally
go build -o mk-addrlist-generator .

# Build with version info (matching CI)
go build -ldflags "-X main.version=ci -X main.buildTime=$(date -u +%Y-%m-%dT%H:M:%SZ) -X main.gitCommit=$(git rev-parse --short HEAD)" -o mk-addrlist-generator .

# Docker build (local)
docker build -t mk-addrlist-generator:test .

# Docker Compose (basic)
docker-compose up -d

# Docker Compose with monitoring stack
docker-compose --profile monitoring up -d
```

## Release flow
- Tags `v*` trigger `.github/workflows/release.yml`
- Uses GoReleaser v2 (`.goreleaser.yaml`), requires `fetch-depth: 0`
- Publishes: GitHub releases, Docker images (amd64+arm64 via manifest), Helm chart to OCI registry, Homebrew tap
- Requires env: `GITHUB_TOKEN`, `HOMEBREW_TAP_TOKEN`, `GITHUB_REPOSITORY_OWNER`

## Gotchas

**Custom duration format.** Config uses a non-standard duration parser (regex-based): `1d`, `12h30m`, `45m30s`, `2d3h45m30s`. `0` means permanent (no timeout). Do NOT replace this with `time.ParseDuration` — it will break config compatibility. The parser is in `pkg/config/types.go:47`.

**Module path.** `go.mod` declares `module mk-addrlist-generator` (no GitHub prefix). All internal imports use `mk-addrlist-generator/...` — not `github.com/shidoh/mk-addrlist-generator/...`.

**Version injection.** Three variables (`version`, `buildTime`, `gitCommit`) are set via ldflags at build time. They propagate through `cmd.SetVersionInfo()` → `cmd.Version` and are also duplicated in `pkg/api` package-level vars. Don't remove the duplicate in `pkg/api` — it's used directly by the `/health` and `/info` endpoints.

**CIDR aggregation is opt-in.** Both CLI (`-a`/`--aggregate`) and HTTP (`?aggregate=true`) require explicit enable. Default is no aggregation. Deduplication is on by default (`deduplicate=true`).

**Generator is stateful.** `Generator` holds a `statsCache` map protected by `sync.RWMutex`. Stats only reflect the last successful generation per list. Lists never generated show zero counts.

**No code generation.** No `go generate` hooks, no protobuf, no mock generation in this codebase (though `go.uber.org/mock` is an indirect dep).

**Test patterns.** All tests are table-driven or simple assertion tests. No external test dependencies, no test fixtures on disk. Server tests use `httptest.NewRecorder()` with direct `router.ServeHTTP()` calls.

**Docker.** Multi-stage build, non-root `appuser`, binary at `/app/mk-addrlist-generator`, default config at `/app/config.yaml`. Health check uses `wget` (Alpine, no curl).

**Goreleaser ignores Windows+ARM.** The `.goreleaser.yaml` explicitly ignores `windows/arm` and `windows/arm64` builds.
