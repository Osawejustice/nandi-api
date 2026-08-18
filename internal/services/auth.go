package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/repositories"
	"github.com/yourorg/nandi/internal/utils"
)

type RegisterInput struct {
	Organization string
	Name         string
	Email        string
	Password     string
}

type LoginInput struct {
	Email      string
	Password   string
	TenantSlug string
}

type TenantChoice struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type MultiTenantError struct {
	Tenants []TenantChoice
}

func (e *MultiTenantError) Error() string {
	return "email belongs to multiple tenants; provide tenant_slug"
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *models.User
	Tenant       *models.Tenant
}

type Principal struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	Role     string
	AuthType string
	APIKeyID uuid.UUID
	HasUser  bool
}

type AuthService struct {
	db      *gorm.DB
	jwt     *utils.JWTManager
	log     zerolog.Logger
	users   *repositories.UserRepo
	tenants *repositories.TenantRepo
	tokens  *repositories.RefreshTokenRepo
	keys    *repositories.APIKeyRepo
}

func NewAuthService(db *gorm.DB, jwt *utils.JWTManager, log zerolog.Logger) *AuthService {
	svc := &AuthService{db: db, jwt: jwt, log: log}
	if db != nil {
		svc.users = repositories.NewUserRepo(db)
		svc.tenants = repositories.NewTenantRepo(db)
		svc.tokens = repositories.NewRefreshTokenRepo(db)
		svc.keys = repositories.NewAPIKeyRepo(db)
	}
	return svc
}

func (s *AuthService) ready() error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	return nil
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}

	email := utils.NormalizeEmail(in.Email)
	name := strings.TrimSpace(in.Name)
	org := strings.TrimSpace(in.Organization)
	if org == "" || name == "" || email == "" || len(in.Password) < 8 {
		return nil, ErrValidation
	}

	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	slug, err := s.uniqueSlug(ctx, utils.Slugify(org))
	if err != nil {
		return nil, err
	}

	tenant := &models.Tenant{
		ID:     models.NewID(),
		Name:   org,
		Slug:   slug,
		Status: models.TenantStatusActive,
	}
	user := &models.User{
		ID:           models.NewID(),
		TenantID:     tenant.ID,
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         models.RoleOwner,
		AgentStatus:  models.AgentStatusOffline,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := repositories.NewTenantRepo(tx).Create(ctx, tenant); err != nil {
			if repositories.IsUniqueViolation(err) {
				return ErrSlugTaken
			}
			return err
		}
		if err := repositories.NewUserRepo(tx).Create(ctx, user); err != nil {
			if repositories.IsUniqueViolation(err) {
				return ErrEmailTaken
			}
			return err
		}
		settings := &models.TenantSetting{
			ID:           models.NewID(),
			TenantID:     tenant.ID,
			FeatureFlags: models.JSONMap{"sentiment": true, "campaigns": true},
			Preferences:  models.JSONMap{},
		}
		return tx.Create(settings).Error
	})
	if err != nil {
		return nil, err
	}

	return s.issueSession(ctx, user, tenant)
}

