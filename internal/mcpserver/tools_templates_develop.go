package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

type templateDevelopArgs struct {
	TargetType              string                   `json:"target_type"`
	TargetID                int64                    `json:"target_id"`
	Mode                    string                   `json:"mode"`
	ChangeType              string                   `json:"change_type"`
	BaseVersionID           int64                    `json:"base_version_id,omitempty"`
	TargetTemplateVersionID int64                    `json:"target_template_version_id,omitempty"`
	Patches                 []fibe.TemplatePatchEdit `json:"patches,omitempty"`
	Edits                   []fibe.TemplatePatchEdit `json:"edits,omitempty"`
	TemplateBody            string                   `json:"template_body,omitempty"`
	TemplateBodyPath        string                   `json:"template_body_path,omitempty"`
	Changelog               string                   `json:"changelog,omitempty"`
	Public                  *bool                    `json:"public,omitempty"`
	SwitchVariables         map[string]any           `json:"switch_variables,omitempty"`
	RegenerateVariables     []string                 `json:"regenerate_variables,omitempty"`
	ConfirmWarnings         bool                     `json:"confirm_warnings,omitempty"`
	PostApply               string                   `json:"post_apply,omitempty"`
	Wait                    bool                     `json:"wait,omitempty"`
	WaitTimeoutSeconds      int64                    `json:"wait_timeout_seconds,omitempty"`
	DiagnoseOnFailure       *bool                    `json:"diagnose_on_failure,omitempty"`
	ResponseMode            string                   `json:"response_mode,omitempty"`
}

type templateDevelopTarget struct {
	templateID   int64
	playspecID   *int64
	playgroundID *int64
	marqueeID    *int64
	jobMode      bool
	baseVersion  int64
}

func (s *Server) registerTemplateDevelopTools() {
	schema, _, _, _ := resourceschema.SchemaFor("template", "develop")
	inputSchema, _ := schema.(map[string]any)
	s.addTool(&toolImpl{
		name:        "fibe_templates_develop",
		description: "[MODE:BROWNFIELD] Preview or apply template changes, switch playspecs/playgrounds/tricks, and optionally roll out or trigger a fresh trick run.",
		tier:        tierBrownfield,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			if _, _, err := resourceschema.ValidatePayload("template", "develop", args); err != nil {
				return nil, err
			}
			var in templateDevelopArgs
			if err := bindArgs(args, &in); err != nil {
				return nil, err
			}
			return runTemplateDevelop(ctx, c, &in)
		},
	}, mcp.NewTool("fibe_templates_develop",
		mcp.WithDescription("[MODE:BROWNFIELD] Preview or apply brownfield template changes. Patch or overwrite a template version, switch a playspec/playground/trick, and optionally roll out a playground or trigger a fresh trick run."),
		withRawInputSchema(inputSchema),
	))
}

func runTemplateDevelop(ctx context.Context, c *fibe.Client, in *templateDevelopArgs) (any, error) {
	if err := normalizeTemplateDevelopArgs(in); err != nil {
		return nil, err
	}
	target, err := resolveTemplateDevelopTarget(ctx, c, in)
	if err != nil {
		return nil, err
	}
	if err := validateTemplateDevelopCombination(in, target); err != nil {
		return nil, err
	}
	switch in.ChangeType {
	case "switch_existing":
		return runTemplateDevelopSwitch(ctx, c, in, target)
	case "patch", "overwrite":
		return runTemplateDevelopPatch(ctx, c, in, target)
	default:
		return nil, fmt.Errorf("unsupported change_type %q", in.ChangeType)
	}
}

func normalizeTemplateDevelopArgs(in *templateDevelopArgs) error {
	if in.TargetType == "" {
		return fmt.Errorf("required field 'target_type' not set")
	}
	if in.TargetID <= 0 {
		return fmt.Errorf("required field 'target_id' must be greater than zero")
	}
	if in.Mode == "" {
		return fmt.Errorf("required field 'mode' not set")
	}
	if in.Mode != "preview" && in.Mode != "apply" {
		return fmt.Errorf("mode must be preview or apply")
	}
	if in.ChangeType == "" {
		return fmt.Errorf("required field 'change_type' not set")
	}
	if in.PostApply == "" {
		in.PostApply = "none"
	}
	if in.ResponseMode == "" {
		in.ResponseMode = "summary"
	}
	if in.ResponseMode != "summary" && in.ResponseMode != "full" {
		return fmt.Errorf("response_mode must be summary or full")
	}
	if in.WaitTimeoutSeconds <= 0 {
		in.WaitTimeoutSeconds = 180
	}
	return nil
}

