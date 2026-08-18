package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
)

type MessageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Create(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *MessageRepo) Update(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Save(msg).Error
}

func (r *MessageRepo) ListByConversation(ctx context.Context, tenantID, conversationID uuid.UUID) ([]models.Message, error) {
	var items []models.Message
	if err := scoped(r.db.WithContext(ctx), tenantID).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *MessageRepo) FindByProviderMessageID(ctx context.Context, tenantID uuid.UUID, provider, providerMessageID string) (*models.Message, error) {
	if providerMessageID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var msg models.Message
	if err := scoped(r.db.WithContext(ctx), tenantID).
		Where("provider = ? AND provider_message_id = ?", provider, providerMessageID).
		First(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *MessageRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Message, error) {
	var msg models.Message
	if err := scoped(r.db.WithContext(ctx), tenantID).First(&msg, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *MessageRepo) VolumeSince(ctx context.Context, tenantID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	err := scoped(r.db.WithContext(ctx).Model(&models.Message{}), tenantID).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

func (r *MessageRepo) AgentReplyCounts(ctx context.Context, tenantID uuid.UUID, since time.Time) ([]AgentMetric, error) {
	var rows []AgentMetric
	err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Select("sender_id as user_id, count(*) as replies").
		Where("tenant_id = ? AND direction = ? AND sender_id IS NOT NULL AND created_at >= ?", tenantID, models.DirectionOutbound, since).
		Group("sender_id").
		Scan(&rows).Error
	return rows, err
}

type AgentMetric struct {
	UserID  uuid.UUID `json:"user_id"`
	Replies int64     `json:"replies"`
}
