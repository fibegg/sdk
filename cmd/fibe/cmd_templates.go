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
  get <id>                                   Show template with versions
  create                                     Create a new template
  update <id>                                Update template metadata
  delete <id>                                Delete a template
  search                                     Search templates
  versions <id>                              List template versions (alias of "versions list <id>")
  versions list <id>                         List template versions
  versions create <id>                       Create a new version
  versions destroy <id> <ver-id>             Delete a version
  versions toggle-public <id> <ver-id>       Toggle version public visibility
  fork <id>                                  Fork a template into your account
  upload-image <id>                          Upload a cover image
  launch <id>                                Launch a playground from template`,
	}
	cmd.AddCommand(
		tplListCmd(), tplGetCmd(), tplCreateCmd(), tplUpdateCmd(), tplDeleteCmd(),
		tplSearchCmd(), tplVersionsCmd(),
		tplCreateVersionCmd(), tplDestroyVersionCmd(), tplTogglePublicCmd(),
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
			if query != "" { params.Q = query }
			if name != "" { params.Name = name }
			if categoryID > 0 { params.CategoryID = categoryID }
			if system == "true" { t := true; params.System = &t } else if system == "false" { f := false; params.System = &f }
			if sort != "" { params.Sort = sort }
			if flagPage > 0 { params.Page = flagPage }
			if flagPerPage > 0 { params.PerPage = flagPerPage }
			tpls, err := c.ImportTemplates.List(ctx(), params)
			if err != nil { return err }
			if effectiveOutput() != "table" { outputJSON(tpls); return nil }
			headers := []string{"ID", "NAME", "AUTHOR", "CATEGORY"}
			rows := make([][]string, len(tpls.Data))
			for i, t := range tpls.Data { rows[i] = []string{fmtInt64Ptr(t.ID), t.Name, fmtStr(t.Author), fmtStr(t.Category)} }
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
		Use: "get <id>", Short: "Show template details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			tpl, err := c.ImportTemplates.Get(ctx(), id)
			if err != nil { return err }
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
			if err := applyFromFile(params); err != nil { return err }
			if cmd.Flags().Changed("name") { params.Name = name }
			if cmd.Flags().Changed("description") { params.Description = desc }
			if cmd.Flags().Changed("category-id") { params.CategoryID = catID }
			if cmd.Flags().Changed("body") { params.TemplateBody = body }

			if params.Name == "" { return fmt.Errorf("required field 'name' not set") }
			if params.TemplateBody == "" { return fmt.Errorf("required field 'body' not set") }
			
			tpl, err := c.ImportTemplates.Create(ctx(), params)
			if err != nil { return err }
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
		Use: "update <id>", Short: "Update template metadata", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.ImportTemplateUpdateParams{}
			if err := applyFromFile(params); err != nil { return err }
			if cmd.Flags().Changed("name") { params.Name = &name }
			_, err := c.ImportTemplates.Update(ctx(), id, params)
			if err != nil { return err }
			fmt.Printf("Updated template %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	return cmd
}

func tplDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a template", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			must(c.ImportTemplates.Delete(ctx(), id))
			fmt.Printf("Template %d deleted\n", id)
			return nil
		},
	}
}

func tplSearchCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use: "search", Short: "Search templates",
		Long: "Search templates by name or description.\n\nREQUIRED FLAGS:\n  --query   Search query\n\nEXAMPLES:\n  fibe templates search --query node",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			results, err := c.ImportTemplates.Search(ctx(), query, nil)
			if err != nil { return err }
			outputJSON(results)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query (required)")
	cmd.MarkFlagRequired("query")
	return cmd
}

func tplVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions [list|create|destroy|toggle-public] <id> ...",
		Short: "Manage template versions (list, create, destroy, toggle-public)",
		Long: `Manage versions of an import template.

Without a subcommand, lists versions for the given template ID
(equivalent to "versions list <id>") for backward compatibility.

SUBCOMMANDS:
  list <id>                          List versions of a template
  create <id>                        Create a new version (--body required)
  destroy <id> <ver-id>              Delete a version
  toggle-public <id> <ver-id>        Flip a version's public visibility

EXAMPLES:
  fibe templates versions 5
  fibe templates versions list 5
  fibe templates versions create 5 --body @template.yaml --public
  fibe templates versions destroy 5 12
  fibe templates versions toggle-public 5 12`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			c := newClient()
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid template id %q: %w", args[0], err)
			}
			versions, err := c.ImportTemplates.ListVersions(ctx(), id, nil)
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
		Use:   "list <template-id>",
		Short: "List template versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			versions, err := c.ImportTemplates.ListVersions(ctx(), id, nil)
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
	var public bool
	cmd := &cobra.Command{
		Use:   "create <template-id>",
		Short: "Create a new version for an import template",
		Long: `Create a new version of an existing template.

REQUIRED FLAGS:
  --body    Template YAML body (use @path/to/file to read from disk)

