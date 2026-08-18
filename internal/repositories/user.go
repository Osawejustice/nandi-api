package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Osawejustice/nandi-api/internal/models"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := scoped(r.db.WithContext(ctx), tenantID).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*models.User, error) {
	var user models.User
	if err := scoped(r.db.WithContext(ctx), tenantID).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindAllByEmail(ctx context.Context, email string) ([]models.User, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) EmailTaken(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	var count int64
	if err := scoped(r.db.WithContext(ctx).Model(&models.User{}), tenantID).
		Where("email = ?", email).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, role string) ([]models.User, error) {
	var users []models.User
	q := scoped(r.db.WithContext(ctx), tenantID)
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if err := q.Order("created_at ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepo) UpdateAgentStatus(ctx context.Context, tenantID, userID uuid.UUID, status string) error {
	return scoped(r.db.WithContext(ctx).Model(&models.User{}), tenantID).
		Where("id = ?", userID).
		Update("agent_status", status).Error
}
