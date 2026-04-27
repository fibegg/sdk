# Fibe MCP Tools

Total Tools: 42

## `fibe_agents_duplicate`
**Tier:** overseer | **Destructive:** False | **Idempotent:** True

### Description
[MODE:OVERSEER] Duplicate an agent configuration.

## `fibe_agents_runtime_status`
**Tier:** overseer | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:OVERSEER] Check agent runtime reachability, authentication, queue, and processing state.

## `fibe_agents_send_message`
**Tier:** overseer | **Destructive:** False | **Idempotent:** False

### Description
[MODE:OVERSEER] Send one text message to an agent runtime chat.

## `fibe_agents_start_chat`
**Tier:** overseer | **Destructive:** False | **Idempotent:** False

### Description
[MODE:SIDEEFFECTS] Start or reconnect an agent runtime chat on the current Marquee.

## `fibe_auth_set`
**Tier:** other | **Destructive:** False | **Idempotent:** False

### Description
[MODE:SIDEEFFECTS] Configure session-scoped authentication credentials for multi-tenant setups in case you have to work with multiple FIBE_API_KEY+FIBE_DOMAIN combinations

## `fibe_call`
**Tier:** meta | **Destructive:** False | **Idempotent:** False

### Description
[MODE:SIDEEFFECTS] Dynamically invoke any registered Fibe tool by name that is not advertised or hidden or not listed by ToolSearch. Use fibe_tools_catalog to list all hidden tools

## `fibe_doctor`
**Tier:** meta | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Run self-diagnostic checks: verify API key, connectivity, and display user profile

## `fibe_feedbacks_get`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:OVERSEER] Get one feedback entry for an agent, including player comments about artefacts or mutters.

## `fibe_feedbacks_list`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:OVERSEER] List all feedback entries associated with an agent.

## `fibe_find_github_repos`
**Tier:** other | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Search GitHub repositories across all connected installations. Returns deduplicated results.

## `fibe_get_github_token`
**Tier:** other | **Destructive:** False | **Idempotent:** True

### Description
[MODE:SIDEEFFECTS] Get a GitHub access token for a repository. Auto-resolves the correct installation.

## `fibe_gitea_repos_create`
**Tier:** other | **Destructive:** False | **Idempotent:** False

### Description
[MODE:GREENFIELD] Register and connect a new Gitea repository

## `fibe_github_repos_create`
**Tier:** other | **Destructive:** False | **Idempotent:** False

### Description
[MODE:GREENFIELD] Register and connect a new GitHub repository

## `fibe_greenfield_create`
**Tier:** greenfield | **Destructive:** False | **Idempotent:** False

### Description
[MODE:GREENFIELD] Create a new repository, Prop, app-owned template version, deployed playground, wait for running, and link it locally.

## `fibe_help`
**Tier:** meta | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Display detailed CLI help documentation for a specific Fibe command path. Extremely useful to look up flag descriptions or expected payload shapes.

## `fibe_local_playgrounds_info`
**Tier:** local | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:BROWNFIELD] Get info about a local playground.

## `fibe_local_playgrounds_link`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** True

### Description
[MODE:BROWNFIELD] Link local playground mounts into a working directory.

## `fibe_local_playgrounds_list`
**Tier:** local | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:BROWNFIELD] List playgrounds available locally at /opt/fibe/playgrounds or PLAYROOMS_ROOT.

## `fibe_local_playgrounds_urls`
**Tier:** local | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:BROWNFIELD] Get URLs of a local playground.

## `fibe_monitor_follow`
**Tier:** overseer | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:OVERSEER] Stream agent-produced events as live MCP progress notifications

## `fibe_monitor_list`
**Tier:** overseer | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:OVERSEER] List agent-produced monitor events

## `fibe_mutter`
**Tier:** base | **Destructive:** False | **Idempotent:** False

