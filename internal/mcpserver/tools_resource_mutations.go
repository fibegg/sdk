package mcpserver

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerResourceMutationTools wires the generic create/update-like mutation
// tool plus custom resource actions. Uniform list/get/delete are exposed
// through fibe_resource_list/get/delete in tools_resources.go.
//
// The FIBE_MCP_TOOLS env var / --tools flag filters at advertisement time.
func (s *Server) registerResourceMutationTools() {
	s.registerResourceMutateTool()
	s.registerPlaygroundMutationTools()
	s.registerAgentMutationTools()
	s.registerFeedbackMutationTools()
}

func (s *Server) registerResourceMutateTool() {
	s.addTool(&toolImpl{
		name:        "fibe_resource_mutate",
		description: "[MODE:SIDEEFFECTS] Create, update, or run a supported resource-scoped mutation with a payload validated against fibe_schema before any API request.",
		tier:        tierBase,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			resource := argString(args, "resource")
			if resource == "" {
				return nil, fmt.Errorf("required field 'resource' not set")
			}
			operation := argString(args, "operation")
			if operation == "" {
				return nil, fmt.Errorf("required field 'operation' not set")
			}
			payload, ok := args["payload"].(map[string]any)
			if !ok || payload == nil {
				return nil, fmt.Errorf("required field 'payload' must be an object")
			}
			canonicalResource, canonicalOperation, err := resourceschema.ValidateMutationPayload(resource, operation, payload)
			if err != nil {
				return nil, err
			}
			if argBool(args, "dry_run") {
				return map[string]any{
					"resource":  canonicalResource,
					"operation": canonicalOperation,
					"dry_run":   true,
					"valid":     true,
					"message":   "Payload is valid; no request was sent.",
				}, nil
			}
			return dispatchResourceMutation(ctx, c, canonicalResource, canonicalOperation, payload)
		},
	}, mcp.NewTool("fibe_resource_mutate",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Create, update, or run a supported resource-scoped mutation. Call fibe_schema(resource:<name>, operation:<operation>) for the exact payload schema; this tool validates that payload locally before any API request. Pass dry_run=true to validate only."),
		withRawInputSchema(resourceschema.MutationToolInputSchema()),
	))
}

func dispatchResourceMutation(ctx context.Context, c *fibe.Client, resource, operation string, payload map[string]any) (any, error) {
	switch resource + "." + operation {
	case "agent.create":
		var p fibe.AgentCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Agents.Create(ctx, &p)
	case "agent.update":
		id, _ := argInt64(payload, "agent_id")
		var p fibe.AgentUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Agents.Update(ctx, id, &p)
	case "api_key.create":
		var p fibe.APIKeyCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.APIKeys.Create(ctx, &p)
	case "marquee.create":
		var p fibe.MarqueeCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Marquees.Create(ctx, &p)
	case "marquee.update":
		id, _ := argInt64(payload, "marquee_id")
		var p fibe.MarqueeUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Marquees.Update(ctx, id, &p)
	case "marquee.autoconnect_token":
		var p fibe.AutoconnectTokenParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Marquees.AutoconnectToken(ctx, &p)
	case "marquee.generate_ssh_key":
		id, _ := argInt64(payload, "marquee_id")
		return c.Marquees.GenerateSSHKey(ctx, id)
	case "marquee.test_connection":
		id, _ := argInt64(payload, "marquee_id")
		return c.Marquees.TestConnection(ctx, id)
	case "playground.create":
		var p fibe.PlaygroundCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		if p.MarqueeID == nil {
			envID, err := parseMarqueeIDEnv()
			if err != nil {
				return nil, fmt.Errorf("marquee_id is required either in payload or via FIBE_MARQUEE_ID env var: %w", err)
			}
			p.MarqueeID = &envID
		}
		return c.Playgrounds.Create(ctx, &p)
	case "playground.update":
		id, _ := argInt64(payload, "playground_id")
		var p fibe.PlaygroundUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Playgrounds.Update(ctx, id, &p)
	case "playspec.create":
		var p fibe.PlayspecCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Playspecs.Create(ctx, &p)
	case "playspec.update":
		id, _ := argInt64(payload, "playspec_id")
		var p fibe.PlayspecUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Playspecs.Update(ctx, id, &p)
	case "prop.create":
		var p fibe.PropCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Props.Create(ctx, &p)
	case "prop.update":
		id, _ := argInt64(payload, "prop_id")
		var p fibe.PropUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Props.Update(ctx, id, &p)
	case "prop.attach":
		repoFullName := argString(payload, "repo_full_name")
		if parsed := parseRepoFullName(repoFullName); parsed != "" {
			repoFullName = parsed
		}
		return c.Props.Attach(ctx, repoFullName)
	case "prop.mirror":
		return c.Props.Mirror(ctx, argString(payload, "source_url"), argString(payload, "name"))
	case "prop.sync":
		id, _ := argInt64(payload, "prop_id")
		if err := c.Props.Sync(ctx, id); err != nil {
			return nil, err
		}
		return map[string]any{"prop_id": id, "ok": true}, nil
	case "secret.create":
		var p fibe.SecretCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Secrets.Create(ctx, &p)
	case "secret.update":
		id, _ := argInt64(payload, "secret_id")
		var p fibe.SecretUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Secrets.Update(ctx, id, &p)
	case "template.create":
		var p fibe.ImportTemplateCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.ImportTemplates.Create(ctx, &p)
	case "template.update":
		return mutateTemplateUpdate(ctx, c, payload)
	case "template.fork":
		id, _ := argInt64(payload, "template_id")
		return c.ImportTemplates.Fork(ctx, id)
	case "template.source_refresh":
		id, _ := argInt64(payload, "template_id")
		return c.ImportTemplates.RefreshSource(ctx, id)
	case "template.source_set":
		id, _ := argInt64(payload, "template_id")
		var p fibe.ImportTemplateSourceParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.ImportTemplates.SetSource(ctx, id, &p)
	case "template.upgrade_playspecs":
		templateID, _ := argInt64(payload, "template_id")
		versionID, _ := argInt64(payload, "version_id")
		return c.ImportTemplates.UpgradeLinkedPlayspecs(ctx, templateID, versionID)
	case "template_version.create":
		return mutateTemplateVersionCreate(ctx, c, payload)
	case "template_version.toggle_public":
		templateID, _ := argInt64(payload, "template_id")
		versionID, _ := argInt64(payload, "version_id")
		return c.ImportTemplates.TogglePublic(ctx, templateID, versionID)
	case "trick.trigger":
		var p fibe.TrickTriggerParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Tricks.Trigger(ctx, &p)
	case "trick.rerun":
		id, _ := argInt64(payload, "trick_id")
		return c.Tricks.Rerun(ctx, id)
	case "webhook.create":
		var p fibe.WebhookEndpointCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.WebhookEndpoints.Create(ctx, &p)
	case "webhook.update":
		id, _ := argInt64(payload, "webhook_id")
		var p fibe.WebhookEndpointUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.WebhookEndpoints.Update(ctx, id, &p)
	case "webhook.test":
		id, _ := argInt64(payload, "webhook_id")
		if err := c.WebhookEndpoints.Test(ctx, id); err != nil {
			return nil, err
		}
		return map[string]any{"webhook_id": id, "ok": true}, nil
	case "job_env.create":
		var p fibe.JobEnvSetParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.JobEnv.Set(ctx, &p)
	case "job_env.update":
		id, _ := argInt64(payload, "job_env_id")
		var p fibe.JobEnvUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.JobEnv.Update(ctx, id, &p)
	default:
		return nil, fmt.Errorf("unsupported mutation %s.%s", resource, operation)
	}
}

