# MCP edition discovery

`fibe mcp serve` uses `--edition auto` by default. The stdio and HTTP
transports make an anonymous GET to the API server’s `/route-manifest.json`
with a short timeout. The process caches the result, including failures, for
each domain. Switching profiles or API domains selects the cached result or
probes the new domain.

When the server reports Community, MCP filters the selected `--tools` tiers
by the manifest’s HTTP methods and route templates. The tool catalog, schema,
and help identify the Community edition and omit operations proven absent.
Tools and operations with an uncertain route stay available. Generic tools
refuse known unavailable operations before sending an API operation request.

Enterprise, older hosted servers, and servers with an unreadable manifest
offer the complete configured tiers. To use that same surface with Community,
set `FIBE_MCP_EDITION=full` or pass the flag:

```sh
fibe mcp serve --tools core --edition full
fibe mcp install --client codex --profile staging --edition full
```

A stdio install records the selected mode in the client’s MCP configuration.
For URL-backed installs, set the mode when starting the HTTP MCP server.
