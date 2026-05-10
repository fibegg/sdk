package fibe

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PlaygroundRetemplateParams configures the brownfield analog of greenfield_create:
// take an existing deployed playground, transform it onto a (potentially fresh) template,
// optionally provision new private Gitea-backed Props on the fly, and roll it out.
//
// One of the following must be provided to identify the new template version:
//  1. TemplateVersionID — use an exact existing version.
//  2. TemplateID + (no body) — use the latest version of that template.
//  3. TemplateBody / TemplateBodyPath (+ optional TemplateID) — author a fresh
//     template version on the fly. If TemplateID is omitted, a new ImportTemplate
//     is created (auto-named for the playground) and a first version is published
//     under it.
type PlaygroundRetemplateParams struct {
	PlaygroundID          int64                `json:"playground_id"`
	PlaygroundIdentifier  string               `json:"playground_identifier,omitempty"`
	Mode                  string               `json:"mode,omitempty"` // "preview" | "apply" (default)
	TemplateBody          string               `json:"template_body,omitempty"`
	TemplateID            int64                `json:"template_id,omitempty"`
	TemplateVersionID     int64                `json:"template_version_id,omitempty"`
	TemplateName          string               `json:"template_name,omitempty"`
	Variables             map[string]any       `json:"variables,omitempty"`
	RegenerateVariables   []string             `json:"regenerate_variables,omitempty"`
	ConfirmWarnings       bool                 `json:"confirm_warnings,omitempty"`
	ProvisionMissingProps string               `json:"provision_missing_props,omitempty"`
	ProvisionPrivate      *bool                `json:"provision_private,omitempty"`
	ProvisionInputs       []ProvisionPropInput `json:"provision_inputs,omitempty"`
	ReuseExistingProps    bool                 `json:"reuse_existing_props,omitempty"`
	Wait                  bool                 `json:"wait,omitempty"`
	WaitTimeoutSeconds    int64                `json:"wait_timeout_seconds,omitempty"`
	DiagnoseOnFailure     *bool                `json:"diagnose_on_failure,omitempty"`
	ResponseMode          string               `json:"response_mode,omitempty"`
	Changelog             string               `json:"changelog,omitempty"`
}

// PlaygroundRetemplateResult is the composite response from a playground transform run.
type PlaygroundRetemplateResult struct {
	Mode             string                               `json:"mode"`
	Playground       *Playground                          `json:"playground,omitempty"`
	Template         *ImportTemplate                      `json:"template,omitempty"`
	TemplateVersion  *ImportTemplateVersion               `json:"template_version,omitempty"`
	SwitchResult     *PlayspecTemplateVersionSwitchResult `json:"switch_result,omitempty"`
	ProvisionedProps []ProvisionedPropResult              `json:"provisioned_props,omitempty"`
	WaitResults      []map[string]any                     `json:"wait_results,omitempty"`
	Diagnostics      map[string]any                       `json:"diagnostics,omitempty"`
}

