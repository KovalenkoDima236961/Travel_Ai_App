package httpserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/httpserver/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/validation"
)

const testJWTSecret = "test-secret"

func TestProtectedUserRoutesRequireValidBearerToken(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/users/me/profile", nil),
		httptest.NewRequest(http.MethodPatch, "/users/me/preferences", strings.NewReader(`{"pace":"balanced"}`)),
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected HTTP 401 for missing token, got %d with %s", rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me/profile", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 for invalid token, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestHealthAndReadyRemainPublic(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})

	for _, path := range []string{"/health", "/ready"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to be public with HTTP 200, got %d", path, rec.Code)
		}
	}
}

func TestValidTokenAllowsProfileAndPreferences(t *testing.T) {
	router, repo := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	token := signAccessToken(t, userID, "owner@example.com", testJWTSecret, time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected profile HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	var profileResp struct {
		UserID            string `json:"userId"`
		PreferredCurrency string `json:"preferredCurrency"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &profileResp); err != nil {
		t.Fatalf("decode profile response: %v", err)
	}
	if profileResp.UserID != userID.String() || profileResp.PreferredCurrency != "EUR" {
		t.Fatalf("unexpected profile response: %+v", profileResp)
	}
	if repo.profiles[userID].UserID != userID {
		t.Fatalf("expected stored profile for %s", userID)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/me/profile", strings.NewReader(`{
		"displayName":"Test Traveler",
		"homeCity":"Bratislava",
		"homeCountry":"Slovakia",
		"preferredCurrency":"EUR",
		"preferredLanguage":"en"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected update profile HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/users/me/preferences", strings.NewReader(`{
		"travelStyles":[" budget ","food","budget",""],
		"pace":"balanced",
		"maxWalkingKmPerDay":8
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected patch preferences HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	var preferencesResp struct {
		UserID             string   `json:"userId"`
		TravelStyles       []string `json:"travelStyles"`
		MaxWalkingKmPerDay *float64 `json:"maxWalkingKmPerDay"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &preferencesResp); err != nil {
		t.Fatalf("decode preferences response: %v", err)
	}
	if preferencesResp.UserID != userID.String() || len(preferencesResp.TravelStyles) != 2 {
		t.Fatalf("unexpected preferences response: %+v", preferencesResp)
	}
	if preferencesResp.MaxWalkingKmPerDay == nil || *preferencesResp.MaxWalkingKmPerDay != 8 {
		t.Fatalf("expected maxWalkingKmPerDay=8, got %+v", preferencesResp.MaxWalkingKmPerDay)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/users/me/preferences", strings.NewReader(`{
		"maxWalkingKmPerDay":null,
		"avoid":["nightclubs"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected clear preferences HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	preferencesResp = struct {
		UserID             string   `json:"userId"`
		TravelStyles       []string `json:"travelStyles"`
		MaxWalkingKmPerDay *float64 `json:"maxWalkingKmPerDay"`
	}{}
	if err := json.Unmarshal(rec.Body.Bytes(), &preferencesResp); err != nil {
		t.Fatalf("decode cleared preferences response: %v", err)
	}
	if preferencesResp.MaxWalkingKmPerDay != nil {
		t.Fatalf("expected maxWalkingKmPerDay to clear, got %+v", preferencesResp.MaxWalkingKmPerDay)
	}
}

func newAuthTestRouter(t *testing.T, authCfg config.AuthConfig) (http.Handler, *routeTestRepo) {
	t.Helper()

	repo := &routeTestRepo{
		profiles:    map[uuid.UUID]entity.Profile{},
		preferences: map[uuid.UUID]entity.Preferences{},
	}
	svc := service.New(repo, zap.NewNop())
	validator, err := validation.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}
	userHandler := handler.New(svc, validator, zap.NewNop())
	ready := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return NewRouter(zap.NewNop(), userHandler, nil, ready, config.CORSConfig{}, authCfg, config.InternalConfig{
		ServiceToken: "test-internal-token",
	}), repo
}

type routeTestRepo struct {
	profiles    map[uuid.UUID]entity.Profile
	preferences map[uuid.UUID]entity.Preferences
}

func (r *routeTestRepo) GetProfileByUserID(_ context.Context, userID uuid.UUID) (*entity.Profile, error) {
	profile, ok := r.profiles[userID]
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	return &profile, nil
}

func (r *routeTestRepo) CreateDefaultProfile(_ context.Context, userID uuid.UUID) (*entity.Profile, error) {
	if profile, ok := r.profiles[userID]; ok {
		return &profile, nil
	}
	now := time.Now().UTC()
	profile := entity.Profile{
		UserID:            userID,
		PreferredCurrency: "EUR",
		PreferredLanguage: "en",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	r.profiles[userID] = profile
	return &profile, nil
}

func (r *routeTestRepo) UpsertProfile(_ context.Context, profile *entity.Profile) (*entity.Profile, error) {
	now := time.Now().UTC()
	out := *profile
	if existing, ok := r.profiles[profile.UserID]; ok {
		out.CreatedAt = existing.CreatedAt
	} else {
		out.CreatedAt = now
	}
	out.UpdatedAt = now
	r.profiles[profile.UserID] = out
	return &out, nil
}

func (r *routeTestRepo) GetPreferencesByUserID(_ context.Context, userID uuid.UUID) (*entity.Preferences, error) {
	preferences, ok := r.preferences[userID]
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	return &preferences, nil
}

func (r *routeTestRepo) CreateDefaultPreferences(_ context.Context, userID uuid.UUID) (*entity.Preferences, error) {
	if preferences, ok := r.preferences[userID]; ok {
		return &preferences, nil
	}
	now := time.Now().UTC()
	preferences := entity.Preferences{
		UserID:              userID,
		TravelStyles:        []string{},
		Pace:                "balanced",
		FoodPreferences:     []string{},
		Avoid:               []string{},
		PreferredTransport:  []string{},
		AccommodationStyle:  []string{},
		DietaryRestrictions: []string{},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	r.preferences[userID] = preferences
	return &preferences, nil
}

func (r *routeTestRepo) UpsertPreferences(_ context.Context, preferences *entity.Preferences) (*entity.Preferences, error) {
	now := time.Now().UTC()
	out := *preferences
	if existing, ok := r.preferences[preferences.UserID]; ok {
		out.CreatedAt = existing.CreatedAt
	} else {
		out.CreatedAt = now
	}
	out.UpdatedAt = now
	r.preferences[preferences.UserID] = out
	return &out, nil
}

func signAccessToken(t *testing.T, userID uuid.UUID, email, secret string, ttl time.Duration) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().UTC()
	payloadBytes, err := json.Marshal(map[string]any{
		"sub":   userID.String(),
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(ttl).Unix(),
	})
	if err != nil {
		t.Fatalf("marshal token payload: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(header + "." + payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + payload + "." + signature
}
