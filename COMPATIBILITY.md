# Compatibility policy

v0.3.0 (released 2026-10-15) implements the coordinated domain-name break:
Marquee → Host, Playspec → Spec, Prop → Repository, Trick → Task, and Company →
Team. Library replaces Scrolls; Agent and Audit Logs retain their canonical model
identities. Old commands, fields, routes and resource names fail with replacement
guidance. Installations that still serve the previous vocabulary require a 0.2.x
CLI until their own cutover.

`contracts/go-public-api-v0.2.45.json` remains historical release evidence.
The compatibility check permits only the approved domain rename; unrelated
exported signatures and struct layouts remain checked. The current generated
Go, REST, CLI and MCP contracts describe the v0.3 surface.

A 0.2.x hotfix must come from a branch at the `v0.2.48` tag. Candidate builds of
unreleased source are built without tags as `0.3.0-rc.1+<sdk-sha7>`; they do not
publish a GitHub release or update Homebrew. Tagged releases are published only
by the `v*` tag workflow.

`Dockerfile.e2e` owns the source-built candidate artifact. Pass the full
`FIBE_SDK_SHA` and matching `FIBE_CLI_VERSION`; `candidate-cli` retains a runnable
Linux build image. The `candidate-cli-export` target exports only `fibe`, using
`FIBE_CLI_GOOS` and `FIBE_CLI_GOARCH` for linux/amd64, linux/arm64 or darwin/arm64.
Build from a clean tracked-source snapshot, excluding private MCP configuration,
with the pinned Go image. Record each binary's checksum, target and full source
identity; preserve all three outputs separately. Use an explicitly selected,
bounded builder and serialize these builds. No release or global installation
is performed by either target.

Within the current major version, releases preserve exported Go names and
signatures, struct field ordering, JSON tags, public interfaces, HTTP wire
shapes, CLI commands and flags, and MCP tool names and schemas. Additive APIs
are allowed. The coordinated v0.3 rename accepts no compatibility aliases.

Security defects, crashes, resource leaks, unsafe filesystem behavior,
credential exposure, malformed-input acceptance, and plainly erroneous retry
behavior may be corrected. Such fixes are documented and covered by contract
tests.

Any additional SDK implementation must execute the same language-neutral
fixtures. Docker Compose schemas are a separate product contract.
