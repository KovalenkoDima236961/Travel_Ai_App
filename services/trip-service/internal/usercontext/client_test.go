package usercontext

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientGetUserContext_ForwardsAuthorizationHeader(t *testing.T) {
	seenHeaders := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeaders[r.URL.Path] = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/me/profile":
			fmt.Fprint(w, `{"userId":"11111111-1111-1111-1111-111111111111","displayName":"Test Traveler","preferredCurrency":"EUR","preferredLanguage":"en"}`)
		case "/users/me/preferences":
			fmt.Fprint(w, `{"userId":"11111111-1111-1111-1111-111111111111","travelStyles":["budget","food"],"pace":"balanced","avoid":["nightclubs"],"preferredTransport":["walking"],"accommodationStyle":["budget_hotel"],"dietaryRestrictions":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	got, err := client.GetUserContext(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Profile == nil || got.Profile.DisplayName == nil || *got.Profile.DisplayName != "Test Traveler" {
		t.Fatalf("expected profile, got %+v", got.Profile)
	}
	if got.Preferences == nil || strings.Join(got.Preferences.Avoid, ",") != "nightclubs" {
		t.Fatalf("expected preferences, got %+v", got.Preferences)
	}
	if seenHeaders["/users/me/profile"] != "Bearer access-token" {
		t.Fatalf("expected profile Authorization header to be forwarded, got %q", seenHeaders["/users/me/profile"])
	}
	if seenHeaders["/users/me/preferences"] != "Bearer access-token" {
		t.Fatalf("expected preferences Authorization header to be forwarded, got %q", seenHeaders["/users/me/preferences"])
	}
}

func TestClientGetUserContext_ToleratesMissingProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/me/profile":
			http.NotFound(w, r)
		case "/users/me/preferences":
			fmt.Fprint(w, `{"userId":"11111111-1111-1111-1111-111111111111","travelStyles":["food"],"pace":"balanced","foodPreferences":["local"],"avoid":[],"preferredTransport":[],"accommodationStyle":[],"dietaryRestrictions":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	got, err := client.GetUserContext(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Profile != nil {
		t.Fatalf("expected missing profile to be nil, got %+v", got.Profile)
	}
	if got.Preferences == nil || strings.Join(got.Preferences.FoodPreferences, ",") != "local" {
		t.Fatalf("expected preferences, got %+v", got.Preferences)
	}
}

func TestClientGetUserContext_AuthFailureIsTypedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := client.GetUserContext(context.Background(), "access-token")
	assertUserContextError(t, err, ErrorTypeAuth, http.StatusUnauthorized)
}

func TestClientGetUserContext_ServiceFailureIsTypedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := client.GetUserContext(context.Background(), "access-token")
	assertUserContextError(t, err, ErrorTypeService, http.StatusBadGateway)
}

func TestClientGetUserContext_MalformedJSONIsTypedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"userId":`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := client.GetUserContext(context.Background(), "access-token")
	assertUserContextError(t, err, ErrorTypeInvalidJSON, http.StatusOK)
}

func TestClientGetMyProfile_DoesNotAcceptMissingToken(t *testing.T) {
	client := newTestClient(t, "http://user-service:8083")

	_, err := client.GetMyProfile(context.Background(), " ")
	assertUserContextError(t, err, ErrorTypeAuth, 0)
}

func TestClientDoesNotLogOrReturnTokenInErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := client.GetUserContext(context.Background(), "secret-token")
	if err == nil {
		t.Fatal("expected error")
	}
	encoded, marshalErr := json.Marshal(err.Error())
	if marshalErr != nil {
		t.Fatalf("marshal error string: %v", marshalErr)
	}
	if strings.Contains(string(encoded), "secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(baseURL, &http.Client{Timeout: time.Second})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

func assertUserContextError(t *testing.T, err error, wantType ErrorType, wantStatus int) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error")
	}
	userContextErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if userContextErr.Type != wantType {
		t.Fatalf("expected error type %s, got %s", wantType, userContextErr.Type)
	}
	if userContextErr.StatusCode != wantStatus {
		t.Fatalf("expected status %d, got %d", wantStatus, userContextErr.StatusCode)
	}
}
