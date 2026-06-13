package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

const playgroundSwitchTemplateToolName = "fibe_playgrounds_switch_template"

func (s *Server) registerPlaygroundSwitchTemplateTools() {
	s.registerPlaygroundSwitchTemplateTool(playgroundSwitchTemplateToolName)
}

func (s *Server) registerPlaygroundSwitchTemplateTool(name string) {
	schema, _, _, _ := resourceschema.SchemaFor("playground", "switch_template")
	inputSchema, _ := schema.(map[string]any)

	description := "[MODE:BROWNFIELD] Switch a deployed playground end-to-end: preserve the playground id, swap it onto a new template shape, provision missing private Gitea/GitHub-backed Props for new repos, roll it out, wait, and diagnose failures. Single-call brownfield analog of fibe_greenfield_create. Apply mode requires a funded Marquee and fails with MARQUEE_NOT_FUNDED when unpaid."

	s.addTool(&toolImpl{
		name:        name,
		description: description,
		tier:        tierBrownfield,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			if _, _, err := resourceschema.ValidatePayload("playground", "switch_template", args); err != nil {
				return nil, err
			}
			mode := strings.ToLower(strings.TrimSpace(argString(args, "mode")))
			if mode == "" {
				mode = "apply"
			}
			if mode == "apply" && !s.cfg.Yolo && !yoloFromContext(ctx) && !argBool(args, "confirm") {
				return nil, &confirmRequiredError{tool: name}
			}
			params, err := buildSwitchTemplateParams(args, mode)
			if err != nil {
				return nil, err
			}
			return c.SwitchPlaygroundTemplate(ctx, params)
		},
	}, mcp.NewTool(name,
		mcp.WithDescription(description),
		withRawInputSchema(inputSchema),
	))
}

func buildSwitchTemplateParams(args map[string]any, mode string) (*fibe.PlaygroundTemplateSwitchParams, error) {
	identifier, err := requiredIdentifier(args, "id_or_name", "")
	if err != nil {
		return nil, err
	}
	inlineBody := argString(args, "template_body")
	bodyPath := argString(args, "template_body_path")
	versionID, versionIDHasValue := argInt64(args, "template_version_id")
	if inlineBody != "" && bodyPath != "" {
		return nil, fmt.Errorf("template_body cannot be combined with template_body_path")
	}
	if versionIDHasValue && versionID > 0 && (inlineBody != "" || bodyPath != "") {
		return nil, fmt.Errorf("template_version_id cannot be combined with template_body or template_body_path")
	}
	body := inlineBody
	if body == "" {
		read, perr := readInlineOrPathTextArgOptional(args, "template_body", "template_body_path")
		if perr != nil {
			return nil, perr
		}
		body = read
	}

	templateIdentifier, _ := argIdentifier(args, "template_id_or_name", "")
	templateID, _ := argInt64(args, "template_id_or_name")
	if body == "" && templateIdentifier == "" && versionID == 0 {
		return nil, fmt.Errorf("must provide template_body, template_body_path, template_id_or_name, or template_version_id")
	}

	params := &fibe.PlaygroundTemplateSwitchParams{
		PlaygroundIdentifier:  identifier,
		Mode:                  mode,
		TemplateBody:          body,
		TemplateID:            templateID,
		TemplateIdentifier:    templateIdentifier,
		TemplateVersionID:     versionID,
		TemplateName:          argString(args, "template_name"),
		Variables:             argMap(args, "variables"),
		RegenerateVariables:   argStringSlice(args, "regenerate_variables"),
		ConfirmWarnings:       argBool(args, "confirm_warnings"),
		ProvisionMissingProps: argString(args, "provision_missing_props"),
		Wait:                  argBoolDefault(args, "wait", true),
		ResponseMode:          argString(args, "response_mode"),
		Changelog:             argString(args, "changelog"),
		ReuseExistingProps:    argBool(args, "reuse_existing_props"),
	}
	if v, ok := args["provision_private"]; ok {
		if b, ok := v.(bool); ok {
			params.ProvisionPrivate = &b
		}
	}
	if v, ok := argInt64(args, "wait_timeout_seconds"); ok {
		params.WaitTimeoutSeconds = v
	}
	if v, ok := args["diagnose_on_failure"]; ok {
		if b, ok := v.(bool); ok {
			params.DiagnoseOnFailure = &b
		}
	}
	if inputs, ok := args["provision_inputs"].([]any); ok {
		for _, raw := range inputs {
			if m, ok := raw.(map[string]any); ok {
				input := fibe.ProvisionPropInput{SourceRepoURL: argString(m, "source_repo_url")}
				if input.SourceRepoURL == "" {
					continue
				}
				if v := argString(m, "name_override"); v != "" {
					input.NameOverride = &v
				}
				if v := argString(m, "default_branch"); v != "" {
					input.DefaultBranch = &v
				}
				if v := argString(m, "description"); v != "" {
					input.Description = &v
				}
				if v, ok := m["auto_init"]; ok {
					if b, ok := v.(bool); ok {
						input.AutoInit = &b
					}
				}
				params.ProvisionInputs = append(params.ProvisionInputs, input)
			}
		}
	}

	if params.ProvisionMissingProps == "" {
		params.ProvisionMissingProps = "gitea"
	}

	return params, nil
}

func argBoolDefault(args map[string]any, key string, def bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func argMap(args map[string]any, key string) map[string]any {
	if v, ok := args[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func argStringSlice(args map[string]any, key string) []string {
	if v, ok := args[key]; ok {
		if list, ok := v.([]any); ok {
			out := make([]string, 0, len(list))
			for _, raw := range list {
				if s, ok := raw.(string); ok && s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}
