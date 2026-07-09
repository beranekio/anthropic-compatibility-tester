# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project purpose

`anthropic-compatibility-tester` is a Go CLI (and Docker image) that checks whether an HTTP endpoint is compatible with the [Anthropic API](https://docs.anthropic.com/en/api/) by exercising it through the [official Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go) (`github.com/anthropics/anthropic-sdk-go`).

A suite **passes** when:

1. The SDK can issue the request without client-side errors.
2. The SDK can parse the response (or stream events) into typed structs.
3. Basic response validation rules in the suite are satisfied.

The process exits `0` when all selected suites pass, `1` when any suite fails compatibility checks, and `2` on configuration or runner errors.

## Repository layout

```
cmd/anthropic-compatibility-tester/   CLI entrypoint
cmd/mockserver/                     Standalone mock server binary
internal/
  config/                           Env/flag parsing, suite selection, validation
  runner/                           SDK client setup, suite orchestration, reporting
  suites/                           One file per suite; shared helpers (stream, output, tools)
  mockserver/                       In-process Anthropic-compatible HTTP server for tests
```

There is no `pkg/` export surface. Keep new code in `internal/`.

## Architecture

```
main → config.Load → runner.RunAll → suites.Suite.Run (per suite)
                              ↓
                    anthropic.NewClient(option.WithBaseURL, WithAPIKey, WithMaxRetries(0))
```

Each suite implements:

```go
type Suite interface {
    Name() string
    Description() string
    Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error
}
```

Register new suites in `internal/suites/suite.go` (`All()`), `internal/suitespec/names.go`, and `config.FullSuites` — keep all three in sync via tests. Update `validateModelsForSuites()` when model config is required. For deprecated APIs, implement `DeprecatedSuite` and ensure `printSuites()` labels them `(deprecated)`.

## Adding a new test suite

Follow this checklist for every new suite:

1. **Create** `internal/suites/<name>.go` with a stateless struct.
2. **Use the official SDK** — call `client.<Service>.<Method>`; do not hand-craft HTTP requests in suites.
3. **Validate** parsed responses with `fail(suite, message)` from `errors.go`; wrap transport/SDK errors with `fmt.Errorf("...: %w", err)`.
4. **Register** the suite in `suites.All()` and update:
   - `config.DefaultSuites` (only if it should run by default)
   - `config.ExtendedSuites` and `config.FullSuites`
   - `internal/suitespec/names.go`
   - `config.validateModelsForSuites()` (if a model env var is needed)
   - `config.Load()` flags/env vars (if new settings are required)
5. **Extend** `internal/mockserver/server.go` with a handler so CI stays offline.
6. **Test** — add or extend `internal/runner/runner_test.go` to run the new suite against the mock server. If config changed, update `internal/config/config_test.go` too.
7. **Document** — add the suite to the table in [`docs/suites.md`](docs/suites.md).

### Suite design principles

- **Minimal requests** — use the smallest prompt/input that exercises the endpoint.
- **Lenient where providers differ** — accept `refusal` stop reasons as valid outcomes.
- **Streaming** — reuse `validateEventStreamContentType` from `stream.go`; always check for a terminal event (`message_stop` for messages).
- **No retries** — the runner sets `option.WithMaxRetries(0)`; suites should not enable retries.
- **No live Anthropic calls in unit tests** — use `mockserver` only.
- **Per-suite timeout** — suites receive a context from `runner` bounded by `cfg.RequestTimeout`.

## Configuration conventions

| Env var | Purpose |
|---------|---------|
| `ANTHROPIC_BASE_URL` | Required. Conventionally `https://api.anthropic.com` (no `/v1` suffix); SDK appends paths. No query params. |
| `ANTHROPIC_API_KEY` | Required when running suites. Not required for `--list-suites`. |
| `ANTHROPIC_MODEL` | Messages suites (default `claude-sonnet-4-6`) |
| `ANTHROPIC_COMPLETION_MODEL` | Legacy completions (defaults to `claude-2.1` when selected) |
| `ANTHROPIC_VISION_MODEL` | Vision messages (defaults to `ANTHROPIC_MODEL`) |
| `TEST_SUITES` | Comma-separated names, or preset: `all`/`default`, `extended`, `full` |
| `REQUEST_TIMEOUT` | Per-suite timeout (default `2m`) |
| `ALLOW_INSECURE_HTTP` | Allow non-loopback `http://` |

## Testing

```bash
go test ./...
go build -o bin/anthropic-compatibility-tester ./cmd/anthropic-compatibility-tester
```

`internal/config/config_test.go` covers flag/env parsing. `internal/runner/runner_test.go` runs suites against `mockserver.New()` and `mockserver.BrokenServer()`.

**Every new suite must have a mock handler.** CI runs `go test ./...`, builds the binary, and builds the Docker image.

## CI and Docker

- GitHub Actions (`.github/workflows/ci.yml`): `go test ./...`, binary build, Docker build on every PR/push to `main`.
- Pushes to `main` publish multi-architecture images to GHCR.
- Dockerfile: multi-stage, distroless nonroot image, entrypoint is the binary.

Do not break the Docker entrypoint contract (no shell wrapper; flags/env only).

## Code style

- Go 1.24+ (`go.mod`). Match existing package naming and file layout.
- Stateless suite structs with value receivers for `Name`/`Description`/`Run`.
- Wrap errors with context; use `fail()` for compatibility validation failures.
- Keep suites focused — one SDK method family per suite file.

## Common pitfalls

- **Base URL** — do not include `/v1`; the SDK appends `v1/messages` etc. Query strings are rejected.
- **Content filter / refusal** — `stop_reason: refusal` with empty text is a pass, not a fail.
- **Mock parity** — forgetting to update `mockserver` breaks CI even if suite code is correct.
- **SDK version** — bump `github.com/anthropics/anthropic-sdk-go` in `go.mod` only when needed; run `go test ./...` after.

## PR checklist

- [ ] `go test ./...` passes
- [ ] New suite registered in `suite.go` (+ config/README if needed)
- [ ] Mock server handler added
- [ ] `runner_test.go` includes new suite in `TestRunAllPassesAgainstMockServer`
- [ ] `config_test.go` updated if config parsing, validation, or presets changed
- [ ] README updated for user-facing changes; suite table lives in `docs/suites.md`
- [ ] Focused diff — no unrelated changes