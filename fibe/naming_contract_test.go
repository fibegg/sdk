package fibe

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSharedResponseFixturesUseActualSDKDecoder(t *testing.T) {
	data, err := os.ReadFile("../contracts/decoder-fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	var playground Playground
	if err = decodeJSONLimitedProjected(context.Background(), strings.NewReader(string(fixture["api_playground"])), maxResponseBody, "shared playground fixture", &playground); err != nil {
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
}
