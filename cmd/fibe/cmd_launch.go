package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)
func launchCmd() *cobra.Command {
	var name, compose string
	var jobMode, createPlayground, noCreatePlayground bool
	var marqueeID int64
	cmd := &cobra.Command{
		Use:   "launch",
		Short: "One-shot: parse compose YAML -> create playspec -> (optionally) deploy playground",
		Long: `One-shot: parse compose YAML -> create playspec -> (optionally) deploy playground on a marquee.
Fastest path from raw Docker Compose YAML to a running environment.

PLAYGROUND CREATION RULES:
  - When --marquee-id is provided, a playground is created on that marquee by default.
  - When --marquee-id is omitted, only the playspec (and any props) are created;
    no playground is deployed and the response carries playground_id=0.
  - --job-mode (trick / CI-Job) REQUIRES --marquee-id; otherwise the trick has nowhere to run.
  - Pass --no-create-playground with a marquee id to skip playground creation explicitly.

REQUIRED FLAGS:
  --name      Playground/trick name
  --compose   Docker-compose YAML content

OPTIONAL FLAGS:
  --marquee-id            Target marquee for the playground/trick
  --job-mode              Create as a trick (job-mode) instead of a playground (requires --marquee-id)
  --no-create-playground  Create only the playspec; skip playground deployment even with --marquee-id

EXAMPLES:
  fibe launch --name my-app --compose @docker-compose.yml --marquee-id 12
  fibe launch --name ci-run --compose @docker-compose.yml --marquee-id 12 --job-mode
  fibe launch --name spec-only --compose @docker-compose.yml --no-create-playground` + generateSchemaDoc(&fibe.LaunchParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.LaunchParams{}
			if err := applyFromFile(params); err != nil { return err }
			if cmd.Flags().Changed("name") { params.Name = name }
			if cmd.Flags().Changed("compose") { params.ComposeYAML = resolveStringValue(compose) }
			if cmd.Flags().Changed("job-mode") && jobMode {
				t := true
				params.JobMode = &t
			}
			if cmd.Flags().Changed("marquee-id") && marqueeID > 0 {
				mid := marqueeID
				params.MarqueeID = &mid
			}
			if cmd.Flags().Changed("create-playground") {
				v := createPlayground
				params.CreatePlayground = &v
			}
			if cmd.Flags().Changed("no-create-playground") && noCreatePlayground {
				v := false
				params.CreatePlayground = &v
			}

			if params.ComposeYAML == "" && len(rawPayload) > 0 {
				params.ComposeYAML = string(rawPayload)
			}

			if params.Name == "" { return fmt.Errorf("required field 'name' not set") }
			if params.ComposeYAML == "" { return fmt.Errorf("required field 'compose' not set") }
			if params.JobMode != nil && *params.JobMode && params.MarqueeID == nil {
				return fmt.Errorf("--job-mode requires --marquee-id (a trick has no marquee to run on otherwise)")
			}

			result, err := c.Launch.Create(ctx(), params)
			if err != nil { return err }
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name (required)")
	cmd.Flags().StringVar(&compose, "compose", "", "Docker-compose YAML (required)")
	cmd.Flags().BoolVar(&jobMode, "job-mode", false, "Create as a trick (job-mode) instead of a playground (requires --marquee-id)")
	cmd.Flags().Int64Var(&marqueeID, "marquee-id", 0, "Target marquee ID. Required when --job-mode is set; without it only the playspec is created.")
	cmd.Flags().BoolVar(&createPlayground, "create-playground", false, "Force playground creation. Defaults to true when --marquee-id is set, false otherwise.")
	cmd.Flags().BoolVar(&noCreatePlayground, "no-create-playground", false, "Skip playground deployment even when --marquee-id is set.")
	return cmd
}

