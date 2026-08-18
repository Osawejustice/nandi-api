package models

import "github.com/google/uuid"

// Tenant is the root isolation boundary. Every business row carries TenantID.
type Tenant struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name   string    `gorm:"size:255;not null" json:"name"`
	Slug   string    `gorm:"size:100;not null;index" json:"slug"`
	Status string    `gorm:"size:32;not null;default:active;index" json:"status"`
	Timestamps
	SoftDelete
}

func (Tenant) TableName() string { return "tenants" }
