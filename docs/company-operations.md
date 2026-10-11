# Company recovery and durable operations

These candidate interfaces require an explicit positive Company ID. SDK methods
add `X-Fibe-Expected-Owner-Type: Team` and `X-Fibe-Expected-Owner-ID`, and reject a
successful response whose owner headers do not match. Existing profile owner
checks remain in force. These expectations do not grant authority or turn a
service-principal credential into a human credential.

Use a Company-owned human credential/profile for runtime recovery, cancellation,
Repository reconciliation/adoption and organization installation management.
The server rechecks current human membership, role, credential scopes and the
relevant installation authority. Read-only visibility also follows server policy.

## CLI

```sh
fibe company --company-id 9 principals list
fibe company --company-id 9 principals create --name "Automation"
fibe company --company-id 9 principals revoke 12
fibe company --company-id 9 principals recover 12 --agent-ids '[23,24]'
fibe company --company-id 9 principals recover 12 --agent-ids '[]'
fibe company --company-id 9 subscriptions list
fibe company --company-id 9 subscriptions cancel 31
fibe company --company-id 9 subscriptions cancel 31 --retry-failed
fibe company --company-id 9 repository-operations list
fibe company --company-id 9 repository-operations reconcile REQUEST_ID --connection-id 42
fibe company --company-id 9 repository-operations adopt REQUEST_ID --connection-id 42
fibe company --company-id 9 git-installations list
fibe company --company-id 9 git-installations connect --installation-id 123 --verification-connection-id 45
fibe company --company-id 9 git-installations remove 42
```

Recovery also accepts `--from-file` with `{"replacement_agent_ids":[]}`. Omission
and `null` are invalid; an explicit empty list creates a replacement runtime
principal without rebinding agents. Agent IDs must be distinct positive integers.
The same completed selection replays the replacement and current adoption
states. A changed selection returns the server's `RECOVERY_CONFLICT` (409).
Adoption may remain pending, failed, unavailable or unknown after recovery has
committed; `recovery_status: completed` does not assert a running replacement.
Replay the identical selection to inspect current adoption states.

Cancellation returns HTTP 202 as an accepted durable intent. It does not assert
that the billing provider has canceled the subscription. Read `subscriptions
list` for `cancellation_state`, request/reconciliation timestamps and stable error
code. Retrying a failed intent requires `--retry-failed` and current human
authority. SDK/CLI/MCP retain provider and authority error states as reported.

Repository operation IDs identify retained requests in this exact Company. Read
the receipt, reconcile the remote identity, then explicitly adopt the known
destination. `outcome_unknown` remains truthful until readback resolves it. Never
overwrite or delete a destination to resolve uncertainty. An optional current
installation connection ID supports the server's reauthorization flow; it is not
an override of the retained destination identity.

## SDK and MCP

SDK entry points are additive `Client` methods: `TeamServicePrincipals`,
`CreateTeamServicePrincipal`, `RevokeTeamServicePrincipal`,
`RecoverRuntimePrincipal`, `CompanySubscriptions`, `CancelCompanySubscription`,
`CompanyRepositoryOperations`, `ReconcileCompanyRepositoryOperation`,
`AdoptCompanyRepositoryOperation`, `CompanyGitInstallations`,
`ConnectCompanyGitInstallation` and `RemoveCompanyGitInstallation`.

The MCP `other` tier registers matching `fibe_company_*` tools. Select `full` or
`other,meta` to advertise them. Every tool requires `company_id`; runtime
recovery requires `replacement_agent_ids`, including explicit `[]`. Revoke,
recover, cancel, adopt and installation removal require `confirm: true` under
the existing destructive-operation gate. This confirmation does not replace
live server authorization. Discovery provides the exact schemas and names.

Public typed responses preserve owner metadata, principal/replacement identity,
Agent adoption states, durable cancellation state, exact Repository destination
and installation authority availability. They expose no runtime key IDs, key
bytes, original recovery authority or installation tokens. Installation token
export is deliberately outside this management interface.

The executable contract fixtures in `contracts/` describe SDK declarations,
REST operations, CLI commands and MCP schemas. Synthetic transport tests exercise
explicit empty recovery, completed replay/conflict, accepted cancellation,
current installation status and matching owner-header proofs.
