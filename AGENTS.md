# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project purpose

`anthropic-compatibility-tester` is a Go CLI (and Docker image) that checks whether an HTTP endpoint is compatible with the [Anthropic API](https://docs.anthropic.com/en/api/) by exercising it through the [official Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go) (`github.com/anthropics/anthropic-sdk-go`).

A suite **passes** when:

1. The SDK can issue the request without client-side errors.
2. The SDK can parse the response (or stream events) into typed structs.
3. Basic response validation rules in the suite are satisfied.

The process exits `0` when all selected suites pass, `1` when any suite fails compatibility checks, and `2` on configuration or runner errors.

### Coverage scope

Suites target **core** Messages / Models / Completions / Message Batches, plus **beta Files and Skills**. Large `client.Beta` platform APIs (agents, sessions, vaults, memory stores, tunnels, deployments, …) are out of scope unless an issue explicitly asks for them. Prefer filling gaps in the core surface before expanding platform coverage.

The suite catalog lives in [`docs/suites.md`](docs/suites.md). Current SDK version is whatever `go.mod` pins — do not hardcode version numbers in docs.

## Repository layout

```
cmd/anthropic-compatibility-tester/   CLI entrypoint
cmd/mockserver/                       Standalone mock server binary
docs/
  suites.md                           Suite catalog (user-facing)
internal/
  config/                             Env/flag parsing, suite selection, validation
  runner/                             SDK client setup, suite orchestration, reporting
  suites/                             One file per suite; shared helpers (stream, output, tools)
  suitespec/                          Suite name registry (keep in sync with suites.All / FullSuites)
  mockserver/                         In-process Anthropic-compatible HTTP server for tests
                                      (handlers in server.go; state in stores.go; payloads.go)
  testutil/                           Shared fixtures (PNG, skill bundles, small text files)
```

There is no `pkg/` export surface. Keep new code in `internal/`. Multimodal and upload suites should use `internal/testutil` fixtures rather than embedding new binaries ad hoc.

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

Register new suites in all of the places listed under [Adding a new test suite](#adding-a-new-test-suite). Update `validateModelsForSuites()` when model config is required. For deprecated APIs, implement `DeprecatedSuite` and ensure `printSuites()` labels them `(deprecated)`.

## Adding a new test suite

Follow this checklist for every new suite:

1. **Create** `internal/suites/<name>.go` with a stateless struct.
2. **Use the official SDK** — call `client.<Service>.<Method>`; do not hand-craft HTTP requests in suites.
3. **Validate** parsed responses with `fail(suite, message)` from `errors.go`; wrap transport/SDK errors with `fmt.Errorf("...: %w", err)`.
4. **Register** the suite everywhere it must appear:

   | Location | When |
   |----------|------|
   | `suites.All()` in `suite.go` | always |
   | `internal/suitespec/names.go` | always |
   | `config.FullSuites` | always |
   | `config.DefaultSuites` | only if it should run by default |
   | `config.ExtendedSuites` | if it belongs in the extended preset |
   | `config.validateModelsForSuites()` | if a model env var is needed |
   | `config.Load()` flags/env | if new settings are required |
   | `internal/runner/runner_test.go` suite list | always (mock full run) |

5. **Extend mockserver** so CI stays offline:
   - Handler in `internal/mockserver/server.go`
   - Stateful resources in `stores.go` when needed
   - Canned shapes in `payloads.go` when needed
6. **Test** — runner tests against `mockserver.New()` / `BrokenServer()`. If config changed, update `internal/config/config_test.go` too. Prefer mockserver unit tests for new handler edge cases.
7. **Document** — add the suite to the table in [`docs/suites.md`](docs/suites.md). Update README only for user-facing config or behavior changes.

### Suite design principles

- **Minimal requests** — use the smallest prompt/input that exercises the endpoint.
- **Lenient where providers differ** — accept `refusal` stop reasons as valid outcomes.
- **Streaming** — reuse helpers from `stream.go` (`validateEventStreamContentType`, start-envelope checks, completion helpers). Always require a terminal event (`message_stop` for messages).
- **Content-block lifecycle** — track the **currently open** block (`content_block_start` → deltas → `content_block_stop`), similar to `messages_tools_stream` / `messages_stream`. Do not use stream-wide “ever seen start/stop” flags that let one block borrow lifecycle events from another.
- **No retries** — the runner sets `option.WithMaxRetries(0)`; suites should not enable retries.
- **No live Anthropic calls in unit tests** — use `mockserver` only.
- **Per-suite timeout** — suites receive a context from `runner` bounded by `cfg.RequestTimeout`.
- **Skills cleanup** — the real Skills API rejects deleting a skill that still has versions. Delete all versions first (see `deleteBetaSkillVersions` / `cleanupBetaSkill`). The mock enforces the same rule.

## Configuration conventions

| Env var | Purpose |
|---------|---------|
| `ANTHROPIC_BASE_URL` | Required. Host-only URL (e.g. `https://api.anthropic.com`). **No path** — values like `/v1` are rejected; the SDK appends API paths. No query params. Trailing slash is trimmed. |
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

**Every new suite must have a mock handler.** CI runs `go test ./...`, builds both binaries, and builds both Docker images.

## CI and Docker

- GitHub Actions (`.github/workflows/ci.yml`): `go test ./...`, binary builds, and Docker builds for **both** the tester and the standalone mockserver on every PR/push to `main`.
- Pushes to `main` publish multi-architecture images to GHCR:
  - tester image (entrypoint: compatibility tester binary)
  - `anthropic-mockserver` image (entrypoint: mock server binary)
- Dockerfile: multi-stage, distroless nonroot targets (`tester`, `mockserver`).

Do not break the Docker entrypoint contract (no shell wrapper; flags/env only). When changing the Dockerfile, preserve both targets.

## Mock server expectations

When extending or reviewing `internal/mockserver`:

- **Streams** — `message_start` should use an empty in-progress envelope; content arrives via `content_block_*` events; `stop_reason` via `message_delta`.
- **Skills** — create / version-create require at least one non-empty multipart **file** part (`MultipartForm.File`); a text form field named `file` is not enough.
- **Errors** — prefer Anthropic-style JSON errors via `writeError` over plain `http.Error`.
- **State** — batch/file/skill state belongs in `stores.go`, not ad-hoc globals in handlers.

## Code style

- Go 1.24+ (`go.mod`). Match existing package naming and file layout.
- Stateless suite structs with value receivers for `Name`/`Description`/`Run`.
- Wrap errors with context; use `fail()` for compatibility validation failures.
- Keep suites focused — one SDK method family per suite file.
- CLI progress and summaries should go through `runner.Output` (not bare `fmt.Print` to stdout) so tests can capture output.

## Common pitfalls

- **Base URL** — must not include a path (`/v1` is rejected). The SDK appends `v1/messages` etc. Query strings and encoded path separators (`%2F`) are also rejected.
- **Content filter / refusal** — `stop_reason: refusal` with empty text is a pass, not a fail.
- **Stream lifecycle** — requiring only “some” `content_block_start`/`stop` on the whole stream is too weak; bind text deltas to an open block.
- **Skills delete order** — deleting a skill while versions remain fails on the live API and on the mock.
- **Mock parity** — forgetting to update `mockserver` (and `stores`/`payloads` when needed) breaks CI even if suite code is correct.
- **Registration drift** — missing `suitespec`, `FullSuites`, `ExtendedSuites`, or `runner_test` suite list causes confusing failures.
- **SDK version** — bump `github.com/anthropics/anthropic-sdk-go` in `go.mod` only when needed; run `go test ./...` after.

## PR checklist

- [ ] `go test ./...` passes
- [ ] New suite registered in `suite.go`, `suitespec/names.go`, and `config.FullSuites` (plus `DefaultSuites` / `ExtendedSuites` if applicable)
- [ ] Mock server handler added (`server.go`; `stores.go` / `payloads.go` if needed)
- [ ] `runner_test.go` includes new suite in `TestRunAllPassesAgainstMockServer`
- [ ] `config_test.go` updated if config parsing, validation, or presets changed
- [ ] Suite table updated in `docs/suites.md`; README updated only for user-facing changes
- [ ] Focused diff — no unrelated changes

Follow these instructions exactly. When working in subdirectories not listed above, check for additional project instruction files (AGENTS.md, Claude.md, etc.).
