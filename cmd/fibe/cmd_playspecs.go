package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func playspecsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "playspecs",
		Aliases: []string{"ps"},
		Short:   "Manage playspecs (service templates)",
		Long: `Manage Fibe playspecs — service composition templates.

A playspec defines the docker-compose configuration, mounted files,
registry credentials, and deployment settings for playgrounds.

SUBCOMMANDS:
  list                   List all playspecs
  get <id>               Show playspec details
  create                 Create a new playspec
  update <id>            Update playspec settings
  delete <id>            Delete a playspec
  services <id>          List playspec services
  validate-compose       Validate a docker-compose YAML`,
	}

	cmd.AddCommand(
		psListCmd(), psGetCmd(), psCreateCmd(), psUpdateCmd(),
		psDeleteCmd(), psServicesCmd(), psValidateCmd(),
	)
	if initPlayspecExtras != nil {
		initPlayspecExtras(cmd)
	}
	return cmd
}

// initPlayspecExtras is populated by cmd_playspecs_extra.go in an init()
// block so the extras ride along without touching this file's main body.
var initPlayspecExtras func(*cobra.Command)

func psListCmd() *cobra.Command {
	var query, name, sort, createdAfter, createdBefore, jobMode, locked string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all playspecs",
		Long: `List all playspecs accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name, description (substring match)
  --job-mode            Filter by job mode. Values: true, false
  --locked              Filter by locked state. Values: true, false
  --name                Filter by name (substring match)

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, name
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, NAME, LOCKED, JOB_MODE, PLAYGROUNDS, CREATED
  Use --output json for full details.

EXAMPLES:
  fibe playspecs list
  fibe ps list -q "web" --job-mode false
  fibe ps list --locked true --sort name_asc -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlayspecListParams{}
			if query != "" {
				params.Q = query
			}
			if jobMode == "true" {
				t := true
				params.JobMode = &t
			} else if jobMode == "false" {
				f := false
				params.JobMode = &f
			}
			if locked == "true" {
				t := true
				params.Locked = &t
			} else if locked == "false" {
				f := false
				params.Locked = &f
			}
			if name != "" {
				params.Name = name
			}
			if createdAfter != "" {
				params.CreatedAfter = createdAfter
			}
			if createdBefore != "" {
				params.CreatedBefore = createdBefore
			}
			if sort != "" {
				params.Sort = sort
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			specs, err := c.Playspecs.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(specs)
				return nil
			}
			headers := []string{"ID", "NAME", "LOCKED", "JOB_MODE", "PLAYGROUNDS", "CREATED"}
			rows := make([][]string, len(specs.Data))
			for i, s := range specs.Data {
				rows[i] = []string{
					fmtInt64Ptr(s.ID), s.Name, fmtBoolPtr(s.Locked),
					fmtBoolPtr(s.JobMode), fmtInt64Ptr(s.PlaygroundCount), fmtTime(s.CreatedAt),
				}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name, description")
	cmd.Flags().StringVar(&jobMode, "job-mode", "", "Filter by job mode (true/false)")
	cmd.Flags().StringVar(&locked, "locked", "", "Filter by locked state (true/false)")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func psGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show playspec details",
		Long: `Get detailed information about a playspec including services, mounted files,
and registry credentials.

EXAMPLES:
  fibe playspecs get 3
  fibe ps get 3 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			spec, err := c.Playspecs.Get(ctx(), id)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(spec)
				return nil
			}
			fmt.Printf("ID:          %s\n", fmtInt64Ptr(spec.ID))
			fmt.Printf("Name:        %s\n", spec.Name)
			fmt.Printf("Description: %s\n", fmtStr(spec.Description))
			fmt.Printf("Locked:      %s\n", fmtBoolPtr(spec.Locked))
			fmt.Printf("Job Mode:    %s\n", fmtBoolPtr(spec.JobMode))
			fmt.Printf("Playgrounds: %s\n", fmtInt64Ptr(spec.PlaygroundCount))
			return nil
		},
	}
}

