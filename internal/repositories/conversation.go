package repositories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Osawejustice/nandi-api/internal/models"
)

type ConversationRepo struct {
	db *gorm.DB
}

func NewConversationRepo(db *gorm.DB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

func (r *ConversationRepo) Create(ctx context.Context, conv *models.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

func (r *ConversationRepo) Update(ctx context.Context, conv *models.Conversation) error {
	return r.db.WithContext(ctx).Save(conv).Error
}

func (r *ConversationRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	if err := scoped(r.db.WithContext(ctx), tenantID).
		Preload("Contact").
		Preload("Assignee").
		First(&conv, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepo) FindOpenByContactChannel(ctx context.Context, tenantID, contactID uuid.UUID, channel string) (*models.Conversation, error) {
	var conv models.Conversation
	err := scoped(r.db.WithContext(ctx), tenantID).
		Where("contact_id = ? AND channel = ? AND status IN ?", contactID, channel, []string{
			models.ConversationOpen, models.ConversationPending,
		}).
		Order("last_message_at DESC NULLS LAST").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

type ConversationFilter struct {
	Status     string
	Channel    string
	AssigneeID string
	Query      string
	Page       int
	PerPage    int
}

func (r *ConversationRepo) List(ctx context.Context, tenantID uuid.UUID, f ConversationFilter) ([]models.Conversation, int64, error) {
	q := scoped(r.db.WithContext(ctx).Model(&models.Conversation{}), tenantID)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Channel != "" {
		q = q.Where("channel = ?", f.Channel)
	}
	if f.AssigneeID == "unassigned" {
		q = q.Where("assignee_id IS NULL")
	} else if f.AssigneeID != "" {
		q = q.Where("assignee_id = ?", f.AssigneeID)
	}
	if strings.TrimSpace(f.Query) != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(f.Query)) + "%"
		q = q.Where(
			"id IN (SELECT conversations.id FROM conversations JOIN contacts ON contacts.id = conversations.contact_id WHERE conversations.tenant_id = ? AND (LOWER(contacts.name) LIKE ? OR contacts.phone LIKE ? OR LOWER(conversations.last_message_preview) LIKE ?))",
			tenantID, like, "%"+strings.TrimSpace(f.Query)+"%", like,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Conversation
	if err := q.Preload("Contact").Preload("Assignee").
		Order("COALESCE(last_message_at, created_at) DESC").
		Offset((f.Page - 1) * f.PerPage).
		Limit(f.PerPage).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ConversationRepo) CountsByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := scoped(r.db.WithContext(ctx).Model(&models.Conversation{}), tenantID).
		Select("status, count(*) as count").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Status] = r.Count
	}
	return out, nil
}

func (r *ConversationRepo) CountsByChannel(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	type row struct {
		Channel string
		Count   int64
	}
	var rows []row
	if err := scoped(r.db.WithContext(ctx).Model(&models.Conversation{}), tenantID).
		Select("channel, count(*) as count").
		Group("channel").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Channel] = r.Count
	}
	return out, nil
}
