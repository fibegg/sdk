package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type ImportTemplateVersionService struct {
	client *Client
}

func (s *ImportTemplateVersionService) Delete(ctx context.Context, templateID int64, id int64) error {
	return s.DeleteByTemplateIdentifier(ctx, int64Identifier(templateID), id)
}

func (s *ImportTemplateVersionService) DeleteByTemplateIdentifier(ctx context.Context, templateIdentifier string, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("%s/versions/%d", identifierPath("/api/import_templates", templateIdentifier), id), nil, nil)
}
