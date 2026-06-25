package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/presence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

func TestPresenceStreamRequiresJWT(t *testing.T) {
	router, _, _ := newPresenceTestRouter(t, presence.Config{Enabled: true})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+uuid.NewString()+"/presence/stream", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestPresenceOwnerCanStreamAndReadSnapshot(t *testing.T) {
	router, _, manager := newPresenceTestRouter(t, presence.Config{
		Enabled:                      true,
		HeartbeatInterval:            time.Second,
		StaleAfter:                   time.Minute,
		MaxConnectionsPerUserPerTrip: 5,
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/presence", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected snapshot HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	streamReq := httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/presence/stream", nil).WithContext(ctx)
	streamReq.Header.Set("Authorization", "Bearer "+ownerToken)
	streamRec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(streamRec, streamReq)
		close(done)
	}()
	waitForPresenceUsers(t, manager, uuid.MustParse(tripID), 1)
	cancel()
	<-done

	if streamRec.Code != http.StatusOK {
		t.Fatalf("expected stream HTTP 200, got %d with %s", streamRec.Code, streamRec.Body.String())
	}
	if got := streamRec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", got)
	}
	if got := streamRec.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("expected X-Accel-Buffering=no, got %q", got)
	}
	body := streamRec.Body.String()
	if !strings.Contains(body, "event: presence.snapshot\n") {
		t.Fatalf("expected presence snapshot event, got %q", body)
	}
	if !strings.Contains(body, `"role":"owner"`) || !strings.Contains(body, `"state":"viewing"`) {
		t.Fatalf("expected owner viewing payload, got %q", body)
	}
}

func TestPresenceStateRequiresAccessAndValidState(t *testing.T) {
	router, _, _ := newPresenceTestRouter(t, presence.Config{Enabled: true})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	editorID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	otherID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	editorToken := signAccessToken(t, editorID, "editor@example.com", testJWTSecret, time.Hour)
	otherToken := signAccessToken(t, otherID, "other@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/presence/state", bytes.NewReader([]byte(`{"state":"away"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid state HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/presence/state", bytes.NewReader([]byte(`{"state":"viewing"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner state HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/presence/state", bytes.NewReader([]byte(`{"state":"viewing"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-collaborator HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	inviteID := invitePresenceCollaborator(t, router, ownerToken, tripID, "editor@example.com", "editor")

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/presence/state", bytes.NewReader([]byte(`{"state":"editing"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected pending collaborator HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/collaborators/"+inviteID+"/accept", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accept HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/presence/state", bytes.NewReader([]byte(`{"state":"editing"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accepted editor HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+tripID+"/collaborators/"+inviteID, nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected remove HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/presence/state", bytes.NewReader([]byte(`{"state":"viewing"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected removed collaborator HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestPresenceDisabledReturns503(t *testing.T) {
	router, _, _ := newPresenceTestRouter(t, presence.Config{Enabled: false})
	userID := uuid.New()
	token := signAccessToken(t, userID, "user@example.com", testJWTSecret, time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+uuid.NewString()+"/presence/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected HTTP 503, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestPresenceMaxConnectionsReturns429(t *testing.T) {
	router, _, manager := newPresenceTestRouter(t, presence.Config{
		Enabled:                      true,
		MaxConnectionsPerUserPerTrip: 1,
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := uuid.MustParse(createCompletedTripForRouteTest(t, router, ownerToken))
	if _, err := manager.Register(context.Background(), presence.PresenceSession{
		SessionID: uuid.NewString(),
		TripID:    tripID,
		UserID:    ownerID,
		Role:      "owner",
		State:     presence.PresenceStateViewing,
	}); err != nil {
		t.Fatalf("register existing presence: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+tripID.String()+"/presence/stream", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP 429, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestPresencePublicShareTokenCannotConnect(t *testing.T) {
	router, _, _ := newPresenceTestRouter(t, presence.Config{Enabled: true})
	publicShareToken := signPublicShareAccessToken(t, "share-token", time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+uuid.NewString()+"/presence/stream", nil)
	req.Header.Set("Authorization", "Bearer "+publicShareToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected public share token HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}
}

func newPresenceTestRouter(t *testing.T, cfg presence.Config) (http.Handler, *routeTestRepo, presence.Manager) {
	t.Helper()

	authCfg := config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	}
	repo := &routeTestRepo{
		trips:             map[uuid.UUID]entity.Trip{},
		collaboratorsByID: map[uuid.UUID]entity.TripCollaborator{},
		sharesByTrip:      map[uuid.UUID]entity.TripShare{},
		sharesByToken:     map[string]entity.TripShare{},
	}
	svc := service.New(
		repo,
		routeTestGenerator{},
		zap.NewNop(),
		service.WithPublicSharing(true, "http://localhost:3000", 32, testPublicShareSecret, 60),
		service.WithUserLookup(routeTestUserLookup{
			usersByEmail: map[string]appdto.UserLookupResult{
				"viewer@example.com": {
					UserID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
					Email:  "viewer@example.com",
				},
				"editor@example.com": {
					UserID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
					Email:  "editor@example.com",
				},
			},
		}),
		service.WithActivity(activity.New(repo, zap.NewNop())),
	)
	validator, err := validation.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}
	presenceCfg := presence.Normalize(cfg)
	manager := presence.NewManager(presenceCfg, zap.NewNop())
	tripHandler := handler.New(svc, validator, zap.NewNop()).EnablePresence(manager, presenceCfg)
	ready := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return NewRouter(zap.NewNop(), tripHandler, ready, config.CORSConfig{}, authCfg), repo, manager
}

func waitForPresenceUsers(t *testing.T, manager presence.Manager, tripID uuid.UUID, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got := len(manager.Snapshot(tripID).Users); got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %d presence users, got %+v", want, manager.Snapshot(tripID))
}

func invitePresenceCollaborator(t *testing.T, router http.Handler, ownerToken, tripID, email, role string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/trips/"+tripID+"/collaborators",
		bytes.NewReader([]byte(`{"email":"`+email+`","role":"`+role+`"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected invite HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var invite struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &invite); err != nil {
		t.Fatalf("decode invite: %v", err)
	}
	return invite.ID
}
