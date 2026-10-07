package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

type specConfigFlags struct {
	scheduleEnabled bool
	scheduleCron    string
	scheduleHostID  string

	triggerEnabled        bool
	triggerEventType      string
	triggerBranch         string
	triggerRepositoryID   string
	triggerHostID         string
	triggerAgentID        string
	triggerMaxRetries     int
	triggerPromptTemplate string

	mutiEnabled        bool
	mutiLanguage       string
	mutiRepositoryID   string
	mutiAgentID        string
	mutiPromptTemplate string
}

func registerSpecConfigFlags(cmd *cobra.Command, flags *specConfigFlags) {
	cmd.Flags().BoolVar(&flags.scheduleEnabled, "schedule-enabled", false, "Enable scheduled job runs")
	cmd.Flags().StringVar(&flags.scheduleCron, "schedule-cron", "", "Cron or Fugit schedule expression")
	cmd.Flags().StringVar(&flags.scheduleHostID, "schedule-host", "", "Target Host ID or name for scheduled runs")

	cmd.Flags().BoolVar(&flags.triggerEnabled, "trigger-enabled", false, "Enable CI trigger runs")
	cmd.Flags().StringVar(&flags.triggerEventType, "trigger-event-type", "", "CI trigger event type: push or pull_request")
	cmd.Flags().StringVar(&flags.triggerBranch, "trigger-branch", "", "CI trigger branch filter")
	cmd.Flags().StringVar(&flags.triggerRepositoryID, "trigger-repository", "", "Repository ID or name whose git events trigger the Spec")
	cmd.Flags().StringVar(&flags.triggerHostID, "trigger-host", "", "Target Host ID or name for CI trigger runs")
	cmd.Flags().StringVar(&flags.triggerAgentID, "trigger-agent-id", "", "Agent ID or name to notify when a CI trigger job fails")
	cmd.Flags().IntVar(&flags.triggerMaxRetries, "trigger-max-retries", 0, "Maximum CI trigger reruns after failure")
	cmd.Flags().StringVar(&flags.triggerPromptTemplate, "trigger-prompt-template", "", "Failure prompt template text or @file path; supports {{logs}}")

	cmd.Flags().BoolVar(&flags.mutiEnabled, "muti-enabled", false, "Enable Muti mutation-cure jobs")
	cmd.Flags().StringVar(&flags.mutiLanguage, "muti-language", "", "Muti mutation language")
	cmd.Flags().StringVar(&flags.mutiRepositoryID, "muti-repository", "", "Repository ID or name whose surviving mutations should be cured")
	cmd.Flags().StringVar(&flags.mutiAgentID, "muti-agent-id", "", "Agent ID or name to notify for surviving mutations")
	cmd.Flags().StringVar(&flags.mutiPromptTemplate, "muti-prompt-template", "", "Muti prompt template text or @file path; supports {{diff}} and mutation metadata placeholders")
}

func applySpecCreateConfigFlags(cmd *cobra.Command, params *fibe.SpecCreateParams, flags specConfigFlags) {
	if specScheduleFlagsChanged(cmd) {
		params.ScheduleConfig = applySpecScheduleFlags(params.ScheduleConfig, cmd, flags)
	}
	if specTriggerFlagsChanged(cmd) {
		params.TriggerConfig = applySpecTriggerFlags(params.TriggerConfig, cmd, flags)
	}
	if specMutiFlagsChanged(cmd) {
		params.MutiConfig = applySpecMutiFlags(params.MutiConfig, cmd, flags)
	}
}

func applySpecUpdateConfigFlags(cmd *cobra.Command, client *fibe.Client, identifier string, params *fibe.SpecUpdateParams, flags specConfigFlags) error {
	var existing *fibe.Spec
	if specUpdateNeedsExistingConfig(cmd, params) {
		var err error
		existing, err = client.Specs.GetByIdentifier(ctx(), identifier)
		if err != nil {
			return fmt.Errorf("load existing spec config: %w", err)
		}
	}

	if specScheduleFlagsChanged(cmd) {
		base := params.ScheduleConfig
		if base == nil && existing != nil {
			base = existing.ScheduleConfig
		}
		params.ScheduleConfig = applySpecScheduleFlags(base, cmd, flags)
	}
	if specTriggerFlagsChanged(cmd) {
		base := params.TriggerConfig
		if base == nil && existing != nil {
			base = existing.TriggerConfig
		}
		params.TriggerConfig = applySpecTriggerFlags(base, cmd, flags)
	}
	if specMutiFlagsChanged(cmd) {
		base := params.MutiConfig
		if base == nil && existing != nil {
			base = existing.MutiConfig
		}
		params.MutiConfig = applySpecMutiFlags(base, cmd, flags)
	}
	return nil
}