func (s *AuthService) uniqueSlug(ctx context.Context, base string) (string, error) {
	slug := base
	for i := 0; i < 6; i++ {
		exists, err := s.tenants.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		suffix, err := utils.RandomHex(2)
		if err != nil {
			return "", err
		}
		slug = fmt.Sprintf("%s-%s", base, suffix)
	}
	return "", ErrSlugTaken
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}

	email := utils.NormalizeEmail(in.Email)
	users, err := s.users.FindAllByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var matches []models.User
	for i := range users {
		if utils.CheckPassword(users[i].PasswordHash, in.Password) {
			matches = append(matches, users[i])
		}
	}
	if len(matches) == 0 {
		return nil, ErrInvalidCredentials
	}

	var user models.User
	if slug := strings.TrimSpace(in.TenantSlug); slug != "" {
		tenant, err := s.tenants.FindBySlug(ctx, slug)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrInvalidCredentials
			}
			return nil, err
		}
		found := false
		for i := range matches {
			if matches[i].TenantID == tenant.ID {
				user = matches[i]
				found = true
				break
			}
		}
		if !found {
			return nil, ErrInvalidCredentials
		}
	} else if len(matches) == 1 {
		user = matches[0]
	} else {
		choices := make([]TenantChoice, 0, len(matches))
		for i := range matches {
			t, err := s.tenants.FindByID(ctx, matches[i].TenantID)
			if err != nil {
				continue
			}
			choices = append(choices, TenantChoice{ID: t.ID, Name: t.Name, Slug: t.Slug})
		}
		return nil, &MultiTenantError{Tenants: choices}
	}

	tenant, err := s.tenants.FindByID(ctx, user.TenantID)
	if err != nil {
		return nil, err
	}
	if tenant.Status == models.TenantStatusSuspended {
		return nil, ErrTenantSuspended
	}

	return s.issueSession(ctx, &user, tenant)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	claims, err := s.jwt.ParseType(refreshToken, utils.TokenTypeRefresh)
	if err != nil {
		return nil, ErrInvalidToken
	}

	stored, err := s.tokens.FindByHash(ctx, utils.SHA256Hex(refreshToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if !stored.IsActive(time.Now().UTC()) {
		return nil, ErrInvalidToken
	}

	userID, err := claims.UserID()
	if err != nil {
		return nil, ErrInvalidToken
	}
	tenantID, err := claims.TenantUUID()
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.users.FindByID(ctx, tenantID, userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	tenant, err := s.tenants.FindByID(ctx, tenantID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if tenant.Status == models.TenantStatusSuspended {
		return nil, ErrTenantSuspended
	}

	if err := s.tokens.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}
	return s.issueSession(ctx, user, tenant)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	stored, err := s.tokens.FindByHash(ctx, utils.SHA256Hex(refreshToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return s.tokens.Revoke(ctx, stored.ID)
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.ready(); err != nil {
		return err
	}
	return s.tokens.RevokeAllForUser(ctx, userID)
}

func (s *AuthService) Me(ctx context.Context, p Principal) (*models.User, *models.Tenant, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	tenant, err := s.tenants.FindByID(ctx, p.TenantID)
	if err != nil {
		return nil, nil, err
	}
	if !p.HasUser {
		return nil, tenant, nil
	}
	user, err := s.users.FindByID(ctx, p.TenantID, p.UserID)
	if err != nil {
		return nil, nil, err
	}
	return user, tenant, nil
}

func (s *AuthService) AuthenticateJWT(token string) (*Principal, error) {
	if s.jwt == nil {
		return nil, ErrUnavailable
	}
	claims, err := s.jwt.ParseType(token, utils.TokenTypeAccess)
	if err != nil {
		return nil, ErrUnauthorized
	}
	userID, err := claims.UserID()
	if err != nil {
		return nil, ErrUnauthorized
	}
	tenantID, err := claims.TenantUUID()
	if err != nil {
		return nil, ErrUnauthorized
	}
	return &Principal{
		UserID:   userID,
		TenantID: tenantID,
		Role:     claims.Role,
		AuthType: models.AuthTypeJWT,
		HasUser:  true,
	}, nil
}

func (s *AuthService) AuthenticateAPIKey(ctx context.Context, raw string) (*Principal, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrUnauthorized
	}
	key, err := s.keys.FindByHash(ctx, utils.SHA256Hex(raw))
	if err != nil {
		return nil, ErrUnauthorized
	}
	_ = s.keys.TouchLastUsed(ctx, key.ID)
	return &Principal{
		TenantID: key.TenantID,
		Role:     key.Role,
		AuthType: models.AuthTypeAPIKey,
		APIKeyID: key.ID,
		HasUser:  false,
	}, nil
}

type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     string
}

func (s *AuthService) CreateUser(ctx context.Context, tenantID uuid.UUID, in CreateUserInput) (*models.User, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	email := utils.NormalizeEmail(in.Email)
	name := strings.TrimSpace(in.Name)
	role := strings.TrimSpace(in.Role)
	if name == "" || email == "" || len(in.Password) < 8 {
		return nil, ErrValidation
	}
	if role == "" {
		role = models.RoleAgent
	}
	if !models.ValidRole(role) || role == models.RoleOwner {
		return nil, ErrValidation
	}
	taken, err := s.users.EmailTaken(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrEmailTaken
	}
	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		ID:           models.NewID(),
		TenantID:     tenantID,
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         role,
		AgentStatus:  models.AgentStatusOffline,
	}
	if err := s.users.Create(ctx, user); err != nil {
		if repositories.IsUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return user, nil
}

func (s *AuthService) CreateAPIKey(ctx context.Context, tenantID, actorID uuid.UUID, name, role string) (*models.APIKey, string, error) {
	if err := s.ready(); err != nil {
		return nil, "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", ErrValidation
	}
	if role == "" {
		role = models.RoleAdmin
	}
	if !models.ValidRole(role) {
		return nil, "", ErrValidation
	}

	raw, prefix, hash, err := utils.NewAPIKey()
	if err != nil {
		return nil, "", err
	}
	key := &models.APIKey{
		ID:        models.NewID(),
		TenantID:  tenantID,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   hash,
		Role:      role,
		CreatedBy: &actorID,
	}
	if err := s.keys.Create(ctx, key); err != nil {
		return nil, "", err
	}
	return key, raw, nil
}

func (s *AuthService) ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]models.APIKey, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.keys.List(ctx, tenantID)
}

func (s *AuthService) RevokeAPIKey(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := s.keys.SoftDelete(ctx, tenantID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *AuthService) issueSession(ctx context.Context, user *models.User, tenant *models.Tenant) (*AuthResult, error) {
	access, _, err := s.jwt.IssueAccess(user.ID, tenant.ID, user.Role)
	if err != nil {
		return nil, err
	}
	refresh, exp, err := s.jwt.IssueRefresh(user.ID, tenant.ID, user.Role)
	if err != nil {
		return nil, err
	}
	row := &models.RefreshToken{
		ID:        models.NewID(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: utils.SHA256Hex(refresh),
		ExpiresAt: exp,
	}
	if err := s.tokens.Create(ctx, row); err != nil {
		return nil, err
	}
	return &AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.jwt.AccessTTL().Seconds()),
		User:         user,
		Tenant:       tenant,
	}, nil
}

func IsMultiTenant(err error) (*MultiTenantError, bool) {
	var mt *MultiTenantError
	if errors.As(err, &mt) {
		return mt, true
	}
	return nil, false
}
