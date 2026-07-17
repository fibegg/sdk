# Compatibility contracts

This directory is the language-neutral compatibility boundary for the Go SDK
and any additional SDK implementation.

- `go-public-api.json` freezes every exported Go type, field order, JSON tag,
  constant, function, method, and public `Unwrap` signature at the hardening
  baseline.
- `go-public-api-v0.2.45.json` is the minimum compatibility floor. CI rejects
  removal or source-signature changes while permitting additive declarations
  made after that tag.
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
