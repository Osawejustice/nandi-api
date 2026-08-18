package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/ai"
	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/providers"
	"github.com/yourorg/nandi/internal/realtime"
	"github.com/yourorg/nandi/internal/repositories"
	"github.com/yourorg/nandi/internal/utils"
)

type InboxService struct {
	db       *gorm.DB
	contacts *ContactService
	convs    *repositories.ConversationRepo
	msgs     *repositories.MessageRepo
	users    *repositories.UserRepo
	router   *providers.Router
	hub      *realtime.Hub
	analyzer ai.Analyzer
	log      zerolog.Logger
}

func NewInboxService(
	db *gorm.DB,
	contacts *ContactService,
	router *providers.Router,
	hub *realtime.Hub,
	analyzer ai.Analyzer,
	log zerolog.Logger,
) *InboxService {
	s := &InboxService{db: db, contacts: contacts, router: router, hub: hub, analyzer: analyzer, log: log}
	if db != nil {
		s.convs = repositories.NewConversationRepo(db)
		s.msgs = repositories.NewMessageRepo(db)
		s.users = repositories.NewUserRepo(db)
	}
	if s.analyzer == nil {
		s.analyzer = ai.NoopAnalyzer{}
	}
	return s
}

func (s *InboxService) ready() error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	return nil
}

type ListConversationsInput struct {
	Status     string
	Channel    string
	AssigneeID string
	Query      string
	Page       int
	PerPage    int
}

func (s *InboxService) ListConversations(ctx context.Context, tenantID uuid.UUID, in ListConversationsInput) ([]models.Conversation, int64, error) {
	if err := s.ready(); err != nil {
		return nil, 0, err
	}
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PerPage < 1 || in.PerPage > 100 {
		in.PerPage = 20
	}
	return s.convs.List(ctx, tenantID, repositories.ConversationFilter(in))
}

