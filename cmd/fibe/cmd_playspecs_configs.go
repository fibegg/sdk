package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

type playspecConfigFlags struct {
	scheduleEnabled   bool
	scheduleCron      string
	scheduleMarqueeID string

	triggerEnabled        bool
	triggerEventType      string
	triggerBranch         string
	triggerPropID         string
	triggerMarqueeID      string
	triggerAgentID        string
	triggerMaxRetries     int
	triggerPromptTemplate string

	mutiEnabled        bool
	mutiLanguage       string
	mutiPropID         string
	mutiAgentID        string
	mutiPromptTemplate string
}

func registerPlayspecConfigFlags(cmd *cobra.Command, flags *playspecConfigFlags) {
	cmd.Flags().BoolVar(&flags.scheduleEnabled, "schedule-enabled", false, "Enable scheduled job runs")
	cmd.Flags().StringVar(&flags.scheduleCron, "schedule-cron", "", "Cron or Fugit schedule expression")
	cmd.Flags().StringVar(&flags.scheduleMarqueeID, "schedule-marquee-id", "", "Target Marquee ID or name for scheduled runs")

	cmd.Flags().BoolVar(&flags.triggerEnabled, "trigger-enabled", false, "Enable CI trigger runs")
	cmd.Flags().StringVar(&flags.triggerEventType, "trigger-event-type", "", "CI trigger event type: push or pull_request")
	cmd.Flags().StringVar(&flags.triggerBranch, "trigger-branch", "", "CI trigger branch filter")
	cmd.Flags().StringVar(&flags.triggerPropID, "trigger-prop-id", "", "Prop ID or name whose git events trigger the Playspec")
	cmd.Flags().StringVar(&flags.triggerMarqueeID, "trigger-marquee-id", "", "Target Marquee ID or name for CI trigger runs")
	cmd.Flags().StringVar(&flags.triggerAgentID, "trigger-agent-id", "", "Agent ID or name to notify when a CI trigger job fails")
	cmd.Flags().IntVar(&flags.triggerMaxRetries, "trigger-max-retries", 0, "Maximum CI trigger reruns after failure")
	cmd.Flags().StringVar(&flags.triggerPromptTemplate, "trigger-prompt-template", "", "Failure prompt template text or @file path; supports {{logs}}")

	cmd.Flags().BoolVar(&flags.mutiEnabled, "muti-enabled", false, "Enable Muti mutation-cure jobs")
	cmd.Flags().StringVar(&flags.mutiLanguage, "muti-language", "", "Muti mutation language")
	cmd.Flags().StringVar(&flags.mutiPropID, "muti-prop-id", "", "Prop ID or name whose surviving mutations should be cured")
	cmd.Flags().StringVar(&flags.mutiAgentID, "muti-agent-id", "", "Agent ID or name to notify for surviving mutations")
	cmd.Flags().StringVar(&flags.mutiPromptTemplate, "muti-prompt-template", "", "Muti prompt template text or @file path; supports {{diff}} and mutation metadata placeholders")
}

func applyPlayspecCreateConfigFlags(cmd *cobra.Command, params *fibe.PlayspecCreateParams, flags playspecConfigFlags) {
	if playspecScheduleFlagsChanged(cmd) {
		params.ScheduleConfig = applyPlayspecScheduleFlags(params.ScheduleConfig, cmd, flags)
	}
	if playspecTriggerFlagsChanged(cmd) {
		params.TriggerConfig = applyPlayspecTriggerFlags(params.TriggerConfig, cmd, flags)
	}
	if playspecMutiFlagsChanged(cmd) {
		params.MutiConfig = applyPlayspecMutiFlags(params.MutiConfig, cmd, flags)
	}
}

