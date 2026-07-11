# Test suites

Every suite calls a real method on the [official Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go). A suite **passes** when the SDK can issue the request, parse the response (or stream), and satisfy basic validation rules. See the [README](../README.md) for exit codes and general usage.

## Presets

| Preset | `TEST_SUITES` value | Scope |
|--------|----------------------|-------|
| Default | `all` or `default` | `models`, `models_get`, `messages`, `messages_stream` |
| Extended | `extended` | default plus tools, JSON, multi-turn, token counting, vision, legacy completions, message batches, beta files, and `error_responses` |
| Full | `full` | every registered suite, including deprecated completions and beta skills |

Deprecated legacy completions suites (`completions`, `completions_stream`) are **opt-in** — included in `extended` and `full`, but not in `default`. They are labeled `(deprecated)` in `--list-suites`.

## Suite reference

| Suite | SDK surface | Endpoint |
|-------|-------------|----------|
| `models` | `client.Models.List` | `GET /v1/models` |
| `models_get` | `client.Models.Get` | `GET /v1/models/{id}` |
| `messages` | `client.Messages.New` | `POST /v1/messages` |
| `messages_stream` | `client.Messages.NewStreaming` | `POST /v1/messages` (stream) |
| `messages_tools` | `client.Messages.New` (with `tools`) | `POST /v1/messages` |
| `messages_tools_stream` | `client.Messages.NewStreaming` (with `tools`) | `POST /v1/messages` (stream) |
| `messages_json` | `client.Messages.New` (`output_config.format` json_schema) | `POST /v1/messages` |
| `messages_multi_turn` | `client.Messages.New` (multi-turn history with tool results) | `POST /v1/messages` |
| `messages_count_tokens` | `client.Messages.CountTokens` | `POST /v1/messages/count_tokens` |
| `messages_vision` | `client.Messages.New` (with image input) | `POST /v1/messages` |
| `(deprecated) completions` | `client.Completions.New` | `POST /v1/complete` |
| `(deprecated) completions_stream` | `client.Completions.NewStreaming` | `POST /v1/complete` (stream) |
| `message_batches_create` | `client.Messages.Batches.New` | `POST /v1/messages/batches` |
| `message_batches_get` | `client.Messages.Batches.Get` | `GET /v1/messages/batches/{id}` |
| `message_batches_cancel` | `client.Messages.Batches.Cancel` | `POST /v1/messages/batches/{id}/cancel` |
| `message_batches_list` | `client.Messages.Batches.List` | `GET /v1/messages/batches` |
| `beta_files` | `client.Beta.Files.Upload`, `List`, `GetMetadata`, `Download`, `Delete` | `POST/GET/DELETE /v1/files?beta=true`, `GET /v1/files/{id}/content?beta=true` |
| `beta_skills` | `client.Beta.Skills.New`, `Get`, `List`, `Delete` | `POST/GET/DELETE /v1/skills?beta=true` |
| `beta_skill_versions` | `client.Beta.Skills.Versions.New`, `Get`, `List` | `POST/GET /v1/skills/{id}/versions?beta=true` |
| `error_responses` | `client.Messages.New` (invalid model) | `POST /v1/messages` |

## Suite-specific model configuration

Most suites reuse `ANTHROPIC_MODEL`. The following variables are only required for the suites listed below.

| Variable | Required by | Default | Notes |
|----------|-------------|---------|-------|
| `ANTHROPIC_VISION_MODEL` | `messages_vision` | same as `ANTHROPIC_MODEL` | Vision-capable model |
| `ANTHROPIC_COMPLETION_MODEL` | `completions`, `completions_stream` | `claude-2.1` when selected | Legacy text completions |

## Examples

The base image is `ghcr.io/beranekio/anthropic-compatibility-tester:latest`. Every example below assumes `ANTHROPIC_BASE_URL` and `ANTHROPIC_API_KEY` are set.

### Default suites

```bash
docker run --rm \
  -e ANTHROPIC_BASE_URL=https://your-endpoint.example \
  -e ANTHROPIC_API_KEY=your-api-key \
  -e ANTHROPIC_MODEL=your-model \
  ghcr.io/beranekio/anthropic-compatibility-tester:latest
```

### Run a subset

```bash
docker run --rm \
  -e ANTHROPIC_BASE_URL=https://your-endpoint.example \
  -e ANTHROPIC_API_KEY=your-api-key \
  -e TEST_SUITES=models,messages \
  ghcr.io/beranekio/anthropic-compatibility-tester:latest
```

### Extended preset

```bash
docker run --rm \
  -e ANTHROPIC_BASE_URL=https://your-endpoint.example \
  -e ANTHROPIC_API_KEY=your-api-key \
  -e ANTHROPIC_MODEL=your-model \
  -e ANTHROPIC_COMPLETION_MODEL=claude-2.1 \
  -e TEST_SUITES=extended \
  ghcr.io/beranekio/anthropic-compatibility-tester:latest
```