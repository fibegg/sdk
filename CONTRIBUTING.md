# Contributing

Thank you for improving the Fibe SDK. Bug reports, documentation fixes, tests,
and focused code changes are welcome.

## Before opening a change

1. Search existing issues and open a focused issue for substantial behavior.
2. Work from a current branch and keep unrelated changes out of the patch.
3. Preserve the compatibility policy in `COMPATIBILITY.md`.
4. Run `mage check`. Live tests additionally require `FIBE_INTEGRATION=1` and
   credentials for a disposable test account.
5. Add or update contract fixtures whenever a supported surface changes.

Commits must include a [Developer Certificate of Origin](DCO.md) sign-off:

```text
Signed-off-by: Your Name <you@example.com>
```

Use `git commit -s` to add it. This project does not require a CLA.

## Code comments

Comments should explain information the code cannot express clearly: wire-format
compatibility, invariants, security boundaries, concurrency constraints, or the
reason for a surprising implementation choice. Do not add comments merely to
repeat a declaration's name or satisfy an exported-symbol documentation quota.
An obvious declaration is clearer without a comment.

## Pull requests

Keep pull requests reviewable and explain behavior, compatibility risk, tests,
and security implications. Generated files must be regenerated using their
documented command. Do not include credentials, customer data, or live API
responses.

Maintainers may ask for a release note under the categorized `Unreleased`
section of `CHANGELOG.md`.
