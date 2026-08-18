package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
)

type APIKeyRepo struct {
	db *gorm.DB
}

func NewAPIKeyRepo(db *gorm.DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

func (r *APIKeyRepo) Create(ctx context.Context, key *models.APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *APIKeyRepo) FindByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	var key models.APIKey
	if err := r.db.WithContext(ctx).First(&key, "key_hash = ?", hash).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepo) List(ctx context.Context, tenantID uuid.UUID) ([]models.APIKey, error) {
	var keys []models.APIKey
	if err := scoped(r.db.WithContext(ctx), tenantID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *APIKeyRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.APIKey, error) {
	var key models.APIKey
	if err := scoped(r.db.WithContext(ctx), tenantID).First(&key, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepo) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	res := scoped(r.db.WithContext(ctx), tenantID).Delete(&models.APIKey{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *APIKeyRepo) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&models.APIKey{}).Where("id = ?", id).Update("last_used_at", now).Error
}
