package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPlaygroundRetemplateTools() {
	schema, _, _, _ := resourceschema.SchemaFor("playground", "retemplate")
	inputSchema, _ := schema.(map[string]any)

	const description = "[MODE:BROWNFIELD] Retemplate a deployed playground in-place: preserves the playground id, swaps it onto a (potentially fresh) template, optionally provisions new private Gitea-backed Props for repos the player doesn't yet own, and rolls it out. Single-call brownfield analog of fibe_greenfield_create. Pass template_body to author a new template inline (the server creates the ImportTemplate + first version), or template_id/template_version_id to use an existing one."

	s.addTool(&toolImpl{
		name:        "fibe_playgrounds_retemplate",
		description: description,
		tier:        tierBrownfield,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			if _, _, err := resourceschema.ValidatePayload("playground", "retemplate", args); err != nil {
				return nil, err
			}
			mode := strings.ToLower(strings.TrimSpace(argString(args, "mode")))
			if mode == "" {
				mode = "apply"
			}
			if mode == "apply" && !s.cfg.Yolo && !yoloFromContext(ctx) && !argBool(args, "confirm") {
				return nil, &confirmRequiredError{tool: "fibe_playgrounds_retemplate"}
			}
			params, err := buildRetemplateParams(args, mode)
			if err != nil {
				return nil, err
			}
			return c.Retemplate(ctx, params)
		},
	}, mcp.NewTool("fibe_playgrounds_retemplate",
		mcp.WithDescription(description),
		withRawInputSchema(inputSchema),
	))
}

func buildRetemplateParams(args map[string]any, mode string) (*fibe.PlaygroundRetemplateParams, error) {
	identifier, err := requiredIdentifier(args, "playground_id", "playground_identifier")
	if err != nil {
		return nil, err
	}
	body := argString(args, "template_body")
	if body == "" {
		read, perr := readInlineOrPathTextArgOptional(args, "template_body", "template_body_path")
		if perr != nil {
			return nil, perr
		}
		body = read
	}

	templateID, _ := argInt64(args, "template_id")
	versionID, _ := argInt64(args, "template_version_id")
	if body == "" && templateID == 0 && versionID == 0 {
		return nil, fmt.Errorf("must provide template_body, template_body_path, template_id, or template_version_id")
	}

	playgroundID, _ := argInt64(args, "playground_id")
	params := &fibe.PlaygroundRetemplateParams{
		PlaygroundID:          playgroundID,
		PlaygroundIdentifier:  identifier,
		Mode:                  mode,
		TemplateBody:          body,
		TemplateID:            templateID,
		TemplateVersionID:     versionID,
		TemplateName:          argString(args, "template_name"),
		Variables:             argMap(args, "variables"),
		RegenerateVariables:   argStringSlice(args, "regenerate_variables"),
		ConfirmWarnings:       argBool(args, "confirm_warnings"),
		ProvisionMissingProps: argString(args, "provision_missing_props"),
		Wait:                  argBoolDefault(args, "wait", true),
		ResponseMode:          argString(args, "response_mode"),
		Changelog:             argString(args, "changelog"),
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
