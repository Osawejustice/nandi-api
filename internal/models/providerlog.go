package models

import (
	"time"

	"github.com/google/uuid"
)

// ProviderLog is an audit row for every outbound send attempt (including failover).
type ProviderLog struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	MessageID         *uuid.UUID `gorm:"type:uuid;index" json:"message_id,omitempty"`
	CampaignID        *uuid.UUID `gorm:"type:uuid;index" json:"campaign_id,omitempty"`
	Provider          string     `gorm:"size:64;not null;index" json:"provider"`
	Channel           string     `gorm:"size:32;not null" json:"channel"`
	To                string     `gorm:"size:64;not null" json:"to"`
	Attempt           int        `gorm:"not null;default:1" json:"attempt"`
	Status            string     `gorm:"size:32;not null;index" json:"status"`
	Success           bool       `gorm:"not null;index" json:"success"`
	ProviderMessageID string     `gorm:"size:191" json:"provider_message_id"`
	ErrorMessage      string     `gorm:"type:text" json:"error_message"`
	LatencyMS         int64      `json:"latency_ms"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (ProviderLog) TableName() string { return "provider_logs" }
