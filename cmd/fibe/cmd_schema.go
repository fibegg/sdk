package main

import (
	"encoding/json"
	"fmt"

	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/spf13/cobra"
)

func schemaCmd() *cobra.Command {
	var resourceFlag, operationFlag string

	cmd := &cobra.Command{
		Use:   "schema <resource> [operation]",
		Short: "Show JSON Schema for a resource",
		Long: `Print JSON Schema hints for Fibe resources.

This is the machine-parsable companion to --help. Use --list to discover
the canonical resource names, aliases, and generic resource operations.

Examples:
  fibe schema --list
  fibe schema --resource playground --operation create
  fibe schema playground list
  fibe schema playground create
  fibe schema playground`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			listFlag, _ := cmd.Flags().GetBool("list")
			resource, operation := "", ""
			if len(args) > 0 {
				resource = args[0]
			}
			if len(args) > 1 {
				operation = args[1]
			}
			if resource == "" {
				resource = resourceFlag
			} else if resourceFlag != "" && resourceschema.NormalizeResource(resourceFlag) != resourceschema.NormalizeResource(resource) {
				return fmt.Errorf("--resource %q conflicts with positional resource %q", resourceFlag, resource)
			}
			if operation == "" {
				operation = operationFlag
			} else if operationFlag != "" && operationFlag != operation {
				return fmt.Errorf("--operation %q conflicts with positional operation %q", operationFlag, operation)
			}

			if listFlag || resource == "" {
				return printJSON(resourceschema.CatalogResponse())
			}

			if resourceschema.NormalizeResource(resource) == "list" {
				return printJSON(resourceschema.CatalogResponse())
			}

			schemas, canonical, ok := resourceschema.SchemasFor(resource)
			if !ok {
				return fmt.Errorf("unknown resource %q; supported resources: %s", resource, resourceschema.ResourceNamesString())
			}

			if operation != "" {
				schema, _, op, ok := resourceschema.SchemaFor(canonical, operation)
				if !ok {
					return fmt.Errorf("unknown operation %q for resource %q", op, canonical)
				}
				return printJSON(map[string]any{canonical + "." + op: schema})
			}

			return printJSON(schemas)
		},
	}
	cmd.Flags().Bool("list", false, "List canonical resources, aliases, and generic resource operations")
	cmd.Flags().StringVar(&resourceFlag, "resource", "", "Resource name (compatibility alias for positional resource)")
	cmd.Flags().StringVar(&operationFlag, "operation", "", "Operation name (compatibility alias for positional operation)")
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
