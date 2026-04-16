package fibe

import "time"

type Secret struct {
	ID          *int64     `json:"id"`
	Key         string     `json:"key"`
	Value       *string    `json:"value,omitempty"`
	Description *string    `json:"description"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type SecretCreateParams struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	Description *string `json:"description,omitempty"`
}

func (p *SecretCreateParams) Validate() error {
	v := &validator{}
	v.required("key", p.Key)
	v.secretKey("key", p.Key)
	v.required("value", p.Value)
	return v.err()
}

type SecretUpdateParams struct {
	Value       *string `json:"value,omitempty"`
	Description *string `json:"description,omitempty"`
}


type SecretListParams struct {
	Q             string `url:"q,omitempty"`
	Key           string `url:"key,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}
