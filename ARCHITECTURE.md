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

Local Playground discovery is a consumer of the Core runtime contract. A
present `.fibe-source-plan.json` is authoritative for normalized repository,
exact branch, relative checkout path, member services, production status, and
mount target. The SDK resolves those declared paths and deduplicates physical
checkouts; it never derives or owns the `/opt/fibe` layout. Compose mount
inspection and historical Prop-shaped path parsing are isolated compatibility
fallbacks for pre-manifest Playgrounds only. Human-facing link names preserve
the established rule—repository/Prop name for one branch, branch suffixes on
all links when more than one branch is present—with deterministic owner/digest
qualification for collisions.

Compatibility artifacts in `contracts/` are the language-neutral boundary for
other SDK implementations. Generated MCP docs are derived from the runtime
registry; the Docker Compose schema remains separate.
