# Fibe SDK & CLI

The official Go SDK and command-line interface for the Fibe platform.

Requires Go 1.26.5 or newer when building from source. See
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

## CLI Usage Highlights

The CLI acts as a human-readable entrypoint or an automated integration for LLM agents.

```bash
# General details
fibe doctor
fibe status
fibe auth status
fibe server-info   # server UTC clock + build identity (unauthenticated)

# View JSON schemas for commands (useful for LLM Agents)
fibe schema list
fibe schema agent create

# Create an agent
fibe agents create --name "My Assistant" --provider "claude-code"

# Send one chat turn by agent name
fibe agent chat my-agent "Fix the failing tests"
fibe agent chat my-agent - < prompt.md

# List agents with bounded runtime status for the returned page
fibe agents list --include-runtime-status --per-page 100 -o json

# Work with runtime chat attachments through Rails
fibe agents upload-attachment my-agent --file ./context.zip
fibe agents download-attachment my-agent runtime-context.zip --to ./context.zip

# Watch agent resource events
fibe agents watch --max-events 5 --duration 1m

# Create a playground from an existing Playspec and override one service field
fibe pg create --name demo --playspec starter --marquee next --service web.subdomain=demo
cat payload.yml | fibe pg create -f -   # explicit stdin
fibe pg create < payload.yml           # file redirection

# Inspect service URLs and runtime service status
fibe pg get demo
fibe pg get demo -o json --only service_urls

# Block until a playground starts running by name or ID
fibe wait playground next --status running --timeout 5m
```

## Output Formatting

Use `-o` to change formatting:

```bash
fibe status -o json
fibe status -o yaml
```

You can filter payloads as well using `--only`
```bash
fibe agents list --output yaml --only "id,name"
```

## Go SDK Usage

To interact with the Fibe API from Go:

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

	// Fetch account status
	status, err := client.Status.Get(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Active Playgrounds: %d\n", status.Playgrounds.Active)
}
```

### Reliable Features Included

1. **Bounded retries and rate-limit waiting**: Retryable responses, including HTTP `429 Too Many Requests`, use bounded `Retry-After` or exponential-backoff delays. `fibe.WithRateLimitAutoWait()` also waits before a request when the client's last response showed that its known quota was exhausted.
2. **Optional circuit breaking**: Enable `fibe.WithCircuitBreaker(...)` to stop sending requests after the configured number of transient failures and probe again after its reset interval.
3. **Idempotency**: Mutating requests send an `Idempotency-Key` header. One automatic key is generated per logical operation and reused across all SDK retries. `fibe.WithIdempotencyKey(ctx, key)` overrides it when a caller needs a stable key across separate calls.
4. **Progress hooks**: Long-running SDK operations emit `fibe.ProgressEvent` values through `fibe.WithProgress(...)`. The CLI renders these as single-line spinners in interactive terminals and keeps line-based status output for non-interactive scripts.

Retryable reads and idempotency-protected mutations handle transient transport
failures and retryable server responses. Cancellation, expired deadlines,
validation errors, and non-replayable request bodies are not retried after
transmission begins. Rate-limit and circuit-breaker state belong to one logical
client/session.

## MCP Server

The same `fibe` binary also runs as a local [Model Context Protocol](https://modelcontextprotocol.io) server so LLM agents can drive Fibe without paying the `fork+exec` cost of invoking the CLI per operation.

```bash
# Register Fibe with your MCP client (claude-code | claude-desktop | cursor | vscode | antigravity | codex)
fibe mcp install --client claude-code
fibe mcp install --client claude-code --user # user-level ~/.claude.json instead of project .mcp.json
fibe mcp install --client codex --profile staging

# Run the server manually (stdio, single-tenant, profile-backed)
fibe mcp serve --profile staging

# Serve multiple tenants over SSE with per-request bearer auth
fibe mcp serve --http :8080 --require-auth

