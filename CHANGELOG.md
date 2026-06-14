# Changelog

All notable changes to the Fibe SDK (`sdk`) and CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

_No unreleased changes documented yet._

## [0.2.40] - 2026-06-13

### Added
- **MCP server (`fibe mcp`)**: The `fibe` binary now doubles as a local [Model Context Protocol](https://modelcontextprotocol.io) server so LLM agents can drive Fibe without paying the `fork+exec` cost of invoking the CLI per operation.
  - **`fibe mcp serve`** — stdio transport (default), SSE (`--http :port`), or streamable-HTTP (`--http :port --streamable`).
  - **60 registered dispatcher tools** with a curated agent workflow surface. The default/full tool surface advertises 59 non-hidden tools, while `--tools core` / `FIBE_MCP_TOOLS=core` advertises the 39 meta/base/greenfield/brownfield tools.
  - **`fibe_pipeline`** — compose multiple tool calls in one round-trip with JSONPath bindings, `parallel` blocks, `for_each` fanout, `dry_run` validation. Results cached per session for 5 minutes under a `pipeline_id`; re-query via `fibe_pipeline_result` with a JSONPath projection.
  - **`idempotency_key`** on `fibe_pipeline` is threaded into per-step SDK contexts (sha256 of `key:step_id`) so destructive pipeline retries hit the server-side 24-hour idempotency cache.
  - **Streaming**: `fibe_playgrounds_wait` emits MCP progress notifications per poll tick, and `fibe_logs_follow` streams playground or trick log lines as notifications.
  - **`--yolo` / `FIBE_MCP_YOLO=1`** skips the `confirm:true` gate on destructive tools for non-interactive (CI) use.
  - **Multi-tenant auth** (HTTP transport): resolves API key from a prior session `fibe_auth_use` / `fibe_auth_set`, then `Authorization: Bearer …` or `X-Fibe-API-Key`, then the server-wide default. Each session caches its own `*fibe.Client` with isolated circuit-breaker state.
  - **Structured errors**: `*fibe.APIError` (`code`, `status`, `message`, `request_id`, `details`, `retry_after_seconds`, `idempotent_replayed`) survives intact through MCP tool-error results.
  - **Resources**: `fibe://me`, `fibe://status`, `fibe://schema`, `fibe://schema/{resource}`, `fibe://help/{path}`, `fibe://pipeline/schema`, `fibe://pipelines/{id}`.
  - **Audit log**: set `FIBE_MCP_AUDIT_LOG=/path/to/log.jsonl` (or `stderr`) for one JSON line per tool call with sensitive-arg redaction.
  - **Install helpers**: `fibe mcp install --client {claude-code,claude-desktop,cursor,vscode,antigravity,codex}` merges into existing configs non-destructively; `fibe mcp uninstall` removes cleanly; `fibe mcp config` prints the snippet.
  - Claude Code project installs now target project-root `.mcp.json`, matching Claude Code's project-scoped MCP config.
- **`Client.Playgrounds.LogStream` / `Client.Tricks.LogStream`**: structured ActionCable log streaming backed by the Rails live log channels. The older `LogsStream` line-only wrappers remain for compatibility.
- **`fibe monitor logs <id-or-name>`**: canonical CLI command for continuous playground and trick logs; `playgrounds logs --follow` and `tricks logs --follow` delegate to it.
- **`fibe github-repos create`** CLI command (parity with existing `fibe gitea-repos create`).
- **`fibe playspecs add-mounted-file` / `update-mounted-file` / `remove-mounted-file`** CLI commands.
- **`fibe playspecs add-registry-credential` / `remove-registry-credential`** CLI commands.
- **`fibe schema` command**: LLM agents can now reliably print JSON schemas of what parameters each resource expects.
- **`fibe wait` command**: Block until a playground reaches a specified status without implementing heavy retry loops in agents.
- **`fibe doctor` command**: Easily check SDK configuration, domain, version, and API connectivity.
- **Structured Error Output**: CLI errors are now fully structured JSON or YAML error objects containing an `error` struct with `code`, `message`, `details`, and `request_id`, instead of unstructured text output. Use the `--explain-errors` flag or run with `-o yaml`/`-o json`.
- **System-level properties**: `APIError` now includes `x-request-id` to debug API issues and `x-idempotent-replayed` to verify if a request resulted in cache reuse on the server.
- **Circuit Breaker Protection**: In-memory SDK rate limit tracking and automated circuit breakers are built into the client object for enterprise workloads.
- **Completion command**: Run `fibe completion bash|zsh|fish|powershell` to install shell conveniences.
- **Dashboard Data**: Fetch everything at once using the `/api/status` endpoint and `fibe status`.

### Changed
- **Cross-Service Pagination**: The pagination across all services has been unified to standard offsets via `page` and `per_page` query arguments instead of `limit` and `offset` everywhere.
- **Strong Webhook Security**: `VerifyWebhookSignatureWithMaxAge(r, secret, maxAge)` was added alongside `VerifyWebhookSignature` for replay protection when callers need a payload age limit.
- **HTTP Transport Stability**: Retrying an operation is more resilient now; response bodies are decoded with a 10 MB limit and drained to keep connections reusable.

### Removed
- Removed the experimental `gorilla/websocket` client stream since the standard REST endpoints natively return current playground statuses and history.
- Unnecessary debug flags.