// Retemplate composes ImportTemplate{,Version}.Create + Playspec.SwitchTemplateVersion
// + post-rollout wait into a single brownfield playground transform flow.
func (c *Client) Retemplate(ctx context.Context, params *PlaygroundRetemplateParams) (*PlaygroundRetemplateResult, error) {
	if params == nil {
		return nil, fmt.Errorf("params is required")
	}
	mode := strings.ToLower(strings.TrimSpace(params.Mode))
	if mode == "" {
		mode = "apply"
	}
	if mode != "apply" && mode != "preview" {
		return nil, fmt.Errorf("mode must be apply or preview")
	}

	playgroundIdentifier := params.PlaygroundIdentifier
	if playgroundIdentifier == "" && params.PlaygroundID > 0 {
		playgroundIdentifier = int64Identifier(params.PlaygroundID)
	}
	if playgroundIdentifier == "" {
		return nil, fmt.Errorf("playground_id (or playground_identifier) is required")
	}

	pg, err := c.Playgrounds.GetByIdentifier(ctx, playgroundIdentifier)
	if err != nil {
		return nil, fmt.Errorf("could not load playground: %w", err)
	}
	if pg.PlayspecID == nil || *pg.PlayspecID <= 0 {
		return nil, fmt.Errorf("playground %d has no playspec_id", pg.ID)
	}

	out := &PlaygroundRetemplateResult{Mode: mode, Playground: pg}
	if err := c.ensureRetemplateSourceTemplate(ctx, pg); err != nil {
		return out, err
	}

	templateID, versionID, tmpl, version, err := c.resolveRetemplateTarget(ctx, pg, params)
	if err != nil {
		return out, err
	}
	if tmpl != nil {
		out.Template = tmpl
	}
	if version != nil {
		out.TemplateVersion = version
	}

	switchParams := &PlayspecTemplateVersionSwitchParams{
		TargetTemplateVersionID: versionID,
		Variables:               params.Variables,
		RegenerateVariables:     params.RegenerateVariables,
		ConfirmWarnings:         params.ConfirmWarnings,
		RolloutMode:             retemplateRolloutMode(mode),
		TargetPlaygroundID:      &pg.ID,
		ResponseMode:            params.ResponseMode,
		ProvisionMissingProps:   params.ProvisionMissingProps,
		ProvisionPrivate:        params.ProvisionPrivate,
		ProvisionInputs:         params.ProvisionInputs,
		ReuseExistingProps:      params.ReuseExistingProps,
	}

	if mode == "preview" {
		previewResult, perr := c.Playspecs.PreviewTemplateVersionSwitch(ctx, *pg.PlayspecID, switchParams)
		if perr != nil {
			return out, perr
		}
		out.SwitchResult = previewResult
		if previewResult != nil {
			out.ProvisionedProps = previewResult.ProvisionedProps
		}
		_ = templateID // surface for caller
		return out, nil
	}

	switchResult, serr := c.Playspecs.SwitchTemplateVersion(ctx, *pg.PlayspecID, switchParams)
	if serr != nil {
		return out, serr
	}
	if err := VerifyTemplateVersionSwitchResult(switchResult, versionID); err != nil {
		return out, err
	}
	out.SwitchResult = switchResult
	if switchResult != nil {
		out.ProvisionedProps = switchResult.ProvisionedProps
	}

	if !params.Wait {
		return out, nil
	}

	timeout := time.Duration(params.WaitTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	waitResult := waitForRetemplateRollout(ctx, c, pg.ID, timeout)
	out.WaitResults = []map[string]any{waitResult}
	if diagnoseRetemplate(params) && waitResult["success"] != true {
		refresh := true
		debug, derr := c.Playgrounds.DebugWithParams(ctx, pg.ID, &PlaygroundDebugParams{Mode: "summary", Refresh: &refresh, LogsTail: 50})
		if derr != nil {
			out.Diagnostics = map[string]any{fmt.Sprintf("%d", pg.ID): map[string]any{"error": derr.Error()}}
		} else {
			out.Diagnostics = map[string]any{fmt.Sprintf("%d", pg.ID): debug}
		}
	}
	return out, nil
}

func (c *Client) ensureRetemplateSourceTemplate(ctx context.Context, pg *Playground) error {
	if pg == nil || pg.PlayspecID == nil || *pg.PlayspecID <= 0 {
		return fmt.Errorf("playground has no playspec_id")
	}
	ps, err := c.Playspecs.Get(ctx, *pg.PlayspecID)
	if err != nil {
		return fmt.Errorf("could not load playspec %d before transform: %w", *pg.PlayspecID, err)
	}
	if ps.SourceTemplateVersionID == nil || *ps.SourceTemplateVersionID <= 0 {
		return fmt.Errorf("playground %d cannot be transformed because playspec %d was not launched from a template version", pg.ID, *pg.PlayspecID)
	}
	return nil
}

func retemplateRolloutMode(mode string) string {
	if mode == "apply" {
		return "target"
	}
	return "none"
}

func diagnoseRetemplate(params *PlaygroundRetemplateParams) bool {
	if params == nil || params.DiagnoseOnFailure == nil {
		return true
	}
	return *params.DiagnoseOnFailure
}

func (c *Client) resolveRetemplateTarget(ctx context.Context, pg *Playground, params *PlaygroundRetemplateParams) (int64, int64, *ImportTemplate, *ImportTemplateVersion, error) {
	body := params.TemplateBody

	switch {
	case params.TemplateVersionID > 0:
		return params.TemplateID, params.TemplateVersionID, nil, nil, nil

	case body != "":
		templateID := params.TemplateID
		var tmpl *ImportTemplate
		if templateID <= 0 {
			name := params.TemplateName
			if name == "" {
				name = fmt.Sprintf("playground-%d-transform-%d", pg.ID, time.Now().UnixNano())
			}
			created, err := c.ImportTemplates.Create(ctx, &ImportTemplateCreateParams{
				Name:         name,
				TemplateBody: body,
			})
			if err != nil {
				return 0, 0, nil, nil, fmt.Errorf("could not create import template: %w", err)
			}
			tmpl = created
			if created != nil && created.ID != nil {
				templateID = *created.ID
			}
			if created != nil && created.LatestVersionID != nil {
				return templateID, *created.LatestVersionID, tmpl, nil, nil
			}
		}
		if templateID <= 0 {
			return 0, 0, tmpl, nil, fmt.Errorf("could not resolve template_id after creation")
		}

		changelog := params.Changelog
		var changelogPtr *string
		if changelog != "" {
			changelogPtr = &changelog
		}
		version, err := c.ImportTemplates.CreateVersion(ctx, templateID, &ImportTemplateVersionCreateParams{
			TemplateBody: body,
			Changelog:    changelogPtr,
		})
		if err != nil {
			return templateID, 0, tmpl, nil, fmt.Errorf("could not create template version: %w", err)
		}
		if version == nil || version.ID == nil {
			return templateID, 0, tmpl, nil, fmt.Errorf("template version response missing id")
		}
		return templateID, *version.ID, tmpl, version, nil

	case params.TemplateID > 0:
		tmpl, err := c.ImportTemplates.Get(ctx, params.TemplateID)
		if err != nil {
			return params.TemplateID, 0, nil, nil, fmt.Errorf("could not load template: %w", err)
		}
		if tmpl == nil || tmpl.LatestVersionID == nil {
			return params.TemplateID, 0, tmpl, nil, fmt.Errorf("template %d has no published versions", params.TemplateID)
		}
		return params.TemplateID, *tmpl.LatestVersionID, tmpl, nil, nil

	default:
		return 0, 0, nil, nil, fmt.Errorf("must provide template_version_id, template_id, or template_body")
	}
}

func waitForRetemplateRollout(ctx context.Context, c *Client, playgroundID int64, timeout time.Duration) map[string]any {
	deadline := time.Now().Add(timeout)
	var lastStatus string
	for {
		status, err := c.Playgrounds.Status(ctx, playgroundID)
		if err != nil {
			return map[string]any{"id": playgroundID, "success": false, "error": err.Error(), "last_status": lastStatus}
		}
		lastStatus = status.Status
		if status.Status == "running" || status.Status == "completed" {
			return map[string]any{"id": playgroundID, "success": true, "status": status.Status}
		}
		if status.Status == "error" || status.Status == "failed" || status.Status == "destroyed" {
			return map[string]any{"id": playgroundID, "success": false, "status": status.Status, "failure_diagnostics": status.FailureDiagnostics}
		}
		if time.Now().After(deadline) {
			return map[string]any{"id": playgroundID, "success": false, "status": status.Status, "error": fmt.Sprintf("timeout after %s", timeout)}
		}
		select {
		case <-ctx.Done():
			return map[string]any{"id": playgroundID, "success": false, "status": status.Status, "error": ctx.Err().Error()}
		case <-time.After(3 * time.Second):
		}
	}
}
