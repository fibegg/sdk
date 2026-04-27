package main

import (
	"encoding/json"
	"fmt"

	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/spf13/cobra"
)

func schemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema <resource> [operation]",
		Short: "Show JSON Schema for a resource",
		Long: `Print JSON Schema hints for Fibe resources.

This is the machine-parsable companion to --help. Use --list to discover
the canonical resource names, aliases, and generic resource operations.

Examples:
  fibe schema --list
  fibe schema playground list
  fibe schema playground create
  fibe schema playground`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			listFlag, _ := cmd.Flags().GetBool("list")
			if listFlag || len(args) == 0 {
				return printJSON(resourceschema.CatalogResponse())
			}

			if resourceschema.NormalizeResource(args[0]) == "list" {
				return printJSON(resourceschema.CatalogResponse())
			}

			schemas, canonical, ok := resourceschema.SchemasFor(args[0])
			if !ok {
				return fmt.Errorf("unknown resource %q; supported resources: %s", args[0], resourceschema.ResourceNamesString())
			}

			if len(args) == 2 {
				schema, _, op, ok := resourceschema.SchemaFor(canonical, args[1])
				if !ok {
					return fmt.Errorf("unknown operation %q for resource %q", op, canonical)
				}
				return printJSON(map[string]any{canonical + "." + op: schema})
			}

			return printJSON(schemas)
		},
	}
	cmd.Flags().Bool("list", false, "List canonical resources, aliases, and generic resource operations")
	return cmd
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
