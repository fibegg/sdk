# Fibe MCP Tools Descriptions

| Tool Name | Description |
|-----------|-------------|
| `fibe_agents_activity_get` | Get agent activity |
| `fibe_agents_activity_update` | Replace agent activity |
| `fibe_agents_authenticate` | Authenticate an agent (OAuth code/token exchange) |
| `fibe_agents_chat` | Send a message to an agent (accepts 'text' or legacy 'message') |
| `fibe_agents_create` | Create a new agent |
| `fibe_agents_delete` | Delete an agent |
| `fibe_agents_duplicate` | Duplicate an agent |
| `fibe_agents_get` | Show detailed agent information |
| `fibe_agents_gitea_token` | Get agent's Gitea token |
| `fibe_agents_github_token` | Get agent's GitHub token (optionally for a specific repo) |
| `fibe_agents_list` | List all agents |
| `fibe_agents_messages_get` | Get agent messages |
| `fibe_agents_messages_update` | Replace agent messages |
| `fibe_agents_mounted_file_add` | Attach a file to an agent (accepts 'mount_path' or legacy 'path') |
| `fibe_agents_mounted_file_remove` | Remove an agent's mounted file |
| `fibe_agents_mounted_file_update` | Update an agent's mounted file metadata (accepts 'mount_path' or legacy 'path') |
| `fibe_agents_raw_providers_get` | Get agent raw provider config |
| `fibe_agents_raw_providers_update` | Replace agent raw provider config |
| `fibe_agents_revoke_github` | Revoke an agent's GitHub token |
| `fibe_agents_update` | Update agent settings |
| `fibe_api_keys_create` | Create a new API key |
| `fibe_api_keys_delete` | Revoke an API key |
| `fibe_api_keys_list` | List API keys |
| `fibe_artefacts_create` | Upload an artefact for an agent (accepts 'name' or legacy 'title'; 'content_base64' or legacy 'content') |
| `fibe_artefacts_download` | Download an artefact (returns base64-encoded content) |
| `fibe_artefacts_get` | Show an artefact |
| `fibe_artefacts_list` | List artefacts for an agent |
| `fibe_audit_logs_list` | List audit logs |
| `fibe_auth_set` | Set session-scoped API key and/or domain (multi-tenant HTTP server) |
| `fibe_call` | Invoke any registered tool by name (including tools not advertised in the current tier) |
| `fibe_categories_list` | List template categories |
| `fibe_doctor` | Run self-diagnostic checks (API key validity, connectivity, version) |
| `fibe_feedbacks_create` | Create feedback for an agent. Required: source_type (e.g. "Artefact"), source_id (int64), selected_text, selection_start, selection_end. source_type is the polymorphic class name from the Rails side — known values include "Artefact". Comment/body goes in 'comment'. |
| `fibe_feedbacks_delete` | Delete feedback |
| `fibe_feedbacks_get` | Show a feedback |
| `fibe_feedbacks_list` | List feedbacks for an agent |
| `fibe_feedbacks_update` | Update feedback |
| `fibe_gitea_repos_create` | Register a Gitea repository |
| `fibe_github_repos_create` | Register a GitHub repository |
| `fibe_help` | Return extended help (cobra Long) for a fibe command path |
| `fibe_hunks_get` | Show a hunk |
| `fibe_hunks_ingest` | Trigger hunk ingestion for a prop |
| `fibe_hunks_list` | List hunks for a prop |
| `fibe_hunks_next` | Fetch the next hunk awaiting processing (requires processor_name) |
| `fibe_hunks_update` | Update a hunk |
| `fibe_installations_list` | List GitHub App installations associated with your account |
| `fibe_installations_repos` | List repositories visible to an installation |
| `fibe_installations_token` | Get a scoped GitHub token for an installation + repo |
| `fibe_launch` | One-shot: compose YAML → playspec → playground. Response surfaces playspec_id and playground_id so pipelines can chain off them. |
| `fibe_limits` | Show current quotas, per-parent caps, and API-key rate-limit usage |
| `fibe_marquees_autoconnect_token` | Generate a marquee autoconnect token |
| `fibe_marquees_create` | Register a new marquee |
| `fibe_marquees_delete` | Delete a marquee |
| `fibe_marquees_generate_ssh_key` | Generate SSH key for a marquee |
| `fibe_marquees_get` | Show detailed marquee information |
| `fibe_marquees_list` | List all marquees (servers) |
| `fibe_marquees_test_connection` | Test connectivity to a marquee |
| `fibe_marquees_update` | Update a marquee |
| `fibe_me` | Show the authenticated user's profile |
| `fibe_monitor_list` | List agent-produced monitor events |
| `fibe_monitor_follow` | Stream agent-produced events as MCP progress notifications |
| `fibe_mutations_create` | Create a mutation for a prop. Required: branch, found_commit_sha (CLI flag --sha is accepted as an alias). |
| `fibe_mutations_list` | List mutations for a prop |
| `fibe_mutations_update` | Update a mutation |
| `fibe_mutters_create` | Append an item to an agent's mutter |
| `fibe_mutters_get` | Get an agent's mutter transcript |
| `fibe_pipeline` | Compose multiple fibe_* tool calls in a single round-trip with JSONPath bindings |
| `fibe_pipeline_result` | Look up a cached pipeline result by pipeline_id and JSONPath |
| `fibe_playgrounds_compose` | Get playground docker-compose configuration |
| `fibe_playgrounds_create` | Deploy a playspec blueprint as a running playground |
| `fibe_playgrounds_debug` | Get comprehensive debug information |
| `fibe_playgrounds_delete` | Delete a playground (destructive, irreversible) |
| `fibe_playgrounds_env` | Get playground environment metadata |
| `fibe_playgrounds_extend` | Extend playground expiration time |
| `fibe_playgrounds_get` | Show detailed playground information |
| `fibe_playgrounds_hard_restart` | Hard restart all playground services |
| `fibe_playgrounds_list` | List all playgrounds (excludes tricks) |
| `fibe_playgrounds_logs` | Get service logs from a playground |
| `fibe_playgrounds_logs_follow` | Stream playground service logs as MCP progress notifications |
| `fibe_playgrounds_rollout` | Recreate playground with latest configuration |
| `fibe_playgrounds_status` | Check playground status |
| `fibe_playgrounds_update` | Update playground settings |
| `fibe_playgrounds_wait` | Poll a playground until it reaches a target status |
| `fibe_playspecs_create` | Create a new playspec |
| `fibe_playspecs_delete` | Delete a playspec |
| `fibe_playspecs_get` | Show detailed playspec information |
| `fibe_playspecs_list` | List all playspecs |
| `fibe_playspecs_mounted_file_add` | Attach a file to a playspec (accepts 'mount_path' or legacy 'path') |
| `fibe_playspecs_mounted_file_remove` | Remove a playspec mounted file |
| `fibe_playspecs_mounted_file_update` | Update playspec mounted file metadata (accepts 'mount_path' or legacy 'path') |
| `fibe_playspecs_registry_credential_add` | Add a registry credential to a playspec (registry_type must be one of: ghcr, dockerhub, aws_ecr) |
| `fibe_playspecs_registry_credential_remove` | Remove a registry credential from a playspec |
| `fibe_playspecs_services` | List services defined in a playspec |
| `fibe_playspecs_update` | Update a playspec |
| `fibe_playspecs_validate_compose` | Validate a docker-compose YAML as a Fibe playspec |
| `fibe_props_attach` | Attach an existing GitHub repository to your account as a prop (accepts 'repo_full_name' or 'repository_url') |
| `fibe_props_branches` | List branches for a prop, optionally filtered by query |
| `fibe_props_create` | Register a new prop (git repository) |
| `fibe_props_delete` | Delete a prop |
| `fibe_props_env_defaults` | Read default environment variables from a prop branch |
| `fibe_props_get` | Show detailed prop information |
| `fibe_props_list` | List all props (source code repositories) |
| `fibe_props_manual_link` | Manually link a prop after OAuth reconnection |
| `fibe_props_mirror` | Create a prop by mirroring an external repository (accepts 'source_url' or legacy 'repository_url') |
| `fibe_props_sync` | Sync a prop with its git remote |
| `fibe_props_update` | Update a prop |
| `fibe_props_with_docker_compose` | List props that ship a docker-compose file |
| `fibe_repo_status_check` | Check Fibe's view of multiple GitHub repository URLs (accepts 'github_urls' or legacy 'urls') |
| `fibe_run` | Escape hatch: invoke any fibe CLI command programmatically |
| `fibe_schema` | Return JSON Schema hints for create/update params of a Fibe resource |
| `fibe_secrets_create` | Create a secret |
| `fibe_secrets_delete` | Delete a secret |
| `fibe_secrets_get` | Show secret metadata (value is redacted) |
| `fibe_secrets_list` | List all secrets |
| `fibe_secrets_update` | Update a secret |
| `fibe_server_info` | Show Fibe server UTC time, build time, and git commit SHA |
| `fibe_status` | Show account status dashboard (counts across all resources in one request) |
| `fibe_teams_create` | Create a team |
| `fibe_teams_delete` | Delete a team |
| `fibe_teams_get` | Show team details |
| `fibe_teams_leave` | Leave a team (accepts 'id' or 'team_id' for consistency with fibe_teams_members_* tools) |
| `fibe_teams_list` | List teams |
| `fibe_teams_members_accept` | Accept a pending team invite |
| `fibe_teams_members_decline` | Decline a pending team invite |
| `fibe_teams_members_invite` | Invite a user to a team |
| `fibe_teams_members_remove` | Remove a member from a team |
| `fibe_teams_members_update` | Update a team member's role |
| `fibe_teams_resources_contribute` | Contribute a resource to a team |
| `fibe_teams_resources_list` | List resources owned by a team |
| `fibe_teams_resources_remove` | Remove a shared team resource |
| `fibe_teams_transfer_leadership` | Transfer team leadership to another member (accepts 'id' or 'team_id') |
| `fibe_teams_update` | Update a team |
| `fibe_templates_create` | Create an import template |
| `fibe_templates_delete` | Delete an import template |
| `fibe_templates_fork` | Fork an import template |
| `fibe_templates_get` | Show import template details |
| `fibe_templates_launch` | Launch a playground from an import template |
| `fibe_templates_list` | List import templates |
| `fibe_templates_search` | Search import templates |
| `fibe_templates_update` | Update an import template |
| `fibe_templates_upload_image` | Upload a cover image for an import template (required: id, filename, image_data OR content_path) |
| `fibe_templates_versions_create` | Create a new template version (accepts 'template_body' or legacy 'body') |
| `fibe_templates_versions_destroy` | Delete a template version |
| `fibe_templates_versions_list` | List versions of an import template |
| `fibe_templates_versions_toggle_public` | Toggle public visibility of a template version (accepts 'id' or 'template_id' for the template) |
| `fibe_tools_catalog` | List every tool registered on the Fibe MCP server (including tools not advertised in the current tier) |
| `fibe_tricks_delete` | Delete a trick |
| `fibe_tricks_get` | Show detailed trick information |
| `fibe_tricks_list` | List all tricks (job-mode playgrounds) |
| `fibe_tricks_logs` | Get service logs from a trick |
| `fibe_tricks_logs_follow` | Stream trick service logs as MCP progress notifications |
| `fibe_tricks_rerun` | Re-run a completed or failed trick |
| `fibe_tricks_status` | Check trick status and job result |
| `fibe_tricks_trigger` | Run a new trick from a job-mode playspec |
| `fibe_tricks_wait` | Poll a trick until it reaches a target status (e.g., completed) |
| `fibe_webhooks_create` | Create a webhook endpoint. events[] must contain exact event identifiers (e.g. agent.created, playground.running, trick.completed) — call fibe_webhooks_event_types first if you're not sure which strings are valid. |
| `fibe_webhooks_delete` | Delete a webhook endpoint |
| `fibe_webhooks_deliveries_list` | List recent deliveries for a webhook endpoint |
| `fibe_webhooks_event_types` | List webhook event types |
| `fibe_webhooks_get` | Show webhook endpoint details |
| `fibe_webhooks_list` | List webhook endpoints |
| `fibe_webhooks_test` | Send a test event to a webhook endpoint |
| `fibe_webhooks_update` | Update a webhook endpoint |
