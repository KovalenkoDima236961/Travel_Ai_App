package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/emailnotifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/handler"
	internalmw "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/middleware"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/stream"
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

func (f *fakeService) CreateBatch(_ context.Context, inputs []notifications.CreateInput) ([]entity.Notification, error) {
	result, err := f.CreateBatchWithPreferences(context.Background(), inputs, nil)
	if err != nil {
		return nil, err
	}
	return result.Created, nil
}

func (f *fakeService) CreateBatchWithPreferences(_ context.Context, inputs []notifications.CreateInput, gate notifications.InAppPreferenceGate) (*notifications.BatchCreateResult, error) {
	result := &notifications.BatchCreateResult{Requested: len(inputs)}
	out := make([]entity.Notification, 0, len(inputs))
	for _, in := range inputs {
		if in.ActorUserID != nil && *in.ActorUserID == in.UserID {
			result.Skipped++
			continue
		}
		n := entity.Notification{
			ID:          uuid.New(),
			UserID:      in.UserID,
			TripID:      in.TripID,
			ActorUserID: in.ActorUserID,
			Type:        in.Type,
			Title:       in.Title,
			Message:     in.Message,
			Metadata:    in.Metadata,
		}
		result.EmailCandidates = append(result.EmailCandidates, n)
		if gate != nil && !gate.AllowInApp(in.UserID, in.Type) {
			result.Skipped++
			result.SkippedByPreference++
			continue
		}
		f.created = append(f.created, in)
		out = append(out, n)
	}
	result.Created = out
	return result, nil
}

// fakeEmailDispatcher records the notifications it is asked to email and returns
// a canned result/error so the handler's response shaping and error mapping can
// be asserted without real email infrastructure.
type fakeEmailDispatcher struct {
	result emailnotifications.EmailSendResult
	err    error
	gotLen int
}

func (f *fakeEmailDispatcher) SendEmailsForNotifications(_ context.Context, notifications []entity.Notification, _ ...emailnotifications.EmailPreferenceGate) (emailnotifications.EmailSendResult, error) {
	f.gotLen = len(notifications)
	return f.result, f.err
}

type fakePreferenceService struct {
	result     *preferences.PreferencesResult
	set        *preferences.EffectiveSet
	getUser    uuid.UUID
	updateUser uuid.UUID
	updated    []preferences.PreferenceInput
	err        error
}

func (f *fakePreferenceService) GetPreferences(_ context.Context, userID uuid.UUID) (*preferences.PreferencesResult, error) {
	f.getUser = userID
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &preferences.PreferencesResult{Items: []preferences.PreferenceItem{
		{Channel: preferences.ChannelInApp, Category: preferences.CategoryComments, Enabled: true},
		{Channel: preferences.ChannelEmail, Category: preferences.CategoryComments, Enabled: true},
	}}, nil
}

func (f *fakePreferenceService) UpdatePreferences(_ context.Context, userID uuid.UUID, items []preferences.PreferenceInput) (*preferences.PreferencesResult, error) {
	f.updateUser = userID
	f.updated = append([]preferences.PreferenceInput(nil), items...)
	if f.err != nil {
		return nil, f.err
	}
	return f.GetPreferences(context.Background(), userID)
}

func (f *fakePreferenceService) EffectiveForUsers(_ context.Context, userIDs []uuid.UUID) (*preferences.EffectiveSet, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.set != nil {
		return f.set, nil
	}
	return preferences.BuildEffectiveSet(userIDs, nil), nil
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
	return newTestRouterWithEmail(svc, nil)
}

func newTestRouterWithEmail(svc *fakeService, emails *fakeEmailDispatcher) http.Handler {
	return newTestRouterFull(svc, emails, nil)
}

func newTestRouterFull(svc *fakeService, emails *fakeEmailDispatcher, prefs *fakePreferenceService) http.Handler {
	var internal *handler.InternalHandler
	if emails == nil && prefs == nil {
		internal = handler.NewInternal(svc, nil, zap.NewNop())
	} else if emails == nil {
		internal = handler.NewInternal(svc, nil, zap.NewNop(), prefs)
	} else if prefs == nil {
		internal = handler.NewInternal(svc, emails, zap.NewNop())
	} else {
		internal = handler.NewInternal(svc, emails, zap.NewNop(), prefs)
	}
	userHandler := handler.New(svc, zap.NewNop())
	if prefs != nil {
		userHandler = handler.New(svc, zap.NewNop(), prefs)
	}
	return NewRouter(
		zap.NewNop(),
		userHandler,
		internal,
		nil,
		config.CORSConfig{AllowedOrigins: "http://localhost:3000"},
		config.JWTConfig{AccessSecret: testJWTSecret, HeaderName: "Authorization"},
		config.InternalConfig{ServiceToken: testInternalToken},
	)
}

