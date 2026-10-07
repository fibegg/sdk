package mcpserver

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestScalarCoercionContract(t *testing.T) {
	stringInputs := []any{"text", true, json.Number("12"), float64(1.5), float32(2.5), int(1), int8(2), int16(3), int32(4), int64(5), uint(6), uint8(7), uint16(8), uint32(9), uint64(10)}
	for _, input := range stringInputs {
		if _, ok := coerceString(input); !ok {
			t.Fatalf("coerceString(%T) rejected", input)
		}
	}
	if _, ok := coerceString(struct{}{}); ok {
		t.Fatal("coerceString accepted struct")
	}
	for _, input := range []any{true, "yes", "0", float64(1), float64(0), int(1), int(0), int64(1), int64(0)} {
		if _, ok := coerceBool(input); !ok {
			t.Fatalf("coerceBool(%v) rejected", input)
		}
	}
	for _, input := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float64(1), json.Number("1"), "1"} {
		if got, ok := coerceInt64(input); !ok || got != 1 {
			t.Fatalf("coerceInt64(%T) = %d, %v", input, got, ok)
		}
	}
	for _, input := range []any{uint(1), uint8(1), uint16(1), uint32(1), uint64(1), int(1), int64(1), float64(1), json.Number("1"), "1"} {
		if got, ok := coerceUint64(input); !ok || got != 1 {
			t.Fatalf("coerceUint64(%T) = %d, %v", input, got, ok)
		}
	}
	for _, input := range stringInputs[2:] {
		if _, ok := coerceFloat64(input); !ok {
			t.Fatalf("coerceFloat64(%T) rejected", input)
		}
	}
	for _, input := range []any{"maybe", float64(2), int(2), int64(2), nil} {
		if _, ok := coerceBool(input); ok {
			t.Fatalf("coerceBool(%v) accepted", input)
		}
	}
	if _, ok := coerceInt64(math.NaN()); ok {
		t.Fatal("coerceInt64 accepted NaN")
	}
}

func TestArgumentBindingAndIdentifierContract(t *testing.T) {
	type Embedded struct {
		Enabled bool `json:"enabled"`
	}
	type destination struct {
		Embedded
		Count          uint64            `json:"count"`
		Ratio          float64           `json:"ratio"`
		Labels         map[string]string `json:"labels"`
		Items          []int64           `json:"items"`
		Ignored        string            `json:"-"`
		SpecIdentifier string
	}
	var got destination
	err := bindArgs(map[string]any{
		"enabled": "yes",
		"count":   "7",
		"ratio":   "1.5",
		"labels":  map[string]any{"env": 42},
		"items":   []any{"2", float64(3)},
		"unknown": true,
	}, &got)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.Count != 7 || got.Ratio != 1.5 || got.Labels["env"] != "42" || !reflect.DeepEqual(got.Items, []int64{2, 3}) {
		t.Fatalf("bound destination = %#v", got)
	}
	if err := bindIdentifierArgs(map[string]any{"spec_id": "starter"}, &got, "spec_id"); err != nil {
		t.Fatal(err)
	}
	if got.SpecIdentifier != "starter" {
		t.Fatalf("identifier = %q", got.SpecIdentifier)
	}
	if err := setIdentifierField(nil, "spec_id", "fixture"); err == nil {
		t.Fatal("nil identifier destination accepted")
	}
	if err := setIdentifierField(new(string), "spec_id", "fixture"); err == nil {
		t.Fatal("non-struct identifier destination accepted")
	}
	for field := range map[string]bool{
		"build_in_public_playground_id": true, "ci_host_id": true, "host_id": true,
		"playground_id": true, "spec_id": true, "repository_id": true, "source_repository_id": true,
		"target_playground_id": true, "target_spec_id": true,
	} {
		if _, ok := identifierStructField(field); !ok {
			t.Fatalf("identifier field %q missing", field)
		}
	}
	if _, ok := identifierStructField("unknown"); ok {
		t.Fatal("unknown identifier field accepted")
	}
}

func TestLocalFileAndLaunchHelperContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.txt")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	encoded, err := readLocalFileBase64(path)
	if err != nil || encoded != "Zml4dHVyZQ==" {
		t.Fatalf("encoded=%q err=%v", encoded, err)
	}
	if text, err := readInlineOrPathTextArg(map[string]any{"path": path}, "inline", "path"); err != nil || text != "fixture" {
		t.Fatalf("text=%q err=%v", text, err)
	}
	if _, err := readInlineOrPathTextArg(nil, "inline", "path"); err == nil {
		t.Fatal("missing text source accepted")
	}
	if got := filenameFromContentPath(path, "fallback"); got != "fixture.txt" {
		t.Fatalf("filename=%q", got)
	}
	if got := filenameFromContentPath("", "fallback"); got != "fallback" {
		t.Fatalf("fallback filename=%q", got)
	}

	for source, args := range map[string]map[string]any{
		"template":         {"template_id_or_name": "starter"},
		"template_version": {"template_version_id": float64(3)},
		"spec":             {"spec_id_or_name": "starter"},
		"compose":          {"compose_yaml": "services: {}"},
		"repo":             {"repository_url": "owner/repo"},
	} {
		if got, err := mcpLaunchSource(args); err != nil || got != source {
			t.Fatalf("source args=%v got=%q err=%v", args, got, err)
		}
	}
	if _, err := mcpLaunchSource(nil); err == nil {
		t.Fatal("empty launch source accepted")
	}
	if _, err := mcpLaunchSource(map[string]any{"compose_yaml": "x", "repository_url": "owner/repo"}); err == nil {
		t.Fatal("multiple launch sources accepted")
	}
	if !valuePresent(struct{}{}) || valuePresent(" ") || valuePresent(float64(0)) || valuePresent(int64(0)) || valuePresent(int(0)) || valuePresent(nil) {
		t.Fatal("valuePresent contract mismatch")
	}
	if got := argStringMap(map[string]any{"values": map[string]any{"A": 1}}, "values"); got["A"] != "1" {
		t.Fatalf("string map=%v", got)
	}
	if got := serviceConfigArgs(map[string]any{"web": map[string]any{"subdomain": "app"}, "bad": make(chan int)}); got["web"] == nil || got["bad"] != nil {
		t.Fatalf("service config=%#v", got)
	}
	if got := launchVariables(map[string]any{" app-url ": " value ", "empty": " "}); got["app-url"] != "value" || len(got) != 1 {
		t.Fatalf("launch variables=%v", got)
	}
	params := &fibe.LaunchParams{RepositoryMappings: map[string]int64{}, RepositoryMappingIdentifiers: map[string]string{}}
	applyLaunchRepositoryMappings(params, map[string]any{"owner/one": "42", "owner/two": "backend", "empty": " "})
	if params.RepositoryMappings["owner/one"] != 42 || params.RepositoryMappingIdentifiers["owner/two"] != "backend" {
		t.Fatalf("repository mappings=%v identifiers=%v", params.RepositoryMappings, params.RepositoryMappingIdentifiers)
	}
}

func TestTemplateChangeHelperContract(t *testing.T) {
	in := &templateChangeArgs{TargetType: "spec", TargetIdentifier: " starter ", Mode: "preview", ChangeType: "patch"}
	if err := normalizeTemplateChangeArgs(in); err != nil {
		t.Fatal(err)
	}
	if in.TargetIdentifier != "starter" || in.PostApply != "none" || in.ResponseMode != "summary" || in.WaitTimeoutSeconds != 180 {
		t.Fatalf("normalized input=%#v", in)
	}
	target := &templateChangeTarget{templateID: 1, baseVersion: 2}
	if err := validateTemplateChangeCombination(in, target); err == nil {
		t.Fatal("empty patch accepted")
	}
	in.Patches = []fibe.TemplatePatchEdit{{Op: "replace", Path: "/name", Value: "fixture"}}
	if err := validateTemplateChangeCombination(in, target); err != nil {
		t.Fatal(err)
	}
	for postApply, want := range map[string]string{"rollout_target": "target", "rollout_all": "all", "none": "none"} {
		if got := rolloutModeForPostApply(postApply); got != want {
			t.Fatalf("rollout mode %q=%q", postApply, got)
		}
	}
	if !diagnoseTemplateChange(in) || boolPtrValue(nil) {
		t.Fatal("boolean defaults mismatch")
	}
	falseValue := false
	in.DiagnoseOnFailure = &falseValue
	if diagnoseTemplateChange(in) {
		t.Fatal("explicit diagnostics=false ignored")
	}
	if got := anyInt64Slice([]any{float64(1), int(2), int64(3), math.Inf(1), -1}); !reflect.DeepEqual(got, []int64{1, 2, 3}) {
		t.Fatalf("rollout ids=%v", got)
	}
	result := fibe.TemplateVersionPatchResult{"playground_rollout_plan": map[string]any{"rollout": []int64{4, 5}}}
	if got := rolloutIDsFromPatchResult(&result); !reflect.DeepEqual(got, []int64{4, 5}) {
		t.Fatalf("patch rollout ids=%v", got)
	}
	if got := rolloutIDsFromAny(map[string]any{"playground_rollout_plan": map[string]any{"rollout": []any{float64(6)}}}); !reflect.DeepEqual(got, []int64{6}) {
		t.Fatalf("generic rollout ids=%v", got)
	}
}

