package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func docsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "docs",
		Short: "Print full CLI help for every command",
		Long: `Print the complete --help output for every CLI command in one shot.

Useful for generating full documentation or piping into search tools.
The output is identical to running each command with --help sequentially.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			walkCommands(cmd.Root(), os.Stdout)
			return nil
		},
	}
}

func walkCommands(cmd *cobra.Command, w io.Writer) {
	fmt.Fprintf(w, "\n%s\n%s\n%s\n\n",
		strings.Repeat("=", 80),
		cmd.CommandPath(),
		strings.Repeat("=", 80))

	cmd.SetOut(w)
	cmd.Help()

	// Recurse into children, skipping:
	// - hidden commands (aliases etc)
	// - "help" (built-in, excludes itself)
	// - "docs" (this command — avoid self-inclusion)
	for _, child := range cmd.Commands() {
		if child.Hidden || child.Name() == "help" || child.Name() == "docs" {
			continue
		}
		walkCommands(child, w)
	}
}
