package integration

import (
	"os"
	"strconv"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Operator templates are provisioned through Admin, independently of catalogue CRUD.
// This case proves API admission and immutable origin binding; runtime deployment
// remains covered by the ordinary Playground lifecycle tests.
func TestOperatorTemplate_LaunchPinnedVersion(t *testing.T) {
	requireEdition(t)
	fixtureID := func(name string) int64 {
		t.Helper()
		value := os.Getenv(name)
		if value == "" {
			t.Skip("operator template launch requires FIBE_TEST_OPERATOR_TEMPLATE_ID, FIBE_TEST_OPERATOR_TEMPLATE_VERSION and FIBE_TEST_OPERATOR_TEMPLATE_VERSION_ID")
		}
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			t.Fatalf("%s must be a positive integer", name)
		}
		return id
	}
	templateID := fixtureID("FIBE_TEST_OPERATOR_TEMPLATE_ID")
	version := fixtureID("FIBE_TEST_OPERATOR_TEMPLATE_VERSION")
	versionID := fixtureID("FIBE_TEST_OPERATOR_TEMPLATE_VERSION_ID")
	c := userClient(t)
	hostID := testHostID(t)
	if hostID <= 0 {
		t.Skip("operator template launch requires an owned ready Host")
	}
	result, err := c.ImportTemplates.LaunchWithParams(ctx(), templateID, &fibe.ImportTemplateLaunchParams{HostID: hostID, Version: &version, Name: uniqueName("operator-template")})
	requireNoError(t, err, "launch operator template pinned version")
	if result.SpecID <= 0 || result.PlaygroundID <= 0 {
		t.Fatalf("launch must return Spec and Playground IDs: %#v", result)
	}
	t.Cleanup(func() { _ = c.Playgrounds.Delete(ctx(), result.PlaygroundID); _ = c.Specs.Delete(ctx(), result.SpecID) })
	spec, err := c.Specs.Get(ctx(), result.SpecID)
	requireNoError(t, err, "read launched Spec")
	if spec.SourceTemplateVersionID == nil || *spec.SourceTemplateVersionID != versionID {
		t.Fatalf("Spec origin = %v, want selected immutable version %d", spec.SourceTemplateVersionID, versionID)
	}
	playground, err := c.Playgrounds.Get(ctx(), result.PlaygroundID)
	requireNoError(t, err, "read launched Playground")
	if playground.SpecID == nil || *playground.SpecID != result.SpecID || playground.HostID == nil || *playground.HostID != hostID {
		t.Fatalf("Playground must reference the launched Spec and selected Host: %#v", playground)
	}
}
