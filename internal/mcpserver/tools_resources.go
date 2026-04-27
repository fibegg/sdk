package mcpserver

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

type flatResourceTool struct {
	list   func(context.Context, *fibe.Client, map[string]any) (any, error)
	get    func(context.Context, *fibe.Client, int64) (any, error)
	delete func(context.Context, *fibe.Client, int64) error
}

func (s *Server) registerResourceTools() {
	resources := flatResourceTools()
	listResourceSelectors := resourceschema.ResourceSelectorsForOperation("list")
	getResourceSelectors := resourceschema.ResourceSelectorsForOperation("get")
	deleteResourceSelectors := resourceschema.ResourceSelectorsForOperation("delete")

	s.addTool(&toolImpl{
		name:        "fibe_resource_list",
		description: "[MODE:DIALOG] List a supported flat Fibe resource. Use fibe_schema with resource=list to discover resource names, aliases, and list params.",
		tier:        tierBase,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			name, rt, err := resolveFlatResource(resources, args, "list")
			if err != nil {
				return nil, err
			}
			if _, _, err := resourceschema.ValidatePayload(name, "list", canonicalResourceArgs(args, name)); err != nil {
				return nil, err
			}
			params, err := resourceParams(args)
			if err != nil {
				return nil, err
			}
			return rt.list(ctx, c, params)
		},
	}, mcp.NewTool("fibe_resource_list",
		mcp.WithDescription("[MODE:DIALOG] List a supported flat Fibe resource. Use fibe_schema with resource=list to discover resource names, aliases, and list params."),
		mcp.WithString("resource", mcp.Required(), mcp.Enum(listResourceSelectors...), mcp.Description("Canonical resource name or explicit alias, e.g. playground, playspec, prop, api_key.")),
		mcp.WithObject("params", mcp.Description("Resource-specific list filters. Inspect with fibe_schema(resource:<name>, operation:list).")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_resource_get",
		description: "[MODE:DIALOG] Get a supported Fibe resource by ID. Use artefact_attachment to download an artefact's single attached file.",
		tier:        tierBase,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			name, rt, err := resolveFlatResource(resources, args, "get")
			if err != nil {
				return nil, err
			}
			if _, ok := args["reveal"]; ok {
				return nil, fmt.Errorf("field 'reveal' is not supported by fibe_resource_get")
			}
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			if id <= 0 {
				return nil, fmt.Errorf("field 'id' must be greater than zero")
			}
			if _, _, err := resourceschema.ValidatePayload(name, "get", canonicalResourceArgs(args, name)); err != nil {
				return nil, err
			}
			return rt.get(ctx, c, id)
		},
	}, mcp.NewTool("fibe_resource_get",
		mcp.WithDescription("[MODE:DIALOG] Get a supported Fibe resource by ID. Secret and job_env reads do not reveal plaintext values; artefact_attachment returns base64 file content."),
		mcp.WithString("resource", mcp.Required(), mcp.Enum(getResourceSelectors...), mcp.Description("Canonical resource name or explicit alias, e.g. playground, artefact, artefact_attachment, playspec, prop, webhook.")),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("ID of the selected resource.")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_resource_delete",
		description: "[MODE:SIDEEFFECTS] Delete a supported flat Fibe resource by ID.",
		tier:        tierBase,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			name, rt, err := resolveFlatResource(resources, args, "delete")
			if err != nil {
				return nil, err
			}
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			if id <= 0 {
				return nil, fmt.Errorf("field 'id' must be greater than zero")
			}
			validationArgs := canonicalResourceArgs(args, name)
			delete(validationArgs, "confirm")
			if _, _, err := resourceschema.ValidatePayload(name, "delete", validationArgs); err != nil {
				return nil, err
			}
			if err := rt.delete(ctx, c, id); err != nil {
				return nil, err
			}
			return map[string]any{"resource": name, "id": id, "deleted": true}, nil
		},
	}, mcp.NewTool("fibe_resource_delete",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Delete a supported flat Fibe resource by ID."),
		mcp.WithString("resource", mcp.Required(), mcp.Enum(deleteResourceSelectors...), mcp.Description("Canonical resource name or explicit alias, e.g. playground, playspec, prop, api_key.")),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("ID of the selected resource.")),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	))
}

