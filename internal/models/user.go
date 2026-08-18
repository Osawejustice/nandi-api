package models

import "github.com/google/uuid"

// User belongs to exactly one tenant. Email is unique per tenant among live rows.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID     uuid.UUID `gorm:"type:uuid;not null;index;index:idx_users_tenant_email,priority:1" json:"tenant_id"`
	Email        string    `gorm:"size:255;not null;index:idx_users_tenant_email,priority:2" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	Role         string    `gorm:"size:32;not null;index" json:"role"`
	AgentStatus  string    `gorm:"size:32;not null;default:offline" json:"agent_status"`
	Timestamps
	SoftDelete

	Tenant *Tenant `gorm:"foreignKey:TenantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}

func (User) TableName() string { return "users" }

const (
	AgentStatusOnline  = "online"
	AgentStatusBusy    = "busy"
	AgentStatusOffline = "offline"
)