func mutateTemplateUpdate(ctx context.Context, c *fibe.Client, payload map[string]any) (any, error) {
	id, _ := argInt64(payload, "template_id")
	var updateParams fibe.ImportTemplateUpdateParams
	if err := bindArgs(payload, &updateParams); err != nil {
		return nil, err
	}

	var result *fibe.ImportTemplate
	if updateParams.Name != nil || updateParams.Description != nil || updateParams.CategoryID != nil {
		res, err := c.ImportTemplates.Update(ctx, id, &updateParams)
		if err != nil {
			return nil, err
		}
		result = res
	}

	var imageParams fibe.UploadImageParams
	if err := bindArgs(payload, &imageParams); err == nil {
		if imageParams.ImageData == "" {
			if v := argString(payload, "content_base64"); v != "" {
				imageParams.ImageData = v
			} else if path := argString(payload, "content_path"); path != "" {
				data, err := readLocalFileBase64(path)
				if err != nil {
					return nil, err
				}
				imageParams.ImageData = data
			}
		}
		if imageParams.ImageData != "" {
			if imageParams.Filename == "" {
				imageParams.Filename = "cover.png"
			}
			res, err := c.ImportTemplates.UploadImage(ctx, id, &imageParams)
			if err != nil {
				return nil, err
			}
			result = res
		}
	}
	if result == nil {
		return nil, fmt.Errorf("template.update payload must include metadata fields or image_data, content_base64, or content_path")
	}
	return result, nil
}

func mutateTemplateVersionCreate(ctx context.Context, c *fibe.Client, payload map[string]any) (any, error) {
	id, _ := argInt64(payload, "template_id")
	if argString(payload, "template_body") != "" && argString(payload, "template_body_path") != "" {
		return nil, fmt.Errorf("pass only one of template_body or template_body_path")
	}
	if argString(payload, "template_body") == "" && argString(payload, "template_body_path") != "" {
		body, err := readInlineOrPathTextArg(payload, "template_body", "template_body_path")
		if err != nil {
			return nil, err
		}
		payload["template_body"] = body
		delete(payload, "template_body_path")
	}
	var p fibe.ImportTemplateVersionCreateParams
	if err := bindArgs(payload, &p); err != nil {
		return nil, err
	}
	if p.ResponseMode == "" {
		p.ResponseMode = "summary"
	}
	return c.ImportTemplates.CreateVersion(ctx, id, &p)
}
