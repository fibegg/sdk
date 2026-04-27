package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type ImportTemplateVersionService struct {
	client *Client
}

func (s *ImportTemplateVersionService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/import_template_versions/%d", id), nil, nil)
}
