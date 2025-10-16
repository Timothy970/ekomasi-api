package dtos

import "time"

type Role struct {
	RoleID      string        `json:"role_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Count       int           `json:"count"`
	Permissions *[]Permission `json:"permissions,omitempty"`
}
type RoleRequest struct {
	Name          string   `json:"name" validate:"required"`
	Description   string   `json:"description" validate:"required"`
	PermissionIDs []string `json:"permission_ids" validate:"required,min=1,dive,required"`
}

type Permission struct {
	ID          string  `json:"permission_id"`
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description"`
}
type PermissionIDs struct {
	PermissionIDs []string `json:"permission_ids" validate:"required,min=1,dive,required"`
}
