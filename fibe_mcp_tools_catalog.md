# Fibe MCP Tools Catalog

Generated from the MCP registry.

- Registered tools: 60
- Advertised with `FIBE_MCP_TOOLS=full`: 59
- Advertised with `FIBE_MCP_TOOLS=core`: 39
- Hidden dispatcher-only tools: 1

`full` advertises every non-hidden registered tool. Hidden tools remain dispatcher-reachable through `fibe_call` and `fibe_pipeline`, and `fibe_tools_catalog` reports them with `hidden:true`.

## `fibe_agent_defaults_get`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Read the authenticated player's agent default overrides.

## `fibe_agent_defaults_reset`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Clear all player agent default overrides so admin defaults apply.

## `fibe_agent_defaults_update`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Replace the authenticated player's agent default overrides. Use the same agent_defaults JSON shape as the profile UI.

### Input Schema
```json
{
  "properties": {
    "agent_defaults": {
      "description": "Player agent defaults object, including provider_overrides when needed.",
      "properties": {},
      "type": "object"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "agent_defaults"
  ],
  "type": "object"
}
```

## `fibe_agents_activity`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] Read agent activity, optionally scoped to a conversation.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "conversation_id": {
      "description": "Specific conversation/thread ID.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_agents_create_conversation`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Create or upsert an agent conversation.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "conversation_id": {
      "description": "Specific conversation/thread ID.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "title": {
      "description": "Human-readable conversation title. Optional.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name",
    "conversation_id"
  ],
  "type": "object"
}
```

## `fibe_agents_delete_conversation`
**Tier:** overseer | **Hidden:** false | **Destructive:** true | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Delete an agent conversation.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "conversation_id": {
      "description": "Specific conversation/thread ID.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name",
    "conversation_id"
  ],
  "type": "object"
}
```

## `fibe_agents_duplicate`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** false

### Description
[MODE:OVERSEER] Duplicate an agent configuration.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_agents_interrupt`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Interrupt a running agent turn.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "conversation_id": {
      "description": "Specific conversation/thread ID.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_agents_live_state`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] Check conversation-scoped agent live stream state.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "conversation_id": {
      "description": "Specific conversation/thread ID.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_agents_messages`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] Read agent messages, optionally scoped to a conversation.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "conversation_id": {
      "description": "Specific conversation/thread ID.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_agents_runtime_status`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] Check agent reachability, authentication, queue, and processing state. Live checks fail with MARQUEE_NOT_FUNDED when unpaid.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_agents_send_message`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:OVERSEER] Send one text message to an agent chat. Fails with MARQUEE_NOT_FUNDED when the chat Marquee is unpaid.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "attachment_filenames": {
      "description": "Runtime attachment filenames returned by a previous upload. Optional.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "attachment_paths": {
      "description": "Local file paths to upload before sending. Optional.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "busy_policy": {
      "description": "Runtime busy behavior, e.g. queue. Optional.",
      "type": "string"
    },
    "conversation_id": {
      "description": "Specific conversation/thread ID. Optional.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "images": {
      "description": "Image payloads to send to the runtime, such as data URLs. Optional.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "text": {
      "description": "Text to send to the agent.",
      "minLength": 1,
      "type": "string"
    }
  },
  "required": [
    "id_or_name",
    "text"
  ],
  "type": "object"
}
```

## `fibe_agents_start_chat`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Start or reconnect an agent chat on the current Marquee. Requires a funded Marquee; unpaid Marquees fail with MARQUEE_NOT_FUNDED.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "id_or_name": {
      "description": "Agent ID or name.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_artefact_upload`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Upload and save an artefact. Useful when Player asks to create something, implicitly or explicitly

### Input Schema
```json
{
  "properties": {
    "agent_id_or_name": {
      "description": "Optional agent id or name; defaults to FIBE_AGENT_ID when available, otherwise creates a player-owned artefact",
      "type": "string"
    },
    "body": {
      "description": "Inline body for body-only artefacts",
      "type": "string"
    },
    "content_base64": {
      "description": "Base64-encoded file content (alias: 'content')",
      "type": "string"
    },
    "content_path": {
      "description": "Absolute local file path to read (local MCP only)",
      "type": "string"
    },
    "content_text": {
      "description": "Alias for body",
      "type": "string"
    },
    "description": {
      "description": "Optional human-readable description",
      "type": "string"
    },
    "filename": {
      "description": "Target filename — defaults to 'name' when omitted",
      "type": "string"
    },
    "name": {
      "description": "Artefact display name (alias: 'title'). Also used as filename fallback.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "playground_id_or_name": {
      "description": "Optional playground ID or name to associate with the artefact",
      "type": "string"
    },
    "skill": {
      "description": "Expose this artefact as a skill",
      "type": "boolean"
    },
    "skill_enabled": {
      "description": "Enable this artefact skill by default",
      "type": "boolean"
    }
  },
  "required": [
    "name"
  ],
  "type": "object"
}
```

## `fibe_auth_list`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] List local Fibe auth profiles available to this MCP server without revealing API keys.

## `fibe_auth_set`
**Tier:** other | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Configure session-scoped authentication credentials for multi-tenant setups in case you have to work with multiple FIBE_API_KEY+FIBE_DOMAIN combinations

### Input Schema
```json
{
  "properties": {
    "api_key": {
      "description": "Fibe API key, for example fibe_live_... or fibe_test_...",
      "type": "string"
    },
    "domain": {
      "description": "API domain override (default: fibe.gg)",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "validate": {
      "description": "Ping /api/me with the new creds before saving (default: true)",
      "type": "boolean"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_auth_status`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Show the current MCP session auth target and selected profile, if any.

