package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Contact is a person a tenant has spoken with. Phone is the primary identifier
// for SMS/WhatsApp inbound matching.
type Contact struct {
	ID       uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Name     string         `gorm:"size:255;not null" json:"name"`
	Phone    string         `gorm:"size:32;not null" json:"phone"`
	Email    string         `gorm:"size:255" json:"email"`
	Tags     pq.StringArray `gorm:"type:text[]" json:"tags"`
	Metadata JSONMap        `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	Timestamps
	SoftDelete
}

func (Contact) TableName() string { return "contacts" }
