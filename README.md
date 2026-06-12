# Fibe SDK & CLI

The official Go SDK and command-line interface for the Fibe platform.

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
fibe me
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

# Watch agent resource events through AnyCable
fibe agents watch --max-events 5 --duration 1m

# Create a playground from an existing Playspec and override one service field
fibe pg create --name demo --playspec starter --marquee next --service web.subdomain=demo

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
		fibe.WithAPIKey("pk_live_yourkeyhere"),
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

1. **Auto rate-limit retry**: When your workload hits HTTP `429 Too Many Requests`, the SDK will sleep the interval specified in `Retry-After`.
2. **Circuit Breaking**: Failed requests trigger in-memory isolations ensuring that backends do not get DDos'd by your local requests.
3. **Idempotency**: Retries send identical keys to the server (`X-Idempotent-Key`) mitigating unintended double writes.

## MCP Server

The same `fibe` binary also runs as a local [Model Context Protocol](https://modelcontextprotocol.io) server so LLM agents can drive Fibe without paying the `fork+exec` cost of invoking the CLI per operation.

```bash
# Register Fibe with your MCP client (claude-code | claude-desktop | cursor | vscode | antigravity | codex)
fibe mcp install --client claude-code
fibe mcp install --client claude-code --project . # project-root .mcp.json
fibe mcp install --client codex --profile staging

# Run the server manually (stdio, single-tenant, profile-backed)
fibe mcp serve --profile staging

# Serve multiple tenants over SSE with per-request bearer auth
fibe mcp serve --http :8080 --require-auth

# Emit a remote MCP entry for clients that support URL-backed servers
fibe mcp install --client antigravity --transport streamable-http --url https://fibe.example.com/mcp
```

Warning: `fibe mcp serve --http` is intended for trusted local/admin deployments. Do not expose it to untrusted remote callers.

### Tool surface

The server registers a curated tool catalog for agent workflows, with generic resource tools such as `fibe_resource_list`, `fibe_resource_get`, `fibe_resource_delete`, `fibe_resource_mutate`, and `fibe_resource_watch` plus high-value actions such as `fibe_greenfield_create` and `fibe_templates_launch`. Agent list/runtime, attachment, and scheduled poke flows use those generic resource tools: list agents through `fibe_resource_list` with `params.include_runtime_status`, upload attachments through `fibe_resource_mutate` using `agent.upload_attachment`, download runtime files through `fibe_resource_get` using `agent_attachment`, and manage scheduled pokes through the `agent_poke` resource aliases `agent_pokes` and `pokes`. Playspec job automation uses `fibe_resource_mutate` with `playspec.create` or `playspec.update`; inspect `fibe_schema(resource:"playspec", operation:"create")` for `schedule_config`, `trigger_config`, and `muti_config` payload fields. Mutation payload schemas are available through `fibe_schema` and are validated locally before API calls. `FIBE_MCP_TOOLS=full` exposes the full registered catalog; set `FIBE_MCP_TOOLS=core` for a smaller curated subset plus always-visible meta tools.

Safety annotations match MCP hints: `readOnlyHint` on reads, `destructiveHint` on delete/rollout/hard-restart. Destructive tools require `confirm:true` in their args unless the server is launched with `--yolo` (or `FIBE_MCP_YOLO=1`) for non-interactive environments.

### Pipeline composition

`fibe_pipeline` composes multiple tool calls in one round-trip using JSONPath bindings, eliminating the need for the LLM to shuttle intermediate payloads between turns:

```json
{
  "steps": [
    {"id": "pg",   "tool": "fibe_resource_mutate", "args": {"resource": "playground", "operation": "create", "payload": {"name": "ci", "playspec_id": 5}}},
    {"id": "wait", "tool": "fibe_playgrounds_wait",   "args": {"id": "$.pg.id", "status": "running"}},
    {"id": "logs", "tool": "fibe_playgrounds_logs",   "args": {"id": "$.pg.id", "service": "web", "tail": 100}}
  ],
  "return": "$.logs.lines"
}
```

Supports `parallel` blocks for concurrent independent steps and `for_each` for fanout over arrays. Results are cached per session for 5 minutes and re-queryable via `fibe_pipeline_result` with a JSONPath projection — the LLM can keep referencing fields from a big pipeline result without re-running it.

### Streaming

`fibe_playgrounds_wait`, `fibe_monitor_logs_follow`, and the compatibility `fibe_playgrounds_logs_follow` stream updates as MCP progress notifications, letting agents delegate "poll until X" loops to the server instead of burning round-trips. CLI users can run `fibe monitor logs <id-or-name>` for continuous playground or trick logs.

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

1. `Authorization: Bearer <fibe-api-key>` header
2. A prior `fibe_auth_set` tool call in the same session
3. The server-wide profile/API-key fallback (disabled with `--require-auth`)

Each session gets its own `*fibe.Client` instance with isolated circuit-breaker and rate-limit state — one tenant's errors can't open another tenant's breaker.

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
