package models

import "github.com/google/uuid"

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"

	MessageStatusPending   = "pending"
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusFailed    = "failed"
	MessageStatusReceived  = "received"
)

// Message is a single inbound or outbound item in a conversation.
type Message struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	ConversationID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"conversation_id"`
	ContactID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"contact_id"`
	SenderID          *uuid.UUID `gorm:"type:uuid" json:"sender_id,omitempty"`
	Direction         string     `gorm:"size:16;not null;index" json:"direction"`
	Channel           string     `gorm:"size:32;not null;index" json:"channel"`
	Body              string     `gorm:"type:text;not null" json:"body"`
	Status            string     `gorm:"size:32;not null;default:pending" json:"status"`
	Provider          string     `gorm:"size:64" json:"provider"`
	ProviderMessageID string     `gorm:"size:191;index" json:"provider_message_id"`
	SentimentScore    *float64   `json:"sentiment_score"`
	SentimentLabel    string     `gorm:"size:32" json:"sentiment_label"`
	Metadata          JSONMap    `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	ErrorMessage      string     `gorm:"type:text" json:"error_message,omitempty"`
	Timestamps
	SoftDelete

	Conversation *Conversation `gorm:"foreignKey:ConversationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (Message) TableName() string { return "messages" }
