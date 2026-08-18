package models

import "github.com/google/uuid"

// TenantSetting stores feature flags and non-secret tenant preferences.
// Provider secrets stay in process environment / a secret manager.
type TenantSetting struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"tenant_id"`
	FeatureFlags JSONMap   `gorm:"type:jsonb;default:'{}'" json:"feature_flags"`
	Preferences  JSONMap   `gorm:"type:jsonb;default:'{}'" json:"preferences"`
	Timestamps
}

func (TenantSetting) TableName() string { return "tenant_settings" }
