package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/localplaygrounds"
	"github.com/mark3labs/mcp-go/mcp"
)

const greenfieldDefaultLinkDir = "/app/playground"

func (s *Server) registerGreenfieldTools() {
	s.addTool(&toolImpl{
		name: "fibe_greenfield_create", description: "[MODE:GREENFIELD] Create a new repository, Prop, app-owned template version, deployed playground, wait for running, and link it locally.", tier: tierGreenfield,
		annotations: toolAnnotations{Idempotent: false},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			params, waitTimeout, err := greenfieldArgs(args)
			if err != nil {
				return nil, err
			}

			result, err := c.Greenfield.Create(ctx, params)
			if err != nil {
				return nil, err
			}
			if result.Playground == nil || result.Playground.ID == 0 {
				return nil, fmt.Errorf("greenfield create did not return a playground id")
			}

			if waitTimeout > 0 {
				waitArgs := map[string]any{
					"playground_id": result.Playground.ID,
					"status":        "running",
					"timeout":       waitTimeout.String(),
					"interval":      "3s",
				}
				if _, err := s.runWait(ctx, c, waitArgs); err != nil {
					return nil, err
				}
			}
			pg, err := c.Playgrounds.Get(ctx, result.Playground.ID)
			if err != nil {
				return nil, err
			}
			result.Playground = pg

			link, err := localplaygrounds.Link(greenfieldTarget(result), greenfieldDefaultLinkDir)
			if err != nil {
				result.Link = nil
			} else {
				result.Link = link
			}

			return result, nil
		},
	}, mcp.NewTool("fibe_greenfield_create",
		mcp.WithDescription("Create a greenfield app in one call: repo, Prop, app template version, deployed playground, wait until running, and local /app/playground link."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Repository/app name; must be unique.")),
		mcp.WithNumber("template_id", mcp.Description("Template ID to use. Optional; defaults to the base template.")),
		mcp.WithString("version", mcp.Description("Template version tag or number for template_id, e.g. v1. Optional; defaults to latest available version.")),
		mcp.WithString("template_body", mcp.Description("Template YAML body to use directly. Optional; cannot be combined with template_id or version.")),
		mcp.WithString("template_body_path", mcp.Description("Absolute local path to a template YAML file (local MCP only). Optional; cannot be combined with template_body, template_id, or version.")),
		mcp.WithString("git_provider", mcp.Description("Destination git provider: gitea or github. Optional; default: gitea.")),
		mcp.WithNumber("marquee_id", mcp.Description("Target marquee ID. Optional; defaults to the current Marquee from FIBE_MARQUEE_ID.")),
		mcp.WithObject("variables", mcp.Description("Template variables map, e.g. {\"app_name\":\"Tower\"}. Optional.")),
		mcp.WithString("wait_timeout", mcp.Description("Max wait duration, e.g. 10m. Optional; default: 10m.")),
	))
}

func greenfieldArgs(args map[string]any) (*fibe.GreenfieldCreateParams, time.Duration, error) {
	name := argString(args, "name")
	if name == "" {
		return nil, 0, fmt.Errorf("required field 'name' not set")
	}
	if argString(args, "template_body") != "" && argString(args, "template_body_path") != "" {
		return nil, 0, fmt.Errorf("pass only one of template_body or template_body_path")
	}

	gitProvider := argString(args, "git_provider")
	if gitProvider == "" {
		gitProvider = "gitea"
	}

	var templateID *int64
	if id, ok := argInt64(args, "template_id"); ok && id > 0 {
		templateID = &id
	}
	version := argString(args, "version")
	if version != "" && templateID == nil {
		return nil, 0, fmt.Errorf("version requires template_id")
	}
	templateBody, err := readInlineOrPathTextArgOptional(args, "template_body", "template_body_path")
	if err != nil {
		return nil, 0, err
	}
	if templateBody != "" && (templateID != nil || version != "") {
		return nil, 0, fmt.Errorf("template_body cannot be combined with template_id or version")
	}

	marqueeID, ok := argInt64(args, "marquee_id")
	if !ok || marqueeID <= 0 {
		envID, err := parseMarqueeIDEnv()
		if err != nil {
			return nil, 0, err
		}
		marqueeID = envID
	}

	timeout := parseDuration(argString(args, "wait_timeout"), 10*time.Minute)

	params := &fibe.GreenfieldCreateParams{
		Name:         name,
		TemplateID:   templateID,
		Version:      version,
		TemplateBody: templateBody,
		GitProvider:  gitProvider,
		MarqueeID:    &marqueeID,
		Variables:    greenfieldVariables(args["variables"]),
	}
	return params, timeout, nil
}

func parseMarqueeIDEnv() (int64, error) {
	raw := strings.TrimSpace(os.Getenv("FIBE_MARQUEE_ID"))
	if raw == "" {
		return 0, fmt.Errorf("marquee_id is required when the current Marquee is not available (FIBE_MARQUEE_ID is not set)")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("FIBE_MARQUEE_ID must be a positive integer")
	}
	return id, nil
}

func greenfieldVariables(raw any) map[string]any {
	out := map[string]any{}
	values, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for key, value := range values {
		normalized := normalizeVariableKey(key)
		if normalized != "" {
			out[normalized] = value
		}
	}
	return out
}

func normalizeVariableKey(key string) string {
	return strings.TrimLeft(strings.TrimSpace(key), "-")
}

func readInlineOrPathTextArgOptional(args map[string]any, inlineKey, pathKey string) (string, error) {
	if argString(args, inlineKey) == "" && argString(args, pathKey) == "" {
		return "", nil
	}
	return readInlineOrPathTextArg(args, inlineKey, pathKey)
}

func greenfieldTarget(result *fibe.GreenfieldResult) string {
	if result.Playground != nil && result.Playground.PlayspecName != nil && *result.Playground.PlayspecName != "" {
		return *result.Playground.PlayspecName
	}
	if result.Playspec != nil && result.Playspec.Name != "" {
		return result.Playspec.Name
	}
	if result.Playground != nil && result.Playground.Name != "" {
		return result.Playground.Name
	}
	return result.Name
}
