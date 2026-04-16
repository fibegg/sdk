package fibe

import "time"

type Team struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	CreatorID    int64      `json:"creator_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MembersCount int64     `json:"members_count"`

	// Detail fields
	Memberships []TeamMembership `json:"memberships,omitempty"`
	Resources   []TeamResource   `json:"resources,omitempty"`
}

type TeamMembership struct {
	ID           int64   `json:"id"`
	PlayerID     int64   `json:"player_id"`
	GithubHandle *string `json:"github_handle"`
	Username     *string `json:"username"`
	Role         string  `json:"role"`
	Status       string  `json:"status"`
}

type TeamResource struct {
	ID                 int64   `json:"id"`
	ResourceType       string  `json:"resource_type"`
	ResourceID         int64   `json:"resource_id"`
	ResourceName       string  `json:"resource_name"`
	PermissionLevel    string  `json:"permission_level"`
	ContributedByID    *int64  `json:"contributed_by_id"`
	ContributedByHandle *string `json:"contributed_by_handle"`
}

type TeamCreateParams struct {
	Name string `json:"name"`
}

type TeamUpdateParams struct {
	Name string `json:"name"`
}

type TeamResourceParams struct {
	ResourceType    string `json:"resource_type"`
	ResourceID      int64  `json:"resource_id"`
	PermissionLevel string `json:"permission_level,omitempty"`
}


type TeamListParams struct {
	Q       string `url:"q,omitempty"`
	Name    string `url:"name,omitempty"`
	Sort    string `url:"sort,omitempty"`
	Page    int    `url:"page,omitempty"`
	PerPage int    `url:"per_page,omitempty"`
}
