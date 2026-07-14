# Governance and releases

Fibe maintainers steward the repository, review changes, enforce compatibility
and security policy, and cut releases. Decisions favor user safety, stable
contracts, maintainability, and evidence from tests and benchmarks.

Changes are reviewed through pull requests. A maintainer who authored a
security-sensitive or contract-sensitive change should seek another review
when practical. Maintainers may revert a change that breaks the compatibility
corpus or creates operational risk.

Releases follow semantic versioning. A release is made from a clean tagged
commit after `mage check`, cross-build, release verification, and any explicitly
enabled live tests pass. Artifacts include checksums, SBOMs, and provenance.
The checklist in `RELEASE_CHECKLIST.md` is authoritative.
