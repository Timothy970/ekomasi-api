package dtos

import "time"

type Role struct {
	RoleID      string    `json:"role_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Count       int       `json:"count"`
}
type RoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
