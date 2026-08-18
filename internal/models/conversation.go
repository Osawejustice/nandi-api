package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ConversationOpen     = "open"
	ConversationPending  = "pending"
	ConversationResolved = "resolved"
	ConversationClosed   = "closed"

	ChannelSMS      = "sms"
	ChannelWhatsApp = "whatsapp"
)

func ValidConversationStatus(status string) bool {
	switch status {
	case ConversationOpen, ConversationPending, ConversationResolved, ConversationClosed:
		return true
	default:
		return false
	}
}

func ValidChannel(channel string) bool {
	switch channel {
	case ChannelSMS, ChannelWhatsApp:
		return true
	default:
		return false
	}
}

// Conversation is a thread between a tenant and one contact on one channel.
type Conversation struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID           uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	ContactID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"contact_id"`
	AssigneeID         *uuid.UUID `gorm:"type:uuid;index" json:"assignee_id"`
	Status             string     `gorm:"size:32;not null;default:open;index" json:"status"`
	Channel            string     `gorm:"size:32;not null;index" json:"channel"`
	LastMessageAt      *time.Time `json:"last_message_at"`
	LastMessagePreview string     `gorm:"size:280" json:"last_message_preview"`
	UnreadCount        int        `gorm:"not null;default:0" json:"unread_count"`
	SentimentScore     *float64   `json:"sentiment_score"`
	SentimentLabel     string     `gorm:"size:32" json:"sentiment_label"`
	Summary            string     `gorm:"type:text" json:"summary"`
	Timestamps
	SoftDelete

	Contact  *Contact `gorm:"foreignKey:ContactID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"contact,omitempty"`
	Assignee *User    `gorm:"foreignKey:AssigneeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"assignee,omitempty"`
}

func (Conversation) TableName() string { return "conversations" }
