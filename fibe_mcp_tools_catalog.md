# Fibe MCP Tools

Total Tools: 163

## `fibe_agents_activity_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get agent activity

## `fibe_agents_activity_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Replace agent activity

## `fibe_agents_authenticate`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Authenticate an agent (OAuth code/token exchange)

## `fibe_agents_chat`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Send a message to an agent (accepts 'text' or legacy 'message')

## `fibe_agents_create`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a new agent

## `fibe_agents_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete an agent

## `fibe_agents_duplicate`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Duplicate an agent

## `fibe_agents_get`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show detailed agent information

## `fibe_agents_gitea_token`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get agent's Gitea token

## `fibe_agents_github_token`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get agent's GitHub token (optionally for a specific repo)

## `fibe_agents_list`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List all agents

## `fibe_agents_messages_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get agent messages

## `fibe_agents_messages_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Replace agent messages

## `fibe_agents_mounted_file_add`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Attach a file to an agent (accepts 'mount_path' or legacy 'path')

## `fibe_agents_mounted_file_remove`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Remove an agent's mounted file

## `fibe_agents_mounted_file_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update an agent's mounted file metadata (accepts 'mount_path' or legacy 'path')

## `fibe_agents_raw_providers_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get agent raw provider config

## `fibe_agents_raw_providers_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Replace agent raw provider config

## `fibe_agents_revoke_github`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Revoke an agent's GitHub token

## `fibe_agents_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update agent settings

## `fibe_api_keys_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a new API key

## `fibe_api_keys_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Revoke an API key

## `fibe_api_keys_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List API keys

## `fibe_artefacts_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Upload an artefact for an agent (accepts 'name' or legacy 'title'; 'content_base64' or legacy 'content')

## `fibe_artefacts_download`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Download an artefact (returns base64-encoded content)

## `fibe_artefacts_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show an artefact

## `fibe_artefacts_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List artefacts for an agent

## `fibe_audit_logs_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List audit logs

## `fibe_auth_set`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Set session-scoped API key and/or domain (multi-tenant HTTP server)

## `fibe_call`
**Tier:** meta | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Invoke any registered tool by name (including tools not advertised in the current tier)

## `fibe_categories_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List template categories

## `fibe_doctor`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Run self-diagnostic checks (API key validity, connectivity, version)

## `fibe_feedbacks_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create feedback for an agent. Required: source_type (e.g. "Artefact"), source_id (int64), selected_text, selection_start, selection_end. source_type is the polymorphic class name from the Rails side — known values include "Artefact". Comment/body goes in 'comment'.

## `fibe_feedbacks_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete feedback

## `fibe_feedbacks_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show a feedback

## `fibe_feedbacks_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List feedbacks for an agent

## `fibe_feedbacks_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update feedback

## `fibe_gitea_repos_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Register a Gitea repository

## `fibe_github_repos_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Register a GitHub repository

## `fibe_help`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Return extended help (cobra Long) for a fibe command path

## `fibe_hunks_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show a hunk

## `fibe_hunks_ingest`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Trigger hunk ingestion for a prop

## `fibe_hunks_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List hunks for a prop

## `fibe_hunks_next`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Fetch the next hunk awaiting processing (requires processor_name)

## `fibe_hunks_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a hunk

## `fibe_installations_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List GitHub App installations associated with your account

## `fibe_installations_repos`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List repositories visible to an installation

## `fibe_installations_token`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get a scoped GitHub token for an installation + repo

## `fibe_launch`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
One-shot: compose YAML → playspec → playground. Response surfaces playspec_id and playground_id so pipelines can chain off them.

## `fibe_limits`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show current quotas, per-parent caps, and API-key rate-limit usage

## `fibe_marquees_autoconnect_token`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Generate a marquee autoconnect token

## `fibe_marquees_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Register a new marquee

## `fibe_marquees_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a marquee

## `fibe_marquees_generate_ssh_key`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Generate SSH key for a marquee

## `fibe_marquees_get`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show detailed marquee information

