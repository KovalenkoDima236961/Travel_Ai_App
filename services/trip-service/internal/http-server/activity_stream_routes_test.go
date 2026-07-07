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
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activitystream"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/validation"
)

func TestActivityStreamRequiresJWT(t *testing.T) {
	router, _, _ := newActivityStreamTestRouter(t, activitystream.Config{Enabled: true})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+uuid.NewString()+"/activity/stream", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestActivityStreamOwnerReceivesCreatedEvent(t *testing.T) {
	router, _, manager := newActivityStreamTestRouter(t, activitystream.Config{
		Enabled:                      true,
		HeartbeatInterval:            time.Hour,
		WriteTimeout:                 time.Second,
		MaxConnectionsPerUserPerTrip: 5,
		ClientBufferSize:             20,
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)

	ctx, cancel := context.WithCancel(context.Background())
	streamReq := httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/activity/stream", nil).WithContext(ctx)
	streamReq.Header.Set("Authorization", "Bearer "+ownerToken)
	streamRec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(streamRec, streamReq)
		close(done)
	}()
	waitForActivityClients(t, manager, uuid.MustParse(tripID), 1)

	rec := doJSON(t, router, http.MethodPost, "/trips/"+tripID+"/comments", ownerToken,
		`{"dayNumber":1,"itemIndex":0,"body":"Live update"}`)
	if rec.Code != http.StatusCreated {
		cancel()
		<-done
		t.Fatalf("expected create comment HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}

	time.Sleep(50 * time.Millisecond)
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
	if !strings.Contains(body, "event: activity.created\n") {
		t.Fatalf("expected activity.created event, got %q", body)
	}
	if !strings.Contains(body, `"eventType":"comment_created"`) {
		t.Fatalf("expected comment_created payload, got %q", body)
	}
}

func TestActivityStreamAcceptedCollaboratorsCanConnect(t *testing.T) {
	for _, tc := range []struct {
		name  string
		email string
		role  string
		user  uuid.UUID
	}{
		{
			name:  "editor",
			email: "editor@example.com",
			role:  "editor",
			user:  uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		},
		{
			name:  "viewer",
			email: "viewer@example.com",
			role:  "viewer",
			user:  uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, _, manager := newActivityStreamTestRouter(t, activitystream.Config{
				Enabled:                      true,
				HeartbeatInterval:            time.Hour,
				WriteTimeout:                 time.Second,
				MaxConnectionsPerUserPerTrip: 5,
			})
			ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
			ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
			collaboratorToken := signAccessToken(t, tc.user, tc.email, testJWTSecret, time.Hour)
			tripID := createCompletedTripForRouteTest(t, router, ownerToken)
			inviteID := inviteActivityCollaborator(t, router, ownerToken, tripID, tc.email, tc.role)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/collaborators/"+inviteID+"/accept", nil)
			req.Header.Set("Authorization", "Bearer "+collaboratorToken)
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected accept HTTP 200, got %d with %s", rec.Code, rec.Body.String())
			}

			ctx, cancel := context.WithCancel(context.Background())
			streamReq := httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/activity/stream", nil).WithContext(ctx)
			streamReq.Header.Set("Authorization", "Bearer "+collaboratorToken)
			streamRec := httptest.NewRecorder()
			done := make(chan struct{})
			go func() {
				router.ServeHTTP(streamRec, streamReq)
				close(done)
			}()
			waitForActivityClients(t, manager, uuid.MustParse(tripID), 1)
			manager.Publish(context.Background(), uuid.MustParse(tripID), activity.EventDTO{
				ID:        uuid.New(),
				TripID:    uuid.MustParse(tripID),
				EventType: activity.EventCommentCreated,
				CreatedAt: time.Now().UTC(),
			})
			time.Sleep(50 * time.Millisecond)
			cancel()
			<-done

			if streamRec.Code != http.StatusOK {
				t.Fatalf("expected stream HTTP 200, got %d with %s", streamRec.Code, streamRec.Body.String())
			}
			if !strings.Contains(streamRec.Body.String(), "event: activity.created\n") {
				t.Fatalf("expected activity event, got %q", streamRec.Body.String())
			}
		})
	}
}

func TestActivityStreamAccessDeniedCases(t *testing.T) {
	router, _, _ := newActivityStreamTestRouter(t, activitystream.Config{Enabled: true})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	editorID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	otherID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	editorToken := signAccessToken(t, editorID, "editor@example.com", testJWTSecret, time.Hour)
	otherToken := signAccessToken(t, otherID, "other@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/activity/stream", nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-collaborator HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	inviteID := inviteActivityCollaborator(t, router, ownerToken, tripID, "editor@example.com", "editor")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/activity/stream", nil)
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
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+tripID+"/collaborators/"+inviteID, nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected remove HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/activity/stream", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected removed collaborator HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestActivityStreamDisabledReturns503(t *testing.T) {
	router, _, _ := newActivityStreamTestRouter(t, activitystream.Config{Enabled: false})
	userID := uuid.New()
	token := signAccessToken(t, userID, "user@example.com", testJWTSecret, time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+uuid.NewString()+"/activity/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected HTTP 503, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestActivityStreamMaxConnectionsReturns429(t *testing.T) {
	router, _, manager := newActivityStreamTestRouter(t, activitystream.Config{
		Enabled:                      true,
		MaxConnectionsPerUserPerTrip: 1,
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := uuid.MustParse(createCompletedTripForRouteTest(t, router, ownerToken))
	if _, err := manager.Register(context.Background(), activitystream.RegisterClientInput{
		ConnectionID: uuid.NewString(),
		TripID:       tripID,
		UserID:       ownerID,
		Role:         "owner",
	}); err != nil {
		t.Fatalf("register existing activity stream: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+tripID.String()+"/activity/stream", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP 429, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestActivityStreamPublicShareTokenCannotConnect(t *testing.T) {
	router, _, _ := newActivityStreamTestRouter(t, activitystream.Config{Enabled: true})
	publicShareToken := signPublicShareAccessToken(t, "share-token", time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+uuid.NewString()+"/activity/stream", nil)
	req.Header.Set("Authorization", "Bearer "+publicShareToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected public share token HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}
}

func newActivityStreamTestRouter(t *testing.T, cfg activitystream.Config) (http.Handler, *routeTestRepo, activitystream.Manager) {
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
	activityStreamCfg := activitystream.Normalize(cfg)
	manager := activitystream.NewManager(activityStreamCfg, zap.NewNop())
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
		service.WithActivity(activity.New(repo, zap.NewNop(), activity.WithPublisher(manager))),
	)
	validator, err := validation.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}
	tripHandler := handler.New(svc, validator, zap.NewNop()).EnableActivityStream(manager, activityStreamCfg)
	ready := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return NewRouter(zap.NewNop(), tripHandler, ready, config.CORSConfig{}, authCfg, config.OpsConfig{}), repo, manager
}

func waitForActivityClients(t *testing.T, manager activitystream.Manager, tripID uuid.UUID, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got := manager.ClientCount(tripID); got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %d activity stream clients, got %d", want, manager.ClientCount(tripID))
}

func inviteActivityCollaborator(t *testing.T, router http.Handler, ownerToken, tripID, email, role string) string {
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
