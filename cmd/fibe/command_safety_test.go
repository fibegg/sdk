package main

import "testing"

func TestEveryRunnableCommandHasSafetyMetadata(t *testing.T) {
	root := RootCmd()
	for _, command := range allCommands(root) {
		if !command.Runnable() {
			continue
		}
		classification := command.Annotations[commandSafetyAnnotation]
		switch classification {
		case "read-only", "mutating", "destructive", "local-only":
		default:
			t.Errorf("command %q classification=%q", command.CommandPath(), classification)
		}
	}
}

func TestRequiredDestructiveCommandNamesAreClassified(t *testing.T) {
	found := 0
	for _, command := range allCommands(RootCmd()) {
		if destructiveCommandName(command.Name()) && command.Runnable() {
			found++
			if got := command.Annotations[commandSafetyAnnotation]; got != "destructive" {
				t.Errorf("command %q classification=%q want destructive", command.CommandPath(), got)
			}
		}
	}
	if found < 10 {
		t.Fatalf("found only %d destructive commands; expected the full CLI surface", found)
	}
}
