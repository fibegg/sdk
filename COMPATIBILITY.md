# Compatibility policy

The v0.3 candidate implements the coordinated domain-name break: Marquee →
Host, Playspec → Spec, Prop → Repository, and Trick → Task. Library replaces
Scrolls; Agent and Audit Logs retain their canonical model identities. Old
commands, fields, routes and resource names fail with replacement guidance.

`contracts/go-public-api-v0.2.45.json` remains historical release evidence.
The compatibility check permits only the approved domain rename; unrelated
exported signatures and struct layouts remain checked. The current generated
Go, REST, CLI and MCP contracts describe the candidate surface.

A production 0.2.x hotfix must come from a branch at the last released tag.
The candidate CLI is built without tags as `0.3.0-rc.1+<sdk-sha7>`; it does not
publish a GitHub release or update Homebrew.

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
