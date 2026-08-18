package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/Osawejustice/nandi-api/internal/models"
)

type RegisterRequest struct {
	Organization string `json:"organization" binding:"required,min=2,max=120" example:"Acme Ltd"`
	Name         string `json:"name" binding:"required,min=1,max=120" example:"Jane Doe"`
	Email        string `json:"email" binding:"required,email" example:"jane@acme.com"`
	Password     string `json:"password" binding:"required,min=8,max=72" example:"changeme1"`
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email" example:"jane@acme.com"`
	Password   string `json:"password" binding:"required" example:"changeme1"`
	TenantSlug string `json:"tenant_slug" example:"acme-ltd"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required,min=1,max=120" example:"Billing worker"`
	Role string `json:"role" example:"admin"`
}

type TenantDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type UserDTO struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	AgentStatus string    `json:"agent_status"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuthData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type" example:"Bearer"`
	ExpiresIn    int64     `json:"expires_in" example:"900"`
	Tenant       TenantDTO `json:"tenant"`
	User         UserDTO   `json:"user"`
}

type MeData struct {
	AuthType string    `json:"auth_type"`
	Role     string    `json:"role"`
	Tenant   TenantDTO `json:"tenant"`
	User     *UserDTO  `json:"user"`
}

type APIKeyDTO struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Role       string     `json:"role"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type APIKeyCreatedData struct {
	APIKeyDTO
	Key string `json:"key"`
}

func tenantDTO(t *models.Tenant) TenantDTO {
	if t == nil {
		return TenantDTO{}
	}
	return TenantDTO{ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status, CreatedAt: t.CreatedAt}
}

func userDTO(u *models.User) UserDTO {
	if u == nil {
		return UserDTO{}
	}
	return UserDTO{
		ID: u.ID, TenantID: u.TenantID, Email: u.Email, Name: u.Name,
		Role: u.Role, AgentStatus: u.AgentStatus, CreatedAt: u.CreatedAt,
	}
}

type CreateUserRequest struct {
	Name     string `json:"name" binding:"required" example:"Amina Agent"`
	Email    string `json:"email" binding:"required,email" example:"amina@acme.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"changeme1"`
	Role     string `json:"role" example:"agent"`
}

type ContactDTO struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Phone     string         `json:"phone"`
	Email     string         `json:"email"`
	Tags      []string       `json:"tags"`
	Metadata  models.JSONMap `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type MessageDTO struct {
	ID                uuid.UUID      `json:"id"`
	ConversationID    uuid.UUID      `json:"conversation_id"`
	ContactID         uuid.UUID      `json:"contact_id"`
	SenderID          *uuid.UUID     `json:"sender_id,omitempty"`
	Direction         string         `json:"direction"`
	Channel           string         `json:"channel"`
	Body              string         `json:"body"`
	Status            string         `json:"status"`
	Provider          string         `json:"provider,omitempty"`
	ProviderMessageID string         `json:"provider_message_id,omitempty"`
	SentimentScore    *float64       `json:"sentiment_score"`
	SentimentLabel    string         `json:"sentiment_label"`
	Metadata          models.JSONMap `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
}

type ConversationDTO struct {
	ID                 uuid.UUID   `json:"id"`
	TenantID           uuid.UUID   `json:"tenant_id"`
	ContactID          uuid.UUID   `json:"contact_id"`
	AssigneeID         *uuid.UUID  `json:"assignee_id"`
	Status             string      `json:"status"`
	Channel            string      `json:"channel"`
	LastMessageAt      *time.Time  `json:"last_message_at"`
	LastMessagePreview string      `json:"last_message_preview"`
	UnreadCount        int         `json:"unread_count"`
	SentimentScore     *float64    `json:"sentiment_score"`
	SentimentLabel     string      `json:"sentiment_label"`
	Summary            string      `json:"summary,omitempty"`
	Contact            *ContactDTO `json:"contact,omitempty"`
	Assignee           *UserDTO    `json:"assignee,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type ConversationDetailDTO struct {
	Conversation ConversationDTO `json:"conversation"`
	Messages     []MessageDTO    `json:"messages"`
}

func contactDTO(c *models.Contact) ContactDTO {
	if c == nil {
		return ContactDTO{}
	}
	tags := []string(c.Tags)
	if tags == nil {
		tags = []string{}
	}
	meta := c.Metadata
	if meta == nil {
		meta = models.JSONMap{}
	}
	return ContactDTO{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Phone: c.Phone, Email: c.Email,
		Tags: tags, Metadata: meta, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func contactDTOs(items []models.Contact) []ContactDTO {
	out := make([]ContactDTO, 0, len(items))
	for i := range items {
		out = append(out, contactDTO(&items[i]))
	}
	return out
}

func messageDTO(m models.Message) MessageDTO {
	meta := m.Metadata
	if meta == nil {
		meta = models.JSONMap{}
	}
	return MessageDTO{
		ID: m.ID, ConversationID: m.ConversationID, ContactID: m.ContactID, SenderID: m.SenderID,
		Direction: m.Direction, Channel: m.Channel, Body: m.Body, Status: m.Status,
		Provider: m.Provider, ProviderMessageID: m.ProviderMessageID,
		SentimentScore: m.SentimentScore, SentimentLabel: m.SentimentLabel,
		Metadata: meta, CreatedAt: m.CreatedAt,
	}
}

func messageDTOs(items []models.Message) []MessageDTO {
	out := make([]MessageDTO, 0, len(items))
	for _, m := range items {
		out = append(out, messageDTO(m))
	}
	return out
}

func conversationDTO(c *models.Conversation) ConversationDTO {
	if c == nil {
		return ConversationDTO{}
	}
	out := ConversationDTO{
		ID: c.ID, TenantID: c.TenantID, ContactID: c.ContactID, AssigneeID: c.AssigneeID,
		Status: c.Status, Channel: c.Channel, LastMessageAt: c.LastMessageAt,
		LastMessagePreview: c.LastMessagePreview, UnreadCount: c.UnreadCount,
		SentimentScore: c.SentimentScore, SentimentLabel: c.SentimentLabel,
		Summary: c.Summary, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.Contact != nil {
		dto := contactDTO(c.Contact)
		out.Contact = &dto
	}
	if c.Assignee != nil {
		dto := userDTO(c.Assignee)
		out.Assignee = &dto
	}
	return out
}

func conversationDTOs(items []models.Conversation) []ConversationDTO {
	out := make([]ConversationDTO, 0, len(items))
	for i := range items {
		out = append(out, conversationDTO(&items[i]))
	}
	return out
}

func apiKeyDTO(k models.APIKey) APIKeyDTO {
	return APIKeyDTO{
		ID: k.ID, Name: k.Name, KeyPrefix: k.KeyPrefix, Role: k.Role,
		LastUsedAt: k.LastUsedAt, CreatedAt: k.CreatedAt,
	}
}