## `fibe_auth_use`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Switch this MCP session to a local Fibe auth profile by name, rebuilding the session client immediately.

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "profile": {
      "description": "Local Fibe auth profile name, for example default, staging, local, or a feature-env profile.",
      "type": "string"
    },
    "validate": {
      "description": "Ping /api/me with the selected profile before saving (default: true).",
      "type": "boolean"
    }
  },
  "required": [
    "profile"
  ],
  "type": "object"
}
```

## `fibe_call`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Invoke a registered Fibe tool that is hidden by the current tool tier. Prefer direct tool calls when the concrete tool is advertised; use fibe_tools_catalog/fibe_schema only when the hidden tool name or args are unclear.

### Input Schema
```json
{
  "properties": {
    "args": {
      "description": "The target tool's args object",
      "properties": {},
      "type": "object"
    },
    "confirm": {
      "description": "Forwarded as args.confirm for destructive tools",
      "type": "boolean"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "tool": {
      "description": "The target tool name, e.g. fibe_playgrounds_debug",
      "type": "string"
    }
  },
  "required": [
    "tool"
  ],
  "type": "object"
}
```

## `fibe_doctor`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Run self-diagnostic checks: verify API key, connectivity, and display user profile

## `fibe_feedbacks_get`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] Get one feedback entry for an agent, including player comments about artefacts or mutters.

### Input Schema
```json
{
  "properties": {
    "feedback_id": {
      "description": "Feedback ID",
      "minimum": 1,
      "type": "number"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "feedback_id"
  ],
  "type": "object"
}
```

## `fibe_feedbacks_list`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] List all feedback entries associated with an agent.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "created_after": {
      "description": "Filter to feedback created at or after this timestamp.",
      "type": "string"
    },
    "created_before": {
      "description": "Filter to feedback created at or before this timestamp.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "page": {
      "description": "Page number.",
      "minimum": 1,
      "type": "integer"
    },
    "per_page": {
      "description": "Number of results per page.",
      "minimum": 1,
      "type": "integer"
    },
    "playground_id_or_name": {
      "description": "Optional playground ID or slug-safe name filter.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "query": {
      "description": "Search across comment, selected_text, and context.",
      "type": "string"
    },
    "sort": {
      "description": "Sort order, for example created_at_desc.",
      "type": "string"
    },
    "source_id": {
      "description": "Filter by source identifier.",
      "type": "string"
    },
    "source_type": {
      "description": "Filter by feedback source type.",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `fibe_find_github_repos`
**Tier:** other | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Search GitHub repositories across all connected installations. Returns deduplicated results.

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "page": {
      "description": "Page number (default: 1)",
      "minimum": 1,
      "type": "number"
    },
    "per_page": {
      "description": "Results per page (default: 30, max: 100)",
      "minimum": 1,
      "type": "number"
    },
    "q": {
      "description": "Search query (filters by repo name). Optional; omit to list all accessible repos.",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_get_github_token`
**Tier:** other | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Get a GitHub access token for a repository. Auto-resolves the correct installation.

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "repo": {
      "description": "Full repo name, e.g. owner/repo",
      "type": "string"
    }
  },
  "required": [
    "repo"
  ],
  "type": "object"
}
```

## `fibe_gitea_repos_create`
**Tier:** greenfield | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:GREENFIELD] Create a managed Gitea repo and matching Prop. For multi-service switches, batch independent repo creation with fibe_pipeline before seeding source and applying fibe_playgrounds_switch_template.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "auto_init": {
      "description": "Initialize the repository with default files.",
      "type": [
        "null",
        "boolean"
      ]
    },
    "description": {
      "description": "Optional description.",
      "type": [
        "null",
        "string"
      ]
    },
    "name": {
      "description": "Name.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "private": {
      "description": "Whether the repository is private.",
      "type": [
        "null",
        "boolean"
      ]
    }
  },
  "required": [
    "name"
  ],
  "type": "object"
}
```

## `fibe_github_repos_create`
**Tier:** greenfield | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:GREENFIELD] Register and connect a new GitHub repository

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "auto_init": {
      "description": "Initialize the repository with default files.",
      "type": [
        "null",
        "boolean"
      ]
    },
    "description": {
      "description": "Optional description.",
      "type": [
        "null",
        "string"
      ]
    },
    "name": {
      "description": "Name.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "private": {
      "description": "Whether the repository is private.",
      "type": [
        "null",
        "boolean"
      ]
    }
  },
  "required": [
    "name"
  ],
  "type": "object"
}
```

## `fibe_greenfield_create`
**Tier:** greenfield | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:GREENFIELD] Create one or more repositories/Props, an app-owned template version, deployed playground, wait for running, and link it locally. Deployment requires a funded Marquee.

### Input Schema
```json
{
  "properties": {
    "config_path": {
      "description": "Config file path inside the GitHub repository. Optional; defaults to fibe.yml, fibe.yaml, docker-compose.yml, docker-compose.yaml.",
      "type": "string"
    },
    "git_provider": {
      "description": "Destination git provider: gitea or github. Optional; default: gitea.",
      "enum": [
        "gitea",
        "github"
      ],
      "type": "string"
    },
    "github_account": {
      "description": "GitHub App installation account owner to use when multiple installations are connected.",
      "type": "string"
    },
    "github_installation_id": {
      "description": "GitHub App installation ID to use when multiple installations are connected.",
      "minimum": 1,
      "type": "number"
    },
    "github_ref": {
      "description": "Git branch, tag, or commit for the config file. Optional.",
      "type": "string"
    },
    "marquee_id_or_name": {
      "description": "Target marquee ID or name. Optional; defaults to the current Marquee from FIBE_MARQUEE_ID. Must be funded.",
      "type": "string"
    },
    "name": {
      "description": "Repository/app name; must be unique. Optional when repository_url is provided; inferred from repo name.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "private": {
      "description": "Create destination repository as private. Optional; Fibe defaults Gitea greenfield repos to private.",
      "type": "boolean"
    },
    "repository_url": {
      "description": "GitHub repository as owner/repo, owner/repo@ref, or https://github.com/owner/repo. Optional alternative to template inputs.",
      "type": "string"
    },
    "service_subdomains": {
      "description": "Exposed service subdomain overrides, e.g. {\"app\":\"my-app\",\"admin\":\"my-app-admin\"}. Optional.",
      "properties": {},
      "type": "object"
    },
    "template_body": {
      "description": "Template YAML body to use directly. Optional; cannot be combined with template_id_or_name, template_version_id, or version.",
      "type": "string"
    },
    "template_body_path": {
      "description": "Absolute local path to a template YAML file (local MCP only). Optional; cannot be combined with template_body, template_id_or_name, template_version_id, or version.",
      "type": "string"
    },
    "template_id_or_name": {
      "description": "Template ID or name to use. Optional; defaults to the base template.",
      "type": "string"
    },
    "template_version_id": {
      "description": "Exact template version ID to use. Optional; cannot be combined with template_id_or_name or version.",
      "minimum": 1,
      "type": "number"
    },
    "variables": {
      "description": "Template variables map, e.g. {\"app_name\":\"Tower\"}. Optional.",
      "properties": {},
      "type": "object"
    },
    "version": {
      "description": "Template version tag or number for template_id_or_name, e.g. v1. Optional; defaults to latest available version.",
      "type": "string"
    },
    "wait_timeout": {
      "description": "Max wait duration, e.g. 10m. Optional; default: 10m.",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_help`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Display detailed CLI help documentation for a specific Fibe command path. Extremely useful to look up flag descriptions or expected payload shapes.

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "path": {
      "description": "Space-separated command path, e.g. \"playgrounds create\". Empty = root help.",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_launch`
**Tier:** greenfield | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:GREENFIELD] Launch from exactly one source: template, template version, playspec, compose YAML, or repository config. Deployment requires a funded Marquee; unpaid Marquees return MARQUEE_NOT_FUNDED.

### Input Schema
```json
{
  "properties": {
    "compose_yaml": {
      "description": "Docker Compose or Fibe YAML content. Optional when repository_url is provided.",
      "type": "string"
    },
    "compose_yaml_path": {
      "description": "Absolute local path to Docker Compose or Fibe YAML (local MCP only). Optional when repository_url is provided.",
      "type": "string"
    },
    "config_path": {
      "description": "Config file path inside the GitHub repository. Optional; defaults to fibe.yml, fibe.yaml, docker-compose.yml, docker-compose.yaml.",
      "type": "string"
    },
    "create_playground": {
      "description": "Force playground creation. Defaults to true when marquee_id_or_name is set, false otherwise.",
      "type": "boolean"
    },
    "diagnose_on_failure": {
      "description": "Diagnose failed waits where supported.",
      "type": "boolean"
    },
    "env_overrides": {
      "description": "Runtime environment overrides for the created Playground.",
      "properties": {},
      "type": "object"
    },
    "github_account": {
      "description": "GitHub App installation account owner to use when multiple installations are connected.",
      "type": "string"
    },
    "github_installation_id": {
      "description": "GitHub App installation ID to use when multiple installations are connected.",
      "minimum": 1,
      "type": "number"
    },
    "github_ref": {
      "description": "Git branch, tag, or commit for the config file. Optional.",
      "type": "string"
    },
    "job_mode": {
      "description": "Create as a trick/job instead of a playground. Requires marquee_id_or_name.",
      "type": "boolean"
    },
    "marquee_id_or_name": {
      "description": "Target marquee ID or name. Required for template/playspec launch; compose/repo can omit to create only the playspec unless create_playground is true.",
      "type": "string"
    },
    "name": {
      "description": "Launch name. Optional when repository_url is provided; inferred from repo name.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "persist_volumes": {
      "description": "Persist Docker volumes across trick/playground recreations. Optional; omitted means the server infers from named compose volumes.",
      "type": "boolean"
    },
    "playspec_id_or_name": {
      "description": "Existing playspec ID or name. Mutually exclusive with other source fields.",
      "type": "string"
    },
    "prop_mappings": {
      "description": "Map repository URL to Prop ID or name. Optional.",
      "properties": {},
      "type": "object"
    },
    "repository_url": {
      "description": "GitHub repository as owner/repo, owner/repo@ref, or https://github.com/owner/repo. Optional alternative to compose_yaml.",
      "type": "string"
    },
    "response_mode": {
      "description": "Server response detail mode where supported.",
      "enum": [
        "summary",
        "full"
      ],
      "type": "string"
    },
    "service_subdomains": {
      "description": "Per-service subdomain overrides.",
      "properties": {},
      "type": "object"
    },
    "services": {
      "description": "Per-service runtime Playground configuration overrides.",
      "properties": {},
      "type": "object"
    },
    "template_id_or_name": {
      "description": "Template ID or name. Mutually exclusive with other source fields; without version/template_version_id, latest version is used.",
      "type": "string"
    },
    "template_version_id": {
      "description": "Exact template version ID. Mutually exclusive with other source fields.",
      "minimum": 1,
      "type": "number"
    },
    "variables": {
      "description": "Template variables map for Fibe template compilation. Optional.",
      "properties": {},
      "type": "object"
    },
    "wait": {
      "description": "Wait for created playground to reach running where supported.",
      "type": "boolean"
    },
    "wait_timeout_seconds": {
      "description": "Wait timeout in seconds.",
      "minimum": 1,
      "type": "number"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_local_conversations_get`
**Tier:** local | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] View one local Codex or Claude conversation by UUID or UUID prefix.

### Input Schema
```json
{
  "properties": {
    "assistant_message_limit": {
      "description": "Maximum characters per assistant message preview. Default 10000; pass 0 for no limit.",
      "type": "number"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "user_message_limit": {
      "description": "Maximum characters per user message preview. Default 5000; pass 0 for no limit.",
      "type": "number"
    },
    "uuid": {
      "description": "Conversation UUID or unique UUID prefix from fibe_local_conversations_list.",
      "type": "string"
    },
    "view": {
      "description": "Output view: messages (default), chat, user-messages, or full.",
      "enum": [
        "messages",
        "chat",
        "user-messages",
        "full"
      ],
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "type": "object"
}
```

## `fibe_local_conversations_get_message`
**Tier:** local | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] View one full local conversation message by conversation UUID and message ID.

