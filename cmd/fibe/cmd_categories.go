package main

import (

	"github.com/spf13/cobra"
)
func categoriesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "categories",
		Short: "List template categories",
		Long: `List all template categories available for organizing import templates.

EXAMPLES:
  fibe categories`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			cats, err := c.TemplateCategories.List(ctx(), listParams())
			if err != nil { return err }
			if effectiveOutput() != "table" { outputJSON(cats); return nil }
			headers := []string{"ID", "NAME", "SLUG"}
			rows := make([][]string, len(cats.Data))
			for i, cat := range cats.Data { rows[i] = []string{fmtInt64(cat.ID), cat.Name, cat.Slug} }
			outputTable(headers, rows)
			return nil
		},
	}
}

