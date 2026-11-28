# Fibe SDK and CLI

The official Go SDK and command-line interface for the Fibe platform.

The SDK uses the domain names Hosts, Specs, Repositories, Tasks and Teams.
Agent, Audit Log, conversation, document and memory operations keep their model
identities. Candidate binaries of unreleased source are built from matched
source without release tags. See [`COMPATIBILITY.md`](COMPATIBILITY.md) for the
compatibility contract.

Requires Go 1.26.11 or newer when building from source. See
[`COMPATIBILITY.md`](COMPATIBILITY.md) for the stable contract and
[`SECURITY.md`](SECURITY.md) for private vulnerability reporting.

## SDK integration and E2E tests

Set `FIBE_DOMAIN` to the API URL of a disposable test environment and
`FIBE_API_KEY` to an explicitly scoped key for that environment.

| Target edition | Command | Live coverage |
| --- | --- | --- |
| Community | `mage testCommunity` | Shared cases |
| Enterprise | `mage testEnterprise` | Shared and Enterprise-only cases |

See [test editions](TEST_EDITIONS.md) for offline listings and fixture requirements.

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

Use separate profiles for different API targets. For the example below, set
`STAGING_API_URL` and `STAGING_API_KEY` for your test environment.

```bash
fibe auth login --profile staging --domain "$STAGING_API_URL" --api-key "$STAGING_API_KEY"
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

fibe pg create --name demo --spec starter --host next --service web.subdomain=demo
cat payload.yml | fibe pg create -f -
fibe pg create < payload.yml

fibe pg get demo
fibe pg get demo -o json --only service_urls

fibe wait playground demo --status running --timeout 5m
```

Waiting for a running Playground checks service readiness by default: the requested
configuration must be applied, builds must have no warnings, long-running services
must be ready, and one-shot dependencies must have exited successfully. Use
`--readiness lifecycle` when you only need the lifecycle status. For a slow
startup, increase `--timeout` to cover the service's Compose healthcheck budget.
Inspect deployment failures with
`fibe playgrounds debug <id-or-name> --build-logs --output json`. New builds retain
their complete logs; older truncated logs cannot be reconstructed.

`wait --timeout` bounds the entire wait, including blocked HTTP requests, final
resource reads and polling sleeps. `--timeout` and `--interval` must be positive.
Cancelling the command cancels its requests and wait, preserving a typed context
error; it does not cancel work already running on the server. Created-Playground
waits and SDK async polling likewise give their requests the polling deadline.

`launch` rejects explicitly supplied flags that its selected source cannot apply:

| Source | Additional supported flags |
| --- | --- |
| Existing Spec | `--service` runtime overrides; use these instead of `--var`, `--env`, `--subdomain` or `--persist-volumes` |
| Template | `--version`, `--var`, `--env`, `--subdomain`, `--persist-volumes`, `--service` |
| Exact template version | Same overrides as Template, without `--version` |
| Compose | Template-style overrides plus `--repository`, `--job-mode`, `--create-playground` or `--no-create-playground` |
| Repository | Compose options plus `--file`, `--ref`, `--github-account`, `--github-installation-id` |

Common `--name`, `--host`, `--env-pack-attachments` and `--wait` options remain
available. `--wait` requires Playground creation. The two creation flags cannot
be supplied together. Supported boolean flags preserve explicit false values.
`--repository REPOSITORY=ID_OR_NAME` requires nonempty, unique keys and positive
numeric targets. ID-only arguments in job ENV, installations, template versions,
webhooks, feedback and API-key deletion reject invalid, zero, negative and
overflowing IDs before making a request. Names remain supported where the command
advertises an ID-or-name argument.

Text options documented as `@path` read that file or return an error before the
request; they no longer fall back to sending an unreadable `@path` literally.
Inline text and readable files retain their existing behavior. For `launch`,
`--from-file`/redirected regular-file input supplies the Compose source body;
it is not a generic launch-options payload. Pipes require explicit `--from-file -`.

Repository mirroring is asynchronous. `fibe wait repository <id-or-name>` waits for
`mirror_status: completed` and reports the persisted failure reason. The Go SDK
exposes these diagnostics through `Playgrounds.RuntimeStatusByIdentifier`,
`Playgrounds.DebugWithBuildLogsByIdentifier`, and
`Repositories.GetMirrorStateByIdentifier`, preserving existing response struct layouts.

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
playground, and process capabilities are denied over HTTP by default: `fibe_run`,
`fibe_auth_list`, `fibe_auth_use`, `fibe_memorize`, every `fibe_local_*` tool, and
any call that passes a local path argument. A loopback-only deployment may opt in
with `--allow-local-access`; stdio retains the existing local tools.

