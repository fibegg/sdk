package fibe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// TrickService provides operations on job-mode playgrounds (tricks).
// Tricks are ad-hoc workloads that run to completion, as opposed to
// long-running playground environments.
type TrickService struct {
	client *Client
}

// List returns only job-mode playgrounds (tricks).
func (s *TrickService) List(ctx context.Context, params *PlaygroundListParams) (*ListResult[Playground], error) {
	if params == nil {
		params = &PlaygroundListParams{}
	}
	t := true
	params.JobMode = &t
	path := "/api/playgrounds" + buildQuery(params)
	return doList[Playground](s.client, ctx, path)
}

// Get returns detailed information about a trick by ID.
func (s *TrickService) Get(ctx context.Context, id int64) (*Playground, error) {
	return s.client.Playgrounds.Get(ctx, id)
}

// Trigger creates a new trick run from a job-mode playspec.
// If params.Name is empty, a name is auto-generated as "{playspec-name}-{random}".
func (s *TrickService) Trigger(ctx context.Context, params *TrickTriggerParams) (*Playground, error) {
	name := params.Name
	if name == "" {
		// Fetch playspec name for auto-generation
		spec, err := s.client.Playspecs.Get(ctx, params.PlayspecID)
		if err != nil {
			return nil, fmt.Errorf("fibe: fetch playspec for trick name: %w", err)
		}
		name = spec.Name + "-" + randomHex(4)
	}

	createParams := &PlaygroundCreateParams{
		Name:       name,
		PlayspecID: params.PlayspecID,
		MarqueeID:  params.MarqueeID,
	}

	return s.client.Playgrounds.Create(ctx, createParams)
}

// Rerun creates a new trick run by copying the playspec and marquee
// from an existing trick.
func (s *TrickService) Rerun(ctx context.Context, sourceID int64) (*Playground, error) {
	source, err := s.client.Playgrounds.Get(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("fibe: fetch source trick for rerun: %w", err)
	}

	if source.PlayspecID == nil {
		return nil, fmt.Errorf("fibe: source trick %d has no playspec", sourceID)
	}

	// Fetch playspec to get the name for auto-generation
	spec, err := s.client.Playspecs.Get(ctx, *source.PlayspecID)
	if err != nil {
		return nil, fmt.Errorf("fibe: fetch playspec for rerun name: %w", err)
	}

	createParams := &PlaygroundCreateParams{
		Name:       spec.Name + "-" + randomHex(4),
		PlayspecID: *source.PlayspecID,
		MarqueeID:  source.MarqueeID,
	}

	return s.client.Playgrounds.Create(ctx, createParams)
}

// Delete deletes a trick.
func (s *TrickService) Delete(ctx context.Context, id int64) error {
	return s.client.Playgrounds.Delete(ctx, id)
}

// Status returns the current status and job result for a trick.
func (s *TrickService) Status(ctx context.Context, id int64) (*PlaygroundStatus, error) {
	return s.client.Playgrounds.Status(ctx, id)
}

// Logs returns logs for a specific service in a trick.
func (s *TrickService) Logs(ctx context.Context, id int64, service string, tail *int) (*PlaygroundLogs, error) {
	return s.client.Playgrounds.Logs(ctx, id, service, tail)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
