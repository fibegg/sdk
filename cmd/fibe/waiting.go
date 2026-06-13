package main

import (
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func waitForCreatedPlayground(c *fibe.Client, id int64, timeout time.Duration) error {
	if id <= 0 {
		return nil
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	_, err := waitForPlayground(ctx(), c, id, "running", timeout, 3*time.Second, nil)
	if err != nil {
		return fmt.Errorf("playground %d did not reach running: %w", id, err)
	}
	return nil
}
