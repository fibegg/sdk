package fibe

import (
	"context"
	"net/http"
)

func (c *Client) OwnerContext(ctx context.Context) (*CredentialContext, error) {
	var result CredentialContext
	err := c.do(ctx, http.MethodGet, "/api/owner_context", nil, &result)
	return &result, err
}

func (c *Client) OwnerContexts(ctx context.Context) (*OwnerContextList, error) {
	var result OwnerContextList
	err := c.do(ctx, http.MethodGet, "/api/owner_contexts", nil, &result)
	return &result, err
}