# Emit a remote MCP entry for clients that support URL-backed servers
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

The server registers a curated tool catalog for agent workflows, with generic resource tools such as `fibe_resource_list`, `fibe_resource_get`, `fibe_resource_delete`, `fibe_resource_mutate`, and `fibe_resource_watch` plus high-value actions such as `fibe_greenfield_create` and `fibe_launch`. Agent list/runtime, attachment, and scheduled poke flows use those generic resource tools: list agents through `fibe_resource_list` with `params.include_runtime_status`, upload attachments through `fibe_resource_mutate` using `agent.upload_attachment`, download runtime files through `fibe_resource_get` using `agent_attachment`, and manage scheduled pokes through the `agent_poke` resource aliases `agent_pokes` and `pokes`. Inspect Playground URLs and services through `fibe_resource_get` with `resource:"playground"` and `id_or_name:"..."`; the detailed response includes `service_urls` and `services`, while `fibe_playgrounds_debug` remains the deeper diagnostics surface for raw compose/routes/log context. Playspec job automation uses `fibe_resource_mutate` with `playspec.create` or `playspec.update`; inspect `fibe_schema(resource:"playspec", operation:"create")` for `schedule_config`, `trigger_config`, and `muti_config` payload fields. Mutation payload schemas are available through `fibe_schema` and are validated locally before API calls.

The generated registry docs currently list 60 registered dispatcher tools. By default, `fibe mcp serve` uses the `full` tool surface and advertises the 59 non-hidden tools. Use `--tools core` or `FIBE_MCP_TOOLS=core` to narrow the native surface to the 39 meta/base/greenfield/brownfield tools, or pass a comma-separated tier list such as `--tools other,meta`. Hidden tools are not advertised natively even in `full`, but remain dispatcher-reachable through `fibe_call` and `fibe_pipeline` when the caller already knows the tool name. Use `fibe_tools_catalog` to inspect `advertised` and `hidden` flags for a running server. Regenerate `fibe_mcp_tools_catalog.md` and `fibe_tools_table.md` deterministically from the Go MCP registry with:

```bash
go run ./scripts/docs
go run ./scripts/docs --check
```

Safety annotations match MCP hints: `readOnlyHint` on reads, `destructiveHint` on delete/rollout/hard-restart. Destructive tools require `confirm:true` in their args unless the server is launched with `--yolo` (or `FIBE_MCP_YOLO=1`) for non-interactive environments.

### Pipeline composition

`fibe_pipeline` composes multiple tool calls in one round-trip using JSONPath bindings, eliminating the need for the LLM to shuttle intermediate payloads between turns:

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

Supports `parallel` blocks for concurrent independent steps and `for_each` for fanout over arrays. Results are cached per session for 5 minutes and re-queryable via `fibe_pipeline_result` with a JSONPath projection — the LLM can keep referencing fields from a big pipeline result without re-running it.

### Streaming

`fibe_playgrounds_wait` and `fibe_logs_follow` stream updates as MCP progress notifications, letting agents delegate "poll until X" loops to the server instead of burning round-trips. Long-running SDK-backed operations, including async request polling and template-switch rollout waits, also forward progress notifications when the MCP client provides a progress token. CLI users can run `fibe logs follow <id-or-name>` for continuous playground or trick logs.

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

- `fibe_auth_list` — list local profiles without exposing API keys
- `fibe_auth_use` — switch this MCP session to another profile
- `fibe_auth_status` — show the current MCP auth target
- `fibe_auth_set` — advanced raw API key/domain override

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

### Audit Log

`FIBE_MCP_AUDIT_LOG` is experimental. It writes one JSON line per tool call for debugging/admin use, and its schema and redaction behavior may evolve.

## Shell Completions

You can generate shell completions natively using the CLI.

**Zsh:**
```bash
fibe completion zsh > "${fpath[1]}/_fibe"
```

**Bash:**
```bash
source <(fibe completion bash)
```