func psCreateCmd() *cobra.Command {
	var name, compose, description string
	var persistVolumes, jobMode bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new playspec",
		Long: `Create a new playspec from a docker-compose YAML definition.

CORE CONCEPTS:
  - Default services are purely static abstractions of the compose YAML.
  - "Dynamic" services directly attach an external Source Code Repository (Prop_ID) to that service to hot-mount working trees.
  - Job Mode: Set job_mode=true to run playgrounds as headless tasks without long-running domains.
  - Automated Jobs: Use trigger_config to bind this playspec to trigger autonomously on GitHub pushes.

ZERO-DOWNTIME & ROUTING CONSTRAINTS:
  - When zerodowntime=true, static compose array 'ports:' are strictly forbidden (it prevents rolling coexist conflicts). You must define 'services[X].exposure_port' instead.
  - Path-based routing (path_rule) bypasses strict subdomain collision limits. Only Traefik matchers like PathPrefix() or PathRegexp() are permitted natively.
  - If a dev-server returns 403 Invalid Host via Traefik routing, you MUST ensure 'allowedHosts: true' (Webpack/Vite) is set inside the framework.

REQUIRED FLAGS:
  --name              Playspec name
  --compose           Docker-compose YAML content or @file path

OPTIONAL FLAGS:
  --description       Playspec description
  --persist-volumes   Persist Docker volumes across recreations
  --job-mode          Headless job-mode playspec (used by 'fibe tricks')

For complex 'services', 'trigger_config', 'muti_config', use --from-file.

EXAMPLES:
  fibe playspecs create --name my-spec --compose @docker-compose.yml
  fibe ps create --name api --compose @docker-compose.yml --description "API server"
  fibe ps create --name ci --compose @ci.yml --job-mode
  fibe playspecs create -f payload.json` + generateSchemaDoc(&fibe.PlayspecCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlayspecCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}

			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("compose") {
				params.BaseComposeYAML = resolveStringValue(compose)
			}
			if cmd.Flags().Changed("description") {
				params.Description = &description
			}
			if cmd.Flags().Changed("persist-volumes") {
				params.PersistVolumes = &persistVolumes
			}
			if cmd.Flags().Changed("job-mode") {
				params.JobMode = &jobMode
			}

			if params.BaseComposeYAML == "" && len(rawPayload) > 0 {
				params.BaseComposeYAML = string(rawPayload)
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}
			if params.BaseComposeYAML == "" {
				return fmt.Errorf("required field 'compose' not set")
			}

			spec, err := c.Playspecs.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(spec)
				return nil
			}
			fmt.Printf("Created playspec %s (%s)\n", fmtInt64Ptr(spec.ID), spec.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Playspec name (required)")
	cmd.Flags().StringVar(&compose, "compose", "", "Docker-compose YAML (required, use @file)")
	cmd.Flags().StringVar(&description, "description", "", "Playspec description")
	cmd.Flags().BoolVar(&persistVolumes, "persist-volumes", false, "Persist Docker volumes across recreations")
	cmd.Flags().BoolVar(&jobMode, "job-mode", false, "Headless job-mode playspec")
	return cmd
}

func psUpdateCmd() *cobra.Command {
	var name, description, baseCompose string
	var persistVolumes, jobMode bool

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update playspec settings",
		Long: `Update an existing playspec's configuration.

OPTIONAL FLAGS:
  --name              New playspec name
  --description       New description
  --base-compose      New base compose YAML (use @file to read from disk)
  --persist-volumes   Update persist-volumes setting
  --job-mode          Update job-mode setting

For complex 'services', 'trigger_config', 'muti_config', use --from-file.

EXAMPLES:
  fibe playspecs update 42 --name new-name
  fibe ps update 42 --description "Updated description" --persist-volumes
  fibe ps update 42 --base-compose @updated-compose.yml
  fibe ps update 42 -f updates.yml` + generateSchemaDoc(&fibe.PlayspecUpdateParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.PlayspecUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("description") {
				params.Description = &description
			}
			if cmd.Flags().Changed("base-compose") {
				v := resolveStringValue(baseCompose)
				params.BaseComposeYAML = &v
			}
			if cmd.Flags().Changed("persist-volumes") {
				params.PersistVolumes = &persistVolumes
			}
			if cmd.Flags().Changed("job-mode") {
				params.JobMode = &jobMode
			}
			spec, err := c.Playspecs.Update(ctx(), id, params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(spec)
				return nil
			}
			fmt.Printf("Updated playspec %s\n", fmtInt64Ptr(spec.ID))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New playspec name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVar(&baseCompose, "base-compose", "", "New base compose YAML (use @file)")
	cmd.Flags().BoolVar(&persistVolumes, "persist-volumes", false, "Update persist-volumes setting")
	cmd.Flags().BoolVar(&jobMode, "job-mode", false, "Update job-mode setting")
	return cmd
}

func psDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a playspec",
		Long: `Delete a playspec. Cannot delete if active playgrounds exist.

EXAMPLES:
  fibe playspecs delete 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Playspecs.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Playspec %d deleted\n", id)
			return nil
		},
	}
}

func psServicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "services <id>",
		Short: "List playspec services",
		Long: `List the services defined in a playspec's docker-compose configuration.

EXAMPLES:
  fibe playspecs services 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			svcs, err := c.Playspecs.Services(ctx(), id)
			if err != nil {
				return err
			}
			outputJSON(svcs)
			return nil
		},
	}
}

func psValidateCmd() *cobra.Command {
	var compose string

	cmd := &cobra.Command{
		Use:   "validate-compose",
		Short: "Validate docker-compose YAML",
		Long: `Validate a docker-compose YAML without creating a playspec.

Returns validation errors and warnings if any.

REQUIRED FLAGS:
  --compose   Docker-compose YAML content

EXAMPLES:
  fibe playspecs validate-compose --compose "version: '3'..."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			result, err := c.Playspecs.ValidateCompose(ctx(), compose)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&compose, "compose", "", "Docker-compose YAML (required)")
	cmd.MarkFlagRequired("compose")
	return cmd
}
