# Fibe SDK and CLI

The official Go SDK and command-line interface for the Fibe platform.

Requires Go 1.26.8 or newer when building from source. See
[`COMPATIBILITY.md`](COMPATIBILITY.md) for the stable contract and
[`SECURITY.md`](SECURITY.md) for private vulnerability reporting.

## Install the CLI

### Homebrew

```bash
brew install --cask fibegg/sdk/fibe
```

If you prefer the two-step flow:

```bash
brew tap fibegg/sdk
brew install --cask fibe
```

The executable name is still `fibe`.

### Go install

```bash
go install github.com/fibegg/sdk/cmd/fibe@latest
```

## Setup

Authenticate the CLI with a named local profile. The default profile targets
`https://fibe.gg`, so most users only need:

```bash
fibe login --api-key "fibe_live_yourkeyhere"
```

Use additional profiles for staging, local, or feature environments:

```bash
fibe auth login --profile staging --domain next.fibe.live --api-key "fibe_test_..."
fibe auth use staging
fibe --profile default doctor
```

Credentials are stored in `~/.config/fibe/credentials.json`; non-secret profile
metadata is stored in `~/.config/fibe/config.json`. `FIBE_API_KEY` and
`FIBE_DOMAIN` remain supported as CI fallbacks when no profile is configured,
but they do not override an active profile.

## CLI examples

Use the CLI directly or from agent workflows.

```bash
fibe doctor
fibe status
fibe auth status
fibe server-info

fibe schema list
fibe schema agent create

fibe agents create --name "My Assistant" --provider "claude-code"

fibe agent chat my-agent "Fix the failing tests"
fibe agent chat my-agent - < prompt.md

fibe agents list --include-runtime-status --per-page 100 -o json

fibe agents upload-attachment my-agent --file ./context.zip
fibe agents download-attachment my-agent runtime-context.zip --to ./context.zip

fibe agents watch --max-events 5 --duration 1m

fibe pg create --name demo --playspec starter --marquee next --service web.subdomain=demo
cat payload.yml | fibe pg create -f -
fibe pg create < payload.yml

fibe pg get demo
fibe pg get demo -o json --only service_urls

fibe wait playground next --status running --timeout 5m
```

Waiting for a running Playground checks service readiness by default: the requested
configuration must be applied, builds must have no warnings, long-running services
must be ready, and one-shot dependencies must have exited successfully. Use
`--readiness lifecycle` when you only need the lifecycle status. For a slow
startup, increase `--timeout` to cover the service's Compose healthcheck budget.
Inspect deployment failures with
`fibe playgrounds debug <id-or-name> --build-logs --output json`. New builds retain
their complete logs; older truncated logs cannot be reconstructed.

Repository mirroring is asynchronous. `fibe wait prop <id-or-name>` waits for
`mirror_status: completed` and reports the persisted failure reason. The Go SDK
exposes these diagnostics through `Playgrounds.RuntimeStatusByIdentifier`,
`Playgrounds.DebugWithBuildLogsByIdentifier`, and
`Props.GetMirrorStateByIdentifier`, preserving existing response struct layouts.

## Output formatting

Use `-o` to change formatting:

```bash
fibe status -o json
fibe status -o yaml
```

Use `--only` to filter payloads:
```bash
fibe agents list --output yaml --only "id,name"
```

## Go SDK

Create a client and call the API:

```go
package main

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
)

func main() {
	client := fibe.NewClient(
		fibe.WithAPIKey("fibe_live_yourkeyhere"),
		fibe.WithRateLimitAutoWait(),
	)

	status, err := client.Status.Get(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Active Playgrounds: %d\n", status.Playgrounds.Active)
}
```

### Reliability

