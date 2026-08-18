package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/repositories"
)

type AnalyticsService struct {
	db    *gorm.DB
	convs *repositories.ConversationRepo
	msgs  *repositories.MessageRepo
	users *repositories.UserRepo
	set   *repositories.SettingRepo
}

func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
	s := &AnalyticsService{db: db}
	if db != nil {
		s.convs = repositories.NewConversationRepo(db)
		s.msgs = repositories.NewMessageRepo(db)
		s.users = repositories.NewUserRepo(db)
		s.set = repositories.NewSettingRepo(db)
	}
	return s
}

type Overview struct {
	ConversationsByStatus  map[string]int64           `json:"conversations_by_status"`
	ConversationsByChannel map[string]int64           `json:"conversations_by_channel"`
	MessagesLast7Days      int64                      `json:"messages_last_7_days"`
	MessagesLast30Days     int64                      `json:"messages_last_30_days"`
	AgentRepliesLast7Days  []repositories.AgentMetric `json:"agent_replies_last_7_days"`
}

func (s *AnalyticsService) Overview(ctx context.Context, tenantID uuid.UUID) (*Overview, error) {
	if s.db == nil {
		return nil, ErrUnavailable
	}
	byStatus, err := s.convs.CountsByStatus(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	byChannel, err := s.convs.CountsByChannel(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	m7, err := s.msgs.VolumeSince(ctx, tenantID, time.Now().UTC().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	m30, err := s.msgs.VolumeSince(ctx, tenantID, time.Now().UTC().AddDate(0, 0, -30))
	if err != nil {
		return nil, err
	}
	agents, err := s.msgs.AgentReplyCounts(ctx, tenantID, time.Now().UTC().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	return &Overview{
		ConversationsByStatus:  byStatus,
		ConversationsByChannel: byChannel,
		MessagesLast7Days:      m7,
		MessagesLast30Days:     m30,
		AgentRepliesLast7Days:  agents,
	}, nil
}

func (s *AnalyticsService) GetSettings(ctx context.Context, tenantID uuid.UUID) (*models.TenantSetting, error) {
	if s.db == nil {
		return nil, ErrUnavailable
	}
	row, err := s.set.Get(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = &models.TenantSetting{
				ID:           models.NewID(),
				TenantID:     tenantID,
				FeatureFlags: models.JSONMap{"sentiment": true, "campaigns": true},
				Preferences:  models.JSONMap{},
			}
			if err := s.set.Upsert(ctx, row); err != nil {
				return nil, err
			}
			return row, nil
		}
		return nil, err
	}
	return row, nil
}

func (s *AnalyticsService) UpdateSettings(ctx context.Context, tenantID uuid.UUID, flags, prefs models.JSONMap) (*models.TenantSetting, error) {
	row, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if flags != nil {
		row.FeatureFlags = flags
	}
	if prefs != nil {
		row.Preferences = prefs
	}
	if err := s.set.Upsert(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *AnalyticsService) SetAgentStatus(ctx context.Context, tenantID, userID uuid.UUID, status string) error {
	if s.db == nil {
		return ErrUnavailable
	}
	switch status {
	case models.AgentStatusOnline, models.AgentStatusBusy, models.AgentStatusOffline:
	default:
		return ErrValidation
	}
	return s.users.UpdateAgentStatus(ctx, tenantID, userID, status)
}

func (s *AnalyticsService) ListAgents(ctx context.Context, tenantID uuid.UUID) ([]models.User, error) {
	if s.db == nil {
		return nil, ErrUnavailable
	}
	return s.users.ListByTenant(ctx, tenantID, "")
}
