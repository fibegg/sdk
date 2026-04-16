package fibe

import "time"

type Mutation struct {
	ID             *int64         `json:"id"`
	PropID         int64          `json:"prop_id"`
	PlayspecID     *int64         `json:"playspec_id"`
	GitDiff        string         `json:"git_diff"`
	Status         string         `json:"status"`
	FoundCommitSHA string         `json:"found_commit_sha"`
	CuredCommitSHA *string        `json:"cured_commit_sha"`
	Branch         string         `json:"branch"`
	Metadata       map[string]any `json:"metadata"`
	CureAttempts   *int64         `json:"cure_attempts"`
	CuredAt        *time.Time     `json:"cured_at"`
	KilledAt       *time.Time     `json:"killed_at"`
	CreatedAt      *time.Time     `json:"created_at"`
	UpdatedAt      *time.Time     `json:"updated_at"`
}

type MutationCreateParams struct {
	GitDiff        string         `json:"git_diff,omitempty"`
	FoundCommitSHA string         `json:"found_commit_sha,omitempty"`
	Branch         string         `json:"branch,omitempty"`
	PlayspecID     *int64         `json:"playspec_id,omitempty"`
	Diff           string         `json:"diff,omitempty"`
	CommitSHA      string         `json:"commit_sha,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type MutationUpdateParams struct {
	Status         *string `json:"status,omitempty"`
	CuredCommitSHA *string `json:"cured_commit_sha,omitempty"`
}

