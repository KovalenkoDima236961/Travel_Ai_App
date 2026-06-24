package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type activityResponse struct {
	Items []struct {
		ID        string         `json:"id"`
		TripID    string         `json:"tripId"`
		EventType string         `json:"eventType"`
		Metadata  map[string]any `json:"metadata"`
		CreatedAt string         `json:"createdAt"`
	} `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

func TestActivityEndpoint_OwnerSeesRecordedEvents(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)

	// Seeding creates the trip (trip_created) and sets its itinerary
	// (itinerary_updated).
	tripID := seedCommentableTrip(t, router, ownerToken)

	// Exercise two more record sites through the real service path.
	if rec := doJSON(t, router, http.MethodPost, "/trips/"+tripID+"/comments", ownerToken,
		`{"dayNumber":1,"itemIndex":0,"body":"Looks great"}`); rec.Code != http.StatusCreated {
		t.Fatalf("expected create comment HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, router, http.MethodPost, "/trips/"+tripID+"/share", ownerToken, ""); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("expected create share HTTP 2xx, got %d with %s", rec.Code, rec.Body.String())
	}

	rec := doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/activity?limit=100", ownerToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected activity HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var resp activityResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode activity response: %v", err)
	}
	if len(resp.Items) < 4 {
		t.Fatalf("expected at least 4 recorded events, got %+v", resp.Items)
	}

	types := map[string]bool{}
	for _, item := range resp.Items {
		types[item.EventType] = true
		if item.TripID != tripID {
			t.Fatalf("event trip id mismatch: %s", item.TripID)
		}
	}
	for _, want := range []string{"trip_created", "itinerary_updated", "comment_created", "share_created"} {
		if !types[want] {
			t.Fatalf("expected %s in activity feed, got %+v", want, types)
		}
	}

	// Newest first: itinerary_updated happened after trip_created.
	if resp.Items[len(resp.Items)-1].EventType != "trip_created" {
		t.Fatalf("expected trip_created to be the oldest (last) event, got %+v", resp.Items)
	}
}

func TestActivityEndpoint_Unauthenticated(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := seedCommentableTrip(t, router, ownerToken)

	rec := doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/activity", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestActivityEndpoint_InvalidCursorRejected(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := seedCommentableTrip(t, router, ownerToken)

	rec := doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/activity?cursor=not-valid$$$", ownerToken, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid cursor HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestActivityEndpoint_NonCollaboratorForbidden(t *testing.T) {
	router, _ := newAuthTestRouter(t, commentAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	otherToken := signAccessToken(t, otherID, "other@example.com", testJWTSecret, time.Hour)
	tripID := seedCommentableTrip(t, router, ownerToken)

	rec := doJSON(t, router, http.MethodGet, "/trips/"+tripID+"/activity", otherToken, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-collaborator HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}
