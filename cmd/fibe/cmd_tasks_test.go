package main

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestTaskResultDoesNotAssumeCompletedMeansSuccess(t *testing.T) {
	if got := taskResult(fibe.Playground{Status: "completed"}); got != "?" {
		t.Fatalf("completed task without job_result rendered %q, want ?", got)
	}

	success := true
	if got := taskResult(fibe.Playground{Status: "completed", JobResult: &fibe.JobResult{Success: &success}}); got != "✓" {
		t.Fatalf("completed successful task rendered %q, want ✓", got)
	}

	success = false
	if got := taskResult(fibe.Playground{Status: "completed", JobResult: &fibe.JobResult{Success: &success}}); got != "✗" {
		t.Fatalf("completed failed task rendered %q, want ✗", got)
	}

	resultStatus := "failed"
	if got := taskResult(fibe.Playground{Status: "completed", ResultStatus: &resultStatus}); got != "✗" {
		t.Fatalf("completed failed result_status rendered %q, want ✗", got)
	}
}
