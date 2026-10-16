package main

import (
	"io"
	"strings"
	"testing"
)

func TestPresetsRemainWebAndAPIOnly(t *testing.T) {
	setupAuthTest(t)
	for _, command := range allCommands(RootCmd()) {
		if strings.Contains(strings.ToLower(command.CommandPath()), "preset") {
			t.Fatalf("unexpected preset CLI command: %s", command.CommandPath())
		}
	}
	for _, name := range []string{"presets", "agent-presets"} {
		root := RootCmd()
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs([]string{name, "list"})
		if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("%s unexpectedly accepted: %v", name, err)
		}
	}
}
