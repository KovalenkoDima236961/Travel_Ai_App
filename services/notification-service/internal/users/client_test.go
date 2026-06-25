package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLookupByIDsSendsTokenAndResolves(t *testing.T) {
	present := uuid.New()
	absent := uuid.New()

	var gotToken, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get(internalServiceTokenHeader)
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"userId":"` + present.String() + `","email":"anna@example.com","displayName":"Anna"}]}`))
	}))
	defer srv.Close()

	client, err := New(Config{BaseURL: srv.URL, Token: "secret-token", TimeoutSeconds: 2})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	profiles, err := client.LookupByIDs(context.Background(), []uuid.UUID{present, absent})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if gotToken != "secret-token" {
		t.Errorf("expected internal token forwarded, got %q", gotToken)
	}
	if gotPath != "/internal/users/batch" {
		t.Errorf("unexpected path %q", gotPath)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 resolved profile (absent omitted), got %d", len(profiles))
	}
	got, ok := profiles[present]
	if !ok || got.Email != "anna@example.com" || got.DisplayName != "Anna" {
		t.Fatalf("unexpected profile: %+v", profiles)
	}
	if _, ok := profiles[absent]; ok {
		t.Fatal("absent user should not be present in the map")
	}
}

func TestLookupByIDsEmptyIsNoOp(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, _ := New(Config{BaseURL: srv.URL, Token: "t"})
	profiles, err := client.LookupByIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected empty result, got %d", len(profiles))
	}
	if called {
		t.Fatal("expected no HTTP call for an empty id list")
	}
}

func TestLookupByIDsNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	client, _ := New(Config{BaseURL: srv.URL, Token: "t"})
	if _, err := client.LookupByIDs(context.Background(), []uuid.UUID{uuid.New()}); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestLookupByIDsTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, _ := New(Config{BaseURL: srv.URL, Token: "t", TimeoutSeconds: 1})
	// Force a tiny client timeout to exercise the transport-error path.
	client.httpClient.Timeout = 5 * time.Millisecond
	if _, err := client.LookupByIDs(context.Background(), []uuid.UUID{uuid.New()}); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestNewValidatesConfig(t *testing.T) {
	if _, err := New(Config{BaseURL: "", Token: "t"}); err == nil {
		t.Fatal("expected error for empty base URL")
	}
	if _, err := New(Config{BaseURL: "ftp://nope", Token: "t"}); err == nil {
		t.Fatal("expected error for non-http scheme")
	}
	if _, err := New(Config{BaseURL: "http://auth:8082", Token: ""}); err == nil {
		t.Fatal("expected error for empty token")
	}
}
