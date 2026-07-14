# Architecture

The repository has three supported entry points:

- `fibe/` is the public Go SDK. A client owns transport, auth, retry,
  idempotency, rate limiting, circuit breaking, and service instances.
- `cmd/fibe/` is the CLI. It preserves command/flag/output contracts while
  delegating remote operations to the SDK and local operations to focused
  internal packages.
- `internal/mcpserver/` exposes the same capabilities over MCP. A central
  dispatcher applies authentication, tenant isolation, safety metadata,
  local-capability policy, audit redaction, and pipeline limits.

`internal/resourceschema/` is shared by CLI and MCP validation.
`internal/localconversations/` and `internal/localplaygrounds/` isolate local
filesystem behavior. `fibetest/` supplies a hermetic mock server.

Compatibility artifacts in `contracts/` are the translation boundary for a
future TypeScript implementation. Generated MCP docs are derived from the
runtime registry; the Docker Compose schema remains separate.
