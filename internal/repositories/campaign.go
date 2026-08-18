package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
)

type CampaignRepo struct {
	db *gorm.DB
}

func NewCampaignRepo(db *gorm.DB) *CampaignRepo {
	return &CampaignRepo{db: db}
}

func (r *CampaignRepo) Create(ctx context.Context, c *models.Campaign) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CampaignRepo) Update(ctx context.Context, c *models.Campaign) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *CampaignRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Campaign, error) {
	var c models.Campaign
	if err := scoped(r.db.WithContext(ctx), tenantID).First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CampaignRepo) List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]models.Campaign, int64, error) {
	q := scoped(r.db.WithContext(ctx).Model(&models.Campaign{}), tenantID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.Campaign
	if err := q.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *CampaignRepo) ClaimDue(ctx context.Context, limit int) ([]models.Campaign, error) {
	var due []models.Campaign
	if err := r.db.WithContext(ctx).
		Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)", models.CampaignQueued, time.Now().UTC()).
		Order("created_at ASC").
		Limit(limit).
		Find(&due).Error; err != nil {
		return nil, err
	}
	claimed := make([]models.Campaign, 0, len(due))
	for i := range due {
		res := r.db.WithContext(ctx).Model(&models.Campaign{}).
			Where("id = ? AND status = ?", due[i].ID, models.CampaignQueued).
			Updates(map[string]any{"status": models.CampaignSending, "started_at": time.Now().UTC()})
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			continue
		}
		due[i].Status = models.CampaignSending
		claimed = append(claimed, due[i])
	}
	return claimed, nil
}

func (r *CampaignRepo) CreateRecipients(ctx context.Context, rows []models.CampaignRecipient) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *CampaignRepo) ListRecipients(ctx context.Context, tenantID, campaignID uuid.UUID) ([]models.CampaignRecipient, error) {
	var rows []models.CampaignRecipient
	if err := scoped(r.db.WithContext(ctx), tenantID).
		Where("campaign_id = ?", campaignID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *CampaignRepo) UpdateRecipient(ctx context.Context, row *models.CampaignRecipient) error {
	return r.db.WithContext(ctx).Save(row).Error
}

type SettingRepo struct {
	db *gorm.DB
}

func NewSettingRepo(db *gorm.DB) *SettingRepo {
	return &SettingRepo{db: db}
}

func (r *SettingRepo) Get(ctx context.Context, tenantID uuid.UUID) (*models.TenantSetting, error) {
	var s models.TenantSetting
	if err := r.db.WithContext(ctx).First(&s, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingRepo) Upsert(ctx context.Context, s *models.TenantSetting) error {
	if s.ID == uuid.Nil {
		s.ID = models.NewID()
	}
	return r.db.WithContext(ctx).Save(s).Error
}
