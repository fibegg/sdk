package main

import (
	"strings"
	"testing"
)

func TestPlaygroundsHelpMatchesLifecycleSurface(t *testing.T) {
	help := commandHelp(t, playgroundsCmd())

	for _, want := range []string{
		"start <id-or-name>",
		"stop <id-or-name>",
		"has_changes",
		"completed",
		"stopping",
		"stopped",
		"destroying",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("playgrounds help missing %q:\n%s", want, help)
		}
	}
}

func TestPlaygroundActionCommandsAreRegistered(t *testing.T) {
	cmd := playgroundsCmd()

	for _, args := range [][]string{
		{"start", "example"},
		{"stop", "example"},
		{"rollout", "example"},
		{"hard-restart", "example"},
	} {
		found, _, err := cmd.Find(args)
		if err != nil {
			t.Fatalf("find %v: %v", args, err)
		}
		if found == nil {
			t.Fatalf("find %v returned nil command", args)
		}
		if found.Use == "" {
			t.Fatalf("find %v returned command without use", args)
		}
	}
}
