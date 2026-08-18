package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/repositories"
	"github.com/yourorg/nandi/internal/utils"
)

type ContactService struct {
	db   *gorm.DB
	repo *repositories.ContactRepo
}

func NewContactService(db *gorm.DB) *ContactService {
	s := &ContactService{db: db}
	if db != nil {
		s.repo = repositories.NewContactRepo(db)
	}
	return s
}

type ContactInput struct {
	Name     string
	Phone    string
	Email    string
	Tags     []string
	Metadata models.JSONMap
}

func (s *ContactService) ready() error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	return nil
}

func (s *ContactService) Create(ctx context.Context, tenantID uuid.UUID, in ContactInput) (*models.Contact, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	phone := utils.NormalizePhone(in.Phone)
	name := strings.TrimSpace(in.Name)
	if phone == "" || name == "" {
		return nil, ErrValidation
	}
	if existing, err := s.repo.FindByPhone(ctx, tenantID, phone); err == nil {
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	c := &models.Contact{
		ID:       models.NewID(),
		TenantID: tenantID,
		Name:     name,
		Phone:    phone,
		Email:    utils.NormalizeEmail(in.Email),
		Tags:     pq.StringArray(in.Tags),
		Metadata: in.Metadata,
	}
	if c.Metadata == nil {
		c.Metadata = models.JSONMap{}
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ContactService) Update(ctx context.Context, tenantID, id uuid.UUID, in ContactInput) (*models.Contact, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	c, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		c.Name = name
	}
	if phone := utils.NormalizePhone(in.Phone); phone != "" {
		c.Phone = phone
	}
	if in.Email != "" {
		c.Email = utils.NormalizeEmail(in.Email)
	}
	if in.Tags != nil {
		c.Tags = pq.StringArray(in.Tags)
	}
	if in.Metadata != nil {
		c.Metadata = in.Metadata
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ContactService) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.Contact, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	c, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *ContactService) List(ctx context.Context, tenantID uuid.UUID, query, tag string, page, perPage int) ([]models.Contact, int64, error) {
	if err := s.ready(); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, tenantID, query, tag, page, perPage)
}

func (s *ContactService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, tenantID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *ContactService) FindOrCreateByPhone(ctx context.Context, tenantID uuid.UUID, phone, name string) (*models.Contact, error) {
	phone = utils.NormalizePhone(phone)
	if phone == "" {
		return nil, ErrValidation
	}
	c, err := s.repo.FindByPhone(ctx, tenantID, phone)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = phone
	}
	return s.Create(ctx, tenantID, ContactInput{Name: name, Phone: phone})
}
