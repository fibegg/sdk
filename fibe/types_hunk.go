package fibe

import "time"

type Hunk struct {
	ID             *int64         `json:"id"`
	PropID         int64          `json:"prop_id"`
	CommitSHA      string         `json:"commit_sha"`
	ParentSHA      *string        `json:"parent_sha"`
	AuthorName     string         `json:"author_name"`
	AuthorEmail    string         `json:"author_email"`
	CommittedAt    time.Time      `json:"committed_at"`
	CommitMessage  *string        `json:"commit_message"`
	FilePath       string         `json:"file_path"`
	ChangeType     string         `json:"change_type"`
	OldFilePath    *string        `json:"old_file_path"`
	HunkIndex      int            `json:"hunk_index"`
	Additions      *int           `json:"additions"`
	Deletions      *int           `json:"deletions"`
	Status         string         `json:"status"`
	ProcessorName  *string        `json:"processor_name"`
	ProcessedAt    *time.Time     `json:"processed_at"`
	Ordinal        int            `json:"ordinal"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      *time.Time     `json:"created_at"`

	// Detail fields
	DiffContent   string `json:"diff_content,omitempty"`
	DiffSizeBytes int    `json:"diff_size_bytes,omitempty"`
}

type HunkListParams struct {
	FilePath        string `url:"file_path,omitempty"`
	ChangeType      string `url:"change_type,omitempty"`
	AuthorEmail     string `url:"author_email,omitempty"`
	AuthorName      string `url:"author_name,omitempty"`
	CommitSHA       string `url:"commit_sha,omitempty"`
	Status          string `url:"status,omitempty"`
	ProcessorName   string `url:"processor_name,omitempty"`
	CommittedAfter  string `url:"committed_after,omitempty"`
	CommittedBefore string `url:"committed_before,omitempty"`
	CreatedAfter    string `url:"created_after,omitempty"`
	CreatedBefore   string `url:"created_before,omitempty"`
	Sort            string `url:"sort,omitempty"`
	Page            int    `url:"page,omitempty"`
	PerPage         int    `url:"per_page,omitempty"`
}

type HunkUpdateParams struct {
	Status        *string        `json:"status,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	SkipReason    *string        `json:"skip_reason,omitempty"`
	ProcessorName *string        `json:"processor_name,omitempty"`
}

