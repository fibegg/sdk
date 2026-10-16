package fibe

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PlaygroundTemplateSwitchParams moves an existing playground to an exact,
// latest, or newly authored template version, with optional private Repository provisioning.
type PlaygroundTemplateSwitchParams struct {
	PlaygroundID                 int64                      `json:"playground_id"`
	PlaygroundIdentifier         string                     `json:"playground_identifier,omitempty"`
	Mode                         string                     `json:"mode,omitempty"` // "preview" | "apply" (default)
	TemplateBody                 string                     `json:"template_body,omitempty"`
	TemplateID                   int64                      `json:"template_id,omitempty"`
	TemplateIdentifier           string                     `json:"template_identifier,omitempty"`
	TemplateVersionID            int64                      `json:"template_version_id,omitempty"`
	TemplateName                 string                     `json:"template_name,omitempty"`
	Variables                    map[string]any             `json:"variables,omitempty"`
	RegenerateVariables          []string                   `json:"regenerate_variables,omitempty"`
	ConfirmWarnings              bool                       `json:"confirm_warnings,omitempty"`
	ProvisionMissingRepositories string                     `json:"provision_missing_repositories,omitempty"`
	ProvisionPrivate             *bool                      `json:"provision_private,omitempty"`
	ProvisionInputs              []ProvisionRepositoryInput `json:"provision_inputs,omitempty"`
	ReuseExistingRepositories    bool                       `json:"reuse_existing_repositories,omitempty"`
	Wait                         bool                       `json:"wait,omitempty"`
	WaitTimeoutSeconds           int64                      `json:"wait_timeout_seconds,omitempty"`
	DiagnoseOnFailure            *bool                      `json:"diagnose_on_failure,omitempty"`
	ResponseMode                 string                     `json:"response_mode,omitempty"`
	Changelog                    string                     `json:"changelog,omitempty"`
}

// PlaygroundTemplateSwitchResult is the composite response from a playground template switch run.
type PlaygroundTemplateSwitchResult struct {
	Mode                    string                           `json:"mode"`
	Playground              *Playground                      `json:"playground,omitempty"`
	Template                *ImportTemplate                  `json:"template,omitempty"`
	TemplateVersion         *ImportTemplateVersion           `json:"template_version,omitempty"`
	SwitchResult            *SpecTemplateVersionSwitchResult `json:"switch_result,omitempty"`
	ProvisionedRepositories []ProvisionedRepositoryResult    `json:"provisioned_repositories,omitempty"`
	WaitResults             []map[string]any                 `json:"wait_results,omitempty"`
	Diagnostics             map[string]any                   `json:"diagnostics,omitempty"`
}

