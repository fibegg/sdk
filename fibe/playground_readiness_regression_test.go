package fibe

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadinessRequiresTheRequestedRuntimeAndSuccessfulDependencyJobs(t *testing.T) {
	for _, tc := range []struct {
		name, fields, reason string
		ready                bool
	}{
		{"applied runtime", ``, "", true},
		{"restored previous runtime", `,"needs_recreation":true`, "not applied", false},
		{"failed replacement build", `,"build_warnings":[{"service":"android","message":"build failed"}]`, "build warnings", false},
		{"successful migration", `,"extra_service":{"name":"migrate","status":"exited","exit_code":0,"completion_expected":true}`, "", true},
		{"failed migration", `,"extra_service":{"name":"migrate","status":"exited","exit_code":1,"completion_expected":true}`, "migrate", false},
		{"unexpected exit", `,"extra_service":{"name":"api","status":"exited","exit_code":0}`, "api", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var response PlaygroundRuntimeStatus
			var extras struct {
				ExtraService *PlaygroundRuntimeServiceInfo `json:"extra_service"`
			}
			if err := json.Unmarshal([]byte(`{"status":"running","services":[{"name":"android","status":"running","health":"healthy"}]`+tc.fields+`}`), &response); err != nil {
				t.Fatal(err)
			}
			data := []byte(`{"status":"running"` + tc.fields + `}`)
			if err := json.Unmarshal(data, &extras); err != nil {
				t.Fatal(err)
			}
			if extras.ExtraService != nil {
				response.RuntimeServices = append(response.RuntimeServices, *extras.ExtraService)
			}
			ready, reason := PlaygroundRuntimeStatusMatchesWaitTarget(&response, "running", PlaygroundWaitReadinessServices)
			if ready != tc.ready || !strings.Contains(reason, tc.reason) {
				t.Fatalf("ready=%v reason=%q", ready, reason)
			}
		})
	}
}
