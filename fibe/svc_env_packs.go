package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type EnvPackService struct{ client *Client }

func (s *EnvPackService) List(ctx context.Context, params *EnvPackListParams) (*ListResult[EnvPack], error) {
	return doList[EnvPack](s.client, ctx, "/api/env_packs"+buildQuery(params))
}
func (s *EnvPackService) Get(ctx context.Context, id int64) (*EnvPack, error) {
	var out EnvPack
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/env_packs/%d", id), nil, &out)
	return &out, err
}
func (s *EnvPackService) Create(ctx context.Context, params *EnvPackCreateParams) (*EnvPack, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var out EnvPack
	err := s.client.do(ctx, http.MethodPost, "/api/env_packs", map[string]any{"env_pack": params}, &out)
	return &out, err
}
func (s *EnvPackService) Update(ctx context.Context, id int64, params *EnvPackUpdateParams) (*EnvPack, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var out EnvPack
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/env_packs/%d", id), map[string]any{"env_pack": params}, &out)
	return &out, err
}
func (s *EnvPackService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/env_packs/%d", id), nil, nil)
}
func envPackTargetPath(target string, id int64) (string, error) {
	if target != "playgrounds" && target != "specs" {
		return "", fmt.Errorf("ENV-pack target must be playgrounds or specs")
	}
	if id <= 0 {
		return "", fmt.Errorf("target ID must be positive")
	}
	return fmt.Sprintf("/api/%s/%d/env_pack_attachments", target, id), nil
}
func (s *EnvPackService) Attachments(ctx context.Context, target string, id int64) (*EnvPackTarget, error) {
	path, err := envPackTargetPath(target, id)
	if err != nil {
		return nil, err
	}
	var out EnvPackTarget
	err = s.client.do(ctx, http.MethodGet, path, nil, &out)
	return &out, err
}
func (s *EnvPackService) ReplaceAttachments(ctx context.Context, target string, id int64, params *EnvPackAttachmentsParams) (*EnvPackTarget, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	path, err := envPackTargetPath(target, id)
	if err != nil {
		return nil, err
	}
	var out EnvPackTarget
	err = s.client.do(ctx, http.MethodPatch, path, params, &out)
	return &out, err
}
func (s *EnvPackService) RenewGrant(ctx context.Context, target string, id, attachmentID int64) (*EnvPackTarget, error) {
	path, err := envPackTargetPath(target, id)
	if err != nil {
		return nil, err
	}
	if attachmentID <= 0 {
		return nil, fmt.Errorf("attachment ID must be positive")
	}
	var out EnvPackTarget
	err = s.client.do(ctx, http.MethodPost, path+"/renew", map[string]any{"attachment_id": attachmentID}, &out)
	return &out, err
}
func (s *PlaygroundService) Rerun(ctx context.Context, id int64, params *PlaygroundRerunParams) (*Playground, error) {
	if params == nil {
		params = &PlaygroundRerunParams{}
	}
	if err := (&EnvPackAttachmentsParams{Attachments: params.EnvPackAttachments}).Validate(); err != nil {
		return nil, err
	}
	var out Playground
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/playgrounds/%d/rerun", id), params, &out)
	return &out, err
}

func (s *EnvPackService) attachmentInputs(ctx context.Context, target string, id int64) ([]EnvPackAttachmentInput, error) {
	current, err := s.Attachments(ctx, target, id)
	if err != nil {
		return nil, err
	}
	rows := make([]EnvPackAttachmentInput, 0, len(current.EnvPackAttachments))
	for _, row := range current.EnvPackAttachments {
		var names *[]string
		if row.ServiceNames != nil {
			copy := append([]string{}, row.ServiceNames...)
			names = &copy
		}
		rows = append(rows, EnvPackAttachmentInput{EnvPackID: row.EnvPackID, ServiceNames: names})
	}
	return rows, nil
}

// ReorderAttachments requires every existing pack exactly once and preserves grants.
func (s *EnvPackService) ReorderAttachments(ctx context.Context, target string, id int64, order []int64) (*EnvPackTarget, error) {
	rows, err := s.attachmentInputs(ctx, target, id)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(order) {
		return nil, fmt.Errorf("order must contain every attached pack exactly once")
	}
	byID := map[int64]EnvPackAttachmentInput{}
	for _, row := range rows {
		byID[row.EnvPackID] = row
	}
	out := make([]EnvPackAttachmentInput, 0, len(order))
	for _, packID := range order {
		row, ok := byID[packID]
		if !ok {
			return nil, fmt.Errorf("order contains a duplicate or unattached pack")
		}
		out = append(out, row)
		delete(byID, packID)
	}
	return s.ReplaceAttachments(ctx, target, id, &EnvPackAttachmentsParams{Attachments: &out})
}
func (s *EnvPackService) DetachAttachment(ctx context.Context, target string, id, packID int64) (*EnvPackTarget, error) {
	rows, err := s.attachmentInputs(ctx, target, id)
	if err != nil {
		return nil, err
	}
	out := make([]EnvPackAttachmentInput, 0, len(rows))
	found := false
	for _, row := range rows {
		if row.EnvPackID == packID {
			found = true
		} else {
			out = append(out, row)
		}
	}
	if !found {
		return nil, fmt.Errorf("pack is not attached")
	}
	return s.ReplaceAttachments(ctx, target, id, &EnvPackAttachmentsParams{Attachments: &out})
}

// RetargetAttachment explicitly renews the selected grant through server authority.
func (s *EnvPackService) RetargetAttachment(ctx context.Context, target string, id, packID int64, names *[]string) (*EnvPackTarget, error) {
	rows, err := s.attachmentInputs(ctx, target, id)
	if err != nil {
		return nil, err
	}
	found := false
	for i := range rows {
		if rows[i].EnvPackID == packID {
			rows[i].ServiceNames = names
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("pack is not attached")
	}
	return s.ReplaceAttachments(ctx, target, id, &EnvPackAttachmentsParams{Attachments: &rows})
}
