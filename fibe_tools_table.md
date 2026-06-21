# Fibe MCP Tools Table

Generated from the MCP registry.

- Registered tools: 60
- Advertised with `FIBE_MCP_TOOLS=full`: 59
- Advertised with `FIBE_MCP_TOOLS=core`: 39
- Hidden dispatcher-only tools: 1

`full` advertises every non-hidden registered tool. Hidden tools remain dispatcher-reachable through `fibe_call` and `fibe_pipeline`, and `fibe_tools_catalog` reports them with `hidden:true`.

| Tool Name | Tier | Advertised in `full` | Description |
|-----------|------|----------------------|-------------|
| `fibe_agent_defaults_get` | base | yes | [MODE:DIALOG] Read the authenticated player's agent default overrides. |
| `fibe_agent_defaults_reset` | base | yes | [MODE:SIDEEFFECTS] Clear all player agent default overrides so admin defaults apply. |
| `fibe_agent_defaults_update` | base | yes | [MODE:SIDEEFFECTS] Replace the authenticated player's agent default overrides. Use the same agent_defaults JSON shape as the profile UI. |
| `fibe_agents_activity` | overseer | yes | [MODE:OVERSEER] Read agent activity, optionally scoped to a conversation. |
| `fibe_agents_create_conversation` | overseer | yes | [MODE:SIDEEFFECTS] Create or upsert an agent conversation. |
| `fibe_agents_delete_conversation` | overseer | yes | [MODE:SIDEEFFECTS] Delete an agent conversation. |
| `fibe_agents_duplicate` | overseer | yes | [MODE:OVERSEER] Duplicate an agent configuration. |
| `fibe_agents_interrupt` | overseer | yes | [MODE:SIDEEFFECTS] Interrupt a running agent turn. |
| `fibe_agents_live_state` | overseer | yes | [MODE:OVERSEER] Check conversation-scoped agent live stream state. |
| `fibe_agents_messages` | overseer | yes | [MODE:OVERSEER] Read agent messages, optionally scoped to a conversation. |
| `fibe_agents_runtime_status` | overseer | yes | [MODE:OVERSEER] Check agent reachability, authentication, queue, and processing state. Live checks fail with MARQUEE_NOT_FUNDED when unpaid. |
| `fibe_agents_send_message` | overseer | yes | [MODE:OVERSEER] Send one text message to an agent chat. Fails with MARQUEE_NOT_FUNDED when the chat Marquee is unpaid. |
| `fibe_agents_start_chat` | overseer | yes | [MODE:SIDEEFFECTS] Start or reconnect an agent chat on the current Marquee. Requires a funded Marquee; unpaid Marquees fail with MARQUEE_NOT_FUNDED. |
| `fibe_artefact_upload` | base | yes | [MODE:SIDEEFFECTS] Upload and save an artefact. Useful when Player asks to create something, implicitly or explicitly |
| `fibe_auth_list` | meta | yes | [MODE:DIALOG] List local Fibe auth profiles available to this MCP server without revealing API keys. |
| `fibe_auth_set` | other | yes | [MODE:SIDEEFFECTS] Configure session-scoped authentication credentials for multi-tenant setups in case you have to work with multiple FIBE_API_KEY+FIBE_DOMAIN combinations |
| `fibe_auth_status` | meta | yes | [MODE:DIALOG] Show the current MCP session auth target and selected profile, if any. |
| `fibe_auth_use` | meta | yes | [MODE:SIDEEFFECTS] Switch this MCP session to a local Fibe auth profile by name, rebuilding the session client immediately. |
| `fibe_call` | meta | yes | [MODE:SIDEEFFECTS] Invoke a registered Fibe tool that is hidden by the current tool tier. Prefer direct tool calls when the concrete tool is advertised; use fibe_tools_catalog/fibe_schema only when the hidden tool name or args are unclear. |
| `fibe_doctor` | meta | yes | [MODE:DIALOG] Run self-diagnostic checks: verify API key, connectivity, and display user profile |
| `fibe_feedbacks_get` | brownfield | yes | [MODE:OVERSEER] Get one feedback entry for an agent, including player comments about artefacts or mutters. |
| `fibe_feedbacks_list` | brownfield | yes | [MODE:OVERSEER] List all feedback entries associated with an agent. |
| `fibe_find_github_repos` | other | yes | [MODE:DIALOG] Search GitHub repositories across all connected installations. Returns deduplicated results. |
| `fibe_get_github_token` | other | yes | [MODE:SIDEEFFECTS] Get a GitHub access token for a repository. Auto-resolves the correct installation. |
| `fibe_gitea_repos_create` | greenfield | yes | [MODE:GREENFIELD] Create a managed Gitea repo and matching Prop. For multi-service switches, batch independent repo creation with fibe_pipeline before seeding source and applying fibe_playgrounds_switch_template. |
| `fibe_github_repos_create` | greenfield | yes | [MODE:GREENFIELD] Register and connect a new GitHub repository |
| `fibe_greenfield_create` | greenfield | yes | [MODE:GREENFIELD] Create one or more repositories/Props, an app-owned template version, deployed playground, wait for running, and link it locally. Deployment requires a funded Marquee. |
| `fibe_help` | meta | yes | [MODE:DIALOG] Display detailed CLI help documentation for a specific Fibe command path. Extremely useful to look up flag descriptions or expected payload shapes. |
| `fibe_launch` | greenfield | yes | [MODE:GREENFIELD] Launch from exactly one source: template, template version, playspec, compose YAML, or repository config. Deployment requires a funded Marquee; unpaid Marquees return MARQUEE_NOT_FUNDED. |
| `fibe_local_conversations_get` | local | yes | [MODE:DIALOG] View one local Codex or Claude conversation by UUID or UUID prefix. |
| `fibe_local_conversations_get_message` | local | yes | [MODE:DIALOG] View one full local conversation message by conversation UUID and message ID. |
| `fibe_local_conversations_list` | local | yes | [MODE:DIALOG] List local Codex, Claude Code, and Claude Desktop conversations from this machine. |
| `fibe_local_playgrounds_info` | brownfield | yes | [MODE:BROWNFIELD] Inspect local playground names, current link state, repo roots, URLs, mounts, or details from /opt/fibe/playgrounds or MARQUEE_ROOT. |
| `fibe_local_playgrounds_link` | brownfield | yes | [MODE:BROWNFIELD] Link local playground mounts into a working directory. |
| `fibe_logs_follow` | brownfield | yes | [MODE:BROWNFIELD] Stream live playground or trick logs as progress notifications. Omitting service streams all services. |
| `fibe_memorize` | base | yes | [MODE:SIDEEFFECTS] Create or update agent-generated memories grounded in one local source conversation. |
| `fibe_monitor_follow` | overseer | yes | [MODE:OVERSEER] Stream agent-produced events as live MCP progress notifications |
| `fibe_monitor_list` | overseer | yes | [MODE:OVERSEER] List agent-produced monitor events |
| `fibe_mutter` | base | yes | [MODE:SIDEEFFECTS] Create one short mutter for an agent: a visible internal note used for progress, proof, blocker, or problem updates. |
| `fibe_mutters_get` | overseer | yes | [MODE:OVERSEER] Retrieve an agent's mutter stream by id_or_name, with optional query/status/severity/playground filters. |
| `fibe_pipeline` | meta | yes | [MODE:SIDEEFFECTS] Execute multiple tool calls sequentially in a single round-trip using JSONPath bindings. The most powerful tool by far! Use to eliminate roundtrip latency when creating and waiting for jobs. |
| `fibe_pipeline_result` | meta | yes | [MODE:DIALOG] Look up a cached result from a previous, the most powerful tool, - pipeline execution |
| `fibe_playgrounds_action` | brownfield | yes | [MODE:SIDEEFFECTS] Run one playground lifecycle action: rollout, hard_restart, stop, start, retry_compose, enable_maintenance, or disable_maintenance. Actions that use the Marquee fail with MARQUEE_NOT_FUNDED when unpaid; stop cleanup remains allowed. |
| `fibe_playgrounds_debug` | brownfield | yes | [MODE:DIALOG] Retrieve comprehensive debugging and diagnostic information for a playground. Use when troubleshooting a failing deployment. |
| `fibe_playgrounds_logs` | brownfield | yes | [MODE:DIALOG] Retrieve playground logs. Omitting service returns all services. Live refresh fails with MARQUEE_NOT_FUNDED when the Marquee is unpaid. |
| `fibe_playgrounds_switch_template` | brownfield | yes | [MODE:BROWNFIELD] Switch a deployed playground end-to-end: preserve the playground id, swap it onto a new template shape, provision missing private Gitea/GitHub-backed Props for new repos, roll it out, wait, and diagnose failures. Single-call brownfield analog of fibe_greenfield_create. Apply mode requires a funded Marquee and fails with MARQUEE_NOT_FUNDED when unpaid. |
| `fibe_playgrounds_wait` | brownfield | yes | [MODE:DIALOG] Block and poll until a playground reaches a specified target state and, for running playgrounds by default, reported services are ready. |
| `fibe_repo_status_check` | other | yes | [MODE:DIALOG] Verify GitHub repository readiness, including runtime writeability and fork/mirror guidance. |
| `fibe_resource_delete` | base | yes | [MODE:SIDEEFFECTS] Delete a supported flat Fibe resource by ID, name, or key where supported. |
| `fibe_resource_get` | base | yes | [MODE:DIALOG] Get a supported Fibe resource by ID, name, or key where supported. Playground reads include service_urls and service runtime status. Use artefact_attachment or agent_attachment to download attached runtime file content. |
| `fibe_resource_list` | base | yes | [MODE:DIALOG] List a supported flat Fibe resource. Use fibe_schema with resource=list to discover resource names, aliases, and list params. |
| `fibe_resource_mutate` | base | yes | [MODE:SIDEEFFECTS] Create, update, or run a supported resource-scoped mutation with a payload validated against fibe_schema before any API request. Actions that use a Marquee require it to be funded. |
| `fibe_resource_watch` | base | yes | [MODE:DIALOG] Watch supported Fibe resource events. |
| `fibe_run` | meta | yes | [MODE:SIDEEFFECTS] Last-resort escape hatch: invoke an arbitrary Fibe CLI command when no dedicated MCP tool fits. Use sparingly. |
| `fibe_schema` | meta | yes | [MODE:DIALOG] Return JSON Schema definitions and the schema resource catalog. |
| `fibe_status` | meta | yes | [MODE:DIALOG] Display a comprehensive dashboard of resource counts, quotas, and rate limits across your account. |
| `fibe_templates_change` | brownfield | no | [MODE:BROWNFIELD] Advanced template change primitive: preview or apply template patches/overwrites, switch playspecs/playgrounds/tricks to existing template versions, and optionally roll out or trigger a fresh trick run. Rollout/trigger actions require a funded Marquee and fail with MARQUEE_NOT_FUNDED when unpaid. |
| `fibe_templates_search` | greenfield | yes | [MODE:GREENFIELD] Search the import-template catalog by text or PostgreSQL regex. Regex mode requires a 3+ character literal token for indexed prefiltering. |
| `fibe_tools_catalog` | meta | yes | [MODE:DIALOG] List all tools registered and available on the Fibe MCP server. CRITICAL: Fibe Platform priority is to let you manage **ALL** its capabilities via its tools so you should find anything here. We just can't advertise them all because there are hundreds |
| `fibe_update_name` | base | yes | [MODE:DIALOG] Update your own agent name. |
