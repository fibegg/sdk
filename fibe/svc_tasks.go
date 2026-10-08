package fibe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// TaskService provides operations on job-mode playgrounds (tasks).
// Tasks are ad-hoc workloads that run to completion, as opposed to
// long-running playground environments.
type TaskService struct {
	client *Client
}

// List returns only job-mode playgrounds (tasks).
func (s *TaskService) List(ctx context.Context, params *PlaygroundListParams) (*ListResult[Playground], error) {
	if params == nil {
		params = &PlaygroundListParams{}
	}
	t := true
	params.JobMode = &t
	path := "/api/playgrounds" + buildQuery(params)
	return doList[Playground](s.client, ctx, path)
}

// Get returns detailed information about a task by ID.
func (s *TaskService) Get(ctx context.Context, id int64) (*Playground, error) {
	return s.client.Playgrounds.Get(ctx, id)
}

// GetByIdentifier returns detailed information about a task by numeric ID or slug-safe name.
func (s *TaskService) GetByIdentifier(ctx context.Context, identifier string) (*Playground, error) {
	return s.client.Playgrounds.GetByIdentifier(ctx, identifier)
}

// Trigger creates a new task run from a job-mode spec.
// If params.Name is empty, a name is auto-generated as "{spec-name}-{random}".
func (s *TaskService) Trigger(ctx context.Context, params *TaskTriggerParams) (*Playground, error) {
	name := params.Name
	if name == "" {
		spec, err := s.client.Specs.GetByIdentifier(ctx, params.specIdentifier())
		if err != nil {
			return nil, fmt.Errorf("fibe: fetch spec for task name: %w", err)
		}
		name = spec.Name + "-" + randomHex(4)
	}

	createParams := &PlaygroundCreateParams{
		Name:               name,
		SpecID:             params.SpecID,
		SpecIdentifier:     params.SpecIdentifier,
		HostID:             params.HostID,
		HostIdentifier:     params.HostIdentifier,
		EnvOverrides:       params.EnvOverrides,
		OnlyServices:       params.OnlyServices,
		ExceptServices:     params.ExceptServices,
		EnvPackAttachments: params.EnvPackAttachments,
	}

	return s.client.Playgrounds.Create(ctx, createParams)
}

// Rerun creates a new task run by copying the spec and host
// from an existing task.
func (s *TaskService) Rerun(ctx context.Context, sourceID int64) (*Playground, error) {
	return s.RerunByIdentifier(ctx, int64Identifier(sourceID))
}

// RerunByIdentifier creates a new task run by copying the spec and host
// from an existing task identified by numeric ID or slug-safe name.
func (s *TaskService) RerunByIdentifier(ctx context.Context, sourceIdentifier string) (*Playground, error) {
	return s.RerunWithParamsByIdentifier(ctx, sourceIdentifier, nil)
}

// RerunWithParamsByIdentifier delegates source attachment inheritance to the server.
func (s *TaskService) RerunWithParamsByIdentifier(ctx context.Context, sourceIdentifier string, params *PlaygroundRerunParams) (*Playground, error) {
	source, err := s.client.Playgrounds.GetByIdentifier(ctx, sourceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("fibe: fetch source task for rerun: %w", err)
	}

	return s.client.Playgrounds.Rerun(ctx, source.ID, params)
}

// Delete deletes a task.
func (s *TaskService) Delete(ctx context.Context, id int64) error {
	return s.client.Playgrounds.Delete(ctx, id)
}

func (s *TaskService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.Playgrounds.DeleteByIdentifier(ctx, identifier)
}

// Status returns the current status and job result for a task.
func (s *TaskService) Status(ctx context.Context, id int64) (*PlaygroundStatus, error) {
	return s.client.Playgrounds.Status(ctx, id)
}

func (s *TaskService) StatusByIdentifier(ctx context.Context, identifier string) (*PlaygroundStatus, error) {
	return s.client.Playgrounds.StatusByIdentifier(ctx, identifier)
}

// Logs returns logs for a task. Empty service returns all services.
func (s *TaskService) Logs(ctx context.Context, id int64, service string, tail *int) (*PlaygroundLogs, error) {
	return s.client.Playgrounds.Logs(ctx, id, service, tail)
}

func (s *TaskService) LogsByIdentifier(ctx context.Context, identifier string, service string, tail *int) (*PlaygroundLogs, error) {
	return s.client.Playgrounds.LogsByIdentifier(ctx, identifier, service, tail)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