## `fibe_marquees_list`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List all marquees (servers)

## `fibe_marquees_test_connection`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Test connectivity to a marquee

## `fibe_marquees_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a marquee

## `fibe_me`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show the authenticated user's profile

## `fibe_monitor_list`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List agent-produced monitor events

## `fibe_monitor_follow`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Stream agent-produced events as MCP progress notifications

## `fibe_mutations_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a mutation for a prop. Required: branch, found_commit_sha (CLI flag --sha is accepted as an alias).

## `fibe_mutations_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List mutations for a prop

## `fibe_mutations_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a mutation

## `fibe_mutters_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Append an item to an agent's mutter

## `fibe_mutters_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get an agent's mutter transcript

## `fibe_pipeline`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Compose multiple fibe_* tool calls in a single round-trip with JSONPath bindings

## `fibe_pipeline_result`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Look up a cached pipeline result by pipeline_id and JSONPath

## `fibe_playgrounds_compose`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get playground docker-compose configuration

## `fibe_playgrounds_create`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Deploy a playspec blueprint as a running playground

## `fibe_playgrounds_debug`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get comprehensive debug information

## `fibe_playgrounds_delete`
**Tier:** core | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a playground (destructive, irreversible)

## `fibe_playgrounds_env`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get playground environment metadata

## `fibe_playgrounds_extend`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Extend playground expiration time

## `fibe_playgrounds_get`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show detailed playground information

## `fibe_playgrounds_hard_restart`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Hard restart all playground services

## `fibe_playgrounds_list`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List all playgrounds (excludes tricks)

## `fibe_playgrounds_logs`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get service logs from a playground

## `fibe_playgrounds_logs_follow`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Stream playground service logs as MCP progress notifications

## `fibe_playgrounds_rollout`
**Tier:** core | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Recreate playground with latest configuration

## `fibe_playgrounds_status`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Check playground status

## `fibe_playgrounds_update`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update playground settings

## `fibe_playgrounds_wait`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Poll a playground until it reaches a target status

## `fibe_playspecs_create`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a new playspec

## `fibe_playspecs_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a playspec

## `fibe_playspecs_get`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show detailed playspec information

## `fibe_playspecs_list`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List all playspecs

## `fibe_playspecs_mounted_file_add`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Attach a file to a playspec (accepts 'mount_path' or legacy 'path')

## `fibe_playspecs_mounted_file_remove`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Remove a playspec mounted file

## `fibe_playspecs_mounted_file_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update playspec mounted file metadata (accepts 'mount_path' or legacy 'path')

## `fibe_playspecs_registry_credential_add`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Add a registry credential to a playspec (registry_type must be one of: ghcr, dockerhub, aws_ecr)

## `fibe_playspecs_registry_credential_remove`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Remove a registry credential from a playspec

## `fibe_playspecs_services`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List services defined in a playspec

## `fibe_playspecs_switch_version`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** False

### Description
Switch a template-backed playspec to another template version

## `fibe_playspecs_switch_version_preview`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Preview switching a template-backed playspec to another template version

## `fibe_playspecs_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a playspec

## `fibe_playspecs_validate_compose`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Validate a docker-compose YAML as a Fibe playspec

## `fibe_props_attach`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Attach an existing GitHub repository to your account as a prop (accepts 'repo_full_name' or 'repository_url')

## `fibe_props_branches`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List branches for a prop, optionally filtered by query

## `fibe_props_create`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Register a new prop (git repository)

## `fibe_props_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a prop

## `fibe_props_env_defaults`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Read default environment variables from a prop branch

## `fibe_props_get`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show detailed prop information

## `fibe_props_list`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List all props (source code repositories)

## `fibe_props_manual_link`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Manually link a prop after OAuth reconnection

## `fibe_props_mirror`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Create a prop by mirroring an external repository (accepts 'source_url' or legacy 'repository_url')

## `fibe_props_sync`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Sync a prop with its git remote

## `fibe_props_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a prop

## `fibe_props_with_docker_compose`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List props that ship a docker-compose file

