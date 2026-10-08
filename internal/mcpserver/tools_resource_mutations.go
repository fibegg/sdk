package mcpserver

import (
	"context"
	"fmt"
	"strings"

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
	s.registerPlaygroundSwitchTemplateTools()
	s.registerAgentMutationTools()
	s.registerFeedbackMutationTools()
}

func (s *Server) registerResourceMutateTool() {
	s.addTool(&toolImpl{
		name:        "fibe_resource_mutate",
		description: "[MODE:SIDEEFFECTS] Create, update, or run a supported resource-scoped mutation with a payload validated against fibe_schema before any API request. Actions that use a Host require it to be funded.",
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
			if mutationRequiresConfirm(canonicalResource, canonicalOperation, payload) && !s.cfg.Yolo && !yoloFromContext(ctx) && !argBool(args, "confirm") {
				return nil, &confirmRequiredError{tool: "fibe_resource_mutate"}
			}
			return dispatchResourceMutation(ctx, c, canonicalResource, canonicalOperation, payload)
		},
	}, mcp.NewTool("fibe_resource_mutate",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Create, update, or run a supported resource-scoped mutation. Call fibe_schema(resource:<name>, operation:<operation>) for the exact payload schema; this tool validates that payload locally before any API request. Pass dry_run=true to validate only. Actions that use a Host require it to be funded and fail with HOST_NOT_FUNDED when unpaid."),
		withRawInputSchema(resourceschema.MutationToolInputSchema()),
	))
}

func mutationRequiresConfirm(resource, operation string, payload map[string]any) bool {
	switch resource + "." + operation {
	case "playground.action":
		return true
	case "playground.switch_template":
		mode := strings.ToLower(strings.TrimSpace(argString(payload, "mode")))
		return mode == "" || mode == "apply"
	case "template.change":
		return strings.EqualFold(strings.TrimSpace(argString(payload, "mode")), "apply")
	default:
		return false
	}
}