func newTestRouterFullWithStream(
	svc *fakeService,
	emails *fakeEmailDispatcher,
	prefs *fakePreferenceService,
	streamManager stream.Manager,
	streamCfg stream.Config,
) http.Handler {
	var internal *handler.InternalHandler
	if emails == nil && prefs == nil {
		internal = handler.NewInternal(svc, nil, zap.NewNop())
	} else if emails == nil {
		internal = handler.NewInternal(svc, nil, zap.NewNop(), prefs)
	} else if prefs == nil {
		internal = handler.NewInternal(svc, emails, zap.NewNop())
	} else {
		internal = handler.NewInternal(svc, emails, zap.NewNop(), prefs)
	}
	internal.EnableStream(streamManager)

	userHandler := handler.New(svc, zap.NewNop())
	if prefs != nil {
		userHandler = handler.New(svc, zap.NewNop(), prefs)
	}
	userHandler.EnableStream(streamManager, streamCfg)

	return NewRouter(
		zap.NewNop(),
		userHandler,
		internal,
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

func waitForStreamCount(t *testing.T, manager stream.Manager, userID uuid.UUID, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got := manager.CountForUser(userID); got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected stream count %d for user %s, got %d", want, userID, manager.CountForUser(userID))
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

func TestGetNotificationPreferencesRequiresJWT(t *testing.T) {
	router := newTestRouterFull(&fakeService{}, nil, &fakePreferenceService{})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestGetNotificationPreferencesUsesTokenSubject(t *testing.T) {
	prefs := &fakePreferenceService{}
	router := newTestRouterFull(&fakeService{}, nil, prefs)
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if prefs.getUser != userID {
		t.Fatalf("expected preferences scoped to token subject %s, got %s", userID, prefs.getUser)
	}
	var body struct {
		Items []struct {
			Channel  string `json:"channel"`
			Category string `json:"category"`
			Enabled  bool   `json:"enabled"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) == 0 {
		t.Fatal("expected preference items")
	}
}

func TestPutNotificationPreferencesSavesCurrentUser(t *testing.T) {
	prefs := &fakePreferenceService{}
	router := newTestRouterFull(&fakeService{}, nil, prefs)
	userID := uuid.New()

	body := []byte(`{"items":[{"channel":"email","category":"comments","enabled":false}]}`)
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if prefs.updateUser != userID {
		t.Fatalf("expected update scoped to token subject %s, got %s", userID, prefs.updateUser)
	}
	if len(prefs.updated) != 1 || prefs.updated[0].Channel != preferences.ChannelEmail ||
		prefs.updated[0].Category != preferences.CategoryComments || prefs.updated[0].Enabled {
		t.Fatalf("unexpected updated preferences: %+v", prefs.updated)
	}
}

func TestPutNotificationPreferencesRejectsMissingEnabled(t *testing.T) {
	router := newTestRouterFull(&fakeService{}, nil, &fakePreferenceService{})
	userID := uuid.New()

	body := []byte(`{"items":[{"channel":"email","category":"comments"}]}`)
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
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

func TestStreamRequiresJWT(t *testing.T) {
	streamManager := stream.NewManager(stream.Config{MaxConnectionsPerUser: 5}, nil)
	router := newTestRouterFullWithStream(
		&fakeService{},
		nil,
		nil,
		streamManager,
		stream.Config{Enabled: true, HeartbeatInterval: time.Second, WriteTimeout: time.Second, MaxConnectionsPerUser: 5},
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/notifications/stream", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestStreamWritesHeadersAndHeartbeat(t *testing.T) {
	streamManager := stream.NewManager(stream.Config{MaxConnectionsPerUser: 5}, nil)
	router := newTestRouterFullWithStream(
		&fakeService{},
		nil,
		nil,
		streamManager,
		stream.Config{Enabled: true, HeartbeatInterval: time.Second, WriteTimeout: time.Second, MaxConnectionsPerUser: 5},
	)
	userID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/notifications/stream", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()
	waitForStreamCount(t, streamManager, userID, 1)
	cancel()
	<-done

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", got)
	}
	if got := rec.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("expected X-Accel-Buffering=no, got %q", got)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("event: heartbeat\n")) {
		t.Fatalf("expected initial heartbeat, got %q", rec.Body.String())
	}
}

func TestStreamDisabledReturns503(t *testing.T) {
	streamManager := stream.NewManager(stream.Config{MaxConnectionsPerUser: 5}, nil)
	router := newTestRouterFullWithStream(
		&fakeService{},
		nil,
		nil,
		streamManager,
		stream.Config{Enabled: false, HeartbeatInterval: time.Second, WriteTimeout: time.Second, MaxConnectionsPerUser: 5},
	)
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/notifications/stream", nil)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStreamMaxConnectionsReturns429(t *testing.T) {
	userID := uuid.New()
	streamManager := stream.NewManager(stream.Config{MaxConnectionsPerUser: 1}, nil)
	existing := stream.NewClient(userID)
	if err := streamManager.Register(userID, existing); err != nil {
		t.Fatalf("register existing stream: %v", err)
	}
	defer streamManager.Unregister(userID, existing.ID)

	router := newTestRouterFullWithStream(
		&fakeService{},
		nil,
		nil,
		streamManager,
		stream.Config{Enabled: true, HeartbeatInterval: time.Second, WriteTimeout: time.Second, MaxConnectionsPerUser: 1},
	)
	req := httptest.NewRequest(http.MethodGet, "/notifications/stream", nil)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStreamReceivesPublishedNotification(t *testing.T) {
	userID := uuid.New()
	streamManager := stream.NewManager(stream.Config{MaxConnectionsPerUser: 5}, nil)
	router := newTestRouterFullWithStream(
		&fakeService{},
		nil,
		nil,
		streamManager,
		stream.Config{Enabled: true, HeartbeatInterval: time.Second, WriteTimeout: time.Second, MaxConnectionsPerUser: 5},
	)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/notifications/stream", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+mintAccessToken(t, testJWTSecret, userID, time.Hour))
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()
	waitForStreamCount(t, streamManager, userID, 1)

	streamManager.PublishToUser(context.Background(), userID, stream.StreamEvent{
		Name: stream.EventNotificationCreated,
		Data: map[string]string{"id": "n1"},
	})
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, "event: notification.created\n") {
		t.Fatalf("expected notification.created event, got %q", body)
	}
	if !strings.Contains(body, `"id":"n1"`) {
		t.Fatalf("expected published payload, got %q", body)
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
	var resp batchResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Created != 1 {
		t.Fatalf("expected created=1, got %+v", resp)
	}
	if len(svc.created) != 1 || svc.created[0].UserID != recipient {
		t.Fatalf("expected one create for recipient, got %+v", svc.created)
	}
}

func TestInternalBatchPublishesCreatedNotificationToStream(t *testing.T) {
	svc := &fakeService{}
	streamManager := stream.NewManager(stream.Config{MaxConnectionsPerUser: 5}, nil)
	recipient := uuid.New()
	client := stream.NewClient(recipient)
	if err := streamManager.Register(recipient, client); err != nil {
		t.Fatalf("register stream client: %v", err)
	}
	defer streamManager.Unregister(recipient, client.ID)

	router := newTestRouterFullWithStream(
		svc,
		nil,
		nil,
		streamManager,
		stream.Config{Enabled: true, HeartbeatInterval: time.Second, WriteTimeout: time.Second, MaxConnectionsPerUser: 5},
	)
	rec := postBatch(t, router, recipient, uuid.New())
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	select {
	case got := <-client.Send:
		if got.Name != stream.EventNotificationCreated {
			t.Fatalf("expected notification.created, got %+v", got)
		}
		payload, err := json.Marshal(got.Data)
		if err != nil {
			t.Fatalf("marshal event payload: %v", err)
		}
		if !bytes.Contains(payload, []byte(`"notification"`)) || !bytes.Contains(payload, []byte(recipient.String())) {
			t.Fatalf("unexpected event payload: %s", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for created notification event")
	}
}

// batchResponseBody mirrors the internal batch endpoint's JSON response shape.
type batchResponseBody struct {
	Requested           int `json:"requested"`
	Created             int `json:"created"`
	Skipped             int `json:"skipped"`
	SkippedByPreference int `json:"skippedByPreference"`
	Email               struct {
		Attempted           int `json:"attempted"`
		Sent                int `json:"sent"`
		Skipped             int `json:"skipped"`
		SkippedByPreference int `json:"skippedByPreference"`
		Failed              int `json:"failed"`
	} `json:"email"`
}

func postBatch(t *testing.T, router http.Handler, recipient, actor uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"notifications":[{"userId":%q,"actorUserId":%q,"type":"comment_created","title":"New comment","message":"A collaborator commented on Day 2."}]}`,
		recipient.String(), actor.String())
	req := httptest.NewRequest(http.MethodPost, "/internal/notifications/batch", bytes.NewReader([]byte(body)))
	req.Header.Set(internalmw.InternalServiceTokenHeader, testInternalToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestInternalBatchReturnsEmailStats(t *testing.T) {
	svc := &fakeService{}
	emails := &fakeEmailDispatcher{result: emailnotifications.EmailSendResult{Attempted: 1, Sent: 1}}
	router := newTestRouterWithEmail(svc, emails)

	rec := postBatch(t, router, uuid.New(), uuid.New())
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp batchResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Created != 1 || resp.Email.Attempted != 1 || resp.Email.Sent != 1 {
		t.Fatalf("unexpected response %+v", resp)
	}
	if emails.gotLen != 1 {
		t.Fatalf("expected dispatcher to receive 1 created notification, got %d", emails.gotLen)
	}
}

func TestInternalBatchEmailDisabledCreatesOnly(t *testing.T) {
	// A noop dispatcher (email not wired) reports everything skipped, rows still created.
	svc := &fakeService{}
	rec := postBatch(t, newTestRouter(svc), uuid.New(), uuid.New())
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp batchResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Created != 1 || resp.Email.Skipped != 1 || resp.Email.Sent != 0 {
		t.Fatalf("expected created=1 skipped=1, got %+v", resp)
	}
}

func TestInternalBatchReportsInAppPreferenceSkip(t *testing.T) {
	svc := &fakeService{}
	recipient := uuid.New()
	prefs := &fakePreferenceService{set: preferences.BuildEffectiveSet(
		[]uuid.UUID{recipient},
		[]entity.NotificationPreference{
			{
				UserID:   recipient,
				Channel:  preferences.ChannelInApp,
				Category: preferences.CategoryComments,
				Enabled:  false,
			},
		},
	)}

	rec := postBatch(t, newTestRouterFull(svc, nil, prefs), recipient, uuid.New())
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp batchResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Requested != 1 || resp.Created != 0 || resp.Skipped != 1 || resp.SkippedByPreference != 1 {
		t.Fatalf("expected in-app preference skip stats, got %+v", resp)
	}
	if len(svc.created) != 0 {
		t.Fatalf("expected no in-app create calls, got %+v", svc.created)
	}
	if resp.Email.Skipped != 1 {
		t.Fatalf("expected noop email dispatcher to receive preserved candidate, got %+v", resp.Email)
	}
}

func TestInternalBatchDoesNotPublishWhenInAppPreferenceSkips(t *testing.T) {
	svc := &fakeService{}
	recipient := uuid.New()
	streamManager := stream.NewManager(stream.Config{MaxConnectionsPerUser: 5}, nil)
	client := stream.NewClient(recipient)
	if err := streamManager.Register(recipient, client); err != nil {
		t.Fatalf("register stream client: %v", err)
	}
	defer streamManager.Unregister(recipient, client.ID)

	prefs := &fakePreferenceService{set: preferences.BuildEffectiveSet(
		[]uuid.UUID{recipient},
		[]entity.NotificationPreference{
			{
				UserID:   recipient,
				Channel:  preferences.ChannelInApp,
				Category: preferences.CategoryComments,
				Enabled:  false,
			},
		},
	)}
	router := newTestRouterFullWithStream(
		svc,
		nil,
		prefs,
		streamManager,
		stream.Config{Enabled: true, HeartbeatInterval: time.Second, WriteTimeout: time.Second, MaxConnectionsPerUser: 5},
	)

	rec := postBatch(t, router, recipient, uuid.New())
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	select {
	case got := <-client.Send:
		t.Fatalf("expected no stream event for skipped in-app notification, got %+v", got)
	default:
	}
}

func TestInternalBatchEmailFailOpenReturns201WithFailedCount(t *testing.T) {
	svc := &fakeService{}
	// fail-open: dispatcher returns no error but reports a failed send.
	emails := &fakeEmailDispatcher{result: emailnotifications.EmailSendResult{Attempted: 1, Failed: 1}}
	rec := postBatch(t, newTestRouterWithEmail(svc, emails), uuid.New(), uuid.New())
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on fail-open, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp batchResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Created != 1 || resp.Email.Failed != 1 {
		t.Fatalf("expected created=1 failed=1, got %+v", resp)
	}
}

func TestInternalBatchEmailFailClosedReturns502(t *testing.T) {
	svc := &fakeService{}
	// fail-closed: dispatcher returns an error after rows were created.
	emails := &fakeEmailDispatcher{result: emailnotifications.EmailSendResult{Attempted: 1, Failed: 1}, err: errors.New("smtp down")}
	rec := postBatch(t, newTestRouterWithEmail(svc, emails), uuid.New(), uuid.New())
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on fail-closed email failure, got %d: %s", rec.Code, rec.Body.String())
	}
	// Rows were still created (dispatcher received them).
	if emails.gotLen != 1 {
		t.Fatalf("expected rows created before email failure, got %d", emails.gotLen)
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
