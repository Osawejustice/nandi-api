package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/providers"
	"github.com/yourorg/nandi/internal/realtime"
	"github.com/yourorg/nandi/internal/repositories"
)

type CampaignService struct {
	db       *gorm.DB
	repo     *repositories.CampaignRepo
	contacts *repositories.ContactRepo
	router   *providers.Router
	hub      *realtime.Hub
	log      zerolog.Logger
}

func NewCampaignService(db *gorm.DB, router *providers.Router, hub *realtime.Hub, log zerolog.Logger) *CampaignService {
	s := &CampaignService{db: db, router: router, hub: hub, log: log}
	if db != nil {
		s.repo = repositories.NewCampaignRepo(db)
		s.contacts = repositories.NewContactRepo(db)
	}
	return s
}

func (s *CampaignService) ready() error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	return nil
}

type CreateCampaignInput struct {
	Name            string
	Channel         string
	MessageTemplate string
	Tag             string
	ScheduledAt     *time.Time
	CreatedBy       uuid.UUID
}

func (s *CampaignService) Create(ctx context.Context, tenantID uuid.UUID, in CreateCampaignInput) (*models.Campaign, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	in.MessageTemplate = strings.TrimSpace(in.MessageTemplate)
	if in.Name == "" || in.MessageTemplate == "" {
		return nil, ErrValidation
	}
	if in.Channel == "" {
		in.Channel = models.ChannelSMS
	}
	if !models.ValidChannel(in.Channel) {
		return nil, ErrUnsupportedChannel
	}

	filter := models.JSONMap{}
	if in.Tag != "" {
		filter["tag"] = in.Tag
	}

	c := &models.Campaign{
		ID:              models.NewID(),
		TenantID:        tenantID,
		CreatedBy:       in.CreatedBy,
		Name:            in.Name,
		Channel:         in.Channel,
		MessageTemplate: in.MessageTemplate,
		Status:          models.CampaignDraft,
		AudienceFilter:  filter,
		ScheduledAt:     in.ScheduledAt,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CampaignService) List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]models.Campaign, int64, error) {
	if err := s.ready(); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return s.repo.List(ctx, tenantID, page, perPage)
}

func (s *CampaignService) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.Campaign, error) {
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

func (s *CampaignService) Start(ctx context.Context, tenantID, id uuid.UUID) (*models.Campaign, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if c.Status != models.CampaignDraft && c.Status != models.CampaignFailed {
		return nil, ErrInvalidState
	}

	tag, _ := c.AudienceFilter["tag"].(string)
	audience, err := s.contacts.ListForAudience(ctx, tenantID, tag)
	if err != nil {
		return nil, err
	}
	if len(audience) == 0 {
		return nil, ErrValidation
	}

	rows := make([]models.CampaignRecipient, 0, len(audience))
	for _, contact := range audience {
		rows = append(rows, models.CampaignRecipient{
			ID:         models.NewID(),
			TenantID:   tenantID,
			CampaignID: c.ID,
			ContactID:  contact.ID,
			To:         contact.Phone,
			Status:     models.RecipientQueued,
			CreatedAt:  time.Now().UTC(),
		})
	}
	if err := s.repo.CreateRecipients(ctx, rows); err != nil {
		return nil, err
	}

	c.Status = models.CampaignQueued
	c.TotalCount = len(rows)
	c.SentCount = 0
	c.FailedCount = 0
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CampaignService) RunOnce(ctx context.Context) {
	if s.db == nil || s.router == nil {
		return
	}
	campaigns, err := s.repo.ClaimDue(ctx, 5)
	if err != nil {
		s.log.Error().Err(err).Msg("claim campaigns")
		return
	}
	for i := range campaigns {
		s.execute(ctx, &campaigns[i])
	}
}

func (s *CampaignService) StartWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.RunOnce(ctx)
			}
		}
	}()
	s.log.Info().Dur("interval", interval).Msg("campaign worker started")
}

func (s *CampaignService) execute(ctx context.Context, c *models.Campaign) {
	recipients, err := s.repo.ListRecipients(ctx, c.TenantID, c.ID)
	if err != nil {
		s.failCampaign(ctx, c, err)
		return
	}

	for i := range recipients {
		if recipients[i].Status == models.RecipientSent {
			continue
		}
		recipients[i].Status = models.RecipientSending
		_ = s.repo.UpdateRecipient(ctx, &recipients[i])

		result, sendErr := s.router.Send(ctx, providers.SendRequest{
			TenantID: c.TenantID,
			To:       recipients[i].To,
			Body:     c.MessageTemplate,
			Channel:  c.Channel,
		})
		sentAt := time.Now().UTC()
		if sendErr != nil {
			recipients[i].Status = models.RecipientFailed
			recipients[i].ErrorMessage = sendErr.Error()
			c.FailedCount++
		} else {
			recipients[i].Status = models.RecipientSent
			recipients[i].SentAt = &sentAt
			if result != nil {
				// provider message id recorded on provider_logs
			}
			c.SentCount++
		}
		_ = s.repo.UpdateRecipient(ctx, &recipients[i])
	}

	done := time.Now().UTC()
	c.CompletedAt = &done
	if c.FailedCount == c.TotalCount {
		c.Status = models.CampaignFailed
	} else {
		c.Status = models.CampaignCompleted
	}
	_ = s.repo.Update(ctx, c)

	if s.hub != nil {
		s.hub.Publish(ctx, realtime.Event{
			Event:    realtime.EventCampaignUpdated,
			TenantID: c.TenantID.String(),
			Payload:  map[string]any{"id": c.ID, "status": c.Status, "sent": c.SentCount, "failed": c.FailedCount},
		})
	}
}

func (s *CampaignService) failCampaign(ctx context.Context, c *models.Campaign, err error) {
	c.Status = models.CampaignFailed
	c.ErrorMessage = err.Error()
	_ = s.repo.Update(ctx, c)
}
