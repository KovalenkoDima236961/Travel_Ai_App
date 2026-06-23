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

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
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

func TestUpdateItineraryRequiresValidBearerToken(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	tripID := uuid.New().String()

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "missing token"},
		{name: "invalid token", token: "Bearer invalid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSON())))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected HTTP 401, got %d with %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPartialRegenerationRequiresValidBearerToken(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	tripID := uuid.New().String()

	for _, tc := range []struct {
		name   string
		method string
		path   string
		token  string
	}{
		{name: "day missing token", method: http.MethodPost, path: "/trips/" + tripID + "/itinerary/days/1/regenerate"},
		{name: "day invalid token", method: http.MethodPost, path: "/trips/" + tripID + "/itinerary/days/1/regenerate", token: "Bearer invalid-token"},
		{name: "item missing token", method: http.MethodPost, path: "/trips/" + tripID + "/itinerary/days/1/items/0/regenerate"},
		{name: "item invalid token", method: http.MethodPost, path: "/trips/" + tripID + "/itinerary/days/1/items/0/regenerate", token: "Bearer invalid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected HTTP 401, got %d with %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestItineraryVersionRoutesRequireValidBearerToken(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	tripID := uuid.New().String()
	versionID := uuid.New().String()

	for _, tc := range []struct {
		name   string
		method string
		path   string
		token  string
	}{
		{name: "list missing token", method: http.MethodGet, path: "/trips/" + tripID + "/itinerary/versions"},
		{name: "list invalid token", method: http.MethodGet, path: "/trips/" + tripID + "/itinerary/versions", token: "Bearer invalid-token"},
		{name: "detail missing token", method: http.MethodGet, path: "/trips/" + tripID + "/itinerary/versions/" + versionID},
		{name: "detail invalid token", method: http.MethodGet, path: "/trips/" + tripID + "/itinerary/versions/" + versionID, token: "Bearer invalid-token"},
		{name: "restore missing token", method: http.MethodPost, path: "/trips/" + tripID + "/itinerary/versions/" + versionID + "/restore"},
		{name: "restore invalid token", method: http.MethodPost, path: "/trips/" + tripID + "/itinerary/versions/" + versionID + "/restore", token: "Bearer invalid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected HTTP 401, got %d with %s", rec.Code, rec.Body.String())
			}
		})
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

func TestPartialRegenerationOwnerCanUpdateAndNonOwnerReceives404(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
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
		t.Fatalf("expected create HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/itinerary/days/1/regenerate", bytes.NewReader([]byte(`{"instruction":"make it cheaper"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner regenerate day HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Itinerary struct {
			Days []struct {
				Title string `json:"title"`
			} `json:"days"`
		} `json:"itinerary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode regenerate response: %v", err)
	}
	if len(updated.Itinerary.Days) != 1 || updated.Itinerary.Days[0].Title != "Regenerated Day" {
		t.Fatalf("expected regenerated day in response, got %+v", updated.Itinerary.Days)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/itinerary/days/1/items/0/regenerate", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner regenerate item HTTP 404, got %d with %s", rec.Code, rec.Body.String())
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

func TestUpdateItineraryOwnerCanEditAndChangesPersist(t *testing.T) {
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
		t.Fatalf("expected create HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	tripID := uuid.MustParse(created.ID)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	var updated struct {
		Status    entity.Status `json:"status"`
		Itinerary struct {
			Days []struct {
				Title string `json:"title"`
				Items []struct {
					Name string `json:"name"`
				} `json:"items"`
			} `json:"days"`
		} `json:"itinerary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Status != entity.StatusCompleted {
		t.Fatalf("expected update status COMPLETED, got %s", updated.Status)
	}
	if updated.Itinerary.Days[0].Title != "Edited Day" {
		t.Fatalf("expected edited day title, got %+v", updated.Itinerary.Days)
	}
	if updated.Itinerary.Days[0].Items[0].Name != "Edited Activity" {
		t.Fatalf("expected edited item name, got %+v", updated.Itinerary.Days[0].Items)
	}
	if repo.trips[tripID].Status != entity.StatusCompleted || len(repo.trips[tripID].Itinerary) == 0 {
		t.Fatalf("expected repository to persist completed itinerary, got %+v", repo.trips[tripID])
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner get HTTP 200 after update, got %d with %s", rec.Code, rec.Body.String())
	}
	var fetched struct {
		Status    entity.Status `json:"status"`
		Itinerary struct {
			Days []struct {
				Title string `json:"title"`
				Items []struct {
					Name string `json:"name"`
				} `json:"items"`
			} `json:"days"`
		} `json:"itinerary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if fetched.Status != entity.StatusCompleted ||
		fetched.Itinerary.Days[0].Title != "Edited Day" ||
		fetched.Itinerary.Days[0].Items[0].Name != "Edited Activity" {
		t.Fatalf("expected GET to return persisted edited itinerary, got %+v", fetched)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner update HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateItineraryValidationErrors(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(validCreateTripJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	cases := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing itinerary", body: `{}`},
		{name: "empty days", body: `{"itinerary":{"days":[]}}`},
		{name: "missing item name", body: `{"itinerary":{"days":[{"day":1,"title":"Day","items":[{"time":"09:00","type":"activity","name":" "}]}]}}`},
		{name: "negative estimated cost", body: `{"itinerary":{"days":[{"day":1,"title":"Day","items":[{"time":"09:00","type":"activity","name":"Walk","estimatedCost":-1}]}]}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+ownerToken)
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected HTTP 400, got %d with %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestItineraryVersionHistoryOwnerCanPreviewRestoreAndNonOwnerReceives404(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
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
		t.Fatalf("expected create HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/generate", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected generate HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected edit HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+created.ID+"/itinerary/versions?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected list versions HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID            string                        `json:"id"`
			VersionNumber int                           `json:"versionNumber"`
			Source        entity.ItineraryVersionSource `json:"source"`
			Itinerary     json.RawMessage               `json:"itinerary"`
			Metadata      map[string]any                `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list versions response: %v", err)
	}
	if len(listResp.Items) != 2 {
		t.Fatalf("expected two versions, got %+v", listResp.Items)
	}
	if len(listResp.Items[0].Itinerary) != 0 {
		t.Fatalf("list response must not include itinerary JSON, got %+v", listResp.Items[0])
	}

	var generatedVersionID string
	for _, item := range listResp.Items {
		if item.Source == entity.ItineraryVersionSourceGenerated {
			generatedVersionID = item.ID
			break
		}
	}
	if generatedVersionID == "" {
		t.Fatalf("expected generated version in list, got %+v", listResp.Items)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+created.ID+"/itinerary/versions/"+generatedVersionID, nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected get version HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var detailResp struct {
		Source    entity.ItineraryVersionSource `json:"source"`
		Itinerary struct {
			Days []struct {
				Title string `json:"title"`
			} `json:"days"`
		} `json:"itinerary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("decode version detail: %v", err)
	}
	if detailResp.Source != entity.ItineraryVersionSourceGenerated ||
		len(detailResp.Itinerary.Days) != 1 ||
		detailResp.Itinerary.Days[0].Title != "Arrival" {
		t.Fatalf("expected generated itinerary detail, got %+v", detailResp)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/itinerary/versions/"+generatedVersionID+"/restore", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected restore HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var restoredResp struct {
		Itinerary struct {
			Days []struct {
				Title string `json:"title"`
			} `json:"days"`
		} `json:"itinerary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &restoredResp); err != nil {
		t.Fatalf("decode restore response: %v", err)
	}
	if len(restoredResp.Itinerary.Days) != 1 || restoredResp.Itinerary.Days[0].Title != "Arrival" {
		t.Fatalf("expected current itinerary restored to generated version, got %+v", restoredResp.Itinerary.Days)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+created.ID+"/itinerary/versions", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected list after restore HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode restored versions response: %v", err)
	}
	if len(listResp.Items) != 3 || listResp.Items[0].Source != entity.ItineraryVersionSourceRestored {
		t.Fatalf("expected restore to append latest RESTORED version, got %+v", listResp.Items)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+created.ID+"/itinerary/versions", nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner list HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+created.ID+"/itinerary/versions/"+generatedVersionID, nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner get version HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/itinerary/versions/"+generatedVersionID+"/restore", nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner restore HTTP 404, got %d with %s", rec.Code, rec.Body.String())
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
	trips    map[uuid.UUID]entity.Trip
	versions []entity.ItineraryVersion
	created  *entity.Trip
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

func (r *routeTestRepo) UpdateItineraryByUserIDAndCreateVersion(
	_ context.Context,
	id, userID uuid.UUID,
	itinerary json.RawMessage,
	status entity.Status,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
) (*entity.Trip, *entity.ItineraryVersion, error) {
	trip, err := r.GetByIDAndUserID(context.Background(), id, userID)
	if err != nil {
		return nil, nil, err
	}
	trip.Itinerary = itinerary
	trip.Status = status
	trip.UpdatedAt = time.Now().UTC()
	r.trips[id] = *trip
	version := entity.ItineraryVersion{
		ID:            uuid.New(),
		TripID:        id,
		UserID:        userID,
		VersionNumber: routeTestNextVersionNumber(r.versions, id),
		Source:        source,
		Itinerary:     itinerary,
		Metadata:      metadata,
		CreatedAt:     time.Now().UTC(),
	}
	r.versions = append(r.versions, version)
	return trip, &version, nil
}

func (r *routeTestRepo) ListItineraryVersionsByTripAndUser(_ context.Context, tripID, userID uuid.UUID, limit, offset int) ([]entity.ItineraryVersion, error) {
	versions := make([]entity.ItineraryVersion, 0)
	for i := len(r.versions) - 1; i >= 0; i-- {
		version := r.versions[i]
		if version.TripID == tripID && version.UserID == userID {
			versions = append(versions, version)
		}
	}
	if offset >= len(versions) {
		return []entity.ItineraryVersion{}, nil
	}
	end := offset + limit
	if end > len(versions) {
		end = len(versions)
	}
	return versions[offset:end], nil
}

func (r *routeTestRepo) GetItineraryVersionByIDTripAndUser(_ context.Context, id, tripID, userID uuid.UUID) (*entity.ItineraryVersion, error) {
	for i := range r.versions {
		version := r.versions[i]
		if version.ID == id && version.TripID == tripID && version.UserID == userID {
			return &version, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func routeTestNextVersionNumber(versions []entity.ItineraryVersion, tripID uuid.UUID) int {
	next := 1
	for _, version := range versions {
		if version.TripID == tripID && version.VersionNumber >= next {
			next = version.VersionNumber + 1
		}
	}
	return next
}

type routeTestGenerator struct{}

func (routeTestGenerator) Generate(_ context.Context, input application.GenerateItineraryInput) (*aggregate.Itinerary, error) {
	trip := input.Trip
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

func (routeTestGenerator) RegenerateDay(_ context.Context, input application.RegenerateDayInput) (*aggregate.ItineraryDay, error) {
	return &aggregate.ItineraryDay{
		Day:   input.DayNumber,
		Title: "Regenerated Day",
		Items: []aggregate.ItineraryItem{
			{Time: "11:00", Type: "activity", Name: "Regenerated Activity"},
		},
	}, nil
}

func (routeTestGenerator) RegenerateItem(_ context.Context, input application.RegenerateItemInput) (*aggregate.ItineraryItem, error) {
	return &aggregate.ItineraryItem{Time: "12:30", Type: "food", Name: "Regenerated Item"}, nil
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

func validUpdateItineraryJSON() string {
	return `{
		"itinerary": {
			"days": [
				{
					"day": 1,
					"title": "Edited Day",
					"items": [
						{
							"time": "10:00",
							"type": "activity",
							"name": "Edited Activity",
							"note": "Updated note",
							"estimatedCost": 12
						}
					]
				}
			]
		}
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
