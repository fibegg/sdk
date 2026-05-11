package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func templatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "templates",
		Aliases: []string{"tpl"},
		Short:   "Manage import templates (Pantry)",
		Long: `Manage Fibe import templates — reusable playspec configurations.

Templates can be published publicly or kept private. Each template
can have multiple versions.

SUBCOMMANDS:
  list                                       List all templates
  get <id-or-name>                           Show template with versions
  create                                     Create a new template
  update <id-or-name>                        Update template metadata
  delete <id-or-name>                        Delete a template
  search                                     Search templates
  versions <id-or-name>                      List template versions
  versions list <id-or-name>                 List template versions
  versions create <id-or-name>               Create a new version
  versions destroy <id-or-name> <ver-id>     Delete a version
  versions toggle-public <id-or-name> <ver-id> Toggle version public visibility
  source set <id-or-name>                    Track a Prop file as template source
  source refresh <id-or-name>                Refresh tracked source now
  source clear <id-or-name>                  Clear tracked source
  upgrade-playspecs <id-or-name>             Upgrade linked job Playspecs to a version
  fork <id-or-name>                          Fork a template into your account
  upload-image <id-or-name>                  Upload a cover image
  launch <id-or-name>                        Launch a playground from template`,
	}
	cmd.AddCommand(
		tplListCmd(), tplGetCmd(), tplCreateCmd(), tplUpdateCmd(), tplDeleteCmd(),
		tplSearchCmd(), tplVersionsCmd(),
		tplCreateVersionCmd(), tplDestroyVersionCmd(), tplTogglePublicCmd(),
		tplSourceCmd(), tplUpgradePlayspecsCmd(),
		tplForkCmd(), tplUploadImageCmd(), tplLaunchCmd(),
	)
	return cmd
}

