package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	CampaignDraft     = "draft"
	CampaignQueued    = "queued"
	CampaignSending   = "sending"
	CampaignCompleted = "completed"
	CampaignFailed    = "failed"
	CampaignCancelled = "cancelled"

	RecipientQueued  = "queued"
	RecipientSending = "sending"
	RecipientSent    = "sent"
	RecipientFailed  = "failed"
)

// Campaign is a one-shot outbound blast to a filtered contact audience.
type Campaign struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	CreatedBy       uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	Name            string     `gorm:"size:255;not null" json:"name"`
	Channel         string     `gorm:"size:32;not null" json:"channel"`
	MessageTemplate string     `gorm:"type:text;not null" json:"message_template"`
	Status          string     `gorm:"size:32;not null;default:draft;index" json:"status"`
	AudienceFilter  JSONMap    `gorm:"type:jsonb;default:'{}'" json:"audience_filter"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	TotalCount      int        `gorm:"not null;default:0" json:"total_count"`
	SentCount       int        `gorm:"not null;default:0" json:"sent_count"`
	FailedCount     int        `gorm:"not null;default:0" json:"failed_count"`
	ErrorMessage    string     `gorm:"type:text" json:"error_message,omitempty"`
	Timestamps
	SoftDelete
}

func (Campaign) TableName() string { return "campaigns" }

// CampaignRecipient is one send job inside a campaign.
type CampaignRecipient struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	CampaignID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"campaign_id"`
	ContactID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"contact_id"`
	To           string     `gorm:"size:64;not null" json:"to"`
	Status       string     `gorm:"size:32;not null;default:queued;index" json:"status"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
	MessageID    *uuid.UUID `gorm:"type:uuid" json:"message_id,omitempty"`
	SentAt       *time.Time `json:"sent_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (CampaignRecipient) TableName() string { return "campaign_recipients" }
