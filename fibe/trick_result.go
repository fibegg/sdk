package fibe

import "strings"

const (
	TrickResultSucceeded = "succeeded"
	TrickResultFailed    = "failed"
	TrickResultUnknown   = "unknown"
	TrickResultRunning   = "running"
)

func TrickOutcome(pg Playground) string {
	if pg.ResultStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*pg.ResultStatus)) {
		case TrickResultSucceeded:
			return TrickResultSucceeded
		case TrickResultFailed:
			return TrickResultFailed
		case TrickResultUnknown:
			return TrickResultUnknown
		}
	}
	if pg.JobResult != nil && pg.JobResult.Success != nil {
		if *pg.JobResult.Success {
			return TrickResultSucceeded
		}
		return TrickResultFailed
	}
	switch strings.ToLower(strings.TrimSpace(pg.Status)) {
	case "error", "failed":
		return TrickResultFailed
	case "completed":
		return TrickResultUnknown
	default:
		return TrickResultRunning
	}
}

func TrickStatusOutcome(pg *PlaygroundStatus) string {
	if pg == nil {
		return TrickResultUnknown
	}
	return TrickOutcome(Playground{
		ID:           pg.ID,
		Status:       pg.Status,
		ResultStatus: pg.ResultStatus,
		JobResult:    pg.JobResult,
	})
}

func TrickOutcomeIsFailed(pg Playground) bool {
	return TrickOutcome(pg) == TrickResultFailed
}

func TrickStatusResultFailed(pg *PlaygroundStatus) bool {
	return TrickStatusOutcome(pg) == TrickResultFailed
}
