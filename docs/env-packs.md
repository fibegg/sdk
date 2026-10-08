# ENV packs in the SDK, CLI, and MCP

ENV packs contain ordinary readable string values. Empty strings, quotes, and newlines are preserved. Owner context comes from the selected authenticated profile; pack updates cannot transfer ownership.

## Go SDK

`Client.EnvPacks` provides `List`, `Get`, `Create`, `Update`, and `Delete`. Target methods `Attachments`, `ReplaceAttachments`, `ReorderAttachments`, `DetachAttachment`, `RetargetAttachment`, and `RenewGrant` accept `playgrounds` or `specs` and numeric target IDs. `Playgrounds.Rerun` and `Tasks.RerunWithParamsByIdentifier` use the server's source-reference operation.

`EnvPackAttachmentsParams.Attachments` and creation/update `EnvPackAttachments` fields are pointers to slices. Nil omits the field: creation inherits Spec defaults, updates preserve references, and reruns preserve source references. A pointer to an empty slice sends `[]` and explicitly detaches all packs. `EnvPackAttachmentInput.ServiceNames` nil means all services; a present empty list is rejected. Types expose no writable grantor or position fields.

```go
empty := []fibe.EnvPackAttachmentInput{}
_, err := client.EnvPacks.ReplaceAttachments(ctx, "playgrounds", id,
    &fibe.EnvPackAttachmentsParams{Attachments: &empty})
```

## CLI

Use `fibe env-packs list/get/create/update/delete`. Creation/update accepts `--from-file` JSON/YAML. `--name` overrides the name; update `--version` supplies the expected version. Delete uses the existing confirmation convention (`--yes` for automation). No reveal flag applies.

```sh
fibe env-packs create --from-file pack.json
fibe env-packs update 42 --from-file updated-pack.json --version 3
fibe env-packs attachments get playgrounds 129
fibe env-packs attachments set playgrounds 129 --from-file attachments.json
fibe env-packs attachments reorder playgrounds 129 --order 71,42
fibe env-packs attachments retarget playgrounds 129 42 --services web,jobs
fibe env-packs attachments retarget playgrounds 129 42 --all-services
fibe env-packs attachments detach playgrounds 129 42
fibe env-packs attachments renew playgrounds 129 42
```

`attachments.json` contains `env_pack_attachments`: `[]` detaches all packs, omitted/null preserves references. Playground/Spec creation/update and Task trigger/rerun `--from-file` accept the same optional selection. Renew resolves the attached pack ID into its attachment ID without fabricating grant identity.

## MCP

`fibe_schema(resource:"env_pack")` discovers schemas. Generic `fibe_resource_list/get/delete` supports `env_pack`. Generic `fibe_resource_mutate` supports pack `create`, `update`, `attachments_replace`, `attachments_reorder`, `attachment_detach`, `attachment_retarget`, and `grant_renew`, plus Playground `rerun` and Task `trigger`/`rerun` selections. `fibe_env_pack_attachments_get` reads target provenance. Mutation dry-runs validate the same nil/empty, key/value, and grant-field rules locally.

All clients share server precedence, authority, and operation snapshots. Pack/list changes take effect only on the next explicit operation. Automatic repair retains the last applied values.

Launch recipient selection is also available on `LaunchParams`,
`ImportTemplateLaunchParams`, and `GreenfieldCreateParams` through the optional
`EnvPackAttachments` pointer. Nil inherits/defaults; a pointer to an empty slice
opts out. Template authors do not store recipient references.

`fibe launch --env-pack-attachments '[{"env_pack_id":7,"service_names":["web"]}]'`
accepts an ordered JSON array or `@file`. Use `[]` to opt out. Omit the option or
use `null` to inherit. The MCP `fibe_launch` tool accepts the same field for
compose, repository, Template, Template-version and Spec sources. Unknown grantor
fields, duplicate IDs and empty selected-service arrays fail before a request.
