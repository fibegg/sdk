package fibe

import "strings"

const (
	TaskResultSucceeded = "succeeded"
	TaskResultFailed    = "failed"
	TaskResultUnknown   = "unknown"
	TaskResultRunning   = "running"
)

func TaskOutcome(pg Playground) string {
	if pg.ResultStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*pg.ResultStatus)) {
		case TaskResultSucceeded:
			return TaskResultSucceeded
		case TaskResultFailed:
			return TaskResultFailed
		case TaskResultUnknown:
			return TaskResultUnknown
		}
	}
	if pg.JobResult != nil && pg.JobResult.Success != nil {
		if *pg.JobResult.Success {
			return TaskResultSucceeded
		}
		return TaskResultFailed
	}
	switch strings.ToLower(strings.TrimSpace(pg.Status)) {
	case "error", "failed":
		return TaskResultFailed
	case "completed":
		return TaskResultUnknown
	default:
		return TaskResultRunning
	}
}

func TaskStatusOutcome(pg *PlaygroundStatus) string {
	if pg == nil {
		return TaskResultUnknown
	}
	return TaskOutcome(Playground{
		ID:           pg.ID,
		Status:       pg.Status,
		ResultStatus: pg.ResultStatus,
		JobResult:    pg.JobResult,
	})
}

func TaskOutcomeIsFailed(pg Playground) bool {
	return TaskOutcome(pg) == TaskResultFailed
}

func TaskStatusResultFailed(pg *PlaygroundStatus) bool {
	return TaskStatusOutcome(pg) == TaskResultFailed
}
