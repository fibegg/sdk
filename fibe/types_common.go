package fibe

import "time"

type ListParams struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}
