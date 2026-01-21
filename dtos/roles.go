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
	Name           string   `json:"name" validate:"required"`
	Description    string   `json:"description" validate:"required"`
	PermissionKeys []string `json:"permission_keys" validate:"required,min=1,dive,required"`
}

type Permission struct {
	ID          string  `json:"permission_master_id"`
	Description *string `json:"description"`
	Category    string  `json:"category" validate:"required"`
	Key         string  `json:"key" validate:"required"`
}
type PermissionKeys struct {
	PermissionKeys []string `json:"permission_keys" validate:"required,min=1,dive,required"`
}

type AvailablePermission struct {
	Category    string `json:"category" validate:"required"`
	Key         string `json:"key" validate:"required"`
	Description string `json:"description" validate:"required"`
}

type UpdateAvailablePermission struct {
	Category       string `json:"category" validate:"required"`
	Key            string `json:"key" validate:"required"`
	Description    string `json:"description" validate:"required"`
	NewCategory    string `json:"new_category" validate:"required"`
	NewKey         string `json:"new_key" validate:"required"`
	NewDescription string `json:"new_description" validate:"required"`
}
