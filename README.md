# anthropic-compatibility-tester

Docker container that checks whether an arbitrary HTTP endpoint is compatible with the [Anthropic API](https://docs.anthropic.com/en/api/) by exercising it through the [official Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go).

Each test suite calls a real SDK method (models list, messages, streaming, token counting, message batches, beta files, and more). If the endpoint returns payloads the SDK cannot parse, or responses that fail basic validation, the process exits with a non-zero status code — making the image suitable for CI gates and compatibility smoke tests.

## Quick start

```bash
docker run --rm \
  -e ANTHROPIC_BASE_URL=https://your-endpoint.example \
  -e ANTHROPIC_API_KEY=your-api-key \
  -e ANTHROPIC_MODEL=your-model \
  ghcr.io/beranekio/anthropic-compatibility-tester:latest
```

This runs the default suites (`models`, `models_get`, `messages`, `messages_stream`). To see every available suite:

```bash
docker run --rm ghcr.io/beranekio/anthropic-compatibility-tester:latest --list-suites
```

## Configuration

All settings can be passed as environment variables or CLI flags.

| Variable | Flag | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `ANTHROPIC_BASE_URL` | `--base-url` | yes | — | API base URL (e.g. `https://api.anthropic.com`). The SDK appends paths like `/v1/messages`. Query parameters are not supported. |
| `ANTHROPIC_API_KEY` | `--api-key` | yes | — | API key sent to the endpoint |
| `ANTHROPIC_MODEL` | `--model` | no | `claude-sonnet-4-6` | Model for messages suites and the model ID fetched by `models_get` |
| `TEST_SUITES` | `--suites` | no | `all` | Comma-separated suite names, or preset: `all`/`default`, `extended`, `full` |
| `REQUEST_TIMEOUT` | `--timeout` | no | `2m` | Per-suite request timeout |
| `ALLOW_INSECURE_HTTP` | `--allow-insecure-http` | no | `false` | Allow plaintext `http://` to non-loopback hosts (loopback HTTP is always permitted) |

Some suites require additional model variables. See the [suite-specific configuration](docs/suites.md#suite-specific-model-configuration) section in `docs/suites.md`.

## Selecting suites

Use `TEST_SUITES` with a preset or an explicit comma-separated list.

| Preset | Scope |
|--------|-------|
| `all` / `default` | Core models and messages suites |
| `extended` | default plus tools, JSON, multi-turn, token counting, vision, legacy completions, message batches, beta files, and error responses |
| `full` | every registered suite, including deprecated completions and beta skills |

```bash
# A subset
docker run --rm \
  -e ANTHROPIC_BASE_URL=https://your-endpoint.example \
  -e ANTHROPIC_API_KEY=your-api-key \
  -e TEST_SUITES=models,messages \
  ghcr.io/beranekio/anthropic-compatibility-tester:latest
```

For the complete suite catalog, presets, and per-suite examples, see **[docs/suites.md](docs/suites.md)**.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | All selected suites passed |
| `1` | One or more suites failed compatibility checks |
| `2` | Configuration or runner error |

## Mock server

For testing gateways and SDK clients without a real backend, a standalone image of the in-process mock server is published. It implements the same Anthropic-compatible surface the test suite runs against (messages, models, completions, message batches, beta files, beta skills), but with deterministic canned responses. State is in memory, there is no authentication, and everything is lost on restart — suitable for a single-replica test backend, not production.

```bash
docker run --rm -p 8080:8080 ghcr.io/beranekio/anthropic-mockserver:latest
```

Point a client (or this tester) at `http://127.0.0.1:8080` on the host. When running the tester in a container that needs to reach the mock server on the host, use `host.docker.internal` with `--add-host` (Docker Desktop provides this automatically; Linux Docker Engine needs the flag) and allow plaintext HTTP to the non-loopback address:

```bash
docker run --rm \
  --add-host=host.docker.internal:host-gateway \
  -e ANTHROPIC_BASE_URL=http://host.docker.internal:8080 \
  -e ANTHROPIC_API_KEY=anything \
  -e ALLOW_INSECURE_HTTP=true \
  ghcr.io/beranekio/anthropic-compatibility-tester:latest
```

The listen address can be changed with `MOCK_ADDR` (or the `-addr` flag):

```bash
docker run --rm -p 9090:9090 -e MOCK_ADDR=:9090 ghcr.io/beranekio/anthropic-mockserver:latest
```

## Development

```bash
go test ./...
go build -o bin/anthropic-compatibility-tester ./cmd/anthropic-compatibility-tester
go build -o bin/mockserver ./cmd/mockserver

ANTHROPIC_BASE_URL=http://127.0.0.1:8080 \
ANTHROPIC_API_KEY=test \
./bin/anthropic-compatibility-tester
```

Build the containers locally:

```bash
docker build -t anthropic-compatibility-tester .
docker build --target mockserver -t anthropic-mockserver .
```

## CI and publishing

GitHub Actions runs unit tests and builds both Docker images on every push and pull request to `main`. When tests pass on a push to `main`, multi-architecture images (`linux/amd64`, `linux/arm64`) are published to GHCR:

- `ghcr.io/beranekio/anthropic-compatibility-tester:latest` — the compatibility tester
- `ghcr.io/beranekio/anthropic-mockserver:latest` — the standalone mock server