func (s *InboxService) GetConversation(ctx context.Context, tenantID, id uuid.UUID) (*models.Conversation, []models.Message, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	conv, err := s.convs.FindByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	msgs, err := s.msgs.ListByConversation(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	return conv, msgs, nil
}

type UpdateConversationInput struct {
	Status     *string
	AssigneeID *string
}

func (s *InboxService) UpdateConversation(ctx context.Context, tenantID, id uuid.UUID, in UpdateConversationInput) (*models.Conversation, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	conv, err := s.convs.FindByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if in.Status != nil {
		if !models.ValidConversationStatus(*in.Status) {
			return nil, ErrValidation
		}
		if !validTransition(conv.Status, *in.Status) {
			return nil, ErrInvalidState
		}
		conv.Status = *in.Status
	}
	if in.AssigneeID != nil {
		raw := strings.TrimSpace(*in.AssigneeID)
		if raw == "" || raw == "null" {
			conv.AssigneeID = nil
		} else {
			aid, err := uuid.Parse(raw)
			if err != nil {
				return nil, ErrValidation
			}
			if _, err := s.users.FindByID(ctx, tenantID, aid); err != nil {
				return nil, ErrNotFound
			}
			conv.AssigneeID = &aid
		}
	}
	if err := s.convs.Update(ctx, conv); err != nil {
		return nil, err
	}
	s.publish(ctx, tenantID, realtime.EventConversationUpdated, conv.ID, uuid.Nil, map[string]any{
		"status": conv.Status, "assignee_id": conv.AssigneeID,
	})
	return conv, nil
}

func validTransition(from, to string) bool {
	if from == to {
		return true
	}
	// MVP state machine: any non-closed status can move to any other, closed is terminal unless reopened to open.
	if from == models.ConversationClosed && to != models.ConversationOpen {
		return false
	}
	return models.ValidConversationStatus(to)
}

func (s *InboxService) Reply(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, conversationID uuid.UUID, body string) (*models.Message, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrValidation
	}
	conv, err := s.convs.FindByID(ctx, tenantID, conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if conv.Contact == nil {
		return nil, ErrNotFound
	}

	msg := &models.Message{
		ID:             models.NewID(),
		TenantID:       tenantID,
		ConversationID: conv.ID,
		ContactID:      conv.ContactID,
		SenderID:       actor,
		Direction:      models.DirectionOutbound,
		Channel:        conv.Channel,
		Body:           body,
		Status:         models.MessageStatusPending,
		Metadata:       models.JSONMap{},
	}
	if err := s.msgs.Create(ctx, msg); err != nil {
		return nil, err
	}

	if s.router != nil {
		result, sendErr := s.router.Send(ctx, providers.SendRequest{
			TenantID:  tenantID,
			To:        conv.Contact.Phone,
			Body:      body,
			Channel:   conv.Channel,
			MessageID: msg.ID,
		})
		if sendErr != nil {
			msg.Status = models.MessageStatusFailed
			msg.ErrorMessage = sendErr.Error()
			_ = s.msgs.Update(ctx, msg)
			return msg, ErrProvider
		}
		msg.Status = models.MessageStatusSent
		msg.Provider = result.Provider
		msg.ProviderMessageID = result.ProviderMessageID
		_ = s.msgs.Update(ctx, msg)
	} else {
		msg.Status = models.MessageStatusSent
		msg.Provider = "none"
		_ = s.msgs.Update(ctx, msg)
	}

	s.touchConversation(ctx, conv, msg, false)
	s.publish(ctx, tenantID, realtime.EventNewMessage, conv.ID, msg.ID, map[string]any{
		"direction": msg.Direction, "channel": msg.Channel, "status": msg.Status,
	})
	return msg, nil
}

type InboundInput struct {
	Phone             string
	Name              string
	Body              string
	Channel           string
	Provider          string
	ProviderMessageID string
}

func (s *InboxService) IngestInbound(ctx context.Context, tenantID uuid.UUID, in InboundInput) (*models.Conversation, *models.Message, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" || utils.NormalizePhone(in.Phone) == "" {
		return nil, nil, ErrValidation
	}
	if in.Channel == "" {
		in.Channel = models.ChannelSMS
	}
	if !models.ValidChannel(in.Channel) {
		return nil, nil, ErrUnsupportedChannel
	}

	if in.ProviderMessageID != "" {
		if existing, err := s.msgs.FindByProviderMessageID(ctx, tenantID, in.Provider, in.ProviderMessageID); err == nil {
			conv, cerr := s.convs.FindByID(ctx, tenantID, existing.ConversationID)
			if cerr != nil {
				return nil, existing, nil
			}
			return conv, existing, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
	}

	contact, err := s.contacts.FindOrCreateByPhone(ctx, tenantID, in.Phone, in.Name)
	if err != nil {
		return nil, nil, err
	}

	created := false
	conv, err := s.convs.FindOpenByContactChannel(ctx, tenantID, contact.ID, in.Channel)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
		conv = &models.Conversation{
			ID:        models.NewID(),
			TenantID:  tenantID,
			ContactID: contact.ID,
			Status:    models.ConversationOpen,
			Channel:   in.Channel,
		}
		if err := s.convs.Create(ctx, conv); err != nil {
			return nil, nil, err
		}
		conv.Contact = contact
		created = true
	} else {
		conv.Contact = contact
	}

	msg := &models.Message{
		ID:                models.NewID(),
		TenantID:          tenantID,
		ConversationID:    conv.ID,
		ContactID:         contact.ID,
		Direction:         models.DirectionInbound,
		Channel:           in.Channel,
		Body:              in.Body,
		Status:            models.MessageStatusReceived,
		Provider:          in.Provider,
		ProviderMessageID: in.ProviderMessageID,
		Metadata:          models.JSONMap{},
	}
	if err := s.msgs.Create(ctx, msg); err != nil {
		if repositories.IsUniqueViolation(err) && in.ProviderMessageID != "" {
			existing, findErr := s.msgs.FindByProviderMessageID(ctx, tenantID, in.Provider, in.ProviderMessageID)
			if findErr == nil {
				return conv, existing, nil
			}
		}
		return nil, nil, err
	}

	s.touchConversation(ctx, conv, msg, true)
	if created {
		s.publish(ctx, tenantID, realtime.EventConversationCreated, conv.ID, uuid.Nil, map[string]any{
			"contact_id": contact.ID, "channel": conv.Channel,
		})
	}
	s.publish(ctx, tenantID, realtime.EventNewMessage, conv.ID, msg.ID, map[string]any{
		"direction": msg.Direction, "channel": msg.Channel, "contact_id": contact.ID,
	})

	go s.scoreSentiment(msg.ID, conv.ID, tenantID, msg.Body)
	return conv, msg, nil
}

