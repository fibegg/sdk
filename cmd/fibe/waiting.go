package main

import (
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func waitForCreatedPlayground(c *fibe.Client, id int64, timeout time.Duration, progress func(string)) (*fibe.Playground, error) {
	if id <= 0 {
		return nil, nil
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	pg, err := waitForPlayground(ctx(), c, id, "running", timeout, waitIntervalForTimeout(timeout), progress)
	if err != nil {
		return nil, fmt.Errorf("playground %d did not reach running: %w", id, err)
	}
	return pg, nil
}

func waitIntervalForTimeout(timeout time.Duration) time.Duration {
	interval := 3 * time.Second
	if timeout <= 0 || timeout >= interval {
		return interval
	}
	interval = timeout / 10
	if interval < 50*time.Millisecond {
		return 50 * time.Millisecond
	}
	return interval
}