func applyPlayspecUpdateConfigFlags(cmd *cobra.Command, client *fibe.Client, identifier string, params *fibe.PlayspecUpdateParams, flags playspecConfigFlags) error {
	var existing *fibe.Playspec
	if playspecUpdateNeedsExistingConfig(cmd, params) {
		var err error
		existing, err = client.Playspecs.GetByIdentifier(ctx(), identifier)
		if err != nil {
			return fmt.Errorf("load existing playspec config: %w", err)
		}
	}

	if playspecScheduleFlagsChanged(cmd) {
		base := params.ScheduleConfig
		if base == nil && existing != nil {
			base = existing.ScheduleConfig
		}
		params.ScheduleConfig = applyPlayspecScheduleFlags(base, cmd, flags)
	}
	if playspecTriggerFlagsChanged(cmd) {
		base := params.TriggerConfig
		if base == nil && existing != nil {
			base = existing.TriggerConfig
		}
		params.TriggerConfig = applyPlayspecTriggerFlags(base, cmd, flags)
	}
	if playspecMutiFlagsChanged(cmd) {
		base := params.MutiConfig
		if base == nil && existing != nil {
			base = existing.MutiConfig
		}
		params.MutiConfig = applyPlayspecMutiFlags(base, cmd, flags)
	}
	return nil
}

func playspecUpdateNeedsExistingConfig(cmd *cobra.Command, params *fibe.PlayspecUpdateParams) bool {
	return (playspecScheduleFlagsChanged(cmd) && params.ScheduleConfig == nil) ||
		(playspecTriggerFlagsChanged(cmd) && params.TriggerConfig == nil) ||
		(playspecMutiFlagsChanged(cmd) && params.MutiConfig == nil)
}

func applyPlayspecScheduleFlags(base map[string]any, cmd *cobra.Command, flags playspecConfigFlags) map[string]any {
	config := cloneStringAnyMap(base)
	if cmd.Flags().Changed("schedule-enabled") {
		config["enabled"] = flags.scheduleEnabled
	}
	if cmd.Flags().Changed("schedule-cron") {
		config["cron"] = flags.scheduleCron
	}
	if cmd.Flags().Changed("schedule-marquee-id") {
		config["marquee_id"] = flags.scheduleMarqueeID
	}
	return config
}

func applyPlayspecTriggerFlags(base map[string]any, cmd *cobra.Command, flags playspecConfigFlags) map[string]any {
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
	if cmd.Flags().Changed("trigger-prop-id") {
		config["prop_id"] = flags.triggerPropID
	}
	if cmd.Flags().Changed("trigger-marquee-id") {
		config["marquee_id"] = flags.triggerMarqueeID
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

func applyPlayspecMutiFlags(base map[string]any, cmd *cobra.Command, flags playspecConfigFlags) map[string]any {
	config := cloneStringAnyMap(base)
	if cmd.Flags().Changed("muti-enabled") {
		config["enabled"] = flags.mutiEnabled
	}
	if cmd.Flags().Changed("muti-language") {
		config["language"] = flags.mutiLanguage
	}
	if cmd.Flags().Changed("muti-prop-id") {
		config["prop_id"] = flags.mutiPropID
	}
	if cmd.Flags().Changed("muti-agent-id") {
		config["agent_id"] = flags.mutiAgentID
	}
	if cmd.Flags().Changed("muti-prompt-template") {
		config["prompt_template"] = resolveStringValue(flags.mutiPromptTemplate)
	}
	return config
}

func playspecScheduleFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("schedule-enabled") ||
		cmd.Flags().Changed("schedule-cron") ||
		cmd.Flags().Changed("schedule-marquee-id")
}

func playspecTriggerFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("trigger-enabled") ||
		cmd.Flags().Changed("trigger-event-type") ||
		cmd.Flags().Changed("trigger-branch") ||
		cmd.Flags().Changed("trigger-prop-id") ||
		cmd.Flags().Changed("trigger-marquee-id") ||
		cmd.Flags().Changed("trigger-agent-id") ||
		cmd.Flags().Changed("trigger-max-retries") ||
		cmd.Flags().Changed("trigger-prompt-template")
}

func playspecMutiFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("muti-enabled") ||
		cmd.Flags().Changed("muti-language") ||
		cmd.Flags().Changed("muti-prop-id") ||
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