### Input Schema
```json
{
  "properties": {
    "message_id": {
      "description": "Message id from fibe_local_conversations_get. Numeric positions such as 1 are also accepted.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "uuid": {
      "description": "Conversation UUID or unique UUID prefix from fibe_local_conversations_list.",
      "type": "string"
    }
  },
  "required": [
    "uuid",
    "message_id"
  ],
  "type": "object"
}
```

## `fibe_local_conversations_list`
**Tier:** local | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] List local Codex, Claude Code, and Claude Desktop conversations from this machine.

### Input Schema
```json
{
  "properties": {
    "cursor": {
      "description": "Opaque cursor returned as next_cursor by a previous list call. Use it to continue to the next page.",
      "type": "string"
    },
    "limit": {
      "description": "Maximum conversations to return. Default 25; pass 0 for no limit.",
      "type": "number"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "query": {
      "description": "Case-insensitive substring search across local transcript file content and conversation UUIDs.",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_local_playgrounds_info`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:BROWNFIELD] Inspect local playground names, URLs, mounts, or details from /opt/fibe/playgrounds or MARQUEE_ROOT.

### Input Schema
```json
{
  "properties": {
    "id_or_name": {
      "description": "Local playground ID, name, compose project, playspec, or unique playspec prefix. Omit for view=names.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "view": {
      "description": "Output view: names, urls, mounts, or details.",
      "enum": [
        "names",
        "urls",
        "mounts",
        "details"
      ],
      "type": "string"
    }
  },
  "required": [
    "view"
  ],
  "type": "object"
}
```

## `fibe_local_playgrounds_link`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** false

### Description
[MODE:BROWNFIELD] Link local playground mounts into a working directory.

### Input Schema
```json
{
  "properties": {
    "id_or_name": {
      "description": "Local playground ID, name, compose project, playspec, or unique playspec prefix",
      "type": "string"
    },
    "link_dir": {
      "description": "Target directory for symlinks (default: /app/playground)",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_logs_follow`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:BROWNFIELD] Stream live playground or trick logs as progress notifications. Omitting service streams all services.