func flatResourceTools() map[string]flatResourceTool {
	return map[string]flatResourceTool{
		"playground": {
			list: listResource[fibe.PlaygroundListParams](func(ctx context.Context, c *fibe.Client, p *fibe.PlaygroundListParams) (any, error) {
				f := false
				if p.JobMode == nil {
					p.JobMode = &f
				}
				return c.Playgrounds.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Playgrounds.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.Playgrounds.Delete(ctx, id)
			},
		},
		"trick": {
			list: listResource[fibe.PlaygroundListParams](func(ctx context.Context, c *fibe.Client, p *fibe.PlaygroundListParams) (any, error) {
				return c.Tricks.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Tricks.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.Tricks.Delete(ctx, id)
			},
		},
		"agent": {
			list: listResource[fibe.AgentListParams](func(ctx context.Context, c *fibe.Client, p *fibe.AgentListParams) (any, error) {
				return c.Agents.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Agents.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.Agents.Delete(ctx, id)
			},
		},
		"artefact": {
			list: listResource[fibe.ArtefactListParams](func(ctx context.Context, c *fibe.Client, p *fibe.ArtefactListParams) (any, error) {
				return c.Artefacts.ListAll(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Artefacts.GetByID(ctx, id)
			},
		},
		"artefact_attachment": {
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				body, filename, contentType, err := c.Artefacts.DownloadByID(ctx, id)
				if err != nil {
					return nil, err
				}
				defer body.Close()
				data, err := io.ReadAll(body)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"resource":       "artefact_attachment",
					"artefact_id":    id,
					"filename":       filename,
					"content_type":   contentType,
					"content_base64": base64.StdEncoding.EncodeToString(data),
					"size":           len(data),
				}, nil
			},
		},
		"playspec": {
			list: listResource[fibe.PlayspecListParams](func(ctx context.Context, c *fibe.Client, p *fibe.PlayspecListParams) (any, error) {
				return c.Playspecs.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Playspecs.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.Playspecs.Delete(ctx, id)
			},
		},
		"prop": {
			list: listResource[fibe.PropListParams](func(ctx context.Context, c *fibe.Client, p *fibe.PropListParams) (any, error) {
				return c.Props.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Props.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.Props.Delete(ctx, id)
			},
		},
		"marquee": {
			list: listResource[fibe.MarqueeListParams](func(ctx context.Context, c *fibe.Client, p *fibe.MarqueeListParams) (any, error) {
				return c.Marquees.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Marquees.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.Marquees.Delete(ctx, id)
			},
		},
		"secret": {
			list: listResource[fibe.SecretListParams](func(ctx context.Context, c *fibe.Client, p *fibe.SecretListParams) (any, error) {
				return c.Secrets.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.Secrets.Get(ctx, id, false)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.Secrets.Delete(ctx, id)
			},
		},
		"api_key": {
			list: listResource[fibe.APIKeyListParams](func(ctx context.Context, c *fibe.Client, p *fibe.APIKeyListParams) (any, error) {
				return c.APIKeys.List(ctx, p)
			}),
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.APIKeys.Delete(ctx, id)
			},
		},
		"webhook": {
			list: listResource[fibe.WebhookEndpointListParams](func(ctx context.Context, c *fibe.Client, p *fibe.WebhookEndpointListParams) (any, error) {
				return c.WebhookEndpoints.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.WebhookEndpoints.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.WebhookEndpoints.Delete(ctx, id)
			},
		},
		"webhook_delivery": {
			list: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
				id, ok := argInt64(args, "webhook_id")
				if !ok {
					return nil, fmt.Errorf("required field 'params.webhook_id' not set")
				}
				if id <= 0 {
					return nil, fmt.Errorf("field 'params.webhook_id' must be greater than zero")
				}
				var p fibe.ListParams
				if err := bindArgs(args, &p); err != nil {
					return nil, err
				}
				return c.WebhookEndpoints.ListDeliveries(ctx, id, &p)
			},
		},
		"template": {
			list: listResource[fibe.ImportTemplateListParams](func(ctx context.Context, c *fibe.Client, p *fibe.ImportTemplateListParams) (any, error) {
				return c.ImportTemplates.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.ImportTemplates.Get(ctx, id)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.ImportTemplates.Delete(ctx, id)
			},
		},
		"template_version": {
			list: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
				id, ok := argInt64(args, "template_id")
				if !ok {
					return nil, fmt.Errorf("required field 'params.template_id' not set")
				}
				if id <= 0 {
					return nil, fmt.Errorf("field 'params.template_id' must be greater than zero")
				}
				var p fibe.ListParams
				if err := bindArgs(args, &p); err != nil {
					return nil, err
				}
				return c.ImportTemplates.ListVersions(ctx, id, &p)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.ImportTemplateVersions.Delete(ctx, id)
			},
		},
		"template_source": {
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				_, err := c.ImportTemplates.ClearSource(ctx, id)
				return err
			},
		},
		"job_env": {
			list: listResource[fibe.JobEnvListParams](func(ctx context.Context, c *fibe.Client, p *fibe.JobEnvListParams) (any, error) {
				return c.JobEnv.List(ctx, p)
			}),
			get: func(ctx context.Context, c *fibe.Client, id int64) (any, error) {
				return c.JobEnv.Get(ctx, id, false)
			},
			delete: func(ctx context.Context, c *fibe.Client, id int64) error {
				return c.JobEnv.Delete(ctx, id)
			},
		},
		"audit_log": {
			list: listResource[fibe.AuditLogListParams](func(ctx context.Context, c *fibe.Client, p *fibe.AuditLogListParams) (any, error) {
				return c.AuditLogs.List(ctx, p)
			}),
		},
		"category": {
			list: listResource[fibe.ListParams](func(ctx context.Context, c *fibe.Client, p *fibe.ListParams) (any, error) {
				return c.TemplateCategories.List(ctx, p)
			}),
		},
	}
}

func listResource[P any](fn func(context.Context, *fibe.Client, *P) (any, error)) func(context.Context, *fibe.Client, map[string]any) (any, error) {
	return func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
		var p P
		if err := bindArgs(args, &p); err != nil {
			return nil, err
		}
		return fn(ctx, c, &p)
	}
}

func resolveFlatResource(resources map[string]flatResourceTool, args map[string]any, operation string) (string, flatResourceTool, error) {
	raw := argString(args, "resource")
	if raw == "" {
		return "", flatResourceTool{}, fmt.Errorf("required field 'resource' not set")
	}
	name, ok := resourceschema.CanonicalResource(raw)
	if !ok {
		return "", flatResourceTool{}, fmt.Errorf("unknown resource %q; supported flat resources: %s", raw, resourceschema.FlatResourceNamesString())
	}
	rt, ok := resources[name]
	if !ok {
		return "", flatResourceTool{}, fmt.Errorf("resource %q is not supported by generic resource tools", name)
	}
	switch operation {
	case "list":
		if rt.list == nil {
			return "", flatResourceTool{}, fmt.Errorf("resource %q does not support list", name)
		}
	case "get":
		if rt.get == nil {
			return "", flatResourceTool{}, fmt.Errorf("resource %q does not support get", name)
		}
	case "delete":
		if rt.delete == nil {
			return "", flatResourceTool{}, fmt.Errorf("resource %q does not support delete", name)
		}
	default:
		return "", flatResourceTool{}, fmt.Errorf("unsupported resource operation %q", operation)
	}
	return name, rt, nil
}

func resourceParams(args map[string]any) (map[string]any, error) {
	raw, ok := args["params"]
	if !ok || raw == nil {
		return map[string]any{}, nil
	}
	params, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field 'params' must be an object")
	}
	return params, nil
}

func canonicalResourceArgs(args map[string]any, canonical string) map[string]any {
	out := make(map[string]any, len(args))
	for key, value := range args {
		out[key] = value
	}
	out["resource"] = canonical
	return out
}
