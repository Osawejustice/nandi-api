package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant-scoped roles. Stored as strings so they stay readable in JWTs and the API.
const (
	RoleOwner = "owner"
	RoleAdmin = "admin"
	RoleAgent = "agent"
)

func ValidRole(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleAgent:
		return true
	default:
		return false
	}
}

const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
)

const (
	AuthTypeJWT    = "jwt"
	AuthTypeAPIKey = "api_key"
)

// NewID returns a random UUID for primary keys. Callers set this before Create
// so we do not depend on Postgres-only defaults.
func NewID() uuid.UUID {
	return uuid.New()
}

// SoftDelete is the shared timestamp set used by GORM soft deletes.
type SoftDelete struct {
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Timestamps are UTC create/update times. GORM fills these via NowFunc.
type Timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
