# Compatibility contracts

This directory is the language-neutral compatibility boundary for the Go SDK
and any additional SDK implementation.

Team recovery and durable operation parity is additive to the candidate.
See `docs/team-operations.md` for explicit Team owner proofs, nil versus
empty recovery selection, replay/conflict and accepted cancellation semantics.
Public response types intentionally omit runtime key identities and token bytes.

- `go-public-api.json` freezes every exported Go type, field order, JSON tag,
  constant, function, method, and public `Unwrap` signature for the current
  candidate.
- `go-public-api-v0.2.45.json` is unmodified generator output from tag `v0.2.45`.
  The historical compatibility test projects the approved v0.3 domain renames
  in memory and checks removals and source-signature/layout changes outside
  explicitly approved additive feature fields. New declarations remain additive.
- `decoder-fixtures.json` holds current Playground, source-reference and Agent
  settings examples consumed by the actual SDK response and CLI file decoders.
- `reliability.json` records cross-language transport behavior that cannot be
  inferred from endpoint examples alone.
- `rest.json` inventories every exported service method, its HTTP method/path
  expression, codec, request/response types, delegation, shared headers, query
  encoding, limits, and custom JSON types.
- `cli-tree.json` and `mcp-tools.json` freeze the complete CLI and MCP public
  catalogs, including ordering and schemas.

Regenerate the current manifest with:

```sh
go run ./internal/cmd/apimanifest
go run ./internal/cmd/restcontract
```

The manifest test fails if regeneration changes tracked output. REST, CLI, and
MCP golden fixtures remain executable tests; generated documentation must not
be treated as the Docker Compose schema, which is a separate product surface.
