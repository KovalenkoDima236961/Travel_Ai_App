package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/handler"
	internalmw "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/middleware"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
)

const (
	testJWTSecret     = "test-access-secret"
	testInternalToken = "test-internal-token"
)

// fakeService satisfies both the user-facing and internal handler ports. It
// records the user ids it is asked about so tests can assert scoping.
type fakeService struct {
	created     []notifications.CreateInput
	listedUser  uuid.UUID
	unread      int
	markAllUser uuid.UUID
}

func (f *fakeService) CreateBatch(_ context.Context, inputs []notifications.CreateInput) (int, error) {
	f.created = append(f.created, inputs...)
	return len(inputs), nil
}

func (f *fakeService) List(_ context.Context, in notifications.ListInput) (*notifications.ListResult, error) {
	f.listedUser = in.UserID
	return &notifications.ListResult{Notifications: []entity.Notification{}}, nil
}

func (f *fakeService) CountUnread(_ context.Context, userID uuid.UUID) (int, error) {
	f.listedUser = userID
	return f.unread, nil
}

func (f *fakeService) MarkRead(_ context.Context, _, userID uuid.UUID) (*entity.Notification, error) {
	return &entity.Notification{UserID: userID}, nil
}

func (f *fakeService) MarkAllRead(_ context.Context, userID uuid.UUID) (int, error) {
	f.markAllUser = userID
	return 0, nil
}

func newTestRouter(svc *fakeService) http.Handler {
	return NewRouter(
		zap.NewNop(),
		handler.New(svc, zap.NewNop()),
		handler.NewInternal(svc, zap.NewNop()),
		nil,
		config.CORSConfig{AllowedOrigins: "http://localhost:3000"},
		config.JWTConfig{AccessSecret: testJWTSecret, HeaderName: "Authorization"},
		config.InternalConfig{ServiceToken: testInternalToken},
	)
}

// mintAccessToken builds a minimal HS256 access token accepted by the auth
// middleware (sub + exp claims).
func mintAccessToken(t *testing.T, secret string, userID uuid.UUID, ttl time.Duration) string {
	t.Helper()
	header := base64URL(`{"alg":"HS256","typ":"JWT"}`)
	payload := base64URL(fmt.Sprintf(`{"sub":%q,"exp":%d}`, userID.String(), time.Now().Add(ttl).Unix()))
	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + signature
}

func base64URL(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func TestHealthReturnsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(&fakeService{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected health body: %+v", body)
	}
}

func TestListNotificationsRequiresJWT(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(&fakeService{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/notifications", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestListNotificationsUsesTokenSubject(t *testing.T) {
	svc := &fakeService{}
	router := newTestRouter(svc)
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if svc.listedUser != userID {
		t.Fatalf("expected list scoped to token subject %s, got %s", userID, svc.listedUser)
	}
}

func TestListRejectsTokenSignedWithWrongSecret(t *testing.T) {
	router := newTestRouter(&fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, "wrong-secret", uuid.New(), time.Hour))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong-secret token, got %d", rec.Code)
	}
}

func TestInternalBatchRequiresToken(t *testing.T) {
	router := newTestRouter(&fakeService{})
	body := []byte(`{"notifications":[]}`)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/notifications/batch", bytes.NewReader(body)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without internal token, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/notifications/batch", bytes.NewReader(body))
	req.Header.Set(internalmw.InternalServiceTokenHeader, "wrong-token")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong internal token, got %d", rec.Code)
	}
}

func TestInternalBatchCreatesNotifications(t *testing.T) {
	svc := &fakeService{}
	router := newTestRouter(svc)

	recipient := uuid.New()
	actor := uuid.New()
	body := fmt.Sprintf(`{"notifications":[{"userId":%q,"actorUserId":%q,"type":"comment_created","title":"New comment","message":"A collaborator commented on Day 2."}]}`,
		recipient.String(), actor.String())

	req := httptest.NewRequest(http.MethodPost, "/internal/notifications/batch", bytes.NewReader([]byte(body)))
	req.Header.Set(internalmw.InternalServiceTokenHeader, testInternalToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["created"] != 1 {
		t.Fatalf("expected created=1, got %+v", resp)
	}
	if len(svc.created) != 1 || svc.created[0].UserID != recipient {
		t.Fatalf("expected one create for recipient, got %+v", svc.created)
	}
}

func TestInternalBatchRejectsInvalidUUID(t *testing.T) {
	router := newTestRouter(&fakeService{})
	body := `{"notifications":[{"userId":"not-a-uuid","type":"comment_created","title":"t","message":"m"}]}`
	req := httptest.NewRequest(http.MethodPost, "/internal/notifications/batch", bytes.NewReader([]byte(body)))
	req.Header.Set(internalmw.InternalServiceTokenHeader, testInternalToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid uuid, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMarkAllReadUsesTokenSubject(t *testing.T) {
	svc := &fakeService{}
	router := newTestRouter(svc)
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodPatch, "/notifications/read-all", nil)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if svc.markAllUser != userID {
		t.Fatalf("expected mark-all scoped to token subject %s, got %s", userID, svc.markAllUser)
	}
}