func (s *InboxService) touchConversation(ctx context.Context, conv *models.Conversation, msg *models.Message, inbound bool) {
	now := time.Now().UTC()
	conv.LastMessageAt = &now
	conv.LastMessagePreview = utils.Preview(msg.Body, 140)
	if inbound {
		conv.UnreadCount++
		if conv.Status == models.ConversationResolved || conv.Status == models.ConversationClosed {
			conv.Status = models.ConversationOpen
		}
	} else if conv.Status == models.ConversationOpen {
		conv.Status = models.ConversationPending
	}
	_ = s.convs.Update(ctx, conv)
}

func (s *InboxService) scoreSentiment(messageID, conversationID, tenantID uuid.UUID, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := s.analyzer.Analyze(ctx, body)
	if err != nil {
		s.log.Warn().Err(err).Msg("sentiment failed; leaving null")
		return
	}
	if res == nil {
		return
	}
	msg, err := s.msgs.FindByID(ctx, tenantID, messageID)
	if err != nil {
		return
	}
	msg.SentimentScore = &res.Score
	msg.SentimentLabel = res.Label
	_ = s.msgs.Update(ctx, msg)

	conv, err := s.convs.FindByID(ctx, tenantID, conversationID)
	if err != nil {
		return
	}
	conv.SentimentScore = &res.Score
	conv.SentimentLabel = res.Label
	_ = s.convs.Update(ctx, conv)

	s.publish(ctx, tenantID, realtime.EventConversationUpdated, conv.ID, uuid.Nil, map[string]any{
		"sentiment_label": res.Label, "sentiment_score": res.Score,
	})
}

func (s *InboxService) Summarize(ctx context.Context, tenantID, conversationID uuid.UUID) (string, error) {
	if err := s.ready(); err != nil {
		return "", err
	}
	conv, msgs, err := s.GetConversation(ctx, tenantID, conversationID)
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 {
		return "", ErrValidation
	}
	summarizer, ok := s.analyzer.(ai.Summarizer)
	if !ok {
		return "", ErrUnavailable
	}
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Direction)
		b.WriteString(": ")
		b.WriteString(m.Body)
		b.WriteString("\n")
	}
	summary, err := summarizer.Summarize(ctx, b.String())
	if err != nil {
		s.log.Error().Err(err).Msg("summary failed")
		return "", ErrUnavailable
	}
	conv.Summary = summary
	_ = s.convs.Update(ctx, conv)
	return summary, nil
}

func (s *InboxService) MarkRead(ctx context.Context, tenantID, conversationID uuid.UUID) error {
	if err := s.ready(); err != nil {
		return err
	}
	conv, err := s.convs.FindByID(ctx, tenantID, conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	conv.UnreadCount = 0
	return s.convs.Update(ctx, conv)
}

func (s *InboxService) publish(ctx context.Context, tenantID uuid.UUID, event string, conversationID, messageID uuid.UUID, payload any) {
	if s.hub == nil {
		return
	}
	evt := realtime.Event{
		Event:    event,
		TenantID: tenantID.String(),
		Payload:  payload,
	}
	if conversationID != uuid.Nil {
		evt.ConversationID = conversationID.String()
	}
	if messageID != uuid.Nil {
		evt.MessageID = messageID.String()
	}
	s.hub.Publish(ctx, evt)
}
