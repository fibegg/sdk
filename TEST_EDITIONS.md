# SDK integration test editions

The SDK supports both application editions. No client API or assertion changes
with the test edition. An unset `FIBE_TEST_EDITION`, `full`, or `enterprise` runs
the existing complete suite. `community` selects reviewed supported cases in the
actual Go integration and live MCP test executables, before fixture creation.
Unknown values fail. The `enterprise` spelling is an alias for `full`.

```sh
# Offline evaluation: no API, credentials, bootstrap or Host access.
go run ./scripts/test-edition --edition community --list
go run ./scripts/test-edition --edition community --json
go run ./scripts/test-edition --edition community --package integration \
  --run '^TestAPIKeys_CRUD$' --json

# Actual tests; the ordinary live opt-in and credentials are still required.
FIBE_TEST_EDITION=community FIBE_INTEGRATION=1 \
  go test ./integration -run '^TestAPIKeys_CRUD$' -count=1 -parallel=1 -timeout=120s
FIBE_TEST_EDITION=community go test ./integration -list .
FIBE_TEST_EDITION=community mage integrationTest
FIBE_TEST_EDITION=community mage integrationList
```

Set `FIBE_DOMAIN` to the Community API URL and `FIBE_API_KEY` to an explicitly
scoped key. The existing fixture rejects a wildcard key for normal-user tests.
Community has one administrator Player; the same Player can own an explicitly
scoped normal key and a separate optional wildcard `FIBE_ADMIN_API_KEY` for
admin coverage. A second Player is never fabricated. Existing Host, GitHub and
provider prerequisites still apply and retain their named skips when absent.
A live Community invocation reads `/route-manifest.json` without credentials
and fails if it targets an Enterprise executable. Offline listing and default
full runs never require this endpoint.

For a focused no-Host smoke, `TestAPIKeys_CRUD` needs `keys:manage`, `agents:read`
and `playgrounds:read`. It creates, lists and deletes API keys using the real SDK.
Other tests need their own ordinary API scopes; selection does not grant them.
Only explicitly opted-in integration commands may contact a live API.

`internal/testedition/cases.json` records top-level test decisions, known
subtest names and the reason for each review. Mixed suites use `runEdition` at
each case boundary: supported cases run their original assertions, Enterprise
cases report an edition skip, and unknown case names fail. Shared Enterprise
fixtures (Vault secrets and managed Gitea sort rows) are created only by a full
run. API failures in selected cases remain failures. Selection never turns an
unexpected response into a skip.

Both the offline evaluator and Community Go test startup reject unknown tests,
unclassified literal subtests, unguarded mixed cases, invalid regular expressions
and selections containing zero eligible cases. A misspelled subtest suffix also
fails. `-run` uses Go's slash-separated component expressions and alternatives.
Enumeration counts named checks, not guaranteed passes: missing credentials,
Hosts or provider prerequisites remain visible runtime skips. Provider matrices
retain their existing credential and `CHAT_E2E_CASE` / `CHAT_E2E_CASE_EXCEPT` filters.

The MCP package contains hermetic tests as well as live tests. The evaluator
lists only live tests there. Direct Go invocations can still select existing
hermetic tests; a typo matching neither a live nor a hermetic test fails in
Community mode. Use `go test -c` for compile-only Community checks; an intentionally
empty `-run '^$'` is an error, not a qualification.

## Standard runners and the external harness

`mage integrationTest`, `mage chatE2E`, plain `go test ./integration/...`, and the
live GitHub Actions workflow honor the same environment variable. The workflow
also exposes an edition choice, defaulting to `full`. Normal SDK unit tests keep
all assertions for both editions, including client support for Enterprise APIs.

The existing Docker runner accepts offline enumeration before its wrapper or
fixture waits:

```sh
FIBE_TEST_EDITION=community docker/fibe-e2e-go-tests --edition-list
FIBE_TEST_EDITION=community docker/fibe-e2e-go-tests --edition-evaluate \
  --package integration --run '^TestLaunch_Create$'
```

Its ordinary test entry points inherit `FIBE_TEST_EDITION`. Community bootstrap
requests only the primary role and does not configure a managed Gitea fixture.
The normal full bootstrap still requests primary, second-user and rate-limit
fixtures. Container network/Host preparation remains the owning harness's job.
No released SDK artifact is changed by editing this checkout; candidate harness
runs must select a committed SDK revision explicitly. There is no SDK release in
this change.

## Operator-template launch coverage

`TestOperatorTemplate_LaunchPinnedVersion` launches a pre-existing Admin template
through the normal SDK endpoint without creating a Library/catalogue template.
Provide `FIBE_TEST_OPERATOR_TEMPLATE_ID`, `FIBE_TEST_OPERATOR_TEMPLATE_VERSION`
(the immutable version number), `FIBE_TEST_OPERATOR_TEMPLATE_VERSION_ID`
(the immutable row ID), and an owned ready `FIBE_TEST_HOST_ID`. The template must
be self-contained for the fixture Host and require no unprovided variables. The
case requires normal template launch, Spec read/manage and Playground read/manage
scopes. Missing fixtures skip; malformed supplied IDs or API/lineage errors fail.
It checks the returned Spec/Playground pair, selected Host and exact source
version ID, then deletes its own resources. Host runtime deployment is covered
by the existing Playground lifecycle tests.

## Reviewing a new or changed case

Run `go test ./internal/testedition` and offline evaluation after edits. Every new
integration test and live MCP test needs an explicit rule; every literal
subtest needs a listed name or mixed decision. Enterprise-only fixture setup
must stay below the edition gate. Keep retained checks and their assertions.

Table-driven names cannot all be inferred from a literal `t.Run` call. Only
those dynamic test functions carry a `dynamic_review` digest of their canonical
Go AST, excluding comments and unrelated functions. A changed dynamic function
requires reviewing its table, fixture setup and case decisions. After that
review, obtain that function's digest with:

```sh
go run ./scripts/test-edition --fingerprint integration/TestDateFilters_CoverageMatrix
```

Update that rule's `dynamic_review`. Ordinary formatting, comments and edits to
other functions require no digest refresh. `--fingerprint '*'` prints all
current dynamic-function digests for review tooling; it does not update the
manifest or classify any case automatically. The guard tests exercise new test
and literal discovery, mixed-case refusals, unknown names, empty selections,
pattern semantics and Community server verification.