// SwitchPlaygroundTemplate composes ImportTemplate{,Version}.Create + Spec.SwitchTemplateVersion
// + post-rollout wait into a single brownfield playground template switch flow.
func (c *Client) SwitchPlaygroundTemplate(ctx context.Context, params *PlaygroundTemplateSwitchParams) (*PlaygroundTemplateSwitchResult, error) {
	if params == nil {
		return nil, fmt.Errorf("params is required")
	}
	// A composed mutation needs complete intermediate resource responses.
	ctx = WithFields(ctx)
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
		return nil, fmt.Errorf("id_or_name is required")
	}

	pg, err := c.Playgrounds.GetByIdentifier(ctx, playgroundIdentifier)
	if err != nil {
		return nil, fmt.Errorf("could not load playground: %w", err)
	}
	if pg.SpecID == nil || *pg.SpecID <= 0 {
		return nil, fmt.Errorf("playground %d has no spec_id", pg.ID)
	}

	out := &PlaygroundTemplateSwitchResult{Mode: mode, Playground: pg}
	if err := c.ensureTemplateSwitchSource(ctx, pg); err != nil {
		return out, err
	}

	templateID, versionID, tmpl, version, err := c.resolveTemplateSwitchTarget(ctx, pg, params)
	if err != nil {
		return out, err
	}
	if tmpl != nil {
		out.Template = tmpl
	}
	if version != nil {
		out.TemplateVersion = version
	}

	switchParams := &SpecTemplateVersionSwitchParams{
		TargetTemplateVersionID:      versionID,
		Variables:                    params.Variables,
		RegenerateVariables:          params.RegenerateVariables,
		ConfirmWarnings:              params.ConfirmWarnings,
		RolloutMode:                  templateSwitchRolloutMode(mode),
		TargetPlaygroundID:           &pg.ID,
		ResponseMode:                 params.ResponseMode,
		ProvisionMissingRepositories: params.ProvisionMissingRepositories,
		ProvisionPrivate:             params.ProvisionPrivate,
		ProvisionInputs:              params.ProvisionInputs,
		ReuseExistingRepositories:    params.ReuseExistingRepositories,
	}

	if mode == "preview" {
		previewResult, perr := c.Specs.PreviewTemplateVersionSwitch(ctx, *pg.SpecID, switchParams)
		if perr != nil {
			return out, perr
		}
		out.SwitchResult = previewResult
		if previewResult != nil {
			out.ProvisionedRepositories = previewResult.ProvisionedRepositories
		}
		_ = templateID // surface for caller
		return out, nil
	}

	switchResult, serr := c.Specs.SwitchTemplateVersion(ctx, *pg.SpecID, switchParams)
	if serr != nil {
		return out, serr
	}
	if err := VerifyTemplateVersionSwitchResult(switchResult, versionID); err != nil {
		return out, err
	}
	out.SwitchResult = switchResult
	if switchResult != nil {
		out.ProvisionedRepositories = switchResult.ProvisionedRepositories
	}

	if !params.Wait {
		return out, nil
	}

	timeout := time.Duration(params.WaitTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	waitResult := waitForTemplateSwitchRollout(ctx, c, pg.ID, timeout)
	out.WaitResults = []map[string]any{waitResult}
	if diagnoseSwitchPlaygroundTemplate(params) && waitResult["success"] != true {
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

func (c *Client) ensureTemplateSwitchSource(ctx context.Context, pg *Playground) error {
	if pg == nil || pg.SpecID == nil || *pg.SpecID <= 0 {
		return fmt.Errorf("playground has no spec_id")
	}
	ps, err := c.Specs.Get(ctx, *pg.SpecID)
	if err != nil {
		return fmt.Errorf("could not load spec %d before switch-template: %w", *pg.SpecID, err)
	}
	if ps.SourceTemplateVersionID == nil || *ps.SourceTemplateVersionID <= 0 {
		return fmt.Errorf("playground %d cannot be switched because spec %d was not launched from a template version", pg.ID, *pg.SpecID)
	}
	return nil
}

func templateSwitchRolloutMode(mode string) string {
	if mode == "apply" {
		return "target"
	}
	return "none"
}

func diagnoseSwitchPlaygroundTemplate(params *PlaygroundTemplateSwitchParams) bool {
	if params == nil || params.DiagnoseOnFailure == nil {
		return true
	}
	return *params.DiagnoseOnFailure
}

func (c *Client) resolveTemplateSwitchTarget(ctx context.Context, pg *Playground, params *PlaygroundTemplateSwitchParams) (int64, int64, *ImportTemplate, *ImportTemplateVersion, error) {
	body := params.TemplateBody

	switch {
	case params.TemplateVersionID > 0:
		return params.TemplateID, params.TemplateVersionID, nil, nil, nil

	case body != "":
		templateID := params.TemplateID
		templateIdentifier := params.TemplateIdentifier
		var tmpl *ImportTemplate
		if templateID <= 0 && templateIdentifier == "" {
			name := params.TemplateName
			if name == "" {
				name = fmt.Sprintf("playground-%d-switch-template-%d", pg.ID, time.Now().UnixNano())
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
		if templateID <= 0 && templateIdentifier == "" {
			return 0, 0, tmpl, nil, fmt.Errorf("could not resolve template_id_or_name after creation")
		}

		changelog := params.Changelog
		var changelogPtr *string
		if changelog != "" {
			changelogPtr = &changelog
		}
		if templateIdentifier == "" {
			templateIdentifier = int64Identifier(templateID)
		}
		version, err := c.ImportTemplates.CreateVersionByIdentifier(ctx, templateIdentifier, &ImportTemplateVersionCreateParams{
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
		tmpl, err := c.ImportTemplates.GetByIdentifier(ctx, int64Identifier(params.TemplateID))
		if err != nil {
			return params.TemplateID, 0, nil, nil, fmt.Errorf("could not load template: %w", err)
		}
		if tmpl == nil || tmpl.LatestVersionID == nil {
			return params.TemplateID, 0, tmpl, nil, fmt.Errorf("template %d has no published versions", params.TemplateID)
		}
		return params.TemplateID, *tmpl.LatestVersionID, tmpl, nil, nil

	case params.TemplateIdentifier != "":
		tmpl, err := c.ImportTemplates.GetByIdentifier(ctx, params.TemplateIdentifier)
		if err != nil {
			return 0, 0, nil, nil, fmt.Errorf("could not load template: %w", err)
		}
		if tmpl == nil || tmpl.LatestVersionID == nil {
			return 0, 0, tmpl, nil, fmt.Errorf("template %s has no published versions", params.TemplateIdentifier)
		}
		templateID := int64(0)
		if tmpl.ID != nil {
			templateID = *tmpl.ID
		}
		return templateID, *tmpl.LatestVersionID, tmpl, nil, nil

	default:
		return 0, 0, nil, nil, fmt.Errorf("must provide template_version_id, template_id_or_name, or template_body")
	}
}

func waitForTemplateSwitchRollout(ctx context.Context, c *Client, playgroundID int64, timeout time.Duration) map[string]any {
	deadline := time.Now().Add(timeout)
	var lastStatus string
	attempt := 0
	for {
		attempt++
		status, err := c.Playgrounds.RuntimeStatus(ctx, playgroundID)
		if err != nil {
			return map[string]any{"id": playgroundID, "success": false, "error": err.Error(), "last_status": lastStatus}
		}
		lastStatus = status.Status
		c.reportProgress(ctx, c.cfg.progressHook, ProgressEvent{
			Operation: "playground_rollout",
			Status:    status.Status,
			Attempt:   attempt,
		})
		ready, pendingReason := PlaygroundRuntimeStatusMatchesWaitTarget(status, "running", PlaygroundWaitReadinessServices)
		if ready || status.Status == "completed" && !TaskStatusResultFailed(&status.PlaygroundStatus) {
			return map[string]any{"id": playgroundID, "success": true, "status": status.Status}
		}
		if status.Status == "error" || status.Status == "failed" || status.Status == "destroyed" {
			return map[string]any{"id": playgroundID, "success": false, "status": status.Status, "failure_diagnostics": status.FailureDiagnostics}
		}
		if time.Now().After(deadline) {
			return map[string]any{"id": playgroundID, "success": false, "status": status.Status, "error": fmt.Sprintf("timeout after %s: %s", timeout, pendingReason)}
		}
		select {
		case <-ctx.Done():
			return map[string]any{"id": playgroundID, "success": false, "status": status.Status, "error": ctx.Err().Error()}
		case <-time.After(3 * time.Second):
		}
	}
}