func TestTemplateChangeWorkflowVariants(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":1,
			"name":"fixture",
			"latest_version_id":2,
			"source_template_version_id":2,
			"source_template":{"id":1},
			"source_template_version":{"template":{"id":1}},
			"target_template_version":{"id":3},
			"spec":{"id":1,"source_template_version_id":3},
			"spec_id":1,
			"host_id":1,
			"job_mode":false,
			"status":"running",
			"playground_rollout_plan":{"rollout":[1]}
		}`))
	}))
	defer api.Close()
	client := fibe.NewClient(
		fibe.WithDisableAutoConfig(),
		fibe.WithDomain(api.URL),
		fibe.WithAPIKey("fixture"),
		fibe.WithMaxRetries(0),
	)
	patch := []fibe.TemplatePatchEdit{{Op: "replace", Path: "/name", Value: "fixture"}}
	cases := []templateChangeArgs{
		{TargetType: "template", TargetIdentifier: "fixture", Mode: "preview", ChangeType: "patch", Patches: patch},
		{TargetType: "template", TargetIdentifier: "fixture", Mode: "apply", ChangeType: "overwrite", TemplateBody: "services: {}"},
		{TargetType: "spec", TargetIdentifier: "fixture", Mode: "preview", ChangeType: "switch_existing", TargetTemplateVersionID: 3},
		{TargetType: "spec", TargetIdentifier: "fixture", Mode: "apply", ChangeType: "switch_existing", TargetTemplateVersionID: 3},
		{TargetType: "playground", TargetIdentifier: "fixture", Mode: "preview", ChangeType: "patch", Patches: patch},
	}
	for _, input := range cases {
		input := input
		t.Run(input.TargetType+"/"+input.Mode+"/"+input.ChangeType, func(t *testing.T) {
			if _, err := runTemplateChange(context.Background(), client, &input); err != nil {
				t.Fatal(err)
			}
		})
	}

	validTarget := &templateChangeTarget{templateID: 1, baseVersion: 2}
	for name, input := range map[string]templateChangeArgs{
		"post_apply":       {TargetType: "spec", PostApply: "invalid", ChangeType: "patch", Patches: patch},
		"template_rollout": {TargetType: "template", PostApply: "rollout_all", ChangeType: "patch", Patches: patch},
		"switch_version":   {TargetType: "spec", PostApply: "none", ChangeType: "switch_existing"},
		"overwrite_body":   {TargetType: "spec", PostApply: "none", ChangeType: "overwrite"},
	} {
		input := input
		t.Run("reject/"+name, func(t *testing.T) {
			if err := validateTemplateChangeCombination(&input, validTarget); err == nil {
				t.Fatal("invalid combination accepted")
			}
		})
	}
}

func TestLaunchWorkflowVariants(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"fixture","status":"running"}`))
	}))
	defer api.Close()
	client := fibe.NewClient(
		fibe.WithDisableAutoConfig(),
		fibe.WithDomain(api.URL),
		fibe.WithAPIKey("fixture"),
		fibe.WithMaxRetries(0),
	)
	server := New(DefaultConfig())
	cases := map[string]map[string]any{
		"template": {
			"template_id_or_name": "starter", "host_id_or_name": float64(1), "version": "2",
			"variables": map[string]any{"NAME": "demo"}, "persist_volumes": true,
		},
		"template_version": {
			"name": "fixture", "template_version_id": float64(2), "host_id_or_name": "primary",
			"env_overrides": map[string]any{"APP_ENV": "test"},
		},
		"spec": {
			"name": "fixture", "spec_id_or_name": "starter", "host_id_or_name": float64(1),
			"services": map[string]any{"web": map[string]any{"subdomain": "app"}},
		},
		"compose": {
			"name": "fixture", "compose_yaml": "services: {}", "create_playground": false,
			"repository_mappings": map[string]any{"owner/repo": "backend"},
		},
	}
	for name, args := range cases {
		name, args := name, args
		t.Run(name, func(t *testing.T) {
			if _, err := server.runLaunch(context.Background(), client, args); err != nil {
				t.Fatal(err)
			}
		})
	}

	if id, identifier := explicitMCPHost(map[string]any{"host_id": "7"}); id == nil || *id != 7 || identifier != "" {
		t.Fatalf("explicit host id=%v identifier=%q", id, identifier)
	}
	if id, identifier := explicitMCPHost(map[string]any{"host_id_or_name": "primary"}); id != nil || identifier != "primary" {
		t.Fatalf("explicit host id=%v identifier=%q", id, identifier)
	}
	if got := mcpHostCandidateNames([]fibe.Host{{ID: 2}, {ID: 1, Name: "alpha"}}); got != "2, alpha" {
		t.Fatalf("candidate names=%q", got)
	}
	if _, _, err := resolveMCPHost(context.Background(), nil, nil); err == nil {
		t.Fatal("missing host accepted")
	}
}
