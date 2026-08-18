package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Osawejustice/nandi-api/internal/config"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrWrongType    = errors.New("unexpected token type")
)

// Claims are embedded in both access and refresh tokens.
type Claims struct {
	TenantID string `json:"tid"`
	Role     string `json:"role"`
	Typ      string `json:"typ"`
	jwt.RegisteredClaims
}

func (c Claims) UserID() (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}

func (c Claims) TenantUUID() (uuid.UUID, error) {
	return uuid.Parse(c.TenantID)
}

// JWTManager issues and parses HS256 access/refresh tokens.
type JWTManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewJWTManager(cfg config.JWTConfig) *JWTManager {
	access := cfg.AccessTTL
	if access <= 0 {
		access = 15 * time.Minute
	}
	refresh := cfg.RefreshTTL
	if refresh <= 0 {
		refresh = 7 * 24 * time.Hour
	}
	return &JWTManager{
		secret:     []byte(cfg.Secret),
		accessTTL:  access,
		refreshTTL: refresh,
		issuer:     "nandi-api",
	}
}

func (m *JWTManager) AccessTTL() time.Duration  { return m.accessTTL }
func (m *JWTManager) RefreshTTL() time.Duration { return m.refreshTTL }

func (m *JWTManager) IssueAccess(userID, tenantID uuid.UUID, role string) (string, time.Time, error) {
	return m.issue(userID, tenantID, role, TokenTypeAccess, m.accessTTL)
}

func (m *JWTManager) IssueRefresh(userID, tenantID uuid.UUID, role string) (string, time.Time, error) {
	return m.issue(userID, tenantID, role, TokenTypeRefresh, m.refreshTTL)
}

func (m *JWTManager) issue(userID, tenantID uuid.UUID, role, typ string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(ttl)
	claims := Claims{
		TenantID: tenantID.String(),
		Role:     role,
		Typ:      typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, exp, nil
}

func (m *JWTManager) Parse(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected alg", ErrInvalidToken)
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *JWTManager) ParseType(token, expectedType string) (*Claims, error) {
	claims, err := m.Parse(token)
	if err != nil {
		return nil, err
	}
	if claims.Typ != expectedType {
		return nil, ErrWrongType
	}
	return claims, nil
}
