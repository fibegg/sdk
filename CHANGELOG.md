# Changelog

All notable changes to the Fibe SDK (`sdk`) and CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.2] - 2026-10-16

### Fixed

- Local Playground discovery and linking no longer classify a long-running
  environment as a Task because one setup service has `fibe.gg/job_watch=true`.
  Project job mode comes from `x-fibe.gg` runtime metadata; service watch flags
  remain available, and actual Tasks remain hidden from the picker and blocked
  from linking. This fixes mixed environments such as Joan.

## [0.3.0] - 2026-10-15

### Changed

- **Breaking:** coordinated domain rename across the SDK, CLI and MCP server:
  Marquee is now Host, Playspec is now Spec, Prop is now Repository and Trick
  is now Task (Library replaces Scrolls). Commands, flags, fields, SDK services
  and types, MCP tools and environment variables (`HOST_ROOT`,
  `HOST_ROOT_DOMAIN`, `HOST_URL_SCHEME`, `FIBE_HOST_ID`, `FIBE_SPEC_ID`,
  `FIBE_REPOSITORY_ID`) use the new names. Old names fail with replacement
  guidance and no aliases. This release targets installations on the new
  contract; installations still on the previous vocabulary need a 0.2.x CLI.
- **Breaking:** Company is now Team. `fibe company --company-id` is
  `fibe team --team-id`; the twelve `fibe_company_*` MCP tools are
  `fibe_team_*` with a required `team_id`; SDK `Company*` methods and types are
  `Team*`; Repository operations poll `/api/team_repository_operations` and
  decode `team_repository_operation_id`. Old names fail with replacement
  guidance and no aliases.
- ENV pack attachments are per Playground. Spec attachments apply only to
  job-mode Specs (packs for scheduled runs); a non-empty attachment list on a
  service Spec is rejected with 422 `env_pack_spec_job_mode_required`.

### Security

- Require Go 1.26.9 for its standard-library security fixes and pin the E2E
  and candidate CLI build image to that patch release.

### Added

- `fibe playgrounds rerun <id-or-name>` creates a new Playground from an existing
  one through the SDK `Playgrounds.Rerun`, with optional `--name` and
  `--from-file` ENV pack attachment selection (nil preserves the source's
  references, `[]` detaches all). It mirrors `fibe tasks rerun`.
- Live integration coverage for ENV packs across the SDK, CLI binary and MCP
  server (`integration/env_packs_test.go`).
- Team owners, ENV packs and Team repository recovery operations across the
  SDK, CLI and MCP server, including GitHub installation selection.

### Removed

- Product-specific example skill names in CLI help and tests (`fibe-hunks.md`)
  in favour of a neutral `example-skill.md`.

## [0.2.46] - 2026-10-01

- Use Homebrew's default download URL verification, supported by current GoReleaser.

### Added

- Language-neutral public API, CLI, MCP, and reliability contracts for
  additional SDK implementations.
- Security, governance, contribution, support, compatibility, architecture,
  release, SBOM, provenance, and pinned quality-gate infrastructure.

### Changed

- Hardened request execution, retry/idempotency, rate limiting, circuit
  breaking, async polling, multipart upload, downloads, WebSockets, webhooks,
  local stores, MCP authentication, pipelines, audit logging, and filesystem
  operations without changing supported public contracts.
- Updated the minimum toolchain to Go 1.26.8 and upgraded pinned dependencies.

### Fixed

- Repeatable multi-ID granular scopes and explicit empty resource restrictions.
- Current Codex model defaults in credentialed runtime test profiles and help.
- Stable automatic idempotency across retries, negative retry handling,
  bounded body decoding, response draining, same-origin async polling, MCP
  tenant isolation, destructive-command classification, and unsafe local path
  edge cases.

## [0.2.45] - 2026-06-18

### Added

- Local path discovery commands and expanded Git-backed launch integration
  coverage.

### Changed

- Improved local playground linking, repository fixtures, agent defaults, and
  E2E orchestration.

## [0.2.44] - 2026-06-17

### Fixed

- Corrected local playground link discovery and related MCP behavior.

## [0.2.43] - 2026-06-16

### Added

- Progress events and interactive CLI spinners for async operations, waits,
  greenfield creation, launches, and template/playground actions.
- MCP forwarding for SDK progress notifications.

## [0.2.42] - 2026-06-15

### Changed

- Improved `fibe_run` guidance when agents use stale resource syntax.

### Added

- Pipe-hygiene coverage for the MCP stdio transport.

## [0.2.41] - 2026-06-14

### Changed

- Refreshed generated MCP tool documentation and made regeneration
  deterministic.

## [0.2.40] - 2026-06-13

### Added
- **MCP server (`fibe mcp`)**: The `fibe` binary now doubles as a local [Model Context Protocol](https://modelcontextprotocol.io) server so LLM agents can drive Fibe without paying the `fork+exec` cost of invoking the CLI per operation.
  - **`fibe mcp serve`**: stdio transport (default), SSE (`--http :port`), or streamable-HTTP (`--http :port --streamable`).
  - **60 registered dispatcher tools** with a curated agent workflow surface. The default/full tool surface advertises 59 non-hidden tools, while `--tools core` / `FIBE_MCP_TOOLS=core` advertises the 39 meta/base/greenfield/brownfield tools.
  - **`fibe_pipeline`**: compose multiple tool calls in one round-trip with JSONPath bindings, `parallel` blocks, `for_each` fanout, `dry_run` validation. Results cached per session for 5 minutes under a `pipeline_id`; re-query via `fibe_pipeline_result` with a JSONPath projection.
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
