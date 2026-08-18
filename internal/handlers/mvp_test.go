package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/Osawejustice/nandi-api/internal/config"
	"github.com/Osawejustice/nandi-api/internal/database"
	"github.com/Osawejustice/nandi-api/internal/realtime"
)

type envelope map[string]any

func TestMVPGoldenPath(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	email := fmt.Sprintf("owner-%d@acme.test", time.Now().UnixNano())
	reg := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"organization": "Acme Test",
		"name":         "Jane Owner",
		"email":        email,
		"password":     "changeme1",
	})
	if reg.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", reg.Code, reg.Body.String())
	}
	access := jsonString(t, reg.Body.Bytes(), "data", "access_token")
	tenantSlug := jsonString(t, reg.Body.Bytes(), "data", "tenant", "slug")
	if access == "" {
		t.Fatal("missing access token")
	}

	me := doJSON(t, router, http.MethodGet, "/api/v1/auth/me", access, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}

	contact := doJSON(t, router, http.MethodPost, "/api/v1/contacts", access, map[string]any{
		"name": "Ada", "phone": "+254700000001", "tags": []string{"vip"},
	})
	if contact.Code != http.StatusCreated {
		t.Fatalf("contact: %d %s", contact.Code, contact.Body.String())
	}

	inbound := doJSON(t, router, http.MethodPost, "/api/v1/dev/inbound", access, map[string]any{
		"phone": "+254700000099", "name": "Customer", "body": "I am unhappy with delivery", "channel": "sms",
	})
	if inbound.Code != http.StatusCreated {
		t.Fatalf("inbound: %d %s", inbound.Code, inbound.Body.String())
	}
	convID := jsonString(t, inbound.Body.Bytes(), "data", "conversation", "id")

	list := doJSON(t, router, http.MethodGet, "/api/v1/conversations?status=open", access, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}

	thread := doJSON(t, router, http.MethodGet, "/api/v1/conversations/"+convID, access, nil)
	if thread.Code != http.StatusOK {
		t.Fatalf("thread: %d %s", thread.Code, thread.Body.String())
	}

	reply := doJSON(t, router, http.MethodPost, "/api/v1/conversations/"+convID+"/messages", access, map[string]any{
		"body": "Sorry about that — we are looking into it.",
	})
	if reply.Code != http.StatusCreated {
		t.Fatalf("reply: %d %s", reply.Code, reply.Body.String())
	}
	if jsonString(t, reply.Body.Bytes(), "data", "status") != "sent" {
		t.Fatalf("expected sent reply: %s", reply.Body.String())
	}

	hookURL := "/api/v1/webhooks/" + tenantSlug + "/sms/africastalking"
	providerID := "at-" + uuid.NewString()
	first := doForm(t, router, hookURL, map[string]string{
		"from": "+254711111111", "text": "Need help", "id": providerID,
	})
	if first.Code != http.StatusOK {
		t.Fatalf("webhook: %d %s", first.Code, first.Body.String())
	}
	second := doForm(t, router, hookURL, map[string]string{
		"from": "+254711111111", "text": "Need help", "id": providerID,
	})
	if second.Code != http.StatusOK {
		t.Fatalf("webhook retry: %d %s", second.Code, second.Body.String())
	}

	campaign := doJSON(t, router, http.MethodPost, "/api/v1/campaigns", access, map[string]any{
		"name": "Promo", "channel": "sms", "message_template": "Hello from Nandi",
	})
	if campaign.Code != http.StatusCreated {
		t.Fatalf("campaign: %d %s", campaign.Code, campaign.Body.String())
	}
	campID := jsonString(t, campaign.Body.Bytes(), "data", "id")
	start := doJSON(t, router, http.MethodPost, "/api/v1/campaigns/"+campID+"/start", access, nil)
	if start.Code != http.StatusOK {
		t.Fatalf("start: %d %s", start.Code, start.Body.String())
	}

	overview := doJSON(t, router, http.MethodGet, "/api/v1/analytics/overview", access, nil)
	if overview.Code != http.StatusOK {
		t.Fatalf("analytics: %d %s", overview.Code, overview.Body.String())
	}

	// Tenant isolation: second org cannot see first contact.
	other := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"organization": "Other Co",
		"name":         "Other Owner",
		"email":        fmt.Sprintf("other-%d@other.test", time.Now().UnixNano()),
		"password":     "changeme1",
	})
	if other.Code != http.StatusCreated {
		t.Fatalf("other register: %d %s", other.Code, other.Body.String())
	}
	otherToken := jsonString(t, other.Body.Bytes(), "data", "access_token")
	contactID := jsonString(t, contact.Body.Bytes(), "data", "id")
	leak := doJSON(t, router, http.MethodGet, "/api/v1/contacts/"+contactID, otherToken, nil)
	if leak.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on cross-tenant read, got %d %s", leak.Code, leak.Body.String())
	}
}

func newTestRouter(t *testing.T) (http.Handler, func()) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := zerolog.New(os.Stdout).Level(zerolog.Disabled)
	db, err := database.NewPostgres(cfg.Database, log)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := database.AutoMigrate(db, log); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rdb, _ := database.NewRedis(cfg.Redis, log)
	hub := realtime.NewHub(rdb, log)
	deps := Dependencies{DB: db, Redis: rdb, Config: cfg, Log: log, Hub: hub}
	svc := BuildServices(deps)
	router := NewRouter(log, deps, svc)
	cleanup := func() {
		_ = database.ClosePostgres(db)
		_ = database.CloseRedis(rdb)
	}
	return router, cleanup
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func doForm(t *testing.T, h http.Handler, path string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	values := url.Values{}
	for k, v := range fields {
		values.Set(k, v)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func jsonString(t *testing.T, raw []byte, path ...string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("json: %v %s", err, raw)
	}
	cur := v
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("not an object at %v in %s", path, raw)
		}
		cur = m[key]
	}
	s, _ := cur.(string)
	return s
}
