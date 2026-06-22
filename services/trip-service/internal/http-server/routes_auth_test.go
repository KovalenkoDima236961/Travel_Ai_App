package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

const testJWTSecret = "test-secret"

func TestProtectedTripRoutesRequireValidBearerToken(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(validCreateTripJSON())))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 for missing token, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(validCreateTripJSON())))
	req.Header.Set("Content-Type", "application/json")
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

func TestValidTokenCreatesAndScopesTripsToCurrentUser(t *testing.T) {
	router, repo := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	otherToken := signAccessToken(t, otherID, "other@example.com", testJWTSecret, time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(validCreateTripJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	if repo.created == nil || repo.created.UserID == nil || *repo.created.UserID != ownerID {
		t.Fatalf("expected created trip owner %s, got %+v", ownerID, repo.created)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	tripID := uuid.MustParse(created.ID)

	foreignTrip := entity.Trip{
		ID:             uuid.New(),
		UserID:         &otherID,
		Destination:    "Paris",
		Days:           2,
		BudgetCurrency: "EUR",
		Travelers:      1,
		Interests:      []string{},
		Pace:           "balanced",
		Status:         entity.StatusDraft,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.trips[foreignTrip.ID] = foreignTrip

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected list HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].ID != tripID.String() {
		t.Fatalf("expected list to include only owner trip %s, got %+v", tripID, listResp.Items)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner get HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner get HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID.String()+"/generate", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner generate HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID.String()+"/generate", nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner generate HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestAuthDisabledUsesDevUserID(t *testing.T) {
	devUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	router, repo := newAuthTestRouter(t, config.AuthConfig{
		Required:        false,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       devUserID.String(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(validCreateTripJSON())))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 with auth disabled, got %d with %s", rec.Code, rec.Body.String())
	}
	if repo.created == nil || repo.created.UserID == nil || *repo.created.UserID != devUserID {
		t.Fatalf("expected dev user id %s, got %+v", devUserID, repo.created)
	}
}

func newAuthTestRouter(t *testing.T, authCfg config.AuthConfig) (http.Handler, *routeTestRepo) {
	t.Helper()

	repo := &routeTestRepo{trips: map[uuid.UUID]entity.Trip{}}
	gen := routeTestGenerator{}
	svc := service.New(repo, gen, zap.NewNop())
	validator, err := validation.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}
	tripHandler := handler.New(svc, validator, zap.NewNop())
	ready := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return NewRouter(zap.NewNop(), tripHandler, ready, config.CORSConfig{}, authCfg), repo
}

type routeTestRepo struct {
	trips   map[uuid.UUID]entity.Trip
	created *entity.Trip
}

func (r *routeTestRepo) Create(_ context.Context, t *entity.Trip) (*entity.Trip, error) {
	now := time.Now().UTC()
	out := *t
	out.ID = uuid.New()
	out.CreatedAt = now
	out.UpdatedAt = now
	r.trips[out.ID] = out
	r.created = &out
	return &out, nil
}

func (r *routeTestRepo) GetByIDAndUserID(_ context.Context, id, userID uuid.UUID) (*entity.Trip, error) {
	trip, ok := r.trips[id]
	if !ok || trip.UserID == nil || *trip.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	return &trip, nil
}

func (r *routeTestRepo) ListByUser(_ context.Context, userID uuid.UUID, _, _ int) ([]entity.Trip, error) {
	trips := make([]entity.Trip, 0)
	for _, trip := range r.trips {
		if trip.UserID != nil && *trip.UserID == userID {
			trips = append(trips, trip)
		}
	}
	return trips, nil
}

func (r *routeTestRepo) UpdateStatusByUserID(_ context.Context, id, userID uuid.UUID, status entity.Status) (*entity.Trip, error) {
	trip, err := r.GetByIDAndUserID(context.Background(), id, userID)
	if err != nil {
		return nil, err
	}
	trip.Status = status
	trip.UpdatedAt = time.Now().UTC()
	r.trips[id] = *trip
	return trip, nil
}

func (r *routeTestRepo) UpdateItineraryByUserID(_ context.Context, id, userID uuid.UUID, itinerary json.RawMessage, status entity.Status) (*entity.Trip, error) {
	trip, err := r.GetByIDAndUserID(context.Background(), id, userID)
	if err != nil {
		return nil, err
	}
	trip.Itinerary = itinerary
	trip.Status = status
	trip.UpdatedAt = time.Now().UTC()
	r.trips[id] = *trip
	return trip, nil
}

type routeTestGenerator struct{}

func (routeTestGenerator) Generate(_ context.Context, trip entity.Trip) (*aggregate.Itinerary, error) {
	return &aggregate.Itinerary{
		Destination: trip.Destination,
		Days: []aggregate.ItineraryDay{
			{
				Day:   1,
				Title: "Arrival",
				Items: []aggregate.ItineraryItem{
					{Time: "09:00", Type: "activity", Name: "Explore"},
				},
			},
		},
	}, nil
}

func validCreateTripJSON() string {
	return `{
		"destination": "Rome",
		"startDate": "2026-08-10",
		"days": 2,
		"budgetAmount": 500,
		"budgetCurrency": "EUR",
		"travelers": 2,
		"interests": ["food", "history"],
		"pace": "balanced"
	}`
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
