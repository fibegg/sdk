# Release checklist

- Confirm the tree is clean and the changelog contains the release notes.
- Run `mage check` with the pinned Go toolchain and tools.
- Run opted-in live tests only against disposable, namespaced resources.
- Verify Linux, macOS, and Windows builds on amd64 and arm64.
- Run GoReleaser in snapshot mode and inspect archives, checksums, and SBOMs.
- Tag the reviewed commit using semantic versioning.
- Verify the published archive installs and `fibe version`/`fibe help` work.
- Validate checksums, SBOMs, provenance, and one mocked SDK/CLI request.
- Confirm Homebrew publication. Notification failure must not invalidate
  already published artifacts.