### Input Schema
```json
{
  "properties": {
    "duration": {
      "description": "Max follow duration (Go duration, default: 30s)",
      "type": "string"
    },
    "id_or_name": {
      "description": "Playground or trick numeric ID or slug-safe name",
      "type": "string"
    },
    "max_lines": {
      "description": "Stop after N new lines (default: 500)",
      "minimum": 1,
      "type": "number"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "service": {
      "description": "Optional Compose service name, for example web or worker. Omit to stream all services.",
      "type": "string"
    },
    "tail": {
      "description": "Initial lines from history (default: 50)",
      "minimum": 1,
      "type": "number"
    },
    "target": {
      "description": "Target type (default: playground).",
      "enum": [
        "playground",
        "trick"
      ],
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_memorize`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Create or update agent-generated memories grounded in one local source conversation.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "agent_id_or_name": {
      "description": "Optional Fibe Agent ID or name that created the memory. fibe_memorize fills this from FIBE_AGENT_ID when available.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "confidence": {
      "description": "Confidence from 0 to 1.",
      "maximum": 1,
      "minimum": 0,
      "type": "number"
    },
    "content": {
      "description": "Durable memory text.",
      "minLength": 1,
      "type": "string"
    },
    "conversation_id": {
      "description": "Stable local source conversation UUID. This is not a server database ID.",
      "minLength": 1,
      "type": "string"
    },
    "groundings": {
      "description": "Proof references into normalized messages or raw provider events.",
      "items": {
        "additionalProperties": false,
        "properties": {
          "end_character": {
            "description": "End character offset within the normalized message content.",
            "minimum": 0,
            "type": "integer"
          },
          "message_position": {
            "description": "Zero-based source message position.",
            "minimum": 0,
            "type": "integer"
          },
          "metadata": {
            "description": "Optional grounding metadata.",
            "type": "object"
          },
          "provider_message_uuid": {
            "description": "Provider message UUID when available.",
            "type": "string"
          },
          "quote": {
            "description": "Short proof excerpt. Maximum 2000 characters on the server.",
            "type": "string"
          },
          "raw_end_character": {
            "description": "End character offset within raw content or raw event text.",
            "minimum": 0,
            "type": "integer"
          },
          "raw_event_index": {
            "description": "Zero-based raw event index when grounding points to raw events.",
            "minimum": 0,
            "type": "integer"
          },
          "raw_start_character": {
            "description": "Start character offset within raw content or raw event text.",
            "minimum": 0,
            "type": "integer"
          },
          "start_character": {
            "description": "Start character offset within the normalized message content.",
            "minimum": 0,
            "type": "integer"
          }
        },
        "type": "object"
      },
      "type": "array"
    },
    "memory_key": {
      "description": "Optional exact idempotency key. Omit this and the server computes one from content, tags, conversation_id, and groundings.",
      "type": "string"
    },
    "metadata": {
      "description": "Optional memory metadata.",
      "type": "object"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "tags": {
      "description": "Memory tags. The server normalizes tags to lowercase slug-like strings.",
      "items": {
        "type": "string"
      },
      "type": "array"
    }
  },
  "required": [
    "conversation_id",
    "content"
  ],
  "type": "object"
}
```

## `fibe_monitor_follow`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] Stream agent-produced events as live MCP progress notifications

### Input Schema
```json
{
  "properties": {
    "agent": {
      "description": "Comma-separated agent IDs. Empty = all accessible.",
      "type": "string"
    },
    "content_limit": {
      "description": "Advanced: truncate each payload to N bytes (default: 32768, max: 131072)",
      "minimum": 1,
      "type": "number"
    },
    "duration": {
      "description": "Max follow duration as Go duration (default: 30s, max: 30m)",
      "type": "string"
    },
    "max_events": {
      "description": "Stop after N events (default: 100)",
      "minimum": 1,
      "type": "number"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "poll_interval": {
      "description": "Polling interval as Go duration (default: 2s)",
      "type": "string"
    },
    "q": {
      "description": "Full-text search across content",
      "type": "string"
    },
    "since": {
      "description": "Lower bound ISO 8601 (default: now)",
      "type": "string"
    },
    "type": {
      "description": "Comma-separated types: message, activity, mutter, artefact",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_monitor_list`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] List agent-produced monitor events

### Input Schema
```json
{
  "properties": {
    "agent": {
      "description": "Comma-separated agent IDs. Empty = all accessible.",
      "type": "string"
    },
    "content_limit": {
      "description": "Advanced: truncate each payload to N bytes (default: 32768, max: 131072)",
      "minimum": 1,
      "type": "number"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "page": {
      "description": "Page number (default: 1)",
      "minimum": 1,
      "type": "number"
    },
    "per_page": {
      "description": "Page size (default: 25, max: 100)",
      "minimum": 1,
      "type": "number"
    },
    "q": {
      "description": "Full-text search across content",
      "type": "string"
    },
    "since": {
      "description": "Lower bound ISO 8601",
      "type": "string"
    },
    "type": {
      "description": "Comma-separated types: message, activity, mutter, artefact",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_mutter`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Create one short mutter for an agent: a visible internal note used for progress, proof, blocker, or problem updates.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "body": {
      "description": "Mutter body text.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "playground_id_or_name": {
      "description": "Optional playground ID or slug-safe name to associate with the mutter.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "type": {
      "description": "Mutter type label. Common values are info, warning, error, and success; Server accepts arbitrary strings.",
      "type": "string"
    }
  },
  "required": [
    "type",
    "body"
  ],
  "type": "object"
}
```

## `fibe_mutters_get`
**Tier:** overseer | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:OVERSEER] Retrieve an agent's mutter stream by id_or_name, with optional query/status/severity/playground filters.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "id_or_name": {
      "description": "Agent ID or name whose mutter stream should be retrieved.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "page": {
      "description": "Page number for paginated mutter results.",
      "minimum": 1,
      "type": "integer"
    },
    "per_page": {
      "description": "Number of mutter results per page.",
      "minimum": 1,
      "type": "integer"
    },
    "playground_id_or_name": {
      "description": "Optional playground ID or slug-safe name used to filter the mutter stream.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "query": {
      "description": "Optional substring search across mutter values.",
      "type": "string"
    },
    "severity": {
      "description": "Optional severity filter.",
      "type": "string"
    },
    "status": {
      "description": "Optional status filter.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_pipeline`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Execute multiple tool calls sequentially in a single round-trip using JSONPath bindings. The most powerful tool by far! Use to eliminate roundtrip latency when creating and waiting for jobs.

### Input Schema
```json
{
  "properties": {
    "cache": {
      "description": "Set to false to skip caching this pipeline's result (default: true)",
      "type": "boolean"
    },
    "dry_run": {
      "description": "Validate refs + schemas without executing",
      "type": "boolean"
    },
    "idempotency_key": {
      "description": "Optional key threaded through destructive steps for retry safety",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "return": {
      "description": "Optional JSONPath or object spec to project the final return",
      "type": "string"
    },
    "steps": {
      "description": "Ordered list of steps to execute.\n\nEach step is one of:\n  {\"id\": \"\u003cstep_id\u003e\", \"tool\": \"\u003ctool_name\u003e\", \"args\": {...}}              (single tool call)\n  {\"parallel\": [\u003cstep\u003e, \u003cstep\u003e, ...]}                                   (independent concurrent steps)\n  {\"id\": \"\u003cstep_id\u003e\", \"for_each\": \"$.list\", \"as\": \"item\",\n    \"steps\": [\u003cstep\u003e, ...], \"collect\": \"$.something\"}                   (fanout)\n\nArgs may contain JSONPath references starting with \"$.\", resolved against the map of prior step outputs.",
      "items": {
        "description": "A pipeline step: tool call, parallel block, or for_each fanout.",
        "properties": {
          "args": {
            "description": "Argument object or command tokens for the target operation.",
            "type": "object"
          },
          "as": {
            "description": "As for this pipeline request.",
            "type": "string"
          },
          "collect": {
            "description": "Collect for this pipeline request.",
            "type": "string"
          },
          "for_each": {
            "description": "For Each for this pipeline request.",
            "type": "string"
          },
          "id": {
            "description": "ID for this pipeline request.",
            "type": "string"
          },
          "input_path": {
            "description": "Input Path path.",
            "type": "string"
          },
          "on_error": {
            "description": "Error handling mode for this pipeline step.",
            "enum": [
              "abort",
              "continue"
            ],
            "type": "string"
          },
          "output_path": {
            "description": "Output Path path.",
            "type": "string"
          },
          "parallel": {
            "description": "Sub-steps to execute concurrently.",
            "items": {
              "description": "A nested pipeline step.",
              "properties": {
                "args": {
                  "description": "Argument object or command tokens for the target operation.",
                  "type": "object"
                },
                "as": {
                  "description": "As for this pipeline request.",
                  "type": "string"
                },
                "collect": {
                  "description": "Collect for this pipeline request.",
                  "type": "string"
                },
                "for_each": {
                  "description": "For Each for this pipeline request.",
                  "type": "string"
                },
                "id": {
                  "description": "ID for this pipeline request.",
                  "type": "string"
                },
                "input_path": {
                  "description": "Input Path path.",
                  "type": "string"
                },
                "on_error": {
                  "description": "Error handling mode for this pipeline step.",
                  "enum": [
                    "abort",
                    "continue"
                  ],
                  "type": "string"
                },
                "output_path": {
                  "description": "Output Path path.",
                  "type": "string"
                },
                "tool": {
                  "description": "Registered Fibe tool name.",
                  "type": "string"
                }
              },
              "type": "object"
            },
            "type": "array"
          },
          "steps": {
            "description": "Sub-steps executed per for_each iteration.",
            "items": {
              "description": "A nested pipeline step.",
              "properties": {
                "args": {
                  "description": "Argument object or command tokens for the target operation.",
                  "type": "object"
                },
                "as": {
                  "description": "As for this pipeline request.",
                  "type": "string"
                },
                "collect": {
                  "description": "Collect for this pipeline request.",
                  "type": "string"
                },
                "for_each": {
                  "description": "For Each for this pipeline request.",
                  "type": "string"
                },
                "id": {
                  "description": "ID for this pipeline request.",
                  "type": "string"
                },
                "input_path": {
                  "description": "Input Path path.",
                  "type": "string"
                },
                "on_error": {
                  "description": "Error handling mode for this pipeline step.",
                  "enum": [
                    "abort",
                    "continue"
                  ],
                  "type": "string"
                },
                "output_path": {
                  "description": "Output Path path.",
                  "type": "string"
                },
                "tool": {
                  "description": "Registered Fibe tool name.",
                  "type": "string"
                }
              },
              "type": "object"
            },
            "type": "array"
          },
          "tool": {
            "description": "Registered Fibe tool name.",
            "type": "string"
          }
        },
        "type": "object"
      },
      "type": "array"
    }
  },
  "required": [
    "steps"
  ],
  "type": "object"
}
```

## `fibe_pipeline_result`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Look up a cached result from a previous, the most powerful tool, - pipeline execution

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "path": {
      "description": "Optional JSONPath. Rooted at step bindings; falls back to full response for \"$.status\" / \"$.error\" etc.",
      "type": "string"
    },
    "pipeline_id": {
      "description": "ID returned by a prior fibe_pipeline call",
      "type": "string"
    }
  },
  "required": [
    "pipeline_id"
  ],
  "type": "object"
}
```

## `fibe_playgrounds_action`
**Tier:** brownfield | **Hidden:** false | **Destructive:** true | **Idempotent:** true | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Run one playground lifecycle action: rollout, hard_restart, stop, start, retry_compose, enable_maintenance, or disable_maintenance. Actions that use the Marquee fail with MARQUEE_NOT_FUNDED when unpaid; stop cleanup remains allowed.

### Input Schema
```json
{
  "properties": {
    "action_type": {
      "description": "Lifecycle action to perform.",
      "enum": [
        "rollout",
        "hard_restart",
        "stop",
        "start",
        "retry_compose",
        "enable_maintenance",
        "disable_maintenance"
      ],
      "type": "string"
    },
    "confirm": {
      "description": "Must be true unless server is running with --yolo",
      "type": "boolean"
    },
    "force": {
      "description": "Bypass eligible state guards when the server permits forced execution.",
      "type": "boolean"
    },
    "id_or_name": {
      "description": "Playground numeric ID or slug-safe name",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name",
    "action_type"
  ],
  "type": "object"
}
```

## `fibe_playgrounds_debug`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Retrieve comprehensive debugging and diagnostic information for a playground. Use when troubleshooting a failing deployment.

### Input Schema
```json
{
  "properties": {
    "id_or_name": {
      "description": "Playground numeric ID or slug-safe name",
      "type": "string"
    },
    "logs_tail": {
      "description": "Optional number of service log lines to include.",
      "minimum": 1,
      "type": "number"
    },
    "mode": {
      "description": "summary (default) or full",
      "enum": [
        "summary",
        "full"
      ],
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "refresh": {
      "description": "Refresh Docker state before reading diagnostics (default: true)",
      "type": "boolean"
    },
    "service": {
      "description": "Optional Compose service name to focus diagnostics on.",
      "type": "string"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_playgrounds_logs`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Retrieve playground logs. Omitting service returns all services. Live refresh fails with MARQUEE_NOT_FUNDED when the Marquee is unpaid.

### Input Schema
```json
{
  "properties": {
    "id_or_name": {
      "description": "Playground numeric ID or slug-safe name",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "service": {
      "description": "Optional Compose service name, for example web or worker. Omit to return all services.",
      "type": "string"
    },
    "tail": {
      "description": "Number of log lines to return (default: 50)",
      "minimum": 1,
      "type": "number"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_playgrounds_switch_template`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:BROWNFIELD] Switch a deployed playground end-to-end: preserve the playground id, swap it onto a new template shape, provision missing private Gitea/GitHub-backed Props for new repos, roll it out, wait, and diagnose failures. Single-call brownfield analog of fibe_greenfield_create. Apply mode requires a funded Marquee and fails with MARQUEE_NOT_FUNDED when unpaid.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "changelog": {
      "description": "Optional changelog stamped on the freshly created template version (when template_body is provided).",
      "type": "string"
    },
    "confirm": {
      "description": "Required true for mode=apply unless server runs with --yolo.",
      "type": "boolean"
    },
    "confirm_warnings": {
      "description": "Allow apply to proceed when preview reports switch warnings (e.g. dropped services).",
      "type": "boolean"
    },
    "diagnose_on_failure": {
      "description": "When wait fails, attach playground debug summary. Default true.",
      "type": "boolean"
    },
    "id_or_name": {
      "description": "Playground ID or slug-safe name of the deployed playground to switch-template.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "mode": {
      "description": "Preview validates and reports diffs/warnings/required variables without writes; apply commits the change. Defaults to apply.",
      "enum": [
        "preview",
        "apply"
      ],
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "provision_inputs": {
      "description": "Per-URL overrides for provisioning. Each item: {source_repo_url, name_override?, default_branch?, description?, auto_init?}.",
      "items": {
        "additionalProperties": false,
        "properties": {
          "auto_init": {
            "description": "Initialize the repository with default files.",
            "type": "boolean"
          },
          "default_branch": {
            "description": "Default Branch for this playgrounds_switch_template request.",
            "type": "string"
          },
          "description": {
            "description": "Optional description.",
            "type": "string"
          },
          "name_override": {
            "description": "Name Override for this playgrounds_switch_template request.",
            "type": "string"
          },
          "source_repo_url": {
            "description": "Source Repo URL URL.",
            "type": "string"
          }
        },
        "required": [
          "source_repo_url"
        ],
        "type": "object"
      },
      "type": "array"
    },
    "provision_missing_props": {
      "description": "When the new template references repos the player does not yet own a Prop for, automatically provision a fresh git repo (in the player's connected Gitea or GitHub account) and create a Prop for each. Default \"gitea\" when omitted on this tool. Set to \"off\" to disable and require existing Props.",
      "enum": [
        "off",
        "gitea",
        "github"
      ],
      "type": "string"
    },
    "provision_private": {
      "description": "Whether the freshly provisioned repos should be private. Defaults to true.",
      "type": "boolean"
    },
    "regenerate_variables": {
      "description": "Variable names to regenerate from defaults instead of carrying over.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "response_mode": {
      "description": "Response detail mode for the underlying switch result.",
      "enum": [
        "summary",
        "full"
      ],
      "type": "string"
    },
    "reuse_existing_props": {
      "description": "Reuse the playground's existing dynamic service Props for the target template before creating new ones. Matching keeps same service names first, then current service order; extra old Props are retired after the switch when they are no longer referenced.",
      "type": "boolean"
    },
    "template_body": {
      "description": "Inline template YAML. Authoring a new target shape on the fly: a new ImportTemplateVersion is created, then the playground is switched to it. Mutually exclusive with template_version_id.",
      "type": "string"
    },
    "template_body_path": {
      "description": "Absolute local path to template YAML (local MCP only).",
      "type": "string"
    },
    "template_id_or_name": {
      "description": "Existing ImportTemplate ID or name to use. Without template_body, the latest version is selected. With template_body, a new version is published under this template.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "template_name": {
      "description": "Optional name for the freshly created ImportTemplate when template_body is provided and template_id_or_name is not. Auto-generated otherwise.",
      "type": "string"
    },
    "template_version_id": {
      "description": "Exact existing ImportTemplateVersion to switch to. Mutually exclusive with template_body.",
      "minimum": 1,
      "type": "integer"
    },
    "variables": {
      "description": "Template variable values for the new template version.",
      "type": "object"
    },
    "wait": {
      "description": "Wait for the playground to reach a running state after rollout. Default true.",
      "type": "boolean"
    },
    "wait_timeout_seconds": {
      "description": "Max seconds to wait. Default 180.",
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "id_or_name"
  ],
  "type": "object"
}
```

## `fibe_playgrounds_wait`
**Tier:** brownfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Block and poll until a playground reaches a specified target state and, for running playgrounds by default, reported services are ready.

### Input Schema
```json
{
  "properties": {
    "id_or_name": {
      "description": "Playground numeric ID or slug-safe name",
      "type": "string"
    },
    "interval": {
      "description": "Polling interval as Go duration string (default: 3s)",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "readiness": {
      "description": "Readiness mode: services waits for reported service readiness when status is running; lifecycle waits only for the top-level status. Defaults to services for status=running and lifecycle otherwise.",
      "type": "string"
    },
    "status": {
      "description": "Target playground status, for example running, stopped, or has_changes.",
      "type": "string"
    },
    "timeout": {
      "description": "Max wait duration as Go duration string (e.g. \"5m\"; default: 10m)",
      "type": "string"
    }
  },
  "required": [
    "id_or_name",
    "status"
  ],
  "type": "object"
}
```

## `fibe_repo_status_check`
**Tier:** other | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Verify the system's access and view of multiple GitHub repository URLs.

### Input Schema
```json
{
  "properties": {
    "github_urls": {
      "description": "GitHub repository URLs to check.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "github_urls"
  ],
  "type": "object"
}
```

## `fibe_resource_delete`
**Tier:** base | **Hidden:** false | **Destructive:** true | **Idempotent:** true | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Delete a supported flat Fibe resource by ID, name, or key where supported.

### Input Schema
```json
{
  "properties": {
    "agent_id_or_name": {
      "description": "Agent ID or name for resource-specific deletes such as agent_poke.",
      "type": "string"
    },
    "confirm": {
      "description": "Must be true unless server is running with --yolo",
      "type": "boolean"
    },
    "id": {
      "description": "Numeric ID of the selected resource.",
      "minimum": 1,
      "type": "number"
    },
    "id_or_key": {
      "description": "Numeric ID or key for secrets.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Numeric ID or name for named resources.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "resource": {
      "description": "Canonical resource name or explicit alias, e.g. playground, playspec, prop, api_key.",
      "enum": [
        "agent",
        "agent-poke",
        "agent-pokes",
        "agent_poke",
        "agent_pokes",
        "agents",
        "api-key",
        "api-keys",
        "api_key",
        "api_keys",
        "import-template",
        "import-templates",
        "import_template",
        "import_templates",
        "job-env",
        "job-environment",
        "job-environments",
        "job-envs",
        "job_env",
        "job_environment",
        "job_environments",
        "job_envs",
        "marquee",
        "marquees",
        "memories",
        "memory",
        "playground",
        "playgrounds",
        "playspec",
        "playspecs",
        "pokes",
        "prop",
        "props",
        "secret",
        "secrets",
        "template",
        "template-source",
        "template-sources",
        "template-version",
        "template-versions",
        "template_source",
        "template_sources",
        "template_version",
        "template_versions",
        "templates",
        "trick",
        "tricks",
        "webhook",
        "webhook-endpoint",
        "webhook-endpoints",
        "webhook_endpoint",
        "webhook_endpoints",
        "webhooks"
      ],
      "type": "string"
    },
    "template_id_or_name": {
      "description": "Template ID or name for resource-specific deletes such as template_version.",
      "type": "string"
    }
  },
  "required": [
    "resource"
  ],
  "type": "object"
}
```

## `fibe_resource_get`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Get a supported Fibe resource by ID, name, or key where supported. Use artefact_attachment or agent_attachment to download attached runtime file content.

### Input Schema
```json
{
  "properties": {
    "agent_id_or_name": {
      "description": "Agent ID or name for resource-specific reads such as agent_attachment.",
      "type": "string"
    },
    "conversation_id": {
      "description": "Runtime conversation/thread ID for resource-specific reads such as agent_attachment.",
      "type": "string"
    },
    "filename": {
      "description": "Runtime filename for resource-specific reads such as agent_attachment.",
      "type": "string"
    },
    "id": {
      "description": "Numeric ID of the selected resource.",
      "minimum": 1,
      "type": "number"
    },
    "id_or_key": {
      "description": "Numeric ID or key for secrets.",
      "type": "string"
    },
    "id_or_name": {
      "description": "Numeric ID or name for named resources.",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "resource": {
      "description": "Canonical resource name or explicit alias, e.g. playground, artefact, artefact_attachment, playspec, prop, webhook.",
      "enum": [
        "agent",
        "agent-attachment",
        "agent-attachments",
        "agent-poke",
        "agent-pokes",
        "agent-upload",
        "agent-uploads",
        "agent_attachment",
        "agent_attachments",
        "agent_poke",
        "agent_pokes",
        "agent_upload",
        "agent_uploads",
        "agents",
        "artefact",
        "artefact-attachment",
        "artefact-attachments",
        "artefact_attachment",
        "artefact_attachments",
        "artefacts",
        "import-template",
        "import-templates",
        "import_template",
        "import_templates",
        "job-env",
        "job-environment",
        "job-environments",
        "job-envs",
        "job_env",
        "job_environment",
        "job_environments",
        "job_envs",
        "marquee",
        "marquees",
        "memories",
        "memory",
        "playground",
        "playgrounds",
        "playspec",
        "playspecs",
        "pokes",
        "prop",
        "props",
        "secret",
        "secrets",
        "template",
        "templates",
        "trick",
        "tricks",
        "webhook",
        "webhook-endpoint",
        "webhook-endpoints",
        "webhook_endpoint",
        "webhook_endpoints",
        "webhooks"
      ],
      "type": "string"
    }
  },
  "required": [
    "resource"
  ],
  "type": "object"
}
```

## `fibe_resource_list`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] List a supported flat Fibe resource. Use fibe_schema with resource=list to discover resource names, aliases, and list params.

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "params": {
      "description": "Resource-specific list filters. Inspect with fibe_schema(resource:\u003cname\u003e, operation:list).",
      "properties": {},
      "type": "object"
    },
    "resource": {
      "description": "Canonical resource name or explicit alias, e.g. playground, playspec, prop, api_key.",
      "enum": [
        "agent",
        "agent-poke",
        "agent-pokes",
        "agent_poke",
        "agent_pokes",
        "agents",
        "api-key",
        "api-keys",
        "api_key",
        "api_keys",
        "artefact",
        "artefacts",
        "audit-log",
        "audit-logs",
        "audit_log",
        "audit_logs",
        "categories",
        "category",
        "import-template",
        "import-templates",
        "import_template",
        "import_templates",
        "job-env",
        "job-environment",
        "job-environments",
        "job-envs",
        "job_env",
        "job_environment",
        "job_environments",
        "job_envs",
        "marquee",
        "marquees",
        "memories",
        "memory",
        "playground",
        "playgrounds",
        "playspec",
        "playspecs",
        "pokes",
        "prop",
        "props",
        "secret",
        "secrets",
        "template",
        "template-categories",
        "template-category",
        "template-version",
        "template-versions",
        "template_categories",
        "template_category",
        "template_version",
        "template_versions",
        "templates",
        "trick",
        "tricks",
        "webhook",
        "webhook-deliveries",
        "webhook-delivery",
        "webhook-endpoint",
        "webhook-endpoints",
        "webhook_deliveries",
        "webhook_delivery",
        "webhook_endpoint",
        "webhook_endpoints",
        "webhooks"
      ],
      "type": "string"
    }
  },
  "required": [
    "resource"
  ],
  "type": "object"
}
```

## `fibe_resource_mutate`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Create, update, or run a supported resource-scoped mutation with a payload validated against fibe_schema before any API request. Actions that use a Marquee require it to be funded.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "confirm": {
      "description": "Set true for destructive routed operations such as playground.action unless the server runs with --yolo.",
      "type": "boolean"
    },
    "dry_run": {
      "description": "Validate the payload against fibe_schema and return without sending any API request.",
      "type": "boolean"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "operation": {
      "description": "Mutation operation. Supported combinations are listed in fibe_schema(resource:list) and enforced by runtime validation.",
      "enum": [
        "action",
        "attach",
        "autoconnect_token",
        "change",
        "create",
        "fork",
        "generate_ssh_key",
        "mirror",
        "rerun",
        "restart_chat",
        "source_refresh",
        "source_set",
        "switch_template",
        "sync",
        "test",
        "test_connection",
        "toggle_public",
        "trigger",
        "update",
        "upgrade_playspecs",
        "upload_attachment"
      ],
      "type": "string"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "payload": {
      "description": "Operation-specific payload. Validate shape with fibe_schema(resource:\u003cname\u003e, operation:\u003coperation\u003e); the server enforces that schema before any API request.",
      "type": "object"
    },
    "resource": {
      "description": "Resource to mutate. Accepts canonical singular snake_case names and explicit aliases.",
      "enum": [
        "agent",
        "agent-poke",
        "agent-pokes",
        "agent_poke",
        "agent_pokes",
        "agents",
        "api-key",
        "api-keys",
        "api_key",
        "api_keys",
        "import-template",
        "import-templates",
        "import_template",
        "import_templates",
        "job-env",
        "job-environment",
        "job-environments",
        "job-envs",
        "job_env",
        "job_environment",
        "job_environments",
        "job_envs",
        "marquee",
        "marquees",
        "playground",
        "playgrounds",
        "playspec",
        "playspecs",
        "pokes",
        "prop",
        "props",
        "secret",
        "secrets",
        "template",
        "template-version",
        "template-versions",
        "template_version",
        "template_versions",
        "templates",
        "trick",
        "tricks",
        "webhook",
        "webhook-endpoint",
        "webhook-endpoints",
        "webhook_endpoint",
        "webhook_endpoints",
        "webhooks"
      ],
      "type": "string"
    }
  },
  "required": [
    "resource",
    "operation",
    "payload"
  ],
  "type": "object"
}
```

## `fibe_resource_watch`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Watch supported Fibe resource events.

### Input Schema
```json
{
  "properties": {
    "duration": {
      "description": "Max watch duration as Go duration (default: 30s, max: 30m).",
      "type": "string"
    },
    "max_events": {
      "description": "Stop after N events (default: 25).",
      "minimum": 1,
      "type": "number"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "resource": {
      "description": "Canonical resource name or explicit alias.",
      "enum": [
        "agent",
        "agents"
      ],
      "type": "string"
    }
  },
  "required": [
    "resource"
  ],
  "type": "object"
}
```

## `fibe_run`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:SIDEEFFECTS] Last-resort escape hatch: invoke an arbitrary Fibe CLI command when no dedicated MCP tool fits. Use sparingly.

### Input Schema
```json
{
  "properties": {
    "args": {
      "description": "Command args as if typed after `fibe`. Scalar items (string, number, boolean) are accepted and stringified into CLI tokens in-order.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "confirm": {
      "description": "Required for delete/destroy/remove CLI paths unless server runs with --yolo.",
      "type": "boolean"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "timeout_ms": {
      "description": "Optional per-call timeout in milliseconds. Recommended for risky escape-hatch calls.",
      "minimum": 1,
      "type": "number"
    }
  },
  "required": [
    "args"
  ],
  "type": "object"
}
```

## `fibe_schema`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Return JSON Schema definitions and the schema resource catalog.

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "operation": {
      "description": "Operation name. Supported combinations are resource-dependent; pass resource=list for the catalog.",
      "enum": [
        "action",
        "attach",
        "autoconnect_token",
        "change",
        "create",
        "delete",
        "event_types",
        "fork",
        "generate_ssh_key",
        "get",
        "list",
        "memorize",
        "mirror",
        "rerun",
        "restart_chat",
        "source_refresh",
        "source_set",
        "switch_template",
        "sync",
        "test",
        "test_connection",
        "toggle_public",
        "trigger",
        "update",
        "upgrade_playspecs",
        "upload_attachment",
        "validate",
        "watch"
      ],
      "type": "string"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "payload": {
      "description": "Optional operation payload for side-effect-free schema-backed validations such as compose.validate.",
      "properties": {},
      "type": "object"
    },
    "resource": {
      "description": "Resource name, alias, or 'list' for the schema resource catalog.",
      "enum": [
        "agent",
        "agent-attachment",
        "agent-attachments",
        "agent-poke",
        "agent-pokes",
        "agent-upload",
        "agent-uploads",
        "agent_attachment",
        "agent_attachments",
        "agent_poke",
        "agent_pokes",
        "agent_upload",
        "agent_uploads",
        "agents",
        "api-key",
        "api-keys",
        "api_key",
        "api_keys",
        "artefact",
        "artefact-attachment",
        "artefact-attachments",
        "artefact_attachment",
        "artefact_attachments",
        "artefacts",
        "audit-log",
        "audit-logs",
        "audit_log",
        "audit_logs",
        "categories",
        "category",
        "compose",
        "composes",
        "docker-compose",
        "docker_compose",
        "import-template",
        "import-templates",
        "import_template",
        "import_templates",
        "job-env",
        "job-environment",
        "job-environments",
        "job-envs",
        "job_env",
        "job_environment",
        "job_environments",
        "job_envs",
        "list",
        "marquee",
        "marquees",
        "memories",
        "memory",
        "mutter",
        "mutters",
        "playground",
        "playgrounds",
        "playspec",
        "playspecs",
        "pokes",
        "prop",
        "props",
        "secret",
        "secrets",
        "template",
        "template-categories",
        "template-category",
        "template-source",
        "template-sources",
        "template-version",
        "template-versions",
        "template_categories",
        "template_category",
        "template_source",
        "template_sources",
        "template_version",
        "template_versions",
        "templates",
        "trick",
        "tricks",
        "webhook",
        "webhook-deliveries",
        "webhook-delivery",
        "webhook-endpoint",
        "webhook-endpoints",
        "webhook_deliveries",
        "webhook_delivery",
        "webhook_endpoint",
        "webhook_endpoints",
        "webhooks"
      ],
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_status`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] Display a comprehensive dashboard of resource counts, quotas, and rate limits across your account.

## `fibe_templates_change`
**Tier:** brownfield | **Hidden:** true | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:BROWNFIELD] Advanced template change primitive: preview or apply template patches/overwrites, switch playspecs/playgrounds/tricks to existing template versions, and optionally roll out or trigger a fresh trick run. Rollout/trigger actions require a funded Marquee and fail with MARQUEE_NOT_FUNDED when unpaid.

### Input Schema
```json
{
  "additionalProperties": false,
  "properties": {
    "base_version_id": {
      "description": "Base template version ID for patch or overwrite. Defaults from target when possible.",
      "minimum": 1,
      "type": "integer"
    },
    "change_type": {
      "description": "Advanced template change workflow. patch creates a new version of the existing template; overwrite replaces the body of an existing version; switch_existing repoints the playspec/playground/trick at a different template version.",
      "enum": [
        "patch",
        "overwrite",
        "switch_existing"
      ],
      "type": "string"
    },
    "changelog": {
      "description": "Human-readable changelog for a created template version.",
      "type": "string"
    },
    "confirm": {
      "description": "Required for mode=apply unless the MCP server runs with --yolo. Not required for previews.",
      "type": "boolean"
    },
    "confirm_warnings": {
      "description": "Allow apply when preview reports switch warnings.",
      "type": "boolean"
    },
    "diagnose_on_failure": {
      "description": "Attach playground diagnostics when a wait fails.",
      "type": "boolean"
    },
    "edits": {
      "description": "Alias for patches.",
      "items": {
        "type": "object"
      },
      "type": "array"
    },
    "mode": {
      "description": "Preview validates and diffs without writes; apply creates/switches resources.",
      "enum": [
        "preview",
        "apply"
      ],
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "patches": {
      "description": "Patch entries: YAML path set/remove or exact search/replace.",
      "items": {
        "type": "object"
      },
      "type": "array"
    },
    "post_apply": {
      "description": "Optional action after apply. Tricks should use trigger_trick; normal playgrounds can use rollout_target or rollout_all.",
      "enum": [
        "none",
        "rollout_target",
        "rollout_all",
        "trigger_trick"
      ],
      "type": "string"
    },
    "provision_inputs": {
      "description": "Per-URL overrides for provisioning. Each item: {source_repo_url, name_override?, default_branch?, description?, auto_init?}. source_repo_url must match a URL declared by the new template's services.",
      "items": {
        "additionalProperties": false,
        "properties": {
          "auto_init": {
            "description": "Initialize the repository with default files.",
            "type": "boolean"
          },
          "default_branch": {
            "description": "Default Branch for this templates_change request.",
            "type": "string"
          },
          "description": {
            "description": "Optional description.",
            "type": "string"
          },
          "name_override": {
            "description": "Name Override for this templates_change request.",
            "type": "string"
          },
          "source_repo_url": {
            "description": "Source Repo URL URL.",
            "type": "string"
          }
        },
        "required": [
          "source_repo_url"
        ],
        "type": "object"
      },
      "type": "array"
    },
    "provision_missing_props": {
      "description": "When the new template references repos the player does not yet own a Prop for, automatically provision a fresh git repo (in the player's connected Gitea or GitHub account) and create a Prop for each. \"gitea\" provisions Gitea-backed private repos seeded from the template's source URLs. \"off\" (default) keeps today's behaviour: public repos auto-create Props, private/missing repos fail with a manual-creation hint.",
      "enum": [
        "off",
        "gitea",
        "github"
      ],
      "type": "string"
    },
    "provision_private": {
      "description": "Whether the freshly provisioned repos should be private. Defaults to true.",
      "type": "boolean"
    },
    "public": {
      "description": "Make the created template version public.",
      "type": "boolean"
    },
    "regenerate_variables": {
      "description": "Variable names to regenerate while switching.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "response_mode": {
      "description": "Response detail mode.",
      "enum": [
        "summary",
        "full"
      ],
      "type": "string"
    },
    "reuse_existing_props": {
      "description": "Reuse the target playspec/playground's current dynamic service Props before provisioning new ones. Same-name services keep their Prop first; remaining services reuse existing Props in current service order; extras are retired after apply when no longer referenced.",
      "type": "boolean"
    },
    "switch_variables": {
      "description": "Template variables to use when switching a playspec.",
      "type": "object"
    },
    "target_id_or_name": {
      "description": "ID or name of the target object.",
      "oneOf": [
        {
          "minimum": 1,
          "type": "integer"
        },
        {
          "minLength": 1,
          "type": "string"
        }
      ]
    },
    "target_template_version_id": {
      "description": "Existing template version ID to switch the target's playspec to. Required for change_type=switch_existing. Can belong to a completely different template — the server reconciles the prop set and regenerates services.",
      "minimum": 1,
      "type": "integer"
    },
    "target_type": {
      "description": "Object to start from. With target_type=playground and change_type=switch_existing, this performs an advanced template-version switch for a deployed playground.",
      "enum": [
        "template",
        "playspec",
        "playground",
        "trick"
      ],
      "type": "string"
    },
    "template_body": {
      "description": "Full replacement template YAML for overwrite.",
      "type": "string"
    },
    "template_body_path": {
      "description": "Absolute local path to full replacement template YAML.",
      "type": "string"
    },
    "wait": {
      "description": "Wait for rollout targets or triggered trick completion.",
      "type": "boolean"
    },
    "wait_timeout_seconds": {
      "description": "Maximum seconds to wait.",
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "target_type",
    "target_id_or_name",
    "mode",
    "change_type"
  ],
  "type": "object"
}
```

## `fibe_templates_search`
**Tier:** greenfield | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:GREENFIELD] Search the import-template catalog by text or PostgreSQL regex. Regex mode requires a 3+ character literal token for indexed prefiltering.

### Input Schema
```json
{
  "properties": {
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "query": {
      "description": "Search query. In regex mode, this is a PostgreSQL regex pattern.",
      "type": "string"
    },
    "regex": {
      "description": "Treat query as PostgreSQL regex. Requires a 3+ character literal token so the server can prefilter with indexed text search.",
      "type": "boolean"
    },
    "template_id_or_name": {
      "description": "Optional template ID or name filter",
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_tools_catalog`
**Tier:** meta | **Hidden:** false | **Destructive:** false | **Idempotent:** true | **Read-only:** true

### Description
[MODE:DIALOG] List all tools registered and available on the Fibe MCP server. CRITICAL: Fibe Platform priority is to let you manage **ALL** its capabilities via its tools so you should find anything here. We just can't advertise them all because there are hundreds

### Input Schema
```json
{
  "properties": {
    "include_schema": {
      "description": "Include input schemas (larger response)",
      "type": "boolean"
    },
    "name_pattern": {
      "description": "Substring to match in tool name or description",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    },
    "tier": {
      "description": "Filter by named tool tier or shortcut. core means meta, base, greenfield, and brownfield; full/all means every tier.",
      "enum": [
        "meta",
        "base",
        "greenfield",
        "brownfield",
        "overseer",
        "local",
        "other",
        "core",
        "full",
        "all"
      ],
      "type": "string"
    }
  },
  "required": [],
  "type": "object"
}
```

## `fibe_update_name`
**Tier:** base | **Hidden:** false | **Destructive:** false | **Idempotent:** false | **Read-only:** false

### Description
[MODE:DIALOG] Update your own agent name.

### Input Schema
```json
{
  "properties": {
    "name": {
      "description": "Your new name",
      "type": "string"
    },
    "only": {
      "description": "Return only these top-level fields from each result item. Example: only: [\"uuid\",\"title\",\"project\"] on local conversations keeps envelope metadata but trims each conversation.",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "output_path": {
      "description": "JSONPath into the tool result, not a filesystem path. Example: \"$.conversations[0].uuid\" returns the first UUID; \"$.conversations\" returns only the array.",
      "type": "string"
    }
  },
  "required": [
    "name"
  ],
  "type": "object"
}
```

