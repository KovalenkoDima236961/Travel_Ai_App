package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
)

// seedCommentableTrip creates a COMPLETED trip with a day-1/item-0 itinerary for
// the given owner and returns its id.
func seedCommentableTrip(t *testing.T, router http.Handler, ownerToken string) string {
	t.Helper()

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

	return created.ID
}

func commentAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	}
}

func doJSON(t *testing.T, router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func TestCommentEndpoints_OwnerLifecycle(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := seedCommentableTrip(t, router, ownerToken)

	// Create.
	rec := doJSON(t, router, http.MethodPost, "/trips/"+tripID+"/comments", ownerToken,
		`{"dayNumber":1,"itemIndex":0,"body":"Looks great"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create comment HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID        string `json:"id"`
		IsAuthor  bool   `json:"isAuthor"`
		CanEdit   bool   `json:"canEdit"`
		CanDelete bool   `json:"canDelete"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created comment: %v", err)
	}
	if created.ID == "" || !created.IsAuthor || !created.CanEdit || !created.CanDelete {
		t.Fatalf("unexpected created comment payload: %+v", created)
	}

	// List all.
	rec = doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/comments", ownerToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected list HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != created.ID {
		t.Fatalf("expected one listed comment matching created id, got %+v", list.Items)
	}

	// List by item (both params).
	rec = doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/comments?dayNumber=1&itemIndex=0", ownerToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected item list HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode item list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected one item comment, got %d", len(list.Items))
	}

	// Counts route resolves separately from /{commentId}.
	rec = doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/comments/counts", ownerToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected counts HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var counts struct {
		Items []struct {
			DayNumber int `json:"dayNumber"`
			ItemIndex int `json:"itemIndex"`
			Count     int `json:"count"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &counts); err != nil {
		t.Fatalf("decode counts: %v", err)
	}
	if len(counts.Items) != 1 || counts.Items[0].DayNumber != 1 || counts.Items[0].ItemIndex != 0 || counts.Items[0].Count != 1 {
		t.Fatalf("unexpected counts payload: %+v", counts.Items)
	}

	// Update (author).
	rec = doJSON(t, router, http.MethodPatch, "/trips/"+tripID+"/comments/"+created.ID, ownerToken,
		`{"body":"Edited body"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated: %v", err)
	}
	if updated.Body != "Edited body" {
		t.Fatalf("expected updated body, got %q", updated.Body)
	}

	// Delete (author/owner) returns the success envelope.
	rec = doJSON(t, router, http.MethodDelete, "/trips/"+tripID+"/comments/"+created.ID, ownerToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected delete HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var deleteResp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &deleteResp); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if !deleteResp.Success {
		t.Fatalf("expected success envelope, got %s", rec.Body.String())
	}

	// Soft-deleted comment is no longer listed.
	rec = doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/comments", ownerToken, "")
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list after delete: %v", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("expected no comments after delete, got %+v", list.Items)
	}
}

func TestCommentEndpoints_PartialItemFilterRejected(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := seedCommentableTrip(t, router, ownerToken)

	for _, query := range []string{"?dayNumber=1", "?itemIndex=0"} {
		rec := doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/comments"+query, ownerToken, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for %q, got %d with %s", query, rec.Code, rec.Body.String())
		}
	}
}

func TestCommentEndpoints_ValidationAndAuth(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := seedCommentableTrip(t, router, ownerToken)

	// Empty body fails structural validation (400).
	rec := doJSON(t, router, http.MethodPost, "/trips/"+tripID+"/comments", ownerToken,
		`{"dayNumber":1,"itemIndex":0,"body":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected empty body HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}

	// Non-existent itinerary item rejected (400).
	rec = doJSON(t, router, http.MethodPost, "/trips/"+tripID+"/comments", ownerToken,
		`{"dayNumber":9,"itemIndex":0,"body":"Nowhere"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing item HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}

	// Unauthenticated comment list is rejected (401).
	rec = doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/comments", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestCommentEndpoints_CrossTripIsolation(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	otherToken := signAccessToken(t, otherID, "other@example.com", testJWTSecret, time.Hour)

	tripID := seedCommentableTrip(t, router, ownerToken)
	otherTripID := seedCommentableTrip(t, router, otherToken)

	rec := doJSON(t, router, http.MethodPost, "/trips/"+tripID+"/comments", ownerToken,
		`{"dayNumber":1,"itemIndex":0,"body":"Owner comment"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}

	// PATCH/DELETE the first trip's comment through the second (owned) trip's
	// path must 404 — comments are scoped to their trip.
	rec = doJSON(t, router, http.MethodPatch, "/trips/"+otherTripID+"/comments/"+created.ID, otherToken,
		`{"body":"hijack"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected cross-trip PATCH HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodDelete, "/trips/"+otherTripID+"/comments/"+created.ID, otherToken, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected cross-trip DELETE HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}