OPTIONAL FLAGS:
  --public  Mark version as publicly visible (default: false)

EXAMPLES:
  fibe templates versions create 5 --body @template.yaml
  fibe templates versions create 5 --body @template.yaml --public`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.ImportTemplateVersionCreateParams{
				TemplateBody: resolveStringValue(body),
			}
			if cmd.Flags().Changed("public") {
				params.Public = &public
			}
			if params.TemplateBody == "" {
				return fmt.Errorf("required field 'body' not set")
			}
			version, err := c.ImportTemplates.CreateVersion(ctx(), id, params)
			if err != nil {
				return err
			}
			outputJSON(version)
			return nil
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "Template YAML body (required, use @path to read file)")
	cmd.Flags().BoolVar(&public, "public", false, "Mark version as public")
	return cmd
}

func tplVersionsDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy <template-id> <version-id>",
		Short: "Delete a template version",
		Long: `Delete a specific version of a template.

If this is the last version, the template itself will also be deleted.

EXAMPLES:
  fibe templates versions destroy 5 12`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			templateID, _ := strconv.ParseInt(args[0], 10, 64)
			versionID, _ := strconv.ParseInt(args[1], 10, 64)
			if err := c.ImportTemplates.DestroyVersion(ctx(), templateID, versionID); err != nil {
				return err
			}
			fmt.Printf("Version %d deleted from template %d\n", versionID, templateID)
			return nil
		},
	}
}

func tplVersionsTogglePublicCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "toggle-public <template-id> <version-id>",
		Short: "Toggle a version's public visibility",
		Long: `Toggle a template version between public and private.

EXAMPLES:
  fibe templates versions toggle-public 5 12`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			templateID, _ := strconv.ParseInt(args[0], 10, 64)
			versionID, _ := strconv.ParseInt(args[1], 10, 64)
			version, err := c.ImportTemplates.TogglePublic(ctx(), templateID, versionID)
			if err != nil {
				return err
			}
			outputJSON(version)
			return nil
		},
	}
}

func tplLaunchCmd() *cobra.Command {
	var marqueeID int64
	var name string
	var version int64
	cmd := &cobra.Command{
		Use: "launch <id>", Short: "Launch a playground from template", Args: cobra.ExactArgs(1),
		Long: "Create a new playground using this template.\n\nREQUIRED FLAGS:\n  --marquee-id   Target marquee for the new playground\n\nOPTIONAL FLAGS:\n  --name         Override generated playground name\n  --version      Launch a specific template version\n\nEXAMPLES:\n  fibe templates launch 8 --marquee-id 12\n  fibe templates launch 8 --marquee-id 12 --name my-playground --version 3",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.ImportTemplateLaunchParams{MarqueeID: marqueeID}
			if name != "" {
				params.Name = name
			}
			if version > 0 {
				params.Version = &version
			}
			result, err := c.ImportTemplates.LaunchWithParams(ctx(), id, params)
			if err != nil { return err }
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().Int64Var(&marqueeID, "marquee-id", 0, "Target marquee ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Optional playground name")
	cmd.Flags().Int64Var(&version, "version", 0, "Optional template version")
	cmd.MarkFlagRequired("marquee-id")
	return cmd
}

func tplCreateVersionCmd() *cobra.Command {
	cmd := tplVersionsCreateCmd()
	cmd.Use = "create-version <template-id>"
	cmd.Short = "Create a new version (alias of \"versions create\")"
	cmd.Hidden = true
	return cmd
}

func tplDestroyVersionCmd() *cobra.Command {
	cmd := tplVersionsDestroyCmd()
	cmd.Use = "destroy-version <template-id> <version-id>"
	cmd.Short = "Delete a template version (alias of \"versions destroy\")"
	cmd.Hidden = true
	return cmd
}

func tplTogglePublicCmd() *cobra.Command {
	cmd := tplVersionsTogglePublicCmd()
	cmd.Use = "toggle-public <template-id> <version-id>"
	cmd.Short = "Toggle a version's public visibility (alias of \"versions toggle-public\")"
	cmd.Hidden = true
	return cmd
}

func tplForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <id>",
		Short: "Fork a template into your account",
		Long: `Create a personal copy (fork) of an existing template.

The forked template is owned by you and can be edited independently
of the original.

EXAMPLES:
  fibe templates fork 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			tpl, err := c.ImportTemplates.Fork(ctx(), id)
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
		Use:   "upload-image <id>",
		Short: "Upload a cover image for a template",
		Long: `Upload a cover image for an import template.

The file is read from disk, base64-encoded, and sent to the server along
with the filename and detected content type.

REQUIRED FLAGS:
  --file    Path to image file (PNG, JPEG, GIF, etc.)

EXAMPLES:
  fibe templates upload-image 5 --file logo.png`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
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
			tpl, err := c.ImportTemplates.UploadImage(ctx(), id, params)
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
// Teams: missing member/resource management commands
// =============================================================================