func resolveTemplateDevelopTarget(ctx context.Context, c *fibe.Client, in *templateDevelopArgs) (*templateDevelopTarget, error) {
	target := &templateDevelopTarget{baseVersion: in.BaseVersionID}
	switch in.TargetType {
	case "template":
		target.templateID = in.TargetID
		tpl, err := c.ImportTemplates.Get(ctx, in.TargetID)
		if err != nil {
			return nil, err
		}
		if target.baseVersion == 0 && tpl.LatestVersionID != nil {
			target.baseVersion = *tpl.LatestVersionID
		}
	case "playspec":
		ps, err := c.Playspecs.Get(ctx, in.TargetID)
		if err != nil {
			return nil, err
		}
		fillTemplateDevelopTargetFromPlayspec(target, ps)
	case "playground", "trick":
		pg, err := c.Playgrounds.Get(ctx, in.TargetID)
		if err != nil {
			return nil, err
		}
		target.playgroundID = &pg.ID
		target.marqueeID = pg.MarqueeID
		target.jobMode = pg.JobMode
		if pg.PlayspecID == nil || *pg.PlayspecID <= 0 {
			return nil, fmt.Errorf("%s %d has no playspec_id", in.TargetType, in.TargetID)
		}
		ps, err := c.Playspecs.Get(ctx, *pg.PlayspecID)
		if err != nil {
			return nil, err
		}
		fillTemplateDevelopTargetFromPlayspec(target, ps)
		target.jobMode = target.jobMode || boolPtrValue(ps.JobMode)
		if in.TargetType == "trick" && !target.jobMode {
			return nil, fmt.Errorf("target_type trick requires a job-mode playground or playspec")
		}
	default:
		return nil, fmt.Errorf("unsupported target_type %q", in.TargetType)
	}
	if in.BaseVersionID > 0 {
		target.baseVersion = in.BaseVersionID
	}
	if target.templateID <= 0 && in.ChangeType != "switch_existing" {
		return nil, fmt.Errorf("target does not resolve to a source template")
	}
	if target.baseVersion <= 0 && in.ChangeType != "switch_existing" {
		return nil, fmt.Errorf("base_version_id is required because it could not be inferred from target")
	}
	return target, nil
}

func fillTemplateDevelopTargetFromPlayspec(target *templateDevelopTarget, ps *fibe.Playspec) {
	if ps.ID != nil {
		target.playspecID = ps.ID
	}
	target.jobMode = target.jobMode || boolPtrValue(ps.JobMode)
	if target.baseVersion == 0 && ps.SourceTemplateVersionID != nil {
		target.baseVersion = *ps.SourceTemplateVersionID
	}
	if ps.SourceTemplate != nil && ps.SourceTemplate.ID != nil {
		target.templateID = *ps.SourceTemplate.ID
	}
	if target.templateID == 0 && ps.SourceTemplateVersion != nil && ps.SourceTemplateVersion.Template != nil && ps.SourceTemplateVersion.Template.ID != nil {
		target.templateID = *ps.SourceTemplateVersion.Template.ID
	}
}

func validateTemplateDevelopCombination(in *templateDevelopArgs, target *templateDevelopTarget) error {
	switch in.PostApply {
	case "none", "rollout_target", "rollout_all", "trigger_trick":
	default:
		return fmt.Errorf("post_apply must be none, rollout_target, rollout_all, or trigger_trick")
	}
	if in.TargetType == "template" && in.PostApply != "none" {
		return fmt.Errorf("target_type template only supports post_apply=none")
	}
	if in.PostApply == "trigger_trick" && !target.jobMode {
		return fmt.Errorf("post_apply=trigger_trick requires a trick or job-mode playspec")
	}
	if (in.PostApply == "rollout_target" || in.PostApply == "rollout_all") && target.jobMode {
		return fmt.Errorf("job-mode tricks cannot be rolled out; use post_apply=trigger_trick")
	}
	if in.PostApply == "rollout_target" && target.playgroundID == nil {
		return fmt.Errorf("post_apply=rollout_target requires target_type playground")
	}
	if in.ChangeType == "switch_existing" && in.TargetTemplateVersionID <= 0 {
		return fmt.Errorf("target_template_version_id is required for switch_existing")
	}
	if in.ChangeType == "switch_existing" && target.playspecID == nil {
		return fmt.Errorf("switch_existing requires target_type playspec, playground, or trick")
	}
	if in.ChangeType == "patch" && len(in.Patches) == 0 && len(in.Edits) == 0 {
		return fmt.Errorf("patch change_type requires patches or edits")
	}
	if in.ChangeType == "overwrite" && in.TemplateBody == "" && in.TemplateBodyPath == "" {
		return fmt.Errorf("overwrite change_type requires template_body or template_body_path")
	}
	return nil
}