## `fibe_repo_status_check`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Check Fibe's view of multiple GitHub repository URLs (accepts 'github_urls' or legacy 'urls')

## `fibe_run`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Escape hatch: invoke any fibe CLI command programmatically

## `fibe_schema`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Return JSON Schema hints for create/update params of a Fibe resource

## `fibe_secrets_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a secret

## `fibe_secrets_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a secret

## `fibe_secrets_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show secret metadata (value is redacted)

## `fibe_secrets_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List all secrets

## `fibe_secrets_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a secret

## `fibe_server_info`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show Fibe server UTC time, build time, and git commit SHA

## `fibe_status`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show account status dashboard (counts across all resources in one request)

## `fibe_teams_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a team

## `fibe_teams_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a team

## `fibe_teams_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show team details

## `fibe_teams_leave`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Leave a team (accepts 'id' or 'team_id' for consistency with fibe_teams_members_* tools)

## `fibe_teams_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List teams

## `fibe_teams_members_accept`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Accept a pending team invite

## `fibe_teams_members_decline`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Decline a pending team invite

## `fibe_teams_members_invite`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Invite a user to a team

## `fibe_teams_members_remove`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Remove a member from a team

## `fibe_teams_members_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a team member's role

## `fibe_teams_resources_contribute`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Contribute a resource to a team

## `fibe_teams_resources_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List resources owned by a team

## `fibe_teams_resources_remove`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Remove a shared team resource

## `fibe_teams_transfer_leadership`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Transfer team leadership to another member (accepts 'id' or 'team_id')

## `fibe_teams_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a team

## `fibe_templates_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create an import template

## `fibe_templates_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete an import template

## `fibe_templates_fork`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Fork an import template

## `fibe_templates_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show import template details

## `fibe_templates_launch`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Launch a playground from an import template

## `fibe_templates_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List import templates

## `fibe_templates_search`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Search import templates

## `fibe_templates_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update an import template

## `fibe_templates_upload_image`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Upload a cover image for an import template (required: id, filename, image_data OR content_path)

## `fibe_templates_versions_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a new template version (accepts 'template_body' or legacy 'body')

## `fibe_templates_versions_destroy`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a template version

## `fibe_templates_versions_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List versions of an import template

## `fibe_templates_versions_toggle_public`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Toggle public visibility of a template version (accepts 'id' or 'template_id' for the template)

## `fibe_tools_catalog`
**Tier:** meta | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List every tool registered on the Fibe MCP server (including tools not advertised in the current tier)

## `fibe_tricks_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a trick

## `fibe_tricks_get`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show detailed trick information

## `fibe_tricks_list`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List all tricks (job-mode playgrounds)

## `fibe_tricks_logs`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Get service logs from a trick

## `fibe_tricks_logs_follow`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Stream trick service logs as MCP progress notifications

## `fibe_tricks_rerun`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Re-run a completed or failed trick

## `fibe_tricks_status`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Check trick status and job result

## `fibe_tricks_trigger`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Run a new trick from a job-mode playspec

## `fibe_tricks_wait`
**Tier:** core | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Poll a trick until it reaches a target status (e.g., completed)

## `fibe_webhooks_create`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** False

### Description
Create a webhook endpoint. events[] must contain exact event identifiers (e.g. agent.created, playground.running, trick.completed) — call fibe_webhooks_event_types first if you're not sure which strings are valid.

## `fibe_webhooks_delete`
**Tier:** full | **Advertised:** True | **Destructive:** True | **Idempotent:** True

### Description
Delete a webhook endpoint

## `fibe_webhooks_deliveries_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List recent deliveries for a webhook endpoint

## `fibe_webhooks_event_types`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List webhook event types

## `fibe_webhooks_get`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Show webhook endpoint details

## `fibe_webhooks_list`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
List webhook endpoints

## `fibe_webhooks_test`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Send a test event to a webhook endpoint

## `fibe_webhooks_update`
**Tier:** full | **Advertised:** True | **Destructive:** False | **Idempotent:** True

### Description
Update a webhook endpoint