1. **Bounded retries and rate-limit waits.** Retryable responses use bounded `Retry-After` or exponential backoff. `fibe.WithRateLimitAutoWait()` waits before a request when the last response exhausted the known quota.
2. **Circuit breaking.** `fibe.WithCircuitBreaker(...)` stops requests after repeated transient failures, then probes after the reset interval.
3. **Idempotency.** Mutations share one generated `Idempotency-Key` across retries. `fibe.WithIdempotencyKey(ctx, key)` supplies a key that can span calls.
4. **Progress hooks.** `fibe.WithProgress(...)` receives `fibe.ProgressEvent` values. The CLI shows a spinner in terminals and line-based status in scripts.

Retryable reads and idempotency-protected mutations handle transient transport
failures and retryable server responses. Cancellation, expired deadlines,
validation errors, and non-replayable request bodies are not retried after
transmission begins. Rate-limit and circuit-breaker state belong to one logical
client/session.

## MCP Server

The `fibe` binary can also run as a local [Model Context Protocol](https://modelcontextprotocol.io) server, avoiding a new CLI process for every operation.

```bash
fibe mcp install --client claude-code
fibe mcp install --client claude-code --user
fibe mcp install --client codex --profile staging

fibe mcp serve --profile staging

fibe mcp serve --http :8080 --require-auth

fibe mcp install --client antigravity --transport streamable-http --url https://fibe.example.com/mcp
```

HTTP MCP requires authentication on non-loopback binds. Alternate API origins
must be explicitly allowlisted with repeatable `--allowed-domain` flags and use
request/session credentials. Browser origins must match the request host or an
exact `--allowed-domain` origin. Local filesystem, profile, conversation,
playground, and process capabilities are denied over HTTP by default. A
loopback-only deployment may opt in with `--allow-local-access`; stdio retains
the existing local tools.

### Tool surface

The server provides generic resource tools such as `fibe_resource_list`, `fibe_resource_get`, `fibe_resource_delete`, `fibe_resource_mutate`, and `fibe_resource_watch`, plus workflow tools such as `fibe_greenfield_create` and `fibe_launch`.

Agent status, attachments, and scheduled pokes use the resource tools. Set `params.include_runtime_status` when listing agents. Use the `agent.upload_attachment`, `agent_attachment`, and `agent_poke` resource operations for files and pokes. To inspect a Playground, call `fibe_resource_get` with `resource:"playground"` and `id_or_name:"..."`; use `fibe_playgrounds_debug` for raw Compose, route, and log details. Playspec jobs use `playspec.create` or `playspec.update`. `fibe_schema` documents and locally validates mutation payloads.

The generated registry docs currently list 60 registered dispatcher tools. By default, `fibe mcp serve` uses the `full` tool surface and advertises the 59 non-hidden tools. Use `--tools core` or `FIBE_MCP_TOOLS=core` to narrow the native surface to the 39 meta/base/greenfield/brownfield tools, or pass a comma-separated tier list such as `--tools other,meta`. Hidden tools are not advertised natively even in `full`, but remain dispatcher-reachable through `fibe_call` and `fibe_pipeline` when the caller already knows the tool name. Use `fibe_tools_catalog` to inspect `advertised` and `hidden` flags for a running server. Regenerate `fibe_mcp_tools_catalog.md` and `fibe_tools_table.md` deterministically from the Go MCP registry with:

```bash
go run ./scripts/docs
go run ./scripts/docs --check
```

MCP annotations mark reads with `readOnlyHint` and delete, rollout, and hard-restart operations with `destructiveHint`. Destructive tools require `confirm:true` unless the server runs with `--yolo` or `FIBE_MCP_YOLO=1`.

### Pipeline composition

`fibe_pipeline` runs several tool calls in one round trip and passes values between them with JSONPath bindings:

```json
{
  "steps": [
    {"id": "pg",   "tool": "fibe_resource_mutate", "args": {"resource": "playground", "operation": "create", "payload": {"name": "ci", "playspec_id": 5}}},
    {"id": "wait", "tool": "fibe_playgrounds_wait",   "args": {"id_or_name": "$.pg.id", "status": "running"}},
    {"id": "logs", "tool": "fibe_playgrounds_logs",   "args": {"id_or_name": "$.pg.id", "service": "web", "tail": 100}}
  ],
  "return": "$.logs.lines"
}
```

Use `parallel` for independent steps and `for_each` for arrays. Results stay in the session cache for five minutes and can be queried through `fibe_pipeline_result` without rerunning the pipeline.

### Streaming

`fibe_playgrounds_wait` and `fibe_logs_follow` send MCP progress notifications. Long SDK operations do the same when the client supplies a progress token. Use `fibe logs follow <id-or-name>` for continuous Playground or Trick logs from the CLI.

SDK WebSocket streams use the configured HTTP transport, enforce a 10 MiB
frame limit, and close their channels on cancellation or terminal failure.
They do not reconnect transparently because replay could duplicate messages;
callers should reconnect explicitly according to their own delivery semantics.

### Resources

The server also exposes read-only MCP resources agents can load once at session start:

| URI | Contents |
|---|---|
| `fibe://me` | Authenticated user snapshot |
| `fibe://status` | Account status dashboard |
| `fibe://schema` | All resource schemas |
| `fibe://schema/{resource}` | Schema for a specific resource (e.g., `fibe://schema/playground`) |
| `fibe://help/{path}` | cobra Long help for a command path |
| `fibe://pipeline/schema` | `fibe_pipeline` DSL reference |
| `fibe://pipelines/{id}` | Cached pipeline result (5-min TTL) |

### Auth and profiles

Stdio transport is single-tenant by design (one process per client). It starts
with the selected CLI profile, and agents can switch the current MCP session at
runtime with:

- `fibe_auth_list`: list local profiles without exposing API keys
- `fibe_auth_use`: switch this MCP session to another profile
- `fibe_auth_status`: show the current MCP auth target
- `fibe_auth_set`: advanced raw API key/domain override

For HTTP/SSE deployments serving multiple tenants, the server resolves
credentials per request in this order:

1. A prior `fibe_auth_use` or `fibe_auth_set` tool call in the same session
2. `Authorization: Bearer <fibe-api-key>` header, falling back to `X-Fibe-API-Key`
3. The server-wide profile/API-key fallback, only when `--require-auth` is not set

`X-Fibe-Domain` can provide an allowlisted per-request origin only alongside
request/session credentials. Each session gets an isolated client. Credentials
are pinned to the session; conflicting later headers fail authentication. Use a
new session or the transactional `fibe_auth_use` / `fibe_auth_set` tools to
switch deliberately.

### Audit log

`FIBE_MCP_AUDIT_LOG` is experimental. It writes one JSON line per tool call; its schema and redaction may change.

## Shell completions

Generate completions with the CLI.

**Zsh:**
```bash
fibe completion zsh > "${fpath[1]}/_fibe"
```

**Bash:**
```bash
source <(fibe completion bash)
```


## Webhook filters

`EventFilters` and `ToolFilters` are keyed by actual event names returned by
`WebhookEndpoints.EventTypes` (`fibe webhooks event-types`). For example,
`ToolFilters: map[string][]string{"playground.created": {"deploy"}}` requires
that event's `payload.tool_name` to equal `deploy`. There is no
`mcp.tool.executed` event. Built-in lifecycle publishers currently omit tool names,
so tool filters on those events block delivery unless a publisher supplies one.

An omitted event key is unrestricted for that filter dimension; an explicit
empty array matches nothing. Both entity and tool restrictions must match.
Unknown event keys and blank tool names are rejected with HTTP 422. Use
`--from-file` JSON to set empty arrays from the CLI.


API-key resource restrictions distinguish omission from an empty list. Omit a
`GranularScopes` map entry to use that scope's normal ownership access; provide
`[]int64{}` to allow none. A nil slice encodes `null` and is invalid. In the CLI,
`--granular-scope monitor:read=` allows no monitored agents, while
`--granular-scope agents:read=12,15` selects those IDs. Repeat the flag for other
scopes. Rails returns normalized restrictions; the current compatibility-frozen
Go `APIKey` response struct ignores that additive response field.
