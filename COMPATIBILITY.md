# Compatibility policy

The public compatibility floor is `v0.2.45`; the hardening baseline also
freezes the declarations and command/tool surfaces recorded under `contracts/`.

Within the current major version, releases preserve exported Go names and
signatures, struct field ordering, JSON tags, public interfaces, HTTP wire
shapes, CLI commands and flags, and MCP tool names and schemas. Additive APIs
are allowed. Deprecations remain functional until a separately versioned
breaking release.

Security defects, crashes, resource leaks, unsafe filesystem behavior,
credential exposure, malformed-input acceptance, and plainly erroneous retry
behavior may be corrected. Such fixes are documented and covered by contract
tests.

Any additional SDK implementation must execute the same language-neutral
fixtures. Docker Compose schemas are a separate product contract.
