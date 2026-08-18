package repositories

import (
	"context"

	"gorm.io/gorm"

	"github.com/Osawejustice/nandi-api/internal/models"
)

type ProviderLogRepo struct {
	db *gorm.DB
}

func NewProviderLogRepo(db *gorm.DB) *ProviderLogRepo {
	return &ProviderLogRepo{db: db}
}

func (r *ProviderLogRepo) Create(ctx context.Context, row *models.ProviderLog) error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.WithContext(ctx).Create(row).Error
}