func tplListCmd() *cobra.Command {
	var query, name, sort string
	var categoryID int64
	var system string
	cmd := &cobra.Command{
		Use: "list", Short: "List all templates",
		Long: `List all import templates accessible to the authenticated user.

FILTERS:
  -q, --query           Full-text search across name, description (PostgreSQL FTS)
  --name                Filter by name (substring match)
  --category-id         Filter by category ID
  --system              Filter system templates. Values: true, false

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: updated_at, name, created_at
                        Direction: asc, desc
                        Default: updated_at_desc

OUTPUT:
  Columns: ID, NAME, AUTHOR, CATEGORY
  Use --output json for full details.

EXAMPLES:
  fibe templates list
  fibe tpl list -q "node"
  fibe tpl list --category-id 3 --sort name_asc
  fibe tpl list --system true -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.ImportTemplateListParams{}
			if query != "" {
				params.Q = query
			}
			if name != "" {
				params.Name = name
			}
			if categoryID > 0 {
				params.CategoryID = categoryID
			}
			if system == "true" {
				t := true
				params.System = &t
			} else if system == "false" {
				f := false
				params.System = &f
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
			tpls, err := c.ImportTemplates.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tpls)
				return nil
			}
			headers := []string{"ID", "NAME", "AUTHOR", "CATEGORY"}
			rows := make([][]string, len(tpls.Data))
			for i, t := range tpls.Data {
				rows[i] = []string{fmtInt64Ptr(t.ID), t.Name, fmtStr(t.Author), fmtStr(t.Category)}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Full-text search across name, description")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().Int64Var(&categoryID, "category-id", 0, "Filter by category ID")
	cmd.Flags().StringVar(&system, "system", "", "Filter system templates (true/false)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. updated_at_desc)")
	return cmd
}

func tplGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <id-or-name>", Short: "Show template details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			tpl, err := c.ImportTemplates.GetByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			outputJSON(tpl)
			return nil
		},
	}
}

func tplCreateCmd() *cobra.Command {
	var name, desc, body string
	var catID int64
	cmd := &cobra.Command{
		Use: "create", Short: "Create a new template",
		Long: "Create a new import template.\n\nTEMPLATE CONSTRAINTS:\n  - Templates are heavily opinionated YAML mappings used to generate generalized Playspecs.\n  - The 'body' must be a valid predefined YAML template matching Fibe's Template Schema engine.\n\nREQUIRED FLAGS:\n  --name          Template name\n  --body          Template body (YAML)\n\nOPTIONAL FLAGS:\n  --category-id   Category ID (defaults to \"Uncategorized\" on the server when omitted)\n  --description   Free-form description\n\nEXAMPLES:\n  fibe templates create --name \"Node.js\" --body @template.yml\n  fibe templates create --name \"Node.js\" --category-id 1 --body @template.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.ImportTemplateCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("description") {
				params.Description = desc
			}
			if cmd.Flags().Changed("category-id") {
				params.CategoryID = catID
			}
			if cmd.Flags().Changed("body") {
				params.TemplateBody = body
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}
			if params.TemplateBody == "" {
				return fmt.Errorf("required field 'body' not set")
			}

			tpl, err := c.ImportTemplates.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tpl)
				return nil
			}
			fmt.Printf("Created template %s (%s)\n", fmtInt64Ptr(tpl.ID), tpl.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name (required)")
	cmd.Flags().StringVar(&desc, "description", "", "Description")
	cmd.Flags().Int64Var(&catID, "category-id", 0, "Category ID (optional, defaults to Uncategorized)")
	cmd.Flags().StringVar(&body, "body", "", "Template body YAML (required)")
	return cmd
}

func tplUpdateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use: "update <id-or-name>", Short: "Update template metadata", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			identifier := args[0]
			params := &fibe.ImportTemplateUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			tpl, err := c.ImportTemplates.UpdateByIdentifier(ctx(), identifier, params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tpl)
				return nil
			}
			fmt.Printf("Updated template %s\n", identifier)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	return cmd
}

func tplDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id-or-name>", Short: "Delete a template", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			identifier := args[0]
			if err := c.ImportTemplates.DeleteByIdentifier(ctx(), identifier); err != nil {
				return err
			}
			fmt.Printf("Template %s deleted\n", identifier)
			return nil
		},
	}
}

func tplSearchCmd() *cobra.Command {
	var query string
	var regex bool
	cmd := &cobra.Command{
		Use: "search", Short: "Search templates",
		Long: `Search templates by name or description.

Use --regex to treat --query as a PostgreSQL regex. Regex search requires
at least one literal token with 3+ characters so the server can prefilter
with indexed text search before applying regex.

REQUIRED FLAGS:
  --query   Search query

EXAMPLES:
  fibe templates search --query node
  fibe templates search --query 'starter-.*' --regex`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			results, err := c.ImportTemplates.SearchWithParams(ctx(), &fibe.ImportTemplateSearchParams{Query: query, Regex: regex})
			if err != nil {
				return err
			}
			outputJSON(results)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query (required)")
	cmd.Flags().BoolVar(&regex, "regex", false, "Treat query as PostgreSQL regex; requires a 3+ character literal token")
	cmd.MarkFlagRequired("query")
	return cmd
}

func tplVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions [list|create|destroy|toggle-public] <id-or-name> ...",
		Short: "Manage template versions (list, create, destroy, toggle-public)",
		Long: `Manage versions of an import template.

Without a subcommand, lists versions for the given template ID or name
(equivalent to "versions list <id-or-name>") for backward compatibility.

SUBCOMMANDS:
  list <id-or-name>                  List versions of a template
  create <id-or-name>                Create a new version (--body required)
  destroy <id-or-name> <ver-id>      Delete a version
  toggle-public <id-or-name> <ver-id> Flip a version's public visibility

EXAMPLES:
  fibe templates versions rails-starter
  fibe templates versions list rails-starter
  fibe templates versions create rails-starter --body @template.yaml --public
  fibe templates versions destroy rails-starter 12
  fibe templates versions toggle-public rails-starter 12`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			c := newClient()
			versions, err := c.ImportTemplates.ListVersionsByIdentifier(ctx(), args[0], nil)
			if err != nil {
				return err
			}
			outputJSON(versions)
			return nil
		},
	}
	cmd.AddCommand(
		tplVersionsListCmd(),
		tplVersionsCreateCmd(),
		tplVersionsDestroyCmd(),
		tplVersionsTogglePublicCmd(),
	)
	return cmd
}

func tplVersionsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <template-id-or-name>",
		Short: "List template versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			versions, err := c.ImportTemplates.ListVersionsByIdentifier(ctx(), args[0], nil)
			if err != nil {
				return err
			}
			outputJSON(versions)
			return nil
		},
	}
}

func tplVersionsCreateCmd() *cobra.Command {
	var body string
	var changelog string
	var public bool
	cmd := &cobra.Command{
		Use:   "create <template-id-or-name>",
		Short: "Create a new version for an import template",
		Long: `Create a new version of an existing template.

REQUIRED FLAGS:
  --body    Template YAML body (use @path/to/file to read from disk)

OPTIONAL FLAGS:
  --public  Mark version as publicly visible (default: false)

EXAMPLES:
  fibe templates versions create rails-starter --body @template.yaml
  fibe templates versions create rails-starter --body @template.yaml --public`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.ImportTemplateVersionCreateParams{
				TemplateBody: resolveStringValue(body),
			}
			if cmd.Flags().Changed("public") {
				params.Public = &public
			}
			if cmd.Flags().Changed("changelog") {
				params.Changelog = &changelog
			}
			if params.TemplateBody == "" {
				return fmt.Errorf("required field 'body' not set")
			}
			version, err := c.ImportTemplates.CreateVersionByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			outputJSON(version)
			return nil
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "Template YAML body (required, use @path to read file)")
	cmd.Flags().BoolVar(&public, "public", false, "Mark version as public")
	cmd.Flags().StringVar(&changelog, "changelog", "", "Optional changelog for this version")
	return cmd
}

func tplSourceCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "source", Short: "Manage tracked template source"}
	cmd.AddCommand(tplSourceSetCmd(), tplSourceRefreshCmd(), tplSourceClearCmd())
	return cmd
}

func tplSourceSetCmd() *cobra.Command {
	var propID, ciMarqueeID, marqueeID string
	var path, ref string
	var autoRefresh, autoUpgrade, ciEnabled bool
	cmd := &cobra.Command{
		Use:   "set <template-id-or-name>",
		Short: "Track a YAML file from a Prop",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateIdentifier := args[0]
			if propID == "" {
				return fmt.Errorf("required field 'prop-id' not set")
			}
			if path == "" {
				return fmt.Errorf("required field 'path' not set")
			}
			params := &fibe.ImportTemplateSourceParams{
				SourcePropIdentifier: propID,
				SourcePath:           path,
				SourceRef:            ref,
			}
			if cmd.Flags().Changed("auto-refresh") {
				params.SourceAutoRefresh = &autoRefresh
			}
			if cmd.Flags().Changed("auto-upgrade") {
				params.SourceAutoUpgrade = &autoUpgrade
			}
			if cmd.Flags().Changed("ci-enabled") || cmd.Flags().Changed("ci") {
				params.CIEnabled = &ciEnabled
			}
			if cmd.Flags().Changed("ci-marquee-id") {
				params.CIMarqueeIdentifier = ciMarqueeID
			} else if cmd.Flags().Changed("marquee-id") {
				params.CIMarqueeIdentifier = marqueeID
			}
			result, err := newClient().ImportTemplates.SetSourceByIdentifier(ctx(), templateIdentifier, params)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&propID, "prop-id", "", "Source Prop ID or name (required)")
	cmd.Flags().StringVar(&path, "path", "", "Source YAML path, e.g. fibe-ci.yml")
	cmd.Flags().StringVar(&ref, "ref", "", "Source ref/branch")
	cmd.Flags().BoolVar(&autoRefresh, "auto-refresh", true, "Refresh versions from matching pushes")
	cmd.Flags().BoolVar(&autoUpgrade, "auto-upgrade", true, "Auto-upgrade linked job Playspecs")
	cmd.Flags().BoolVar(&ciEnabled, "ci-enabled", false, "Enable CI workflow sync for this template source")
	cmd.Flags().BoolVar(&ciEnabled, "ci", false, "Alias for --ci-enabled")
	cmd.Flags().StringVar(&ciMarqueeID, "ci-marquee-id", "", "Marquee ID or name used by CI workflow sync")
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Alias for --ci-marquee-id")
	return cmd
}

func tplSourceRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh <template-id-or-name>",
		Short: "Refresh tracked template source now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := newClient().ImportTemplates.RefreshSourceByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
}

func tplSourceClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear <template-id-or-name>",
		Short: "Clear tracked template source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := newClient().ImportTemplates.ClearSourceByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
}

func tplUpgradePlayspecsCmd() *cobra.Command {
	var versionID int64
	cmd := &cobra.Command{
		Use:   "upgrade-playspecs <template-id-or-name>",
		Short: "Upgrade linked job Playspecs to a template version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if versionID <= 0 {
				return fmt.Errorf("required field 'target-version-id' not set")
			}
			result, err := newClient().ImportTemplates.UpgradeLinkedPlayspecsByIdentifier(ctx(), args[0], versionID)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().Int64Var(&versionID, "target-version-id", 0, "Target template version ID")
	return cmd
}

func tplVersionsDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy <template-id-or-name> <version-id>",
		Short: "Delete a template version",
		Long: `Delete a specific version of a template.

If this is the last version, the template itself will also be deleted.

EXAMPLES:
  fibe templates versions destroy rails-starter 12`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			versionID, _ := strconv.ParseInt(args[1], 10, 64)
			if err := c.ImportTemplateVersions.Delete(ctx(), versionID); err != nil {
				return err
			}
			fmt.Printf("Version %d deleted from template %s\n", versionID, args[0])
			return nil
		},
	}
}

func tplVersionsTogglePublicCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "toggle-public <template-id-or-name> <version-id>",
		Short: "Toggle a version's public visibility",
		Long: `Toggle a template version between public and private.

EXAMPLES:
  fibe templates versions toggle-public rails-starter 12`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			versionID, _ := strconv.ParseInt(args[1], 10, 64)
			version, err := c.ImportTemplates.TogglePublicByIdentifier(ctx(), args[0], versionID)
			if err != nil {
				return err
			}
			outputJSON(version)
			return nil
		},
	}
}

func tplLaunchCmd() *cobra.Command {
	var marqueeID string
	var name string
	var version int64
	cmd := &cobra.Command{
		Use: "launch <id-or-name>", Short: "Launch a playground from template", Args: cobra.ExactArgs(1),
		Long: "Create a new playground using this template.\n\nREQUIRED FLAGS:\n  --marquee-id   Target marquee for the new playground\n\nOPTIONAL FLAGS:\n  --name         Override generated playground name\n  --version      Launch a specific template version\n\nEXAMPLES:\n  fibe templates launch rails-starter --marquee-id next\n  fibe templates launch rails-starter --marquee-id next --name my-playground --version 3",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.ImportTemplateLaunchParams{MarqueeIdentifier: marqueeID}
			if name != "" {
				params.Name = name
			}
			if version > 0 {
				params.Version = &version
			}
			result, err := c.ImportTemplates.LaunchWithParamsByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Target marquee ID or name (required)")
	cmd.Flags().StringVar(&name, "name", "", "Optional playground name")
	cmd.Flags().Int64Var(&version, "version", 0, "Optional template version")
	cmd.MarkFlagRequired("marquee-id")
	return cmd
}

func tplCreateVersionCmd() *cobra.Command {
	cmd := tplVersionsCreateCmd()
	cmd.Use = "create-version <template-id-or-name>"
	cmd.Short = "Create a new version (alias of \"versions create\")"
	cmd.Hidden = true
	return cmd
}

func tplDestroyVersionCmd() *cobra.Command {
	cmd := tplVersionsDestroyCmd()
	cmd.Use = "destroy-version <template-id-or-name> <version-id>"
	cmd.Short = "Delete a template version (alias of \"versions destroy\")"
	cmd.Hidden = true
	return cmd
}

func tplTogglePublicCmd() *cobra.Command {
	cmd := tplVersionsTogglePublicCmd()
	cmd.Use = "toggle-public <template-id-or-name> <version-id>"
	cmd.Short = "Toggle a version's public visibility (alias of \"versions toggle-public\")"
	cmd.Hidden = true
	return cmd
}

func tplForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <id-or-name>",
		Short: "Fork a template into your account",
		Long: `Create a personal copy (fork) of an existing template.

The forked template is owned by you and can be edited independently
of the original.

EXAMPLES:
  fibe templates fork rails-starter`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			tpl, err := c.ImportTemplates.ForkByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			outputJSON(tpl)
			return nil
		},
	}
}

func tplUploadImageCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "upload-image <id-or-name>",
		Short: "Upload a cover image for a template",
		Long: `Upload a cover image for an import template.

The file is read from disk, base64-encoded, and sent to the server along
with the filename and detected content type.

REQUIRED FLAGS:
  --file    Path to image file (PNG, JPEG, GIF, etc.)

EXAMPLES:
  fibe templates upload-image rails-starter --file logo.png`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if file == "" {
				return fmt.Errorf("required field 'file' not set")
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read image file: %w", err)
			}
			contentType := http.DetectContentType(data)
			params := &fibe.UploadImageParams{
				Filename:    filepath.Base(file),
				ImageData:   base64.StdEncoding.EncodeToString(data),
				ContentType: contentType,
			}
			tpl, err := c.ImportTemplates.UploadImageByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			outputJSON(tpl)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "Path to image file (required)")
	return cmd
}

// =============================================================================
// =============================================================================