`fibe_run` starts the CLI with the session's credentials. When the session uses a
profile (the server's `--profile` or `fibe_auth_use`), the child gets
`--profile <name>`. Any other key (`--api-key`, `FIBE_API_KEY`, `fibe_auth_set` or
an HTTP bearer) reaches the child through its `FIBE_MCP_RUN_API_KEY` environment
variable, never argv or the tool result. `--profile`, `--api-key` or `--domain`
passed in `fibe_run` args take precedence.

### Tool surface

The server provides generic resource tools such as `fibe_resource_list`, `fibe_resource_get`, `fibe_resource_delete`, `fibe_resource_mutate`, and `fibe_resource_watch`, plus workflow tools such as `fibe_greenfield_create` and `fibe_launch`.

Agent status, attachments, and scheduled pokes use the resource tools. Set `params.include_runtime_status` when listing agents. Use the `agent.upload_attachment`, `agent_attachment`, and `agent_poke` resource operations for files and pokes. To inspect a Playground, call `fibe_resource_get` with `resource:"playground"` and `id_or_name:"..."`; use `fibe_playgrounds_debug` for raw Compose, route, and log details. Spec jobs use `spec.create` or `spec.update`. `fibe_schema` documents and locally validates mutation payloads.

Use `--tools core` or `FIBE_MCP_TOOLS=core` to select the
meta, base, greenfield and brownfield tiers. The default `--tools full` selects
every tier. A comma-separated list such as `--tools other,meta` selects named
tiers. Team operations use the `other` tier. See [Team operations](docs/team-operations.md).

By default, `--edition auto` reads the API server’s public route manifest.
For Community, MCP advertises the tools backed by available routes and
reports the edition in `fibe_tools_catalog`, `fibe_schema`, and `fibe_help`.
For Enterprise and servers without a readable manifest, MCP offers the
complete configured tiers. Set `--edition full` or `FIBE_MCP_EDITION=full`
on `fibe mcp serve` or `fibe mcp install` to use the complete configured
tiers on Community as well. See [MCP edition discovery](docs/mcp-editions.md).

Use `fibe_tools_catalog` to inspect `advertised` and `hidden` flags for a
running server. Available tools outside the advertised tiers can be invoked
through `fibe_call` and `fibe_pipeline`. Regenerate the offline catalogs,
which describe the full SDK surface, with:

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
    {"id": "pg",   "tool": "fibe_resource_mutate", "args": {"resource": "playground", "operation": "create", "payload": {"name": "ci", "spec_id": 5}}},
    {"id": "wait", "tool": "fibe_playgrounds_wait",   "args": {"id_or_name": "$.pg.id", "status": "running"}},
    {"id": "logs", "tool": "fibe_playgrounds_logs",   "args": {"id_or_name": "$.pg.id", "service": "web", "tail": 100}}
  ],
  "return": "$.logs.lines"
}
```

Use `parallel` for independent steps and `for_each` for arrays. Results stay in the session cache for five minutes and can be queried through `fibe_pipeline_result` without rerunning the pipeline.

### Streaming

`fibe_playgrounds_wait` and `fibe_logs_follow` send MCP progress notifications. Long SDK operations do the same when the client supplies a progress token. Use `fibe playgrounds logs <id-or-name> --follow` or `fibe tasks logs <id-or-name> --follow` for continuous Playground or Task logs from the CLI.

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
scopes. The Go `APIKey` response struct ignores normalized restrictions when they
are included in the API response.

### Local Playground link directory

The CLI and MCP local Playground tools use `PLAYGROUNDS_DIR` for links, current
selection, repository paths and screenshot artifacts. Agent startup exports an
absolute path beneath its persistent data directory so provider tools and the
file explorer share one location even when their working directories differ.
An explicit `--link-dir` (MCP `link_dir`) takes precedence. Without either, the
standalone CLI keeps `/app/playground` for compatibility.

The link directory's parent must be writable by the agent user: linking stages
and atomically replaces the whole directory. A writable `/app/playground` under
a root-owned `/app` is insufficient. Use a directory under writable persistent
data rather than changing `/app` ownership or permissions.
