package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole enumerates platform-level roles.
type UserRole string

const (
	UserRoleSuperAdmin UserRole = "superadmin"
	UserRoleAdmin      UserRole = "admin"
	UserRoleUser       UserRole = "user"
)

// AuthProvider enumerates login providers.
type AuthProvider string

const (
	AuthProviderEmail  AuthProvider = "email"
	AuthProviderGoogle AuthProvider = "google"
)

// User is a platform user account.
type User struct {
	ID            uuid.UUID    `json:"id"`
	Email         string       `json:"email"`
	PasswordHash  *string      `json:"-"`
	FirstName     *string      `json:"first_name,omitempty"`
	LastName      *string      `json:"last_name,omitempty"`
	Phone         *string      `json:"phone,omitempty"`
	AvatarURL     *string      `json:"avatar_url,omitempty"`
	Role          UserRole     `json:"role"`
	GoogleSub     *string      `json:"-"`
	AuthProvider  AuthProvider `json:"auth_provider"`
	EmailVerified bool         `json:"email_verified"`
	IsActive      bool         `json:"is_active"`
	LastLoginAt   *time.Time   `json:"last_login_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// FullName returns a display name.
func (u *User) FullName() string {
	if u.FirstName != nil && u.LastName != nil {
		return *u.FirstName + " " + *u.LastName
	}
	if u.FirstName != nil {
		return *u.FirstName
	}
	return u.Email
}
