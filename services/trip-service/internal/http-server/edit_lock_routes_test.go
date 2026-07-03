package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/editlocks"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

func TestEditLockRoutesRequireJWT(t *testing.T) {
	router, _ := newEditLockTestRouter(t, editlocks.Config{Enabled: true})
	tripID := uuid.NewString()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/trips/" + tripID + "/edit-lock"},
		{method: http.MethodPost, path: "/trips/" + tripID + "/edit-lock"},
		{method: http.MethodDelete, path: "/trips/" + tripID + "/edit-lock"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s expected HTTP 401, got %d with %s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestEditLockPermissionsAndConflictFlow(t *testing.T) {
	router, _ := newEditLockTestRouter(t, editlocks.Config{
		Enabled: true,
		TTL:     3 * time.Minute,
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	editorID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	viewerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	otherID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	editorToken := signAccessToken(t, editorID, "editor@example.com", testJWTSecret, time.Hour)
	viewerToken := signAccessToken(t, viewerID, "viewer@example.com", testJWTSecret, time.Hour)
	otherToken := signAccessToken(t, otherID, "other@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)
	editorInviteID := inviteEditLockCollaborator(t, router, ownerToken, tripID, "editor@example.com", "editor")
	viewerInviteID := inviteEditLockCollaborator(t, router, ownerToken, tripID, "viewer@example.com", "viewer")
	acceptEditLockInvite(t, router, editorToken, tripID, editorInviteID)
	acceptEditLockInvite(t, router, viewerToken, tripID, viewerInviteID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/edit-lock", bytes.NewReader([]byte(`{"scope":"itinerary"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner acquire HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var ownerAcquire struct {
		Acquired bool `json:"acquired"`
		Renewed  bool `json:"renewed"`
		Lock     struct {
			Locked              bool   `json:"locked"`
			LockedByUserID      string `json:"lockedByUserId"`
			LockedByCurrentUser bool   `json:"lockedByCurrentUser"`
			TTLSeconds          int    `json:"ttlSeconds"`
		} `json:"lock"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &ownerAcquire); err != nil {
		t.Fatalf("decode owner acquire: %v", err)
	}
	if !ownerAcquire.Acquired || !ownerAcquire.Lock.Locked || !ownerAcquire.Lock.LockedByCurrentUser ||
		ownerAcquire.Lock.LockedByUserID != ownerID.String() || ownerAcquire.Lock.TTLSeconds != 180 {
		t.Fatalf("unexpected owner acquire body: %+v", ownerAcquire)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner renew HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var ownerRenew struct {
		Acquired bool `json:"acquired"`
		Renewed  bool `json:"renewed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &ownerRenew); err != nil {
		t.Fatalf("decode owner renew: %v", err)
	}
	if !ownerRenew.Acquired || !ownerRenew.Renewed {
		t.Fatalf("expected renewed owner lock, got %+v", ownerRenew)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected editor conflict HTTP 409, got %d with %s", rec.Code, rec.Body.String())
	}
	var conflict struct {
		Error    string `json:"error"`
		Acquired bool   `json:"acquired"`
		Reason   string `json:"reason"`
		Lock     struct {
			Locked              bool   `json:"locked"`
			LockedByUserID      string `json:"lockedByUserId"`
			LockedByCurrentUser bool   `json:"lockedByCurrentUser"`
		} `json:"lock"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &conflict); err != nil {
		t.Fatalf("decode conflict: %v", err)
	}
	if conflict.Error != "edit_lock_conflict" || conflict.Acquired ||
		conflict.Reason != "locked_by_other_user" || !conflict.Lock.Locked ||
		conflict.Lock.LockedByUserID != ownerID.String() || conflict.Lock.LockedByCurrentUser {
		t.Fatalf("unexpected edit lock conflict body: %+v", conflict)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected viewer GET HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected viewer POST HTTP 403, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-collaborator GET HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected non-owner DELETE HTTP 403, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner DELETE HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var release struct {
		Released bool `json:"released"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &release); err != nil {
		t.Fatalf("decode release: %v", err)
	}
	if !release.Released {
		t.Fatalf("expected released=true, got %+v", release)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected editor acquire after release HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestEditLockPublicShareTokenAndDisabledBehavior(t *testing.T) {
	router, _ := newEditLockTestRouter(t, editlocks.Config{Enabled: false})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)

	publicShareToken := signPublicShareAccessToken(t, "share-token", time.Hour)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+publicShareToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected public share token HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected disabled GET HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var disabledGet struct {
		Locked   bool `json:"locked"`
		Disabled bool `json:"disabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &disabledGet); err != nil {
		t.Fatalf("decode disabled get: %v", err)
	}
	if disabledGet.Locked || !disabledGet.Disabled {
		t.Fatalf("expected disabled unlocked view, got %+v", disabledGet)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected disabled POST HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var disabledPost struct {
		Acquired bool `json:"acquired"`
		Disabled bool `json:"disabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &disabledPost); err != nil {
		t.Fatalf("decode disabled post: %v", err)
	}
	if !disabledPost.Acquired || !disabledPost.Disabled {
		t.Fatalf("expected disabled acquire success, got %+v", disabledPost)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+tripID+"/edit-lock", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected disabled DELETE HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var disabledDelete struct {
		Released bool `json:"released"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &disabledDelete); err != nil {
		t.Fatalf("decode disabled delete: %v", err)
	}
	if disabledDelete.Released {
		t.Fatalf("expected disabled release false, got %+v", disabledDelete)
	}
}

func newEditLockTestRouter(t *testing.T, cfg editlocks.Config) (http.Handler, *routeTestRepo) {
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
	editLockCfg := editlocks.Normalize(cfg)
	tripHandler := handler.New(svc, validator, zap.NewNop()).
		EnableEditLocks(editlocks.NewManager(), editLockCfg)
	ready := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return NewRouter(zap.NewNop(), tripHandler, ready, config.CORSConfig{}, authCfg, config.OpsConfig{}), repo
}

func inviteEditLockCollaborator(t *testing.T, router http.Handler, ownerToken, tripID, email, role string) string {
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

func acceptEditLockInvite(t *testing.T, router http.Handler, token, tripID, inviteID string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/collaborators/"+inviteID+"/accept", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accept HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
}
