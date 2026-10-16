package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/pflag"
)

type cliContract struct {
	Version  int                  `json:"version"`
	Commands []cliCommandContract `json:"commands"`
}

type cliCommandContract struct {
	Path        string            `json:"path"`
	Use         string            `json:"use"`
	Aliases     []string          `json:"aliases,omitempty"`
	Runnable    bool              `json:"runnable"`
	Hidden      bool              `json:"hidden,omitempty"`
	Safety      string            `json:"safety,omitempty"`
	Flags       []cliFlagContract `json:"flags,omitempty"`
	LocalFlags  []cliFlagContract `json:"local_flags,omitempty"`
	ExitOnError bool              `json:"exit_on_error"`
}

type cliFlagContract struct {
	Name       string `json:"name"`
	Shorthand  string `json:"shorthand,omitempty"`
	Type       string `json:"type"`
	Default    string `json:"default"`
	NoOpt      string `json:"no_opt_default,omitempty"`
	Usage      string `json:"usage"`
	Hidden     bool   `json:"hidden,omitempty"`
	Deprecated string `json:"deprecated,omitempty"`
}

func TestCLITreeContract(t *testing.T) {
	contract := cliContract{Version: 1}
	for _, command := range allCommands(RootCmd()) {
		entry := cliCommandContract{
			Path:        command.CommandPath(),
			Use:         command.Use,
			Aliases:     append([]string(nil), command.Aliases...),
			Runnable:    command.Runnable(),
			Hidden:      command.Hidden,
			Safety:      command.Annotations[commandSafetyAnnotation],
			Flags:       snapshotFlags(command.InheritedFlags()),
			LocalFlags:  snapshotFlags(command.NonInheritedFlags()),
			ExitOnError: false,
		}
		contract.Commands = append(contract.Commands, entry)
	}
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := filepath.Join(cliRepositoryRoot(t), "contracts", "cli-tree.json")
	if os.Getenv("UPDATE_CONTRACTS") == "1" {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, want) {
		t.Fatal("CLI contract is stale; run UPDATE_CONTRACTS=1 go test ./cmd/fibe -run TestCLITreeContract")
	}
}

func snapshotFlags(flags *pflag.FlagSet) []cliFlagContract {
	var out []cliFlagContract
	flags.VisitAll(func(flag *pflag.Flag) {
		out = append(out, cliFlagContract{
			Name:       flag.Name,
			Shorthand:  flag.Shorthand,
			Type:       flag.Value.Type(),
			Default:    flag.DefValue,
			NoOpt:      flag.NoOptDefVal,
			Usage:      flag.Usage,
			Hidden:     flag.Hidden,
			Deprecated: flag.Deprecated,
		})
	})
	return out
}

func cliRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