func specUpdateNeedsExistingConfig(cmd *cobra.Command, params *fibe.SpecUpdateParams) bool {
	return (specScheduleFlagsChanged(cmd) && params.ScheduleConfig == nil) ||
		(specTriggerFlagsChanged(cmd) && params.TriggerConfig == nil) ||
		(specMutiFlagsChanged(cmd) && params.MutiConfig == nil)
}

func applySpecScheduleFlags(base map[string]any, cmd *cobra.Command, flags specConfigFlags) map[string]any {
	config := cloneStringAnyMap(base)
	if cmd.Flags().Changed("schedule-enabled") {
		config["enabled"] = flags.scheduleEnabled
	}
	if cmd.Flags().Changed("schedule-cron") {
		config["cron"] = flags.scheduleCron
	}
	if cmd.Flags().Changed("schedule-host") {
		config["host_id"] = flags.scheduleHostID
	}
	return config
}

func applySpecTriggerFlags(base map[string]any, cmd *cobra.Command, flags specConfigFlags) map[string]any {
	config := cloneStringAnyMap(base)
	if cmd.Flags().Changed("trigger-enabled") {
		config["enabled"] = flags.triggerEnabled
	}
	if cmd.Flags().Changed("trigger-event-type") {
		config["event_type"] = flags.triggerEventType
	}
	if cmd.Flags().Changed("trigger-branch") {
		config["branch"] = flags.triggerBranch
	}
	if cmd.Flags().Changed("trigger-repository") {
		config["repository_id"] = flags.triggerRepositoryID
	}
	if cmd.Flags().Changed("trigger-host") {
		config["host_id"] = flags.triggerHostID
	}
	if cmd.Flags().Changed("trigger-agent-id") {
		config["agent_id"] = flags.triggerAgentID
	}
	if cmd.Flags().Changed("trigger-max-retries") {
		config["max_retries"] = flags.triggerMaxRetries
	}
	if cmd.Flags().Changed("trigger-prompt-template") {
		config["prompt_template"] = resolveStringValue(flags.triggerPromptTemplate)
	}
	return config
}

func applySpecMutiFlags(base map[string]any, cmd *cobra.Command, flags specConfigFlags) map[string]any {
	config := cloneStringAnyMap(base)
	if cmd.Flags().Changed("muti-enabled") {
		config["enabled"] = flags.mutiEnabled
	}
	if cmd.Flags().Changed("muti-language") {
		config["language"] = flags.mutiLanguage
	}
	if cmd.Flags().Changed("muti-repository") {
		config["repository_id"] = flags.mutiRepositoryID
	}
	if cmd.Flags().Changed("muti-agent-id") {
		config["agent_id"] = flags.mutiAgentID
	}
	if cmd.Flags().Changed("muti-prompt-template") {
		config["prompt_template"] = resolveStringValue(flags.mutiPromptTemplate)
	}
	return config
}

func specScheduleFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("schedule-enabled") ||
		cmd.Flags().Changed("schedule-cron") ||
		cmd.Flags().Changed("schedule-host")
}

func specTriggerFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("trigger-enabled") ||
		cmd.Flags().Changed("trigger-event-type") ||
		cmd.Flags().Changed("trigger-branch") ||
		cmd.Flags().Changed("trigger-repository") ||
		cmd.Flags().Changed("trigger-host") ||
		cmd.Flags().Changed("trigger-agent-id") ||
		cmd.Flags().Changed("trigger-max-retries") ||
		cmd.Flags().Changed("trigger-prompt-template")
}

func specMutiFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("muti-enabled") ||
		cmd.Flags().Changed("muti-language") ||
		cmd.Flags().Changed("muti-repository") ||
		cmd.Flags().Changed("muti-agent-id") ||
		cmd.Flags().Changed("muti-prompt-template")
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
