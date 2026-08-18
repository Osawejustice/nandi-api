package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken stores the SHA-256 of a refresh JWT so tokens can be rotated and revoked.
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (t RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && t.ExpiresAt.After(now)
}
