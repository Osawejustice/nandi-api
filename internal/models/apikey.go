package models

import (
	"time"

	"github.com/google/uuid"
)

// APIKey authenticates machine clients. The raw key is shown once at creation.
type APIKey struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Name       string     `gorm:"size:120;not null" json:"name"`
	KeyPrefix  string     `gorm:"size:16;not null" json:"key_prefix"`
	KeyHash    string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	Role       string     `gorm:"size:32;not null;default:admin" json:"role"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedBy  *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	Timestamps
	SoftDelete
}

func (APIKey) TableName() string { return "api_keys" }
