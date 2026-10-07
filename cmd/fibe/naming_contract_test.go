package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestRemovedDomainCommandsOfferCanonicalCommands(t *testing.T) {
	for old, replacement := range map[string]string{"marquees": "hosts", "playspecs": "specs", "props": "repositories", "tricks": "tasks", "mq": "hosts"} {
		t.Run(old, func(t *testing.T) {
			root := RootCmd()
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs([]string{old, "list"})
			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), replacement) {
				t.Fatalf("got %v; want replacement %s", err, replacement)
			}
			command, _, _ := root.Find([]string{old})
			if command != root {
				t.Fatalf("removed name %s still resolves to %s", old, command.Name())
			}
		})
	}
}

func TestRemovedDomainFlagsFailBeforeRequests(t *testing.T) {
	for old, replacement := range map[string]string{"--marquee": "--host", "--playspec": "--spec", "--marquee-id": "--host"} {
		root := RootCmd()
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs([]string{"playgrounds", "create", old, "one"})
		err := root.Execute()
		if err == nil || !strings.Contains(err.Error(), replacement) {
			t.Fatalf("%s: got %v", old, err)
		}
	}
}

func TestFromFileRejectsRemovedRuntimeFields(t *testing.T) {
	previous := flagFromFile
	t.Cleanup(func() { flagFromFile = previous; rawPayload = nil })
	flagFromFile = filepath.Join(t.TempDir(), "payload.yml")
	if err := os.WriteFile(flagFromFile, []byte("marquee_id: 1\nplayspec_id: 2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var params fibe.PlaygroundCreateParams
	err := applyFromFile(&params)
	if err == nil || !strings.Contains(err.Error(), "host_id") {
		t.Fatalf("got %v", err)
	}
}

func TestSharedAgentSettingsFixtureUsesActualCLIFileDecoder(t *testing.T) {
	data, err := os.ReadFile("../../contracts/naming/v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	previous := flagFromFile
	t.Cleanup(func() { flagFromFile = previous; rawPayload = nil })
	flagFromFile = filepath.Join(t.TempDir(), "fibe.yml")
	if err = os.WriteFile(flagFromFile, fixture["agent_settings"], 0600); err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err = applyFromFile(&settings); err != nil {
		t.Fatal(err)
	}
	if settings["hostRoot"] != "/opt/fibe" || settings["hostRootDomain"] != "host.example.test" {
		t.Fatalf("unexpected settings %+v", settings)
	}
}
