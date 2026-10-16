package main

import (
	"strings"

	"github.com/spf13/cobra"
)

const commandSafetyAnnotation = "fibe.gg/safety"

func annotateCommandSafety(root *cobra.Command) {
	for _, command := range allCommands(root) {
		if !command.Runnable() {
			continue
		}
		if command.Annotations == nil {
			command.Annotations = map[string]string{}
		}
		command.Annotations[commandSafetyAnnotation] = commandSafetyClass(command)
	}
}

func allCommands(root *cobra.Command) []*cobra.Command {
	commands := []*cobra.Command{root}
	for _, child := range root.Commands() {
		commands = append(commands, allCommands(child)...)
	}
	return commands
}

func commandSafetyClass(command *cobra.Command) string {
	name := strings.ToLower(command.Name())
	if destructiveCommandName(name) {
		return "destructive"
	}
	switch name {
	case "list", "ls", "get", "show", "inspect", "status", "info", "logs", "debug", "doctor", "wait", "follow",
		"download", "download-attachment", "download-upload", "schema", "help", "version", "completion", "docs", "config",
		"names", "current", "repos", "urls", "mounts", "details", "view", "catalog", "whoami":
		return "read-only"
	}
	path := strings.ToLower(command.CommandPath())
	if strings.Contains(path, " local ") || strings.HasPrefix(path, "fibe local ") || strings.Contains(path, " auth ") {
		return "local-only"
	}
	return "mutating"
}

func destructiveCommandName(name string) bool {
	switch name {
	case "stop", "disable", "restart", "hard-restart", "restart-chat", "reset", "rollout", "switch-template", "change-template":
		return true
	}
	for _, prefix := range []string{"delete", "destroy", "remove"} {
		if name == prefix || strings.HasPrefix(name, prefix+"-") {
			return true
		}
	}
	return false
}
