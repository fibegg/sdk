package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func specsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "specs",
		Aliases: []string{"spec"},
		Short:   "Manage specs (service templates)",
		Long: `Manage Fibe specs: service composition templates.

A spec defines the docker-compose configuration, mounted files,
registry credentials, and deployment settings for playgrounds.

SUBCOMMANDS:
  list                   List all specs
  get <id-or-name>       Show spec details
  create                 Create a new spec
  update <id-or-name>    Update spec settings
  delete <id-or-name>    Delete a spec
  services <id-or-name>  List spec services
  validate-compose       Validate a docker-compose YAML`,
	}

	cmd.AddCommand(
		psListCmd(), psGetCmd(), psCreateCmd(), psUpdateCmd(),
		psDeleteCmd(), psServicesCmd(), psValidateCmd(),
	)
	if initSpecExtras != nil {
		initSpecExtras(cmd)
	}
	return cmd
}

// initSpecExtras is populated by cmd_specs_extra.go in an init()
// block so the extras ride along without touching this file's main body.
var initSpecExtras func(*cobra.Command)

func psListCmd() *cobra.Command {
	var query, name, sort, createdAfter, createdBefore, jobMode, locked string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all specs",
		Long: `List all specs accessible to the authenticated user.

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
  fibe specs list
  fibe ps list -q "web" --job-mode false
  fibe ps list --locked true --sort name_asc -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.SpecListParams{}
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
			specs, err := c.Specs.List(ctx(), params)
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
		Use:   "get <id-or-name>",
		Short: "Show spec details",
		Long: `Get detailed information about a spec including services, mounted files,
and registry credentials.

EXAMPLES:
  fibe specs get 3
  fibe ps get 3 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			spec, err := c.Specs.GetByIdentifier(ctx(), args[0])
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
	var configFlags specConfigFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new spec",
		Long: `Create a new spec from a docker-compose YAML definition.

CORE CONCEPTS:
  - Default services are purely static abstractions of the compose YAML.
  - "Dynamic" services directly attach an external Source Code Repository (repository_id) to that service to hot-mount working trees.
  - Job Mode: Set job_mode=true to run playgrounds as headless tasks without long-running domains.
  - Automated Jobs: Use schedule and trigger flags to run job-mode Specs automatically.
  - Agent Prompts: Use --trigger-agent-id/--trigger-prompt-template for CI failure messages, and --muti-* flags for mutation-cure jobs.

ZERO-DOWNTIME & ROUTING CONSTRAINTS:
  - When zerodowntime=true, static compose array 'ports:' are strictly forbidden (it prevents rolling coexist conflicts). You must define 'services[X].exposure_port' instead.
  - Path-based routing (path_rule) bypasses strict subdomain collision limits. Only Traefik matchers like PathPrefix() or PathRegexp() are permitted natively.
  - If a dev-server returns 403 Invalid Host via Traefik routing, you MUST ensure 'allowedHosts: true' (Webpack/Vite) is set inside the framework.

REQUIRED FLAGS:
  --name              Spec name
  --compose           Docker-compose YAML content or @file path

OPTIONAL FLAGS:
  --description       Spec description
  --persist-volumes   Persist Docker volumes across recreations
  --job-mode          Headless job-mode spec (used by 'fibe tasks')

For complex 'services', use --from-file. Schedule, trigger, and Muti configs also accept JSON/YAML through --from-file.

EXAMPLES:
  fibe specs create --name my-spec --compose @docker-compose.yml
  fibe ps create --name api --compose @docker-compose.yml --description "API server"
  fibe ps create --name ci --compose @ci.yml --job-mode
  fibe ps create --name ci --compose @ci.yml --job-mode --trigger-enabled --trigger-repository api --trigger-host ci-runner --trigger-agent-id fixer --trigger-prompt-template @ci-prompt.txt
  fibe ps create --name muti --compose @muti.yml --job-mode --muti-enabled --muti-language ruby --muti-repository api --muti-agent-id fixer
  fibe specs create -f payload.json` + generateSchemaDoc(&fibe.SpecCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.SpecCreateParams{}
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
			applySpecCreateConfigFlags(cmd, params, configFlags)

			if params.BaseComposeYAML == "" && len(rawPayload) > 0 {
				params.BaseComposeYAML = string(rawPayload)
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}
			if params.BaseComposeYAML == "" {
				return fmt.Errorf("required field 'compose' not set")
			}

			spec, err := c.Specs.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(spec)
				return nil
			}
			fmt.Printf("Created spec %s (%s)\n", fmtInt64Ptr(spec.ID), spec.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Spec name (required)")
	cmd.Flags().StringVar(&compose, "compose", "", "Docker-compose YAML (required, use @file)")
	cmd.Flags().StringVar(&description, "description", "", "Spec description")
	cmd.Flags().BoolVar(&persistVolumes, "persist-volumes", false, "Persist Docker volumes across recreations")
	cmd.Flags().BoolVar(&jobMode, "job-mode", false, "Headless job-mode spec")
	registerSpecConfigFlags(cmd, &configFlags)
	return cmd
}

func psUpdateCmd() *cobra.Command {
	var name, description, baseCompose string
	var persistVolumes, jobMode bool
	var configFlags specConfigFlags

	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: "Update spec settings",
		Long: `Update an existing spec's configuration.

OPTIONAL FLAGS:
  --name              New spec name
  --description       New description
  --base-compose      New base compose YAML (use @file to read from disk)
  --persist-volumes   Update persist-volumes setting
  --job-mode          Update job-mode setting

For complex 'services', use --from-file. Schedule, trigger, and Muti configs also accept JSON/YAML through --from-file.

EXAMPLES:
  fibe specs update 42 --name new-name
  fibe ps update 42 --description "Updated description" --persist-volumes
  fibe ps update 42 --base-compose @updated-compose.yml
  fibe ps update 42 --trigger-agent-id fixer --trigger-prompt-template @ci-prompt.txt
  fibe ps update 42 --muti-enabled --muti-language ruby --muti-repository api --muti-agent-id fixer
  fibe ps update 42 -f updates.yml` + generateSchemaDoc(&fibe.SpecUpdateParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.SpecUpdateParams{}
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
			if err := applySpecUpdateConfigFlags(cmd, c, args[0], params, configFlags); err != nil {
				return err
			}
			spec, err := c.Specs.UpdateByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(spec)
				return nil
			}
			fmt.Printf("Updated spec %s\n", fmtInt64Ptr(spec.ID))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New spec name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVar(&baseCompose, "base-compose", "", "New base compose YAML (use @file)")
	cmd.Flags().BoolVar(&persistVolumes, "persist-volumes", false, "Update persist-volumes setting")
	cmd.Flags().BoolVar(&jobMode, "job-mode", false, "Update job-mode setting")
	registerSpecConfigFlags(cmd, &configFlags)
	return cmd
}

func psDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: "Delete a spec",
		Long: `Delete a spec. Cannot delete if active playgrounds exist.

EXAMPLES:
  fibe specs delete 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if err := c.Specs.DeleteByIdentifier(ctx(), args[0]); err != nil {
				return err
			}
			fmt.Printf("Spec %s deleted\n", args[0])
			return nil
		},
	}
}

func psServicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "services <id-or-name>",
		Short: "List spec services",
		Long: `List the services defined in a spec's docker-compose configuration.

EXAMPLES:
  fibe specs services 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			svcs, err := c.Specs.ServicesByIdentifier(ctx(), args[0])
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
		Long: `Validate a docker-compose YAML without creating a spec.

Returns validation errors and warnings if any.

REQUIRED FLAGS:
  --compose   Docker-compose YAML content

EXAMPLES:
  fibe specs validate-compose --compose "version: '3'..."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.ComposeValidateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("compose") {
				var err error
				params.ComposeYAML, err = readTextValue(compose)
				if err != nil {
					return err
				}
			}
			if params.ComposeYAML == "" {
				return fmt.Errorf("required field 'compose_yaml' not set; use --compose or --from-file")
			}
			result, err := c.Specs.ValidateComposeWithParams(ctx(), params)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&compose, "compose", "", "Docker-compose YAML or @path (or compose_yaml in --from-file)")
	return cmd
}
