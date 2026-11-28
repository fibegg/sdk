# SDK integration test editions

Set `FIBE_DOMAIN` to the API URL of a disposable test environment and
`FIBE_API_KEY` to a normal-user key with explicit scopes. Each Mage edition command
opts in to live tests and passes the target, key, and fixture environment to
the Go suites.

| Target edition | Command | Live coverage |
| --- | --- | --- |
| Community | `mage testCommunity` | Shared cases |
| Enterprise | `mage testEnterprise` | Shared and Enterprise-only cases |

For direct Go runs, set `FIBE_TEST_EDITION=community` or
`FIBE_TEST_EDITION=enterprise`. Existing invocations with the edition unset
or set to `full` retain the default complete suite.

```sh
# List reviewed cases without an installation or credentials.
FIBE_TEST_EDITION=community go test ./integration ./internal/mcpserver -list .
FIBE_TEST_EDITION=enterprise go test ./integration ./internal/mcpserver -list .

# Run a focused Community API-key smoke.
FIBE_TEST_EDITION=community FIBE_INTEGRATION=1 \
  go test ./integration -run '^TestAPIKeys_CRUD$' -count=1 -parallel=1 -timeout=120s
```

Community has one administrator Player; the same Player can own an explicitly
scoped normal key and a separate optional wildcard `FIBE_ADMIN_API_KEY` for
admin coverage. A second Player is never fabricated. Existing Host, GitHub and
provider prerequisites still apply and retain their named skips when absent.
A live Community invocation reads `/route-manifest.json` without credentials
and fails if it targets an Enterprise executable. Offline listing and Enterprise
runs omit this check.

For a focused no-Host smoke, `TestAPIKeys_CRUD` needs `keys:manage`, `agents:read`
and `playgrounds:read`. It creates, lists and deletes API keys using the real SDK.
Other tests need their own ordinary API scopes; selection does not grant them.
Only explicitly opted-in integration commands may contact a live API.

`internal/testedition/cases.json` is the edition declaration for each live
test. `shared` checks run in both editions. `enterprise-only` checks need
features such as Teams, invitations, a second Player, or the Library. A
`mixed` suite is a container whose cases have individual declarations. It
uses `runEdition` so each subtest is gated before its fixtures and assertions.
Enterprise fixtures such as Vault secrets and managed Gitea sort rows are
created only in the complete suite.

Explicit edition runs validate the inventory before fixture setup. Invalid
`-run` patterns and selections containing zero eligible cases fail. `-run`
uses Go's slash-separated component expressions and alternatives. Listings
count reviewed cases. Missing credentials, Hosts, or provider prerequisites
can still skip listed cases at runtime. Provider matrices honor the existing
credential and `CHAT_E2E_CASE` / `CHAT_E2E_CASE_EXCEPT` filters.

The MCP package also contains hermetic tests. The explicit edition listings
show live cases only, and direct `-run` invocations can select hermetic tests.
Use `go test -c` for compile-only checks with an explicit edition.

## Standard runners and the external harness

`mage integrationTest`, `mage chatE2E`, plain `go test ./integration/...`, and the
live GitHub Actions workflow honor the same environment variable. The workflow
also exposes an edition choice, defaulting to `full`. Normal SDK unit tests keep
all assertions for both editions, including client support for Enterprise APIs.

The external Docker E2E harness can provision isolated environments and run
the SDK stage for both editions:

```sh
FIBE_WORKSPACE_ROOT="$PWD" node docker-e2e-harness/native/run.mjs --edition both --only sdk
```

Use `--edition community` or `--edition enterprise` to select a single
edition. The harness supplies the API target, credentials, and Host fixtures.
For an existing disposable API target, use the Mage commands above.

The SDK's Docker helper also supports offline enumeration and evaluation:

```sh
FIBE_TEST_EDITION=community bash docker/fibe-e2e-go-tests --edition-list
FIBE_TEST_EDITION=community bash docker/fibe-e2e-go-tests --edition-evaluate \
  --package integration --run '^TestLaunch_Create$'
```

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

Run `go test ./internal/testedition` and offline evaluation after edits. Add a
manifest rule for each new integration test or live MCP test. Start integration
tests with `requireEdition(t)` and use `requireRealServer(t)` for live MCP setup.
List literal subtests in the rule's inventory or assign each its own edition in
a mixed suite. Keep Enterprise-only fixtures below the edition gate.

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

The Library boundary includes Artefacts, Agent-attached Artefacts, Feedbacks and
Mutters. Their live CLI and MCP checks require Enterprise. Agent chat messages,
activity and ordinary mounted files remain Community checks. Mixed Agent flows
gate only the Library operations and retain the surrounding core assertions.
The status-summary case remains Community; its sibling that creates a Vault
secret is Enterprise.

The GitHub fixture launch test can bind the designated PAT through Community's
normal browser login, recent-auth and Git connection endpoint. Supply the
existing operator's `FIBE_NATIVE_ADMIN_EMAIL` and `FIBE_NATIVE_ADMIN_PASSWORD`
(or `FIBE_NATIVE_EMAIL` / `FIBE_NATIVE_PASSWORD`) when running that opted-in
fixture workflow. The browser identity must match the SDK key's Player. This
uses an in-memory cookie jar and does not create another Player or write a CLI
profile. Enterprise uses the existing fixture setup.

Literal keyed action tables whose case names join two fields with a dot are
also enumerated from the Go AST. A missing classification fails offline even
when the dynamic function's review digest is current. Opaque dynamic generators
still require explicit review and reject unknown names at execution time.
