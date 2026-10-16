package fibe

type OwnerContext struct {
	OwnerType string `json:"owner_type,omitempty"`
	OwnerID   int64  `json:"owner_id,omitempty"`
}

type OwnershipMetadata struct {
	OwnerContext
	CreatorID            *int64 `json:"creator_id,omitempty"`
	CreatorSubject       string `json:"creator_subject,omitempty"`
	AuthorizationVersion int    `json:"authorization_version,omitempty"`
}

type CredentialContext struct {
	OwnerContext
	PrincipalType        string `json:"principal_type,omitempty"`
	PrincipalID          int64  `json:"principal_id,omitempty"`
	AuthorizationVersion int    `json:"authorization_version,omitempty"`
	AuthorityRevision    *int64 `json:"authority_revision,omitempty"`
}

type OwnerContextList struct {
	Data                 []OwnerContextDescription `json:"data"`
	CurrentContext       OwnerContext              `json:"current_context"`
	PrincipalType        string                    `json:"principal_type"`
	PrincipalID          int64                     `json:"principal_id"`
	AuthorizationVersion int                       `json:"authorization_version"`
}

type OwnerContextDescription struct {
	OwnerContext
	Name              string `json:"name"`
	AuthorityRevision *int64 `json:"authority_revision,omitempty"`
}
