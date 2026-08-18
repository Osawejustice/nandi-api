package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Osawejustice/nandi-api/internal/config"
)

func TestJWTIssueAndParse(t *testing.T) {
	m := NewJWTManager(config.JWTConfig{Secret: "test-secret", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	userID := uuid.New()
	tenantID := uuid.New()

	access, _, err := m.IssueAccess(userID, tenantID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.ParseType(access, TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != "owner" || claims.TenantID != tenantID.String() {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := m.ParseType(access, TokenTypeRefresh); err == nil {
		t.Fatal("expected wrong type")
	}
}

func TestJWTRejectsBadToken(t *testing.T) {
	m := NewJWTManager(config.JWTConfig{Secret: "test-secret", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	if _, err := m.Parse("not-a-token"); err == nil {
		t.Fatal("expected invalid token")
	}
}
