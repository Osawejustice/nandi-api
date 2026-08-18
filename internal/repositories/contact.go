package repositories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
)

type ContactRepo struct {
	db *gorm.DB
}

func NewContactRepo(db *gorm.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

func (r *ContactRepo) Create(ctx context.Context, contact *models.Contact) error {
	return r.db.WithContext(ctx).Create(contact).Error
}

func (r *ContactRepo) Update(ctx context.Context, contact *models.Contact) error {
	return r.db.WithContext(ctx).Save(contact).Error
}

func (r *ContactRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Contact, error) {
	var contact models.Contact
	if err := scoped(r.db.WithContext(ctx), tenantID).First(&contact, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepo) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*models.Contact, error) {
	var contact models.Contact
	if err := scoped(r.db.WithContext(ctx), tenantID).First(&contact, "phone = ?", phone).Error; err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepo) List(ctx context.Context, tenantID uuid.UUID, query string, tag string, page, perPage int) ([]models.Contact, int64, error) {
	q := scoped(r.db.WithContext(ctx).Model(&models.Contact{}), tenantID)
	if query != "" {
		like := "%" + strings.ToLower(query) + "%"
		q = q.Where("LOWER(name) LIKE ? OR phone LIKE ? OR LOWER(email) LIKE ?", like, "%"+query+"%", like)
	}
	if tag != "" {
		q = q.Where("? = ANY(tags)", tag)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.Contact
	if err := q.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ContactRepo) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	res := scoped(r.db.WithContext(ctx), tenantID).Delete(&models.Contact{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ContactRepo) ListForAudience(ctx context.Context, tenantID uuid.UUID, tag string) ([]models.Contact, error) {
	q := scoped(r.db.WithContext(ctx), tenantID).Where("phone <> ''")
	if tag != "" {
		q = q.Where("? = ANY(tags)", tag)
	}
	var items []models.Contact
	if err := q.Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