func dispatchResourceMutation(ctx context.Context, c *fibe.Client, resource, operation string, payload map[string]any) (any, error) {
	payload = resourceMutationBackendPayload(resource, operation, payload)
	switch resource + "." + operation {
	case "env_pack.create":
		var p fibe.EnvPackCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.EnvPacks.Create(ctx, &p)
	case "env_pack.update":
		id, err := requiredPositiveID(payload, "env_pack_id")
		if err != nil {
			return nil, err
		}
		var p fibe.EnvPackUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.EnvPacks.Update(ctx, id, &p)
	case "env_pack.attachments_replace":
		id, err := requiredPositiveID(payload, "target_id")
		if err != nil {
			return nil, err
		}
		var p fibe.EnvPackAttachmentsParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.EnvPacks.ReplaceAttachments(ctx, argString(payload, "target"), id, &p)
	case "env_pack.grant_renew":
		id, err := requiredPositiveID(payload, "target_id")
		if err != nil {
			return nil, err
		}
		attachment, err := requiredPositiveID(payload, "attachment_id")
		if err != nil {
			return nil, err
		}
		return c.EnvPacks.RenewGrant(ctx, argString(payload, "target"), id, attachment)
	case "env_pack.attachments_reorder", "env_pack.attachment_detach", "env_pack.attachment_retarget":
		id, err := requiredPositiveID(payload, "target_id")
		if err != nil {
			return nil, err
		}
		target := argString(payload, "target")
		if operation == "attachments_reorder" {
			var p struct {
				Order []int64 `json:"order"`
			}
			if err := bindArgs(payload, &p); err != nil {
				return nil, err
			}
			return c.EnvPacks.ReorderAttachments(ctx, target, id, p.Order)
		}
		packID, err := requiredPositiveID(payload, "env_pack_id")
		if err != nil {
			return nil, err
		}
		if operation == "attachment_detach" {
			return c.EnvPacks.DetachAttachment(ctx, target, id, packID)
		}
		var p struct {
			Names *[]string `json:"service_names"`
		}
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.EnvPacks.RetargetAttachment(ctx, target, id, packID, p.Names)
	case "playground.rerun":
		id, err := requiredPositiveID(payload, "playground_id")
		if err != nil {
			return nil, err
		}
		var p fibe.PlaygroundRerunParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Playgrounds.Rerun(ctx, id, &p)
	case "agent_poke.create":
		identifier, err := requiredIdentifier(payload, "agent_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.AgentPokeCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Agents.CreatePokeByIdentifier(ctx, identifier, &p)
	case "agent_poke.update":
		identifier, err := requiredIdentifier(payload, "agent_id", "")
		if err != nil {
			return nil, err
		}
		pokeID, err := requiredPositiveID(payload, "poke_id")
		if err != nil {
			return nil, err
		}
		var p fibe.AgentPokeUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Agents.UpdatePokeByIdentifier(ctx, identifier, pokeID, &p)
	case "agent.create":
		var p fibe.AgentCreateParams
		if err := bindIdentifierArgs(payload, &p, "build_in_public_playground_id"); err != nil {
			return nil, err
		}
		return c.Agents.Create(ctx, &p)
	case "agent.update":
		identifier, err := requiredIdentifier(payload, "agent_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.AgentUpdateParams
		if err := bindIdentifierArgs(payload, &p, "build_in_public_playground_id"); err != nil {
			return nil, err
		}
		return c.Agents.UpdateByIdentifier(ctx, identifier, &p)
	case "agent.restart_chat":
		identifier, err := requiredIdentifier(payload, "agent_id", "")
		if err != nil {
			return nil, err
		}
		return c.Agents.RestartChatByIdentifier(ctx, identifier)
	case "agent.upload_attachment":
		identifier, err := requiredIdentifier(payload, "agent_id", "")
		if err != nil {
			return nil, err
		}
		reader, err := decodeFileSource(payload)
		if err != nil {
			return nil, err
		}
		filename := argString(payload, "filename")
		if filename == "" {
			filename = filenameFromContentPath(argString(payload, "content_path"), "attachment")
		}
		return c.Agents.UploadReaderByIdentifier(ctx, identifier, reader, filename, &fibe.AgentUploadParams{
			ConversationID: argString(payload, "conversation_id"),
		})
	case "api_key.create":
		var p fibe.APIKeyCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.APIKeys.Create(ctx, &p)
	case "host.create":
		var p fibe.HostCreateParams
		if err := bindIdentifierArgs(payload, &p, "repository_id"); err != nil {
			return nil, err
		}
		return c.Hosts.Create(ctx, &p)
	case "host.update":
		identifier, err := requiredIdentifier(payload, "host_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.HostUpdateParams
		if err := bindIdentifierArgs(payload, &p, "repository_id"); err != nil {
			return nil, err
		}
		return c.Hosts.UpdateByIdentifier(ctx, identifier, &p)
	case "host.autoconnect_token":
		var p fibe.AutoconnectTokenParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Hosts.AutoconnectToken(ctx, &p)
	case "host.generate_ssh_key":
		identifier, err := requiredIdentifier(payload, "host_id", "")
		if err != nil {
			return nil, err
		}
		return c.Hosts.GenerateSSHKeyByIdentifier(ctx, identifier)
	case "host.test_connection":
		identifier, err := requiredIdentifier(payload, "host_id", "")
		if err != nil {
			return nil, err
		}
		return c.Hosts.TestConnectionByIdentifier(ctx, identifier)
	case "playground.create":
		var p fibe.PlaygroundCreateParams
		if err := bindIdentifierArgs(payload, &p, "spec_id", "host_id"); err != nil {
			return nil, err
		}
		if p.HostID == nil && p.HostIdentifier == "" {
			hostID, hostIdentifier, err := resolveMCPHost(ctx, c, payload)
			if err != nil {
				return nil, err
			}
			p.HostID = hostID
			p.HostIdentifier = hostIdentifier
		}
		return c.Playgrounds.Create(ctx, &p)
	case "playground.update":
		identifier, err := requiredIdentifier(payload, "playground_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.PlaygroundUpdateParams
		if err := bindIdentifierArgs(payload, &p, "spec_id", "host_id"); err != nil {
			return nil, err
		}
		return c.Playgrounds.UpdateByIdentifier(ctx, identifier, &p)
	case "playground.action":
		identifier, err := requiredIdentifier(payload, "playground_id", "")
		if err != nil {
			return nil, err
		}
		p := &fibe.PlaygroundActionParams{ActionType: argString(payload, "action_type")}
		if _, ok := payload["force"]; ok {
			force := argBool(payload, "force")
			p.Force = &force
		}
		return c.Playgrounds.ActionByIdentifier(ctx, identifier, p)
	case "playground.switch_template":
		mode := strings.ToLower(strings.TrimSpace(argString(payload, "mode")))
		if mode == "" {
			mode = "apply"
		}
		params, err := buildSwitchTemplateParams(payload, mode)
		if err != nil {
			return nil, err
		}
		return c.SwitchPlaygroundTemplate(ctx, params)
	case "spec.create":
		var p fibe.SpecCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Specs.Create(ctx, &p)
	case "spec.update":
		identifier, err := requiredIdentifier(payload, "spec_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.SpecUpdateParams
		if err := bindSpecUpdateArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Specs.UpdateByIdentifier(ctx, identifier, &p)
	case "repository.create":
		var p fibe.RepositoryCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Repositories.Create(ctx, &p)
	case "repository.update":
		identifier, err := requiredIdentifier(payload, "repository_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.RepositoryUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Repositories.UpdateByIdentifier(ctx, identifier, &p)
	case "repository.attach":
		repoFullName := argString(payload, "repo_full_name")
		if parsed := parseRepoFullName(repoFullName); parsed != "" {
			repoFullName = parsed
		}
		return c.Repositories.Attach(ctx, repoFullName)
	case "repository.mirror":
		return c.Repositories.Mirror(ctx, argString(payload, "source_url"), argString(payload, "name"))
	case "repository.sync":
		identifier, err := requiredIdentifier(payload, "repository_id", "")
		if err != nil {
			return nil, err
		}
		if err := c.Repositories.SyncByIdentifier(ctx, identifier); err != nil {
			return nil, err
		}
		return map[string]any{"repository_identifier": identifier, "ok": true}, nil
	case "secret.create":
		var p fibe.SecretCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Secrets.Create(ctx, &p)
	case "secret.update":
		identifier, err := requiredIdentifier(payload, "secret_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.SecretUpdateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Secrets.UpdateByIdentifier(ctx, identifier, &p)
	case "template.create":
		var p fibe.ImportTemplateCreateParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.ImportTemplates.Create(ctx, &p)
	case "template.update":
		return mutateTemplateUpdate(ctx, c, payload)
	case "template.change":
		var in templateChangeArgs
		if err := bindArgs(payload, &in); err != nil {
			return nil, err
		}
		return runTemplateChange(ctx, c, &in)
	case "template.fork":
		identifier, err := requiredIdentifier(payload, "template_id", "")
		if err != nil {
			return nil, err
		}
		return c.ImportTemplates.ForkByIdentifier(ctx, identifier)
	case "template.source_refresh":
		identifier, err := requiredIdentifier(payload, "template_id", "")
		if err != nil {
			return nil, err
		}
		return c.ImportTemplates.RefreshSourceByIdentifier(ctx, identifier)
	case "template.source_set":
		identifier, err := requiredIdentifier(payload, "template_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.ImportTemplateSourceParams
		if err := bindIdentifierArgs(payload, &p, "source_repository_id", "ci_host_id", "host_id"); err != nil {
			return nil, err
		}
		return c.ImportTemplates.SetSourceByIdentifier(ctx, identifier, &p)
	case "template.upgrade_specs":
		identifier, err := requiredIdentifier(payload, "template_id", "")
		if err != nil {
			return nil, err
		}
		versionID, _ := argInt64(payload, "version_id")
		return c.ImportTemplates.UpgradeLinkedSpecsByIdentifier(ctx, identifier, versionID)
	case "template_version.create":
		return mutateTemplateVersionCreate(ctx, c, payload)
	case "template_version.toggle_public":
		identifier, err := requiredIdentifier(payload, "template_id", "")
		if err != nil {
			return nil, err
		}
		versionID, _ := argInt64(payload, "version_id")
		return c.ImportTemplates.TogglePublicByIdentifier(ctx, identifier, versionID)
	case "task.trigger":
		var p fibe.TaskTriggerParams
		if err := bindIdentifierArgs(payload, &p, "spec_id", "host_id"); err != nil {
			return nil, err
		}
		return c.Tasks.Trigger(ctx, &p)
	case "task.rerun":
		identifier, err := requiredIdentifier(payload, "task_id", "")
		if err != nil {
			return nil, err
		}
		var p fibe.PlaygroundRerunParams
		if err := bindArgs(payload, &p); err != nil {
			return nil, err
		}
		return c.Tasks.RerunWithParamsByIdentifier(ctx, identifier, &p)
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
		if err := bindIdentifierArgs(payload, &p, "repository_id"); err != nil {
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

func resourceMutationBackendPayload(resource, operation string, payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload)+8)
	for key, value := range payload {
		out[key] = value
	}
	copyField := func(from, to string) {
		if value, ok := out[from]; ok {
			out[to] = value
		}
	}
	for from, to := range map[string]string{
		"agent_id_or_name":                      "agent_id",
		"build_in_public_playground_id_or_name": "build_in_public_playground_id",
		"ci_host_id_or_name":                    "ci_host_id",
		"host_id_or_name":                       "host_id",
		"playground_id_or_name":                 "playground_id",
		"spec_id_or_name":                       "spec_id",
		"repository_id_or_name":                 "repository_id",
		"source_repository_id_or_name":          "source_repository_id",
		"target_playground_id_or_name":          "target_playground_id",
		"target_spec_id_or_name":                "target_spec_id",
		"template_id_or_name":                   "template_id",
	} {
		copyField(from, to)
	}
	if resource == "secret" {
		copyField("id_or_key", "secret_id")
	}
	if _, ok := out["id_or_name"]; ok {
		switch resource {
		case "agent":
			copyField("id_or_name", "agent_id")
		case "host":
			copyField("id_or_name", "host_id")
		case "playground":
			copyField("id_or_name", "playground_id")
		case "spec":
			copyField("id_or_name", "spec_id")
		case "repository":
			copyField("id_or_name", "repository_id")
		case "template":
			copyField("id_or_name", "template_id")
		case "task":
			copyField("id_or_name", "task_id")
		}
	}
	return out
}

func mutateTemplateUpdate(ctx context.Context, c *fibe.Client, payload map[string]any) (any, error) {
	identifier, err := requiredIdentifier(payload, "template_id", "")
	if err != nil {
		return nil, err
	}
	var updateParams fibe.ImportTemplateUpdateParams
	if err := bindArgs(payload, &updateParams); err != nil {
		return nil, err
	}

	var result *fibe.ImportTemplate
	if updateParams.Name != nil || updateParams.Description != nil || updateParams.CategoryID != nil {
		res, err := c.ImportTemplates.UpdateByIdentifier(ctx, identifier, &updateParams)
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
			res, err := c.ImportTemplates.UploadImageByIdentifier(ctx, identifier, &imageParams)
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
	identifier, err := requiredIdentifier(payload, "template_id", "")
	if err != nil {
		return nil, err
	}
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
	return c.ImportTemplates.CreateVersionByIdentifier(ctx, identifier, &p)
}

func bindSpecUpdateArgs(payload map[string]any, params *fibe.SpecUpdateParams) error {
	cleaned := make(map[string]any, len(payload))
	for key, value := range payload {
		cleaned[key] = value
	}

	type serviceIdentifier struct {
		index      int
		identifier string
	}
	var identifiers []serviceIdentifier
	if services, ok := payload["services"].([]any); ok {
		copiedServices := make([]any, len(services))
		for i, service := range services {
			serviceMap, ok := service.(map[string]any)
			if !ok {
				copiedServices[i] = service
				continue
			}
			copied := make(map[string]any, len(serviceMap))
			for key, value := range serviceMap {
				copied[key] = value
			}
			if value, ok := copied["repository_id_or_name"]; ok {
				copied["repository_id"] = value
				delete(copied, "repository_id_or_name")
			}
			if identifier, ok := stringIdentifierValue(copied["repository_id"]); ok {
				identifiers = append(identifiers, serviceIdentifier{index: i, identifier: identifier})
				delete(copied, "repository_id")
			}
			copiedServices[i] = copied
		}
		cleaned["services"] = copiedServices
	}

	if err := bindArgs(cleaned, params); err != nil {
		return err
	}
	for _, entry := range identifiers {
		if entry.index >= 0 && entry.index < len(params.Services) {
			params.Services[entry.index].RepositoryIdentifier = entry.identifier
		}
	}
	return nil
}
