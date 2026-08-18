package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
)

type TenantRepo struct {
	db *gorm.DB
}

func NewTenantRepo(db *gorm.DB) *TenantRepo {
	return &TenantRepo{db: db}
}

func (r *TenantRepo) Create(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *TenantRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).First(&tenant, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepo) FindBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).First(&tenant, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *TenantRepo) Update(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}