### Description
[MODE:SIDEEFFECTS] Create one short mutter for an agent: a visible internal note used for progress, proof, blocker, or problem updates.

## `fibe_mutters_get`
**Tier:** overseer | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:OVERSEER] Retrieve an agent's mutter stream by agent_id, with optional query/status/severity/playground filters.

## `fibe_pipeline`
**Tier:** meta | **Destructive:** False | **Idempotent:** False

### Description
[MODE:SIDEEFFECTS] Execute multiple tool calls sequentially in a single round-trip using JSONPath bindings. The most powerful tool by far! Use to eliminate roundtrip latency when creating and waiting for jobs.

## `fibe_pipeline_result`
**Tier:** meta | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Look up a cached result from a previous, the most powerful tool, - pipeline execution

## `fibe_playgrounds_action`
**Tier:** brownfield | **Destructive:** True | **Idempotent:** True

### Description
[MODE:SIDEEFFECTS] Run one playground lifecycle action: rollout, hard_restart, stop, start, or retry_compose.

## `fibe_playgrounds_debug`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Retrieve comprehensive debugging and diagnostic information for a playground. Use when troubleshooting a failing deployment.

## `fibe_playgrounds_logs`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Retrieve the consolidated service logs from a playground. Use when troubleshooting startup errors.

## `fibe_playgrounds_logs_follow`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:SIDEEFFECTS] Stream the live service logs from a playground as progress notifications

## `fibe_playgrounds_wait`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Block and poll until a playground reaches a specified target state (has timeout)

## `fibe_repo_status_check`
**Tier:** other | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Verify the system's access and view of multiple GitHub repository URLs.

## `fibe_resource_delete`
**Tier:** base | **Destructive:** True | **Idempotent:** True

### Description
[MODE:SIDEEFFECTS] Delete a supported flat Fibe resource by ID.

## `fibe_resource_get`
**Tier:** base | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Get a supported Fibe resource by ID. Use artefact_attachment to download an artefact's single attached file.

## `fibe_resource_list`
**Tier:** base | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] List a supported flat Fibe resource. Use fibe_schema with resource=list to discover resource names, aliases, and list params.

## `fibe_resource_mutate`
**Tier:** base | **Destructive:** False | **Idempotent:** False

### Description
[MODE:SIDEEFFECTS] Create, update, or run a supported resource-scoped mutation with a payload validated against fibe_schema before any API request.

## `fibe_run`
**Tier:** meta | **Destructive:** False | **Idempotent:** False

### Description
[MODE:SIDEEFFECTS] Last-resort escape hatch: invoke an arbitrary Fibe CLI command when no dedicated MCP tool fits. Use sparingly.

## `fibe_schema`
**Tier:** meta | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Return JSON Schema definitions and the schema resource catalog.

## `fibe_status`
**Tier:** meta | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] Display a comprehensive dashboard of resource counts, quotas, and rate limits across your account.

## `fibe_templates_develop`
**Tier:** brownfield | **Destructive:** False | **Idempotent:** False

### Description
[MODE:BROWNFIELD] Preview or apply template changes, switch playspecs/playgrounds/tricks, and optionally roll out or trigger a fresh trick run.

## `fibe_templates_launch`
**Tier:** greenfield | **Destructive:** False | **Idempotent:** True

### Description
[MODE:GREENFIELD] Bootstrap and launch a new playground directly from an import template.

## `fibe_templates_search`
**Tier:** greenfield | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:GREENFIELD] Search the import-template catalog by text or PostgreSQL regex. Regex mode requires a 3+ character literal token for indexed prefiltering.

## `fibe_tools_catalog`
**Tier:** meta | **Destructive:** False | **Idempotent:** True | **Read-only:** True

### Description
[MODE:DIALOG] List all tools registered and available on the Fibe MCP server. CRITICAL: Fibe Platform priority is to let you manage **ALL** its capabilities via its tools so you should find anything here. We just can't advertise them all because there are hundreds
