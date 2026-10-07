package fibe

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLegacyResponseFieldsFailWithoutChangingDestination(t *testing.T) {
	destination := Playground{ID: 99}
	err := decodeJSONLimitedProjected(context.Background(), strings.NewReader(`{"id":1,"marquee_id":2,"playspec_id":3}`), maxResponseBody, "response", &destination)
	if err == nil || !strings.Contains(err.Error(), "host_id") {
		t.Fatalf("got %v", err)
	}
	if destination.ID != 99 {
		t.Fatalf("failed decode changed destination: %+v", destination)
	}
}

func TestSharedNamingFixtureUsesActualSDKDecoder(t *testing.T) {
	data, err := os.ReadFile("../contracts/naming/v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	var playground Playground
	if err = decodeJSONLimitedProjected(context.Background(), strings.NewReader(string(fixture["api_playground"])), maxResponseBody, "shared naming fixture", &playground); err != nil {
		t.Fatal(err)
	}
	if playground.ID != 20 || playground.HostID == nil || *playground.HostID != 21 || playground.SpecID == nil || *playground.SpecID != 120 || playground.JobMode {
		t.Fatalf("unexpected playground %+v", playground)
	}
	var source map[string]any
	if err = decodeJSONLimitedProjected(context.Background(), strings.NewReader(string(fixture["source_reference"])), maxResponseBody, "source reference", &source); err != nil {
		t.Fatal(err)
	}
	if source["repository_id"] != float64(24) || source["repository_branch_id"] != float64(7) || source["branch"] != "feature/naming" {
		t.Fatalf("unexpected source %+v", source)
	}
	var removed map[string]string
	if err = json.Unmarshal(fixture["removed_keys"], &removed); err != nil {
		t.Fatal(err)
	}
	for old, replacement := range removed {
		input, _ := json.Marshal(map[string]any{old: 1})
		var destination map[string]any
		err = decodeJSONLimitedProjected(context.Background(), strings.NewReader(string(input)), maxResponseBody, "removed field", &destination)
		if err == nil || !strings.Contains(err.Error(), replacement) {
			t.Fatalf("%s: got %v", old, err)
		}
	}
}
