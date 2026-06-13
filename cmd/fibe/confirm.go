package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func confirmDestructive(action string, yes bool) error {
	if yes {
		return nil
	}
	if !stdinIsTerminal() {
		return fmt.Errorf("%s requires --yes in non-interactive mode", action)
	}
	fmt.Fprintf(os.Stderr, "%s? Type yes to continue: ", action)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(line), "yes") {
		return nil
	}
	return fmt.Errorf("aborted")
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
