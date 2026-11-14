package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedAgentSettingsFixtureUsesActualCLIFileDecoder(t *testing.T) {
	data, err := os.ReadFile("../../contracts/decoder-fixtures.json")
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
