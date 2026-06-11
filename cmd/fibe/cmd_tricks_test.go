package main

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestTrickResultDoesNotAssumeCompletedMeansSuccess(t *testing.T) {
	if got := trickResult(fibe.Playground{Status: "completed"}); got != "?" {
		t.Fatalf("completed trick without job_result rendered %q, want ?", got)
	}

	success := true
	if got := trickResult(fibe.Playground{Status: "completed", JobResult: &fibe.JobResult{Success: &success}}); got != "✓" {
		t.Fatalf("completed successful trick rendered %q, want ✓", got)
	}

	success = false
	if got := trickResult(fibe.Playground{Status: "completed", JobResult: &fibe.JobResult{Success: &success}}); got != "✗" {
		t.Fatalf("completed failed trick rendered %q, want ✗", got)
	}

	resultStatus := "failed"
	if got := trickResult(fibe.Playground{Status: "completed", ResultStatus: &resultStatus}); got != "✗" {
		t.Fatalf("completed failed result_status rendered %q, want ✗", got)
	}
}
