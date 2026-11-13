package models

import "time"

// Role constants matching the CHECK constraint on users.role.
const (
	RoleRequester = "requester"
	RoleAgent     = "agent"
	RoleAdmin     = "admin"
)

// User is the application-level user model. password_hash is intentionally
// omitted from JSON so it can never leak through an API response.
type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}
