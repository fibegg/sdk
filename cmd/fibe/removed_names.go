package main

import (
	"fmt"
	"strings"

	"github.com/fibegg/sdk/internal/domainnames"
	"github.com/spf13/cobra"
)

func installRemovedNameErrors(root *cobra.Command) {
	for old, replacement := range domainnames.Resources {
		canonical := strings.ReplaceAll(replacement, "_", "-")
		for _, command := range root.Commands() {
			if command.Name() == canonical {
				command.SuggestFor = append(command.SuggestFor, old)
			}
		}
	}
	var configure func(*cobra.Command)
	configure = func(cmd *cobra.Command) {
		cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
			const prefix = "unknown flag: --"
			if old, ok := strings.CutPrefix(err.Error(), prefix); ok {
				if replacement, ok := domainnames.Replacement(strings.TrimSuffix(old, "-id")); ok {
					return fmt.Errorf("flag --%s was removed; use --%s", old, replacement)
				}
				oldField := strings.ReplaceAll(old, "-", "_")
				if replacement, ok := domainnames.FieldReplacement(oldField); ok {
					return fmt.Errorf("flag --%s was removed; use --%s", old, strings.ReplaceAll(replacement, "_", "-"))
				}
			}
			return err
		})
		for _, child := range cmd.Commands() {
			configure(child)
		}
	}
	configure(root)
}
