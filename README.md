# Fibe SDK & CLI

The official Go SDK and command-line interface for the Fibe platform.

## Install the CLI

### Homebrew

```bash
brew install fibegg/sdk/fibe
```

If you prefer the two-step flow:

```bash
brew tap fibegg/sdk
brew install fibe
```

The executable name is still `fibe`.

### Go install

```bash
go install github.com/fibegg/sdk/cmd/fibe@latest
```

## Setup

Set your API key via an environment variable. Obtain a key at `https://fibe.gg/settings/api-keys`.

```bash
export FIBE_API_KEY="pk_live_yourkeyhere"
```

You can optionally specify `--api-key` in the CLI commands.

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

# Block until playground starts running
fibe wait playground 42 --status running --timeout 5m
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

# Run the server manually (stdio, single-tenant)
FIBE_API_KEY=pk_live_... fibe mcp serve

# Serve multiple tenants over SSE with per-request bearer auth
fibe mcp serve --http :8080 --require-auth

# Emit a remote MCP entry for clients that support URL-backed servers
fibe mcp install --client antigravity --transport streamable-http --url https://fibe.example.com/mcp
```

Warning: `fibe mcp serve --http` is intended for trusted local/admin deployments. Do not expose it to untrusted remote callers.

### Tool surface

The server registers ~100 tools that map 1:1 to CLI leaf commands — `fibe_playgrounds_list`, `fibe_playgrounds_get`, `fibe_launch`, `fibe_tricks_trigger`, etc. Input schemas are derived from the SDK's `*Params` structs so agents get type-checked arguments. `FIBE_MCP_TOOLS=full` is the default parity surface; set `FIBE_MCP_TOOLS=core` for a smaller curated subset plus always-visible meta tools.

Safety annotations match MCP hints: `readOnlyHint` on reads, `destructiveHint` on delete/rollout/hard-restart. Destructive tools require `confirm:true` in their args unless the server is launched with `--yolo` (or `FIBE_MCP_YOLO=1`) for non-interactive environments.

### Pipeline composition

`fibe_pipeline` composes multiple tool calls in one round-trip using JSONPath bindings, eliminating the need for the LLM to shuttle intermediate payloads between turns:

```json
{
  "steps": [
    {"id": "pg",   "tool": "fibe_playgrounds_create", "args": {"name": "ci", "playspec_id": 5}},
    {"id": "wait", "tool": "fibe_playgrounds_wait",   "args": {"id": "$.pg.id", "status": "running"}},
    {"id": "logs", "tool": "fibe_playgrounds_logs",   "args": {"id": "$.pg.id", "service": "web", "tail": 100}}
  ],
  "return": "$.logs.lines"
}
```

Supports `parallel` blocks for concurrent independent steps and `for_each` for fanout over arrays. Results are cached per session for 5 minutes and re-queryable via `fibe_pipeline_result` with a JSONPath projection — the LLM can keep referencing fields from a big pipeline result without re-running it.

### Streaming

`fibe_playgrounds_wait` and `fibe_playgrounds_logs_follow` stream updates as MCP progress notifications, letting agents delegate "poll until X" loops to the server instead of burning round-trips.

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

### Multi-tenant auth

Stdio transport is single-tenant by design (one process per client). For HTTP/SSE deployments serving multiple tenants, the server resolves credentials per request in this order:

1. `Authorization: Bearer <fibe-api-key>` header
2. A prior `fibe_auth_set` tool call in the same session
3. The server-wide `FIBE_API_KEY` fallback (disabled with `--require-auth`)

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
