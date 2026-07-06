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
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

const (
	testJWTSecret         = "test-secret"
	testPublicShareSecret = "test-public-share-secret"
)

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

func TestShareRoutesRequireValidBearerToken(t *testing.T) {
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
		{name: "get missing token", method: http.MethodGet, path: "/trips/" + tripID + "/share"},
		{name: "get invalid token", method: http.MethodGet, path: "/trips/" + tripID + "/share", token: "Bearer invalid-token"},
		{name: "create missing token", method: http.MethodPost, path: "/trips/" + tripID + "/share"},
		{name: "create invalid token", method: http.MethodPost, path: "/trips/" + tripID + "/share", token: "Bearer invalid-token"},
		{name: "patch missing token", method: http.MethodPatch, path: "/trips/" + tripID + "/share"},
		{name: "patch invalid token", method: http.MethodPatch, path: "/trips/" + tripID + "/share", token: "Bearer invalid-token"},
		{name: "delete missing token", method: http.MethodDelete, path: "/trips/" + tripID + "/share"},
		{name: "delete invalid token", method: http.MethodDelete, path: "/trips/" + tripID + "/share", token: "Bearer invalid-token"},
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

func TestPublicTripSharingOwnerFlowAndSanitizedPublicResponse(t *testing.T) {
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
	req = httptest.NewRequest(http.MethodGet, "/trips/"+created.ID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected initial share status HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var initialShare struct {
		Enabled    bool   `json:"enabled"`
		ShareToken string `json:"shareToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &initialShare); err != nil {
		t.Fatalf("decode initial share: %v", err)
	}
	if initialShare.Enabled || initialShare.ShareToken != "" {
		t.Fatalf("expected disabled initial share without token, got %+v", initialShare)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected create share HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var share struct {
		ShareToken string `json:"shareToken"`
		ShareURL   string `json:"shareUrl"`
		Enabled    bool   `json:"enabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatalf("decode share response: %v", err)
	}
	if !share.Enabled || len(share.ShareToken) < 43 || share.ShareURL == "" {
		t.Fatalf("expected enabled secure share response, got %+v", share)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected repeated create share HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var repeatedShare struct {
		ShareToken string `json:"shareToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &repeatedShare); err != nil {
		t.Fatalf("decode repeated share: %v", err)
	}
	if repeatedShare.ShareToken != share.ShareToken {
		t.Fatalf("expected repeated create to return existing token %q, got %q", share.ShareToken, repeatedShare.ShareToken)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner create share HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected public trip HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var publicBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &publicBody); err != nil {
		t.Fatalf("decode public trip: %v", err)
	}
	if publicBody["destination"] != "Rome" {
		t.Fatalf("expected public destination Rome, got %+v", publicBody)
	}
	if _, ok := publicBody["userId"]; ok {
		t.Fatalf("public response must not include userId: %+v", publicBody)
	}
	if _, ok := publicBody["email"]; ok {
		t.Fatalf("public response must not include email: %+v", publicBody)
	}
	if _, ok := publicBody["versionHistory"]; ok {
		t.Fatalf("public response must not include version history: %+v", publicBody)
	}
	// The private trip budget must never appear on the public share, even though
	// the trip was created with a budget.
	if _, ok := publicBody["budgetAmount"]; ok {
		t.Fatalf("public response must not include budgetAmount: %+v", publicBody)
	}
	if _, ok := publicBody["budgetCurrency"]; ok {
		t.Fatalf("public response must not include budgetCurrency: %+v", publicBody)
	}
	itinerary, ok := publicBody["itinerary"].(map[string]any)
	days, daysOK := itinerary["days"].([]any)
	if !ok || !daysOK || len(days) == 0 {
		t.Fatalf("expected public itinerary days, got %+v", publicBody["itinerary"])
	}
	if _, ok := itinerary["totalBudget"]; ok {
		t.Fatalf("public itinerary must not include totalBudget: %+v", itinerary)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+created.ID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected disable share HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected disabled public share HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+created.ID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected idempotent disable share HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected re-enable share HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var reenabledShare struct {
		ShareToken string `json:"shareToken"`
		Enabled    bool   `json:"enabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reenabledShare); err != nil {
		t.Fatalf("decode re-enabled share: %v", err)
	}
	if !reenabledShare.Enabled || reenabledShare.ShareToken != share.ShareToken {
		t.Fatalf("expected re-enable to keep original token, got %+v", reenabledShare)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/unknown-token", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected unknown public share HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestPasswordProtectedShareUnlockFlowAndOldTokensAreBlocked(t *testing.T) {
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
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)
	expiresAt := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)

	rec := httptest.NewRecorder()
	body := `{"expiresAt":"` + expiresAt + `","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/share", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected protected share create HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var share struct {
		ShareToken       string  `json:"shareToken"`
		ShareURL         string  `json:"shareUrl"`
		Enabled          bool    `json:"enabled"`
		ExpiresAt        *string `json:"expiresAt"`
		Expired          bool    `json:"expired"`
		PasswordRequired bool    `json:"passwordRequired"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatalf("decode protected share: %v", err)
	}
	if !share.Enabled || share.ShareToken == "" || share.ShareURL == "" || share.ExpiresAt == nil || share.Expired || !share.PasswordRequired {
		t.Fatalf("expected enabled password-protected expiring share, got %+v", share)
	}
	var rawShare map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rawShare); err != nil {
		t.Fatalf("decode raw protected share: %v", err)
	}
	if _, ok := rawShare["passwordHash"]; ok {
		t.Fatalf("share API must not expose passwordHash: %+v", rawShare)
	}
	stored := repo.sharesByToken[share.ShareToken]
	if stored.PasswordHash == nil || *stored.PasswordHash == "secret123" || !stored.PasswordRequired {
		t.Fatalf("expected stored bcrypt password hash, got %+v", stored)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken+"/status", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected protected status HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var status struct {
		Available        bool `json:"available"`
		PasswordRequired bool `json:"passwordRequired"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !status.Available || !status.PasswordRequired {
		t.Fatalf("expected password-required status, got %+v", status)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected protected public trip without token HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}

	wrongShareTokenAccess := signPublicShareAccessToken(t, "different-token", time.Hour)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	req.Header.Set("Authorization", "Bearer "+wrongShareTokenAccess)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected different-share token HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}

	expiredAccess := signPublicShareAccessToken(t, share.ShareToken, -time.Minute)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	req.Header.Set("Authorization", "Bearer "+expiredAccess)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected expired public token HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/public/trips/"+share.ShareToken+"/unlock", bytes.NewReader([]byte(`{"password":"wrong-password"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected wrong password HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/public/trips/"+share.ShareToken+"/unlock", bytes.NewReader([]byte(`{"password":"secret123"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected unlock HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var unlock struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   string `json:"expiresAt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &unlock); err != nil {
		t.Fatalf("decode unlock: %v", err)
	}
	if unlock.AccessToken == "" || unlock.ExpiresAt == "" {
		t.Fatalf("expected unlock token and expiry, got %+v", unlock)
	}
	claims := decodeJWTPayload(t, unlock.AccessToken)
	if claims["typ"] != "public_share" || claims["shareToken"] != share.ShareToken {
		t.Fatalf("unexpected public share token claims: %+v", claims)
	}
	if _, ok := claims["sub"]; ok {
		t.Fatalf("public share token must not include sub: %+v", claims)
	}
	if _, ok := claims["email"]; ok {
		t.Fatalf("public share token must not include email: %+v", claims)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	req.Header.Set("Authorization", "Bearer "+unlock.AccessToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected unlocked public trip HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID, nil)
	req.Header.Set("Authorization", "Bearer "+unlock.AccessToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected public share token to fail private route with HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/trips/"+tripID+"/share", bytes.NewReader([]byte(`{"clearPassword":true}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner patch HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+tripID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected disable HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	req.Header.Set("Authorization", "Bearer "+unlock.AccessToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected disabled share to block old token with HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestShareSettingsPatchClearAndExpirationRules(t *testing.T) {
	router, repo := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)
	expiresAt := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/share", bytes.NewReader([]byte(`{"expiresAt":"`+expiresAt+`","password":"secret123"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected share create HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var share struct {
		ShareToken string `json:"shareToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatalf("decode share: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/trips/"+tripID+"/share", bytes.NewReader([]byte(`{"clearPassword":true,"clearExpiration":true}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected clear settings HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var cleared struct {
		ExpiresAt        *string `json:"expiresAt"`
		PasswordRequired bool    `json:"passwordRequired"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &cleared); err != nil {
		t.Fatalf("decode cleared settings: %v", err)
	}
	if cleared.ExpiresAt != nil || cleared.PasswordRequired {
		t.Fatalf("expected cleared expiration and password, got %+v", cleared)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected cleared password public trip HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/trips/"+tripID+"/share", bytes.NewReader([]byte(`{"expiresAt":"`+past+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected past expiration HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/trips/"+tripID+"/share", bytes.NewReader([]byte(`{"password":"secret456","clearPassword":true}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected password conflict HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}

	shareRow := repo.sharesByToken[share.ShareToken]
	shareRow.ExpiresAt = timePtr(time.Now().UTC().Add(-time.Minute))
	shareRow.UpdatedAt = time.Now().UTC()
	repo.sharesByToken[shareRow.ShareToken] = shareRow
	repo.sharesByTrip[shareRow.TripID] = shareRow

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken+"/status", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected expired share status HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected expired public trip HTTP 404, got %d with %s", rec.Code, rec.Body.String())
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
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/itinerary/days/1/regenerate", bytes.NewReader([]byte(regenerateDayJSON(1, "make it cheaper"))))
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
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID.String()+"/generate", bytes.NewReader([]byte(expectedRevisionJSON(0))))
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

func TestItineraryRevisionConflictFlow(t *testing.T) {
	router, repo := newAuthTestRouter(t, config.AuthConfig{
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
		ID                string `json:"id"`
		ItineraryRevision int    `json:"itineraryRevision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ItineraryRevision != 0 {
		t.Fatalf("expected new trip revision 0, got %d", created.ItineraryRevision)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(`{"itinerary":{"days":[{"day":1,"title":"Day","items":[{"time":"09:00","type":"activity","name":"Walk"}]}]}}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing revision HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}
	var missingBody struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &missingBody); err != nil {
		t.Fatalf("decode missing revision response: %v", err)
	}
	if missingBody.Error != "expected_itinerary_revision_required" {
		t.Fatalf("unexpected missing revision error body: %+v", missingBody)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSONWithRevision(0))))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected revision 0 update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var updated struct {
		ItineraryRevision int `json:"itineraryRevision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.ItineraryRevision != 1 {
		t.Fatalf("expected update to increment revision to 1, got %d", updated.ItineraryRevision)
	}
	if len(repo.versions) != 1 {
		t.Fatalf("expected one version after successful update, got %d", len(repo.versions))
	}
	activityCount := len(repo.activityEvents)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSONWithRevision(0))))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected stale update HTTP 409, got %d with %s", rec.Code, rec.Body.String())
	}
	var conflictBody struct {
		Error                    string `json:"error"`
		Message                  string `json:"message"`
		CurrentItineraryRevision int    `json:"currentItineraryRevision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &conflictBody); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if conflictBody.Error != "itinerary_conflict" || conflictBody.CurrentItineraryRevision != 1 {
		t.Fatalf("unexpected conflict body: %+v", conflictBody)
	}
	if len(repo.versions) != 1 {
		t.Fatalf("conflict must not create version, got %d", len(repo.versions))
	}
	if len(repo.activityEvents) != activityCount {
		t.Fatalf("conflict must not create activity event")
	}
}

func TestCollaborativePlanningInviteAcceptRolesAndRemoval(t *testing.T) {
	router, _ := newAuthTestRouter(t, config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	})
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	viewerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	viewerToken := signAccessToken(t, viewerID, "viewer@example.com", testJWTSecret, time.Hour)
	tripID := createCompletedTripForRouteTest(t, router, ownerToken)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/collaborators", bytes.NewReader([]byte(`{"email":"viewer@example.com","role":"viewer"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected invite HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var invite struct {
		ID     string `json:"id"`
		UserID string `json:"userId"`
		Role   string `json:"role"`
		Status string `json:"status"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &invite); err != nil {
		t.Fatalf("decode invite: %v", err)
	}
	if invite.UserID != viewerID.String() || invite.Role != "viewer" || invite.Status != "pending" || invite.Email != "viewer@example.com" {
		t.Fatalf("unexpected invite response: %+v", invite)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/collaboration/invitations", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected invitations HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var invitations []struct {
		CollaboratorID string `json:"collaboratorId"`
		TripID         string `json:"tripId"`
		Role           string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &invitations); err != nil {
		t.Fatalf("decode invitations: %v", err)
	}
	if len(invitations) != 1 || invitations[0].CollaboratorID != invite.ID || invitations[0].TripID != tripID || invitations[0].Role != "viewer" {
		t.Fatalf("unexpected invitations: %+v", invitations)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID, nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected pending collaborator get HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/collaborators/"+invite.ID+"/accept", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accept HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/shared-with-me", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected shared-with-me HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var shared []struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &shared); err != nil {
		t.Fatalf("decode shared trips: %v", err)
	}
	if len(shared) != 1 || shared[0].ID != tripID || shared[0].Role != "viewer" {
		t.Fatalf("unexpected shared trips: %+v", shared)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID, nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accepted viewer get HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var viewerTrip struct {
		Access struct {
			Role                   string `json:"role"`
			CanEdit                bool   `json:"canEdit"`
			CanManageCollaborators bool   `json:"canManageCollaborators"`
		} `json:"access"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &viewerTrip); err != nil {
		t.Fatalf("decode viewer trip: %v", err)
	}
	if viewerTrip.Access.Role != "viewer" || viewerTrip.Access.CanEdit || viewerTrip.Access.CanManageCollaborators {
		t.Fatalf("unexpected viewer access: %+v", viewerTrip.Access)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+tripID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSONWithRevision(1))))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected viewer edit HTTP 403, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/trips/"+tripID+"/collaborators/"+invite.ID, bytes.NewReader([]byte(`{"role":"editor"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner role update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+tripID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSONWithRevision(1))))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected editor edit HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected editor share create HTTP 403, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID+"/itinerary/versions", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected owner version list HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var versions struct {
		Items []struct {
			CreatedByUserID string `json:"createdByUserId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &versions); err != nil {
		t.Fatalf("decode versions: %v", err)
	}
	if len(versions.Items) == 0 || versions.Items[0].CreatedByUserID != viewerID.String() {
		t.Fatalf("expected latest version actor %s, got %+v", viewerID, versions.Items)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/trips/"+tripID+"/collaborators/"+invite.ID, nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected remove collaborator HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trips/"+tripID, nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected removed collaborator get HTTP 404, got %d with %s", rec.Code, rec.Body.String())
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
		{name: "invalid opening hours day", body: `{"itinerary":{"days":[{"day":1,"title":"Day","items":[{"time":"09:00","type":"activity","name":"Walk","place":{"provider":"mock","providerPlaceId":"mock-place","name":"Mock Place","address":"Mock address","openingHours":[{"dayOfWeek":0,"open":"09:00","close":"18:00"}]}}]}]}}`},
		{name: "invalid opening time", body: `{"itinerary":{"days":[{"day":1,"title":"Day","items":[{"time":"09:00","type":"activity","name":"Walk","place":{"provider":"mock","providerPlaceId":"mock-place","name":"Mock Place","address":"Mock address","openingHours":[{"dayOfWeek":1,"open":"9:00","close":"18:00"}]}}]}]}}`},
		{name: "invalid closing time", body: `{"itinerary":{"days":[{"day":1,"title":"Day","items":[{"time":"09:00","type":"activity","name":"Walk","place":{"provider":"mock","providerPlaceId":"mock-place","name":"Mock Place","address":"Mock address","openingHours":[{"dayOfWeek":1,"open":"09:00","close":"24:00"}]}}]}]}}`},
		{name: "opening after close", body: `{"itinerary":{"days":[{"day":1,"title":"Day","items":[{"time":"09:00","type":"activity","name":"Walk","place":{"provider":"mock","providerPlaceId":"mock-place","name":"Mock Place","address":"Mock address","openingHours":[{"dayOfWeek":1,"open":"18:00","close":"09:00"}]}}]}]}}`},
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
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/generate", bytes.NewReader([]byte(expectedRevisionJSON(0))))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected generate HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSONWithRevision(1))))
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
	req = httptest.NewRequest(http.MethodPost, "/trips/"+created.ID+"/itinerary/versions/"+generatedVersionID+"/restore", bytes.NewReader([]byte(expectedRevisionJSON(2))))
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
	return newAuthTestRouterWithOptions(t, authCfg)
}

func newAuthTestRouterWithOptions(t *testing.T, authCfg config.AuthConfig, extraOpts ...service.Option) (http.Handler, *routeTestRepo) {
	t.Helper()

	repo := &routeTestRepo{
		trips:                       map[uuid.UUID]entity.Trip{},
		collaboratorsByID:           map[uuid.UUID]entity.TripCollaborator{},
		sharesByTrip:                map[uuid.UUID]entity.TripShare{},
		sharesByToken:               map[string]entity.TripShare{},
		budgetOptimizationProposals: []entity.BudgetOptimizationProposal{},
		tripTravelers:               map[uuid.UUID]entity.TripTraveler{},
	}
	gen := routeTestGenerator{}
	opts := []service.Option{
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
	}
	opts = append(opts, extraOpts...)
	svc := service.New(repo, gen, zap.NewNop(), opts...)
	validator, err := validation.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}
	tripHandler := handler.New(svc, validator, zap.NewNop())
	ready := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return NewRouter(zap.NewNop(), tripHandler, ready, config.CORSConfig{}, authCfg, config.OpsConfig{}), repo
}

type routeTestRepo struct {
	trips                       map[uuid.UUID]entity.Trip
	versions                    []entity.ItineraryVersion
	collaboratorsByID           map[uuid.UUID]entity.TripCollaborator
	sharesByTrip                map[uuid.UUID]entity.TripShare
	sharesByToken               map[string]entity.TripShare
	created                     *entity.Trip
	comments                    []entity.ItineraryComment
	activityEvents              []entity.TripActivityEvent
	calendarSyncs               []entity.TripCalendarSync
	budgetOptimizationProposals []entity.BudgetOptimizationProposal
	workspaceBudgets            map[uuid.UUID]entity.WorkspaceBudget
	tripTravelers               map[uuid.UUID]entity.TripTraveler
}

// --- approval workflow (route tests do not exercise approval endpoints; these
// stubs satisfy the repository interface) ---

func (r *routeTestRepo) GetTripApprovalFields(_ context.Context, tripID uuid.UUID) (*entity.TripApprovalFields, error) {
	return &entity.TripApprovalFields{TripID: tripID, Status: "not_required"}, nil
}

func (r *routeTestRepo) UpdateTripApprovalStatus(_ context.Context, fields *entity.TripApprovalFields) (*entity.TripApprovalFields, error) {
	cp := *fields
	return &cp, nil
}

func (r *routeTestRepo) InsertTripApprovalEvent(_ context.Context, event *entity.TripApprovalEvent) (*entity.TripApprovalEvent, error) {
	cp := *event
	return &cp, nil
}

func (r *routeTestRepo) ListTripApprovalEventsByTrip(_ context.Context, _ uuid.UUID, _ int) ([]entity.TripApprovalEvent, error) {
	return nil, nil
}

func (r *routeTestRepo) ListWorkspaceApprovals(_ context.Context, _ entity.ListWorkspaceApprovalsParams) ([]entity.WorkspaceApprovalRow, error) {
	return nil, nil
}

func (r *routeTestRepo) CountWorkspaceApprovalsByStatus(_ context.Context, _ uuid.UUID) (entity.WorkspaceApprovalCounts, error) {
	return entity.WorkspaceApprovalCounts{}, nil
}

func (r *routeTestRepo) ResetApprovalStatusForTripIfActive(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*entity.ApprovalResetResult, error) {
	return &entity.ApprovalResetResult{Reset: false}, nil
}

func (r *routeTestRepo) CreateTripActivityEvent(_ context.Context, event *entity.TripActivityEvent) (*entity.TripActivityEvent, error) {
	stored := *event
	if stored.CreatedAt.IsZero() {
		// Strictly increasing timestamps so newest-first ordering is stable.
		stored.CreatedAt = time.Now().UTC().Add(time.Duration(len(r.activityEvents)) * time.Millisecond)
	}
	r.activityEvents = append(r.activityEvents, stored)
	return &stored, nil
}

func (r *routeTestRepo) ListTripActivityEvents(
	_ context.Context,
	tripID uuid.UUID,
	limit int,
	cursorCreatedAt *time.Time,
	cursorID *uuid.UUID,
) ([]entity.TripActivityEvent, error) {
	// Filter to the trip, newest first (created_at DESC, id DESC).
	matching := make([]entity.TripActivityEvent, 0, len(r.activityEvents))
	for _, e := range r.activityEvents {
		if e.TripID == tripID {
			matching = append(matching, e)
		}
	}
	sort.Slice(matching, func(i, j int) bool {
		if !matching[i].CreatedAt.Equal(matching[j].CreatedAt) {
			return matching[i].CreatedAt.After(matching[j].CreatedAt)
		}
		return matching[i].ID.String() > matching[j].ID.String()
	})
	out := make([]entity.TripActivityEvent, 0, limit)
	for _, e := range matching {
		if cursorCreatedAt != nil && cursorID != nil {
			older := e.CreatedAt.Before(*cursorCreatedAt) ||
				(e.CreatedAt.Equal(*cursorCreatedAt) && e.ID.String() < cursorID.String())
			if !older {
				continue
			}
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
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

func (r *routeTestRepo) UpdateTripBudget(_ context.Context, id, userID uuid.UUID, amount *float64, currency string) (*entity.Trip, error) {
	trip, ok := r.trips[id]
	if !ok || trip.UserID == nil || *trip.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	trip.BudgetAmount = amount
	trip.BudgetCurrency = currency
	trip.UpdatedAt = time.Now().UTC()
	r.trips[id] = trip
	return &trip, nil
}

func (r *routeTestRepo) UpdateTripAccommodation(_ context.Context, id, userID uuid.UUID, accommodation *aggregate.Accommodation) (*entity.Trip, error) {
	trip, ok := r.trips[id]
	if !ok || trip.UserID == nil || *trip.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	trip.Accommodation = accommodation
	trip.UpdatedAt = time.Now().UTC()
	r.trips[id] = trip
	return &trip, nil
}

func (r *routeTestRepo) ClearTripAccommodation(_ context.Context, id, userID uuid.UUID) (*entity.Trip, error) {
	trip, ok := r.trips[id]
	if !ok || trip.UserID == nil || *trip.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	trip.Accommodation = nil
	trip.UpdatedAt = time.Now().UTC()
	r.trips[id] = trip
	return &trip, nil
}

func (r *routeTestRepo) GetByIDAndUserID(_ context.Context, id, userID uuid.UUID) (*entity.Trip, error) {
	trip, ok := r.trips[id]
	if !ok || trip.UserID == nil || *trip.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	return &trip, nil
}

func (r *routeTestRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Trip, error) {
	trip, ok := r.trips[id]
	if !ok {
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

func (r *routeTestRepo) ListAccessible(_ context.Context, userID uuid.UUID, workspaceIDs []uuid.UUID, scope appdto.TripListScope, workspaceID *uuid.UUID, _, _ int) ([]entity.Trip, error) {
	allowedWorkspaces := make(map[uuid.UUID]struct{}, len(workspaceIDs))
	for _, id := range workspaceIDs {
		allowedWorkspaces[id] = struct{}{}
	}
	trips := make([]entity.Trip, 0)
	for _, trip := range r.trips {
		personalOwner := trip.UserID != nil && *trip.UserID == userID && trip.WorkspaceID == nil
		workspaceAllowed := false
		if trip.WorkspaceID != nil {
			_, workspaceAllowed = allowedWorkspaces[*trip.WorkspaceID]
			if workspaceID != nil && *trip.WorkspaceID != *workspaceID {
				workspaceAllowed = false
			}
		}
		switch scope {
		case appdto.TripListScopePersonal:
			if personalOwner {
				trips = append(trips, trip)
			}
		case appdto.TripListScopeWorkspace:
			if workspaceAllowed {
				trips = append(trips, trip)
			}
		default:
			if personalOwner || workspaceAllowed {
				trips = append(trips, trip)
			}
		}
	}
	return trips, nil
}

func (r *routeTestRepo) CreateWorkspaceBudget(_ context.Context, budget *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error) {
	r.ensureWorkspaceBudgets()
	now := time.Now().UTC()
	out := *budget
	if out.CreatedAt.IsZero() {
		out.CreatedAt = now
	}
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = out.CreatedAt
	}
	if out.IsPrimary {
		r.clearWorkspaceBudgetPrimary(out.WorkspaceID)
	}
	r.workspaceBudgets[out.ID] = out
	return &out, nil
}

func (r *routeTestRepo) GetWorkspaceBudgetByID(_ context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error) {
	r.ensureWorkspaceBudgets()
	budget, ok := r.workspaceBudgets[budgetID]
	if !ok || budget.WorkspaceID != workspaceID {
		return nil, domainerrs.ErrNotFound
	}
	return &budget, nil
}

func (r *routeTestRepo) ListWorkspaceBudgetsByWorkspace(_ context.Context, workspaceID uuid.UUID, status *entity.WorkspaceBudgetStatus) ([]entity.WorkspaceBudget, error) {
	r.ensureWorkspaceBudgets()
	out := make([]entity.WorkspaceBudget, 0)
	for _, budget := range r.workspaceBudgets {
		if budget.WorkspaceID != workspaceID {
			continue
		}
		if status != nil && budget.Status != *status {
			continue
		}
		out = append(out, budget)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsPrimary != out[j].IsPrimary {
			return out[i].IsPrimary
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (r *routeTestRepo) ListActiveWorkspaceBudgetsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]entity.WorkspaceBudget, error) {
	status := entity.WorkspaceBudgetStatusActive
	return r.ListWorkspaceBudgetsByWorkspace(ctx, workspaceID, &status)
}

func (r *routeTestRepo) GetPrimaryWorkspaceBudget(_ context.Context, workspaceID uuid.UUID) (*entity.WorkspaceBudget, error) {
	r.ensureWorkspaceBudgets()
	for _, budget := range r.workspaceBudgets {
		if budget.WorkspaceID == workspaceID && budget.Status == entity.WorkspaceBudgetStatusActive && budget.IsPrimary {
			out := budget
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) UpdateWorkspaceBudget(_ context.Context, budget *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error) {
	r.ensureWorkspaceBudgets()
	current, ok := r.workspaceBudgets[budget.ID]
	if !ok || current.WorkspaceID != budget.WorkspaceID || current.Status != entity.WorkspaceBudgetStatusActive {
		return nil, domainerrs.ErrNotFound
	}
	out := *budget
	if out.IsPrimary {
		r.clearWorkspaceBudgetPrimary(out.WorkspaceID)
	}
	out.UpdatedAt = time.Now().UTC()
	r.workspaceBudgets[out.ID] = out
	return &out, nil
}

func (r *routeTestRepo) ArchiveWorkspaceBudget(_ context.Context, workspaceID, budgetID, actorUserID uuid.UUID) (*entity.WorkspaceBudget, error) {
	r.ensureWorkspaceBudgets()
	budget, ok := r.workspaceBudgets[budgetID]
	if !ok || budget.WorkspaceID != workspaceID || budget.Status == entity.WorkspaceBudgetStatusArchived {
		return nil, domainerrs.ErrNotFound
	}
	now := time.Now().UTC()
	budget.Status = entity.WorkspaceBudgetStatusArchived
	budget.IsPrimary = false
	budget.ArchivedByUserID = &actorUserID
	budget.ArchivedAt = &now
	budget.UpdatedAt = now
	r.workspaceBudgets[budgetID] = budget
	return &budget, nil
}

func (r *routeTestRepo) SetWorkspaceBudgetPrimary(_ context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error) {
	r.ensureWorkspaceBudgets()
	budget, ok := r.workspaceBudgets[budgetID]
	if !ok || budget.WorkspaceID != workspaceID || budget.Status != entity.WorkspaceBudgetStatusActive {
		return nil, domainerrs.ErrNotFound
	}
	r.clearWorkspaceBudgetPrimary(workspaceID)
	budget.IsPrimary = true
	budget.UpdatedAt = time.Now().UTC()
	r.workspaceBudgets[budgetID] = budget
	return &budget, nil
}

func (r *routeTestRepo) CountWorkspaceBudgets(_ context.Context, workspaceID uuid.UUID, status *entity.WorkspaceBudgetStatus) (int, error) {
	r.ensureWorkspaceBudgets()
	count := 0
	for _, budget := range r.workspaceBudgets {
		if budget.WorkspaceID != workspaceID {
			continue
		}
		if status != nil && budget.Status != *status {
			continue
		}
		count++
	}
	return count, nil
}

func (r *routeTestRepo) ensureWorkspaceBudgets() {
	if r.workspaceBudgets == nil {
		r.workspaceBudgets = map[uuid.UUID]entity.WorkspaceBudget{}
	}
}

func (r *routeTestRepo) clearWorkspaceBudgetPrimary(workspaceID uuid.UUID) {
	for id, budget := range r.workspaceBudgets {
		if budget.WorkspaceID == workspaceID && budget.Status == entity.WorkspaceBudgetStatusActive {
			budget.IsPrimary = false
			r.workspaceBudgets[id] = budget
		}
	}
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
	ctx context.Context,
	id, userID uuid.UUID,
	itinerary json.RawMessage,
	status entity.Status,
	expectedItineraryRevision int,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
) (*entity.Trip, *entity.ItineraryVersion, error) {
	return r.UpdateItineraryAndCreateVersion(ctx, id, userID, userID, itinerary, status, expectedItineraryRevision, source, metadata)
}

func (r *routeTestRepo) UpdateItineraryAndCreateVersion(
	_ context.Context,
	id, ownerUserID, actorUserID uuid.UUID,
	itinerary json.RawMessage,
	status entity.Status,
	expectedItineraryRevision int,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
) (*entity.Trip, *entity.ItineraryVersion, error) {
	trip, err := r.GetByIDAndUserID(context.Background(), id, ownerUserID)
	if err != nil {
		return nil, nil, err
	}
	if trip.ItineraryRevision != expectedItineraryRevision {
		return nil, nil, apperrs.NewItineraryConflict(trip.ItineraryRevision)
	}
	trip.Itinerary = itinerary
	trip.Status = status
	trip.ItineraryRevision++
	trip.UpdatedAt = time.Now().UTC()
	r.trips[id] = *trip
	version := entity.ItineraryVersion{
		ID:              uuid.New(),
		TripID:          id,
		UserID:          ownerUserID,
		CreatedByUserID: &actorUserID,
		VersionNumber:   routeTestNextVersionNumber(r.versions, id),
		Source:          source,
		Itinerary:       itinerary,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}
	r.versions = append(r.versions, version)
	return trip, &version, nil
}

func (r *routeTestRepo) ListItineraryVersionsByTripAndUser(_ context.Context, tripID, userID uuid.UUID, limit, offset int) ([]entity.ItineraryVersion, error) {
	return r.ListItineraryVersionsByTrip(context.Background(), tripID, limit, offset)
}

func (r *routeTestRepo) ListItineraryVersionsByTrip(_ context.Context, tripID uuid.UUID, limit, offset int) ([]entity.ItineraryVersion, error) {
	versions := make([]entity.ItineraryVersion, 0)
	for i := len(r.versions) - 1; i >= 0; i-- {
		version := r.versions[i]
		if version.TripID == tripID {
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
	return r.GetItineraryVersionByIDTrip(context.Background(), id, tripID)
}

func (r *routeTestRepo) GetItineraryVersionByIDTrip(_ context.Context, id, tripID uuid.UUID) (*entity.ItineraryVersion, error) {
	for i := range r.versions {
		version := r.versions[i]
		if version.ID == id && version.TripID == tripID {
			return &version, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) UpsertTripCollaborator(_ context.Context, collaborator *entity.TripCollaborator) (*entity.TripCollaborator, error) {
	now := time.Now().UTC()
	for id, existing := range r.collaboratorsByID {
		if existing.TripID == collaborator.TripID && existing.UserID == collaborator.UserID {
			existing.Role = collaborator.Role
			if existing.Status == entity.CollaboratorStatusRemoved {
				existing.Status = entity.CollaboratorStatusPending
				existing.AcceptedAt = nil
				existing.InvitedAt = now
			}
			existing.RemovedAt = nil
			existing.InvitedByUserID = collaborator.InvitedByUserID
			existing.UpdatedAt = now
			r.collaboratorsByID[id] = existing
			return &existing, nil
		}
	}
	out := *collaborator
	out.ID = uuid.New()
	out.Status = entity.CollaboratorStatusPending
	out.InvitedAt = now
	out.UpdatedAt = now
	r.collaboratorsByID[out.ID] = out
	return &out, nil
}

func (r *routeTestRepo) GetTripCollaboratorByTripAndUser(_ context.Context, tripID, userID uuid.UUID) (*entity.TripCollaborator, error) {
	for _, collaborator := range r.collaboratorsByID {
		if collaborator.TripID == tripID && collaborator.UserID == userID {
			out := collaborator
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) GetTripCollaboratorByID(_ context.Context, tripID, collaboratorID uuid.UUID) (*entity.TripCollaborator, error) {
	collaborator, ok := r.collaboratorsByID[collaboratorID]
	if !ok || collaborator.TripID != tripID {
		return nil, domainerrs.ErrNotFound
	}
	return &collaborator, nil
}

func (r *routeTestRepo) ListTripCollaborators(_ context.Context, tripID uuid.UUID) ([]entity.TripCollaborator, error) {
	out := make([]entity.TripCollaborator, 0)
	for _, collaborator := range r.collaboratorsByID {
		if collaborator.TripID == tripID && collaborator.Status != entity.CollaboratorStatusRemoved {
			out = append(out, collaborator)
		}
	}
	return out, nil
}

func (r *routeTestRepo) UpdateTripCollaboratorRole(_ context.Context, tripID, collaboratorID uuid.UUID, role entity.CollaboratorRole) (*entity.TripCollaborator, error) {
	collaborator, ok := r.collaboratorsByID[collaboratorID]
	if !ok || collaborator.TripID != tripID || collaborator.Status == entity.CollaboratorStatusRemoved {
		return nil, domainerrs.ErrNotFound
	}
	collaborator.Role = role
	collaborator.UpdatedAt = time.Now().UTC()
	r.collaboratorsByID[collaboratorID] = collaborator
	return &collaborator, nil
}

func (r *routeTestRepo) RemoveTripCollaborator(_ context.Context, tripID, collaboratorID uuid.UUID) (*entity.TripCollaborator, error) {
	collaborator, ok := r.collaboratorsByID[collaboratorID]
	if !ok || collaborator.TripID != tripID {
		return nil, domainerrs.ErrNotFound
	}
	now := time.Now().UTC()
	collaborator.Status = entity.CollaboratorStatusRemoved
	collaborator.RemovedAt = &now
	collaborator.UpdatedAt = now
	r.collaboratorsByID[collaboratorID] = collaborator
	return &collaborator, nil
}

func (r *routeTestRepo) AcceptTripCollaborator(_ context.Context, tripID, collaboratorID, userID uuid.UUID) (*entity.TripCollaborator, error) {
	collaborator, ok := r.collaboratorsByID[collaboratorID]
	if !ok || collaborator.TripID != tripID || collaborator.UserID != userID || collaborator.Status != entity.CollaboratorStatusPending {
		return nil, domainerrs.ErrNotFound
	}
	now := time.Now().UTC()
	collaborator.Status = entity.CollaboratorStatusAccepted
	collaborator.AcceptedAt = &now
	collaborator.RemovedAt = nil
	collaborator.UpdatedAt = now
	r.collaboratorsByID[collaboratorID] = collaborator
	return &collaborator, nil
}

func (r *routeTestRepo) DeclineTripCollaborator(_ context.Context, tripID, collaboratorID, userID uuid.UUID) (*entity.TripCollaborator, error) {
	collaborator, ok := r.collaboratorsByID[collaboratorID]
	if !ok || collaborator.TripID != tripID || collaborator.UserID != userID || collaborator.Status != entity.CollaboratorStatusPending {
		return nil, domainerrs.ErrNotFound
	}
	now := time.Now().UTC()
	collaborator.Status = entity.CollaboratorStatusRemoved
	collaborator.RemovedAt = &now
	collaborator.UpdatedAt = now
	r.collaboratorsByID[collaboratorID] = collaborator
	return &collaborator, nil
}

func (r *routeTestRepo) ListPendingCollaborationInvitations(_ context.Context, userID uuid.UUID) ([]entity.SharedTrip, error) {
	return r.listSharedTripsByCollaborator(userID, entity.CollaboratorStatusPending)
}

func (r *routeTestRepo) ListSharedTripsByUser(_ context.Context, userID uuid.UUID) ([]entity.SharedTrip, error) {
	return r.listSharedTripsByCollaborator(userID, entity.CollaboratorStatusAccepted)
}

func (r *routeTestRepo) listSharedTripsByCollaborator(userID uuid.UUID, status entity.CollaboratorStatus) ([]entity.SharedTrip, error) {
	out := make([]entity.SharedTrip, 0)
	for _, collaborator := range r.collaboratorsByID {
		if collaborator.UserID != userID || collaborator.Status != status {
			continue
		}
		trip, ok := r.trips[collaborator.TripID]
		if !ok {
			continue
		}
		out = append(out, entity.SharedTrip{Trip: trip, Collaborator: collaborator})
	}
	return out, nil
}

func (r *routeTestRepo) CreateTripTraveler(_ context.Context, traveler *entity.TripTraveler) (*entity.TripTraveler, error) {
	now := time.Now().UTC()
	out := *traveler
	if out.ID == uuid.Nil {
		out.ID = uuid.New()
	}
	out.CreatedAt = now
	out.UpdatedAt = now
	r.tripTravelers[out.ID] = out
	return &out, nil
}

func (r *routeTestRepo) GetTripTravelerByID(_ context.Context, tripID, travelerID uuid.UUID) (*entity.TripTraveler, error) {
	traveler, ok := r.tripTravelers[travelerID]
	if !ok || traveler.TripID != tripID {
		return nil, domainerrs.ErrNotFound
	}
	return &traveler, nil
}

func (r *routeTestRepo) ListTripTravelersByTrip(_ context.Context, tripID uuid.UUID) ([]entity.TripTraveler, error) {
	out := make([]entity.TripTraveler, 0)
	for _, traveler := range r.tripTravelers {
		if traveler.TripID == tripID && traveler.Status != entity.TripTravelerStatusRemoved {
			out = append(out, traveler)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *routeTestRepo) ListActiveTripTravelersByTrip(_ context.Context, tripID uuid.UUID) ([]entity.TripTraveler, error) {
	out := make([]entity.TripTraveler, 0)
	for _, traveler := range r.tripTravelers {
		if traveler.TripID == tripID && traveler.Status == entity.TripTravelerStatusActive {
			out = append(out, traveler)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *routeTestRepo) UpdateTripTraveler(_ context.Context, traveler *entity.TripTraveler) (*entity.TripTraveler, error) {
	existing, ok := r.tripTravelers[traveler.ID]
	if !ok || existing.TripID != traveler.TripID || existing.Status == entity.TripTravelerStatusRemoved {
		return nil, domainerrs.ErrNotFound
	}
	out := *traveler
	out.Status = existing.Status
	out.CreatedAt = existing.CreatedAt
	out.CreatedByUserID = existing.CreatedByUserID
	out.LinkedUserID = existing.LinkedUserID
	out.UpdatedAt = time.Now().UTC()
	r.tripTravelers[out.ID] = out
	return &out, nil
}

func (r *routeTestRepo) RemoveTripTraveler(_ context.Context, tripID, travelerID uuid.UUID) (*entity.TripTraveler, error) {
	traveler, ok := r.tripTravelers[travelerID]
	if !ok || traveler.TripID != tripID || traveler.Status == entity.TripTravelerStatusRemoved {
		return nil, domainerrs.ErrNotFound
	}
	now := time.Now().UTC()
	traveler.Status = entity.TripTravelerStatusRemoved
	traveler.RemovedAt = &now
	traveler.UpdatedAt = now
	r.tripTravelers[travelerID] = traveler
	return &traveler, nil
}

func (r *routeTestRepo) GetTripTravelerByLinkedUser(_ context.Context, tripID, linkedUserID uuid.UUID) (*entity.TripTraveler, error) {
	for _, traveler := range r.tripTravelers {
		if traveler.TripID == tripID &&
			traveler.LinkedUserID != nil &&
			*traveler.LinkedUserID == linkedUserID &&
			traveler.Status == entity.TripTravelerStatusActive {
			return &traveler, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) CountActiveTravelersByTrip(_ context.Context, tripID uuid.UUID) (int, error) {
	count := 0
	for _, traveler := range r.tripTravelers {
		if traveler.TripID == tripID && traveler.Status == entity.TripTravelerStatusActive {
			count++
		}
	}
	return count, nil
}

func (r *routeTestRepo) CreateTripShare(_ context.Context, share *entity.TripShare) (*entity.TripShare, error) {
	if _, ok := r.sharesByTrip[share.TripID]; ok {
		return nil, domainerrs.ErrConflict
	}
	if _, ok := r.sharesByToken[share.ShareToken]; ok {
		return nil, domainerrs.ErrConflict
	}

	now := time.Now().UTC()
	out := *share
	out.ID = uuid.New()
	out.CreatedAt = now
	out.UpdatedAt = now
	r.sharesByTrip[out.TripID] = out
	r.sharesByToken[out.ShareToken] = out
	return &out, nil
}

func (r *routeTestRepo) GetTripShareByTripAndUser(_ context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error) {
	share, ok := r.sharesByTrip[tripID]
	if !ok || share.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	return &share, nil
}

func (r *routeTestRepo) GetTripShareByToken(_ context.Context, shareToken string) (*entity.TripShare, error) {
	share, ok := r.sharesByToken[shareToken]
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	return &share, nil
}

func (r *routeTestRepo) EnableTripShare(_ context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error) {
	share, ok := r.sharesByTrip[tripID]
	if !ok || share.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	share.Enabled = true
	share.DisabledAt = nil
	share.UpdatedAt = time.Now().UTC()
	r.sharesByTrip[tripID] = share
	r.sharesByToken[share.ShareToken] = share
	return &share, nil
}

func (r *routeTestRepo) UpdateTripShareSettings(_ context.Context, tripID, userID uuid.UUID, expiresAt *time.Time, passwordRequired bool, passwordHash *string) (*entity.TripShare, error) {
	share, ok := r.sharesByTrip[tripID]
	if !ok || share.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	share.ExpiresAt = expiresAt
	share.PasswordRequired = passwordRequired
	share.PasswordHash = passwordHash
	share.UpdatedAt = time.Now().UTC()
	r.sharesByTrip[tripID] = share
	r.sharesByToken[share.ShareToken] = share
	return &share, nil
}

func (r *routeTestRepo) DisableTripShare(_ context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error) {
	share, ok := r.sharesByTrip[tripID]
	if !ok || share.UserID != userID {
		return nil, domainerrs.ErrNotFound
	}
	now := time.Now().UTC()
	share.Enabled = false
	share.DisabledAt = &now
	share.UpdatedAt = now
	r.sharesByTrip[tripID] = share
	r.sharesByToken[share.ShareToken] = share
	return &share, nil
}

func (r *routeTestRepo) CreateItineraryComment(_ context.Context, comment *entity.ItineraryComment) (*entity.ItineraryComment, error) {
	out := *comment
	if out.ID == uuid.Nil {
		out.ID = uuid.New()
	}
	if out.Status == "" {
		out.Status = entity.CommentStatusActive
	}
	now := time.Now().UTC()
	out.CreatedAt = now
	out.UpdatedAt = now
	r.comments = append(r.comments, out)
	return &out, nil
}

func (r *routeTestRepo) ListItineraryCommentsByTrip(_ context.Context, tripID uuid.UUID) ([]entity.ItineraryComment, error) {
	out := make([]entity.ItineraryComment, 0)
	for _, c := range r.comments {
		if c.TripID == tripID && c.Status == entity.CommentStatusActive {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *routeTestRepo) ListItineraryCommentsByItem(_ context.Context, tripID uuid.UUID, dayNumber, itemIndex int) ([]entity.ItineraryComment, error) {
	out := make([]entity.ItineraryComment, 0)
	for _, c := range r.comments {
		if c.TripID == tripID && c.Status == entity.CommentStatusActive &&
			c.DayNumber == dayNumber && c.ItemIndex == itemIndex {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *routeTestRepo) GetItineraryCommentByID(_ context.Context, tripID, commentID uuid.UUID) (*entity.ItineraryComment, error) {
	for i := range r.comments {
		if r.comments[i].ID == commentID && r.comments[i].TripID == tripID {
			out := r.comments[i]
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) UpdateItineraryCommentBody(_ context.Context, tripID, commentID uuid.UUID, body string) (*entity.ItineraryComment, error) {
	for i := range r.comments {
		if r.comments[i].ID == commentID && r.comments[i].TripID == tripID &&
			r.comments[i].Status == entity.CommentStatusActive {
			r.comments[i].Body = body
			r.comments[i].UpdatedAt = time.Now().UTC()
			out := r.comments[i]
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) SoftDeleteItineraryComment(_ context.Context, tripID, commentID uuid.UUID) (*entity.ItineraryComment, error) {
	for i := range r.comments {
		if r.comments[i].ID == commentID && r.comments[i].TripID == tripID &&
			r.comments[i].Status == entity.CommentStatusActive {
			now := time.Now().UTC()
			r.comments[i].Status = entity.CommentStatusDeleted
			r.comments[i].DeletedAt = &now
			r.comments[i].UpdatedAt = now
			out := r.comments[i]
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) CountItineraryCommentsByTripGrouped(_ context.Context, tripID uuid.UUID) ([]entity.ItineraryCommentCount, error) {
	type key struct{ day, item int }
	counts := make(map[key]int)
	order := make([]key, 0)
	for _, c := range r.comments {
		if c.TripID == tripID && c.Status == entity.CommentStatusActive {
			k := key{c.DayNumber, c.ItemIndex}
			if _, seen := counts[k]; !seen {
				order = append(order, k)
			}
			counts[k]++
		}
	}
	out := make([]entity.ItineraryCommentCount, 0, len(order))
	for _, k := range order {
		out = append(out, entity.ItineraryCommentCount{DayNumber: k.day, ItemIndex: k.item, Count: counts[k]})
	}
	return out, nil
}

func (r *routeTestRepo) UpsertTripCalendarSync(_ context.Context, sync *entity.TripCalendarSync) (*entity.TripCalendarSync, error) {
	out := *sync
	if out.ID == uuid.Nil {
		out.ID = uuid.New()
	}
	now := time.Now().UTC()
	out.Status = "active"
	out.LastSyncedAt = now
	out.CreatedAt = now
	out.UpdatedAt = now
	for i := range r.calendarSyncs {
		existing := &r.calendarSyncs[i]
		if existing.TripID == out.TripID &&
			existing.UserID == out.UserID &&
			existing.Provider == out.Provider &&
			existing.SyncKey == out.SyncKey {
			r.calendarSyncs[i] = out
			return &out, nil
		}
	}
	r.calendarSyncs = append(r.calendarSyncs, out)
	return &out, nil
}

func (r *routeTestRepo) ListTripCalendarSyncsByTripUserProvider(_ context.Context, tripID, userID uuid.UUID, provider string) ([]entity.TripCalendarSync, error) {
	out := make([]entity.TripCalendarSync, 0)
	for _, sync := range r.calendarSyncs {
		if sync.TripID == tripID && sync.UserID == userID && sync.Provider == provider && sync.Status == "active" && sync.DeletedAt == nil {
			out = append(out, sync)
		}
	}
	return out, nil
}

func (r *routeTestRepo) GetTripCalendarSyncStatus(_ context.Context, tripID, userID uuid.UUID, provider string) (int, *time.Time, int, error) {
	var count int
	var last *time.Time
	var revision int
	for _, sync := range r.calendarSyncs {
		if sync.TripID != tripID || sync.UserID != userID || sync.Provider != provider || sync.Status != "active" || sync.DeletedAt != nil {
			continue
		}
		count++
		if last == nil || sync.LastSyncedAt.After(*last) {
			v := sync.LastSyncedAt
			last = &v
		}
		if sync.ItineraryRevision > revision {
			revision = sync.ItineraryRevision
		}
	}
	return count, last, revision, nil
}

func (r *routeTestRepo) GetActiveTripCalendarSyncByKey(_ context.Context, tripID, userID uuid.UUID, provider, syncKey string) (*entity.TripCalendarSync, error) {
	for _, sync := range r.calendarSyncs {
		if sync.TripID == tripID && sync.UserID == userID && sync.Provider == provider && sync.SyncKey == syncKey && sync.Status == "active" && sync.DeletedAt == nil {
			out := sync
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) MarkTripCalendarSyncDeleted(_ context.Context, tripID, userID uuid.UUID, provider, syncKey string) error {
	now := time.Now().UTC()
	for i := range r.calendarSyncs {
		sync := &r.calendarSyncs[i]
		if sync.TripID == tripID && sync.UserID == userID && sync.Provider == provider && sync.SyncKey == syncKey {
			sync.Status = "deleted"
			sync.DeletedAt = &now
			sync.UpdatedAt = now
		}
	}
	return nil
}

func (r *routeTestRepo) MarkAllTripCalendarSyncsDeleted(_ context.Context, tripID, userID uuid.UUID, provider string) error {
	now := time.Now().UTC()
	for i := range r.calendarSyncs {
		sync := &r.calendarSyncs[i]
		if sync.TripID == tripID && sync.UserID == userID && sync.Provider == provider && sync.Status == "active" {
			sync.Status = "deleted"
			sync.DeletedAt = &now
			sync.UpdatedAt = now
		}
	}
	return nil
}

func (r *routeTestRepo) CreateBudgetOptimizationProposal(_ context.Context, proposal *entity.BudgetOptimizationProposal) (*entity.BudgetOptimizationProposal, error) {
	out := *proposal
	now := time.Now().UTC()
	out.CreatedAt = now
	out.UpdatedAt = now
	r.budgetOptimizationProposals = append(r.budgetOptimizationProposals, out)
	return &out, nil
}

func (r *routeTestRepo) GetBudgetOptimizationProposalByIDAndTrip(_ context.Context, id, tripID uuid.UUID) (*entity.BudgetOptimizationProposal, error) {
	for i := range r.budgetOptimizationProposals {
		proposal := r.budgetOptimizationProposals[i]
		if proposal.ID == id && proposal.TripID == tripID {
			return &proposal, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) ListBudgetOptimizationProposalsByTrip(_ context.Context, tripID uuid.UUID, status *entity.BudgetOptimizationProposalStatus, limit int) ([]entity.BudgetOptimizationProposal, error) {
	out := make([]entity.BudgetOptimizationProposal, 0)
	for _, proposal := range r.budgetOptimizationProposals {
		if proposal.TripID != tripID {
			continue
		}
		if status != nil && proposal.Status != *status {
			continue
		}
		out = append(out, proposal)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *routeTestRepo) ListPendingBudgetOptimizationProposalsByTrip(ctx context.Context, tripID uuid.UUID, limit int) ([]entity.BudgetOptimizationProposal, error) {
	status := entity.BudgetOptimizationProposalStatusPending
	return r.ListBudgetOptimizationProposalsByTrip(ctx, tripID, &status, limit)
}

func (r *routeTestRepo) MarkBudgetOptimizationProposalApplied(_ context.Context, id uuid.UUID, appliedItineraryRevision int) (*entity.BudgetOptimizationProposal, error) {
	for i := range r.budgetOptimizationProposals {
		proposal := &r.budgetOptimizationProposals[i]
		if proposal.ID == id && proposal.Status == entity.BudgetOptimizationProposalStatusPending {
			now := time.Now().UTC()
			proposal.Status = entity.BudgetOptimizationProposalStatusApplied
			proposal.AppliedItineraryRevision = &appliedItineraryRevision
			proposal.AppliedAt = &now
			proposal.UpdatedAt = now
			out := *proposal
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) MarkBudgetOptimizationProposalDiscarded(_ context.Context, id uuid.UUID) (*entity.BudgetOptimizationProposal, error) {
	for i := range r.budgetOptimizationProposals {
		proposal := &r.budgetOptimizationProposals[i]
		if proposal.ID == id && proposal.Status == entity.BudgetOptimizationProposalStatusPending {
			now := time.Now().UTC()
			proposal.Status = entity.BudgetOptimizationProposalStatusDiscarded
			proposal.DiscardedAt = &now
			proposal.UpdatedAt = now
			out := *proposal
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) MarkBudgetOptimizationProposalExpired(_ context.Context, id uuid.UUID) (*entity.BudgetOptimizationProposal, error) {
	for i := range r.budgetOptimizationProposals {
		proposal := &r.budgetOptimizationProposals[i]
		if proposal.ID == id && proposal.Status == entity.BudgetOptimizationProposalStatusPending {
			now := time.Now().UTC()
			proposal.Status = entity.BudgetOptimizationProposalStatusExpired
			proposal.ExpiredAt = &now
			proposal.UpdatedAt = now
			out := *proposal
			return &out, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (r *routeTestRepo) MarkBudgetOptimizationProposalFailed(_ context.Context, id uuid.UUID) (*entity.BudgetOptimizationProposal, error) {
	for i := range r.budgetOptimizationProposals {
		proposal := &r.budgetOptimizationProposals[i]
		if proposal.ID == id {
			proposal.Status = entity.BudgetOptimizationProposalStatusFailed
			proposal.UpdatedAt = time.Now().UTC()
			out := *proposal
			return &out, nil
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

type routeTestUserLookup struct {
	usersByEmail map[string]appdto.UserLookupResult
}

func (l routeTestUserLookup) LookupByEmail(_ context.Context, email string) (*appdto.UserLookupResult, error) {
	user, ok := l.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	return &user, nil
}

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

func (routeTestGenerator) OptimizeBudgetDay(_ context.Context, input budgetoptimization.OptimizeDayInput) (*budgetoptimization.ProposalContent, error) {
	savings := 10.0
	proposedTotal := input.BudgetContext.DayEstimatedTotal - savings
	if proposedTotal < 0 {
		proposedTotal = 0
	}
	proposedDay := input.CurrentDay
	if len(proposedDay.Items) > 0 {
		amount := 0.0
		proposedDay.Items[0].EstimatedCost = &aggregate.EstimatedCost{
			Amount:     &amount,
			Currency:   input.BudgetContext.Currency,
			Category:   "activity",
			Confidence: "medium",
			Source:     "ai",
		}
	}
	return &budgetoptimization.ProposalContent{
		Summary:                   "Reduced day cost with a cheaper option.",
		Scope:                     budgetoptimization.ScopeDay,
		DayNumber:                 input.DayNumber,
		Currency:                  input.BudgetContext.Currency,
		BaseDayEstimatedTotal:     input.BudgetContext.DayEstimatedTotal,
		ProposedDayEstimatedTotal: proposedTotal,
		EstimatedSavingsAmount:    savings,
		Confidence:                budgetoptimization.ConfidenceMedium,
		Changes: []budgetoptimization.ProposalChange{{
			Type:                   budgetoptimization.ChangeReplaceItem,
			EstimatedSavingsAmount: &savings,
			Currency:               input.BudgetContext.Currency,
		}},
		ProposedDay: proposedDay,
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

func validUpdateItineraryJSON() string {
	return validUpdateItineraryJSONWithRevision(0)
}

func validUpdateItineraryJSONWithRevision(revision int) string {
	return `{
		"expectedItineraryRevision": ` + strconv.Itoa(revision) + `,
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

func expectedRevisionJSON(revision int) string {
	return `{"expectedItineraryRevision":` + strconv.Itoa(revision) + `}`
}

func regenerateDayJSON(revision int, instruction string) string {
	if strings.TrimSpace(instruction) == "" {
		return expectedRevisionJSON(revision)
	}
	body, _ := json.Marshal(map[string]any{
		"expectedItineraryRevision": revision,
		"instruction":               instruction,
	})
	return string(body)
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

func createCompletedTripForRouteTest(t *testing.T, router http.Handler, ownerToken string) string {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(validCreateTripJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create trip HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create trip: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+created.ID+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected update itinerary HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	return created.ID
}

func signPublicShareAccessToken(t *testing.T, shareToken string, ttl time.Duration) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().UTC()
	payloadBytes, err := json.Marshal(map[string]any{
		"typ":        "public_share",
		"shareToken": shareToken,
		"aud":        "public-trip-share",
		"iss":        "trip-service",
		"iat":        now.Unix(),
		"exp":        now.Add(ttl).Unix(),
	})
	if err != nil {
		t.Fatalf("marshal public share token payload: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, []byte(testPublicShareSecret))
	_, _ = mac.Write([]byte(header + "." + payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + payload + "." + signature
}

func decodeJWTPayload(t *testing.T, token string) map[string]any {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected JWT with 3 segments, got %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal JWT payload: %v", err)
	}
	return claims
}

func timePtr(t time.Time) *time.Time {
	return &t
}