func runTemplateDevelopSwitch(ctx context.Context, c *fibe.Client, in *templateDevelopArgs, target *templateDevelopTarget) (any, error) {
	params := &fibe.PlayspecTemplateVersionSwitchParams{
		TargetTemplateVersionID: in.TargetTemplateVersionID,
		Variables:               in.SwitchVariables,
		RegenerateVariables:     in.RegenerateVariables,
		ConfirmWarnings:         in.ConfirmWarnings,
		RolloutMode:             rolloutModeForPostApply(in.PostApply),
		TargetPlaygroundID:      target.playgroundID,
		ResponseMode:            in.ResponseMode,
	}
	if in.Mode == "preview" {
		return c.Playspecs.PreviewTemplateVersionSwitch(ctx, *target.playspecID, params)
	}
	result, err := c.Playspecs.SwitchTemplateVersion(ctx, *target.playspecID, params)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"result": result}
	if err := runTemplateDevelopPostApply(ctx, c, in, target, out, rolloutIDsFromAny(result)); err != nil {
		return nil, err
	}
	return out, nil
}

func runTemplateDevelopPatch(ctx context.Context, c *fibe.Client, in *templateDevelopArgs, target *templateDevelopTarget) (any, error) {
	body := in.TemplateBody
	if in.ChangeType == "overwrite" && body == "" {
		read, err := readInlineOrPathTextArg(map[string]any{"template_body_path": in.TemplateBodyPath}, "template_body", "template_body_path")
		if err != nil {
			return nil, err
		}
		body = read
	}
	params := &fibe.TemplateVersionPatchParams{
		BaseVersionID:       target.baseVersion,
		TemplateBody:        body,
		Patches:             in.Patches,
		Edits:               in.Edits,
		Public:              in.Public,
		TargetPlayspecID:    target.playspecID,
		TargetPlaygroundID:  target.playgroundID,
		RolloutMode:         rolloutModeForPostApply(in.PostApply),
		SwitchVariables:     in.SwitchVariables,
		RegenerateVariables: in.RegenerateVariables,
		ConfirmWarnings:     &in.ConfirmWarnings,
		ResponseMode:        in.ResponseMode,
	}
	if in.Changelog != "" {
		params.Changelog = &in.Changelog
	}
	if in.Mode == "preview" {
		return c.ImportTemplates.PatchPreview(ctx, target.templateID, params)
	}
	autoSwitch := target.playspecID != nil
	params.AutoSwitch = &autoSwitch
	result, err := c.ImportTemplates.PatchCreate(ctx, target.templateID, params)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"result": result}
	if err := runTemplateDevelopPostApply(ctx, c, in, target, out, rolloutIDsFromPatchResult(result)); err != nil {
		return nil, err
	}
	return out, nil
}

func runTemplateDevelopPostApply(ctx context.Context, c *fibe.Client, in *templateDevelopArgs, target *templateDevelopTarget, out map[string]any, rolloutIDs []int64) error {
	if in.PostApply == "trigger_trick" {
		if target.playspecID == nil {
			return fmt.Errorf("cannot trigger trick without playspec_id")
		}
		trick, err := c.Tricks.Trigger(ctx, &fibe.TrickTriggerParams{PlayspecID: *target.playspecID, MarqueeID: target.marqueeID})
		if err != nil {
			return err
		}
		out["triggered_trick"] = trick
		if in.Wait {
			result := waitForSingleTemplatePatchRollout(ctx, c, trick.ID, time.Duration(in.WaitTimeoutSeconds)*time.Second)
			out["wait_results"] = []map[string]any{result}
			if diagnoseTemplateDevelop(in) && result["success"] != true {
				refresh := true
				debug, err := c.Playgrounds.DebugWithParams(ctx, trick.ID, &fibe.PlaygroundDebugParams{Mode: "summary", Refresh: &refresh, LogsTail: 50})
				if err != nil {
					out["diagnostics"] = map[string]any{fmt.Sprintf("%d", trick.ID): map[string]any{"error": err.Error()}}
				} else {
					out["diagnostics"] = map[string]any{fmt.Sprintf("%d", trick.ID): debug}
				}
			}
		}
		return nil
	}
	if in.Wait && len(rolloutIDs) > 0 {
		waitResults, diagnostics := waitForTemplatePatchRollouts(ctx, c, rolloutIDs, time.Duration(in.WaitTimeoutSeconds)*time.Second, diagnoseTemplateDevelop(in))
		out["wait_results"] = waitResults
		if len(diagnostics) > 0 {
			out["diagnostics"] = diagnostics
		}
	}
	return nil
}

func rolloutModeForPostApply(postApply string) string {
	switch postApply {
	case "rollout_target":
		return "target"
	case "rollout_all":
		return "all"
	default:
		return "none"
	}
}

func diagnoseTemplateDevelop(in *templateDevelopArgs) bool {
	if in.DiagnoseOnFailure == nil {
		return true
	}
	return *in.DiagnoseOnFailure
}

func boolPtrValue(v *bool) bool {
	return v != nil && *v
}

func rolloutIDsFromAny(v any) []int64 {
	var m map[string]any
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	plan, ok := m["playground_rollout_plan"].(map[string]any)
	if !ok {
		return nil
	}
	return anyInt64Slice(plan["rollout"])
}
