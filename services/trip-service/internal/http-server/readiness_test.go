package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeReadinessDB struct {
	err error
}

func (db fakeReadinessDB) Ping(context.Context) error {
	return db.err
}

func TestReadinessHandlerReadyInMockMode(t *testing.T) {
	handler := NewReadinessHandler(
		fakeReadinessDB{},
		"mock",
		"",
		time.Second,
		nil,
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", recorder.Code)
	}

	body := decodeReadinessBody(t, recorder)
	if body.Status != "ready" {
		t.Fatalf("expected ready status, got %q", body.Status)
	}
	if body.Checks["postgres"] != "ok" {
		t.Fatalf("expected postgres ok, got %q", body.Checks["postgres"])
	}
	if _, ok := body.Checks["aiPlanningService"]; ok {
		t.Fatalf("did not expect aiPlanningService check in mock mode")
	}
}

func TestReadinessHandlerChecksAIPlanningServiceInHTTPMode(t *testing.T) {
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(aiServer.Close)

	handler := NewReadinessHandler(
		fakeReadinessDB{},
		"http",
		aiServer.URL,
		time.Second,
		nil,
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", recorder.Code)
	}

	body := decodeReadinessBody(t, recorder)
	if body.Checks["postgres"] != "ok" {
		t.Fatalf("expected postgres ok, got %q", body.Checks["postgres"])
	}
	if body.Checks["aiPlanningService"] != "ok" {
		t.Fatalf("expected aiPlanningService ok, got %q", body.Checks["aiPlanningService"])
	}
}

func TestReadinessHandlerReturnsUnavailableOnPostgresFailure(t *testing.T) {
	handler := NewReadinessHandler(
		fakeReadinessDB{err: errors.New("database unavailable")},
		"mock",
		"",
		time.Second,
		nil,
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected HTTP 503, got %d", recorder.Code)
	}

	body := decodeReadinessBody(t, recorder)
	if body.Status != "not_ready" {
		t.Fatalf("expected not_ready status, got %q", body.Status)
	}
	if body.Checks["postgres"] != "failed" {
		t.Fatalf("expected postgres failed, got %q", body.Checks["postgres"])
	}
}

type readinessBody struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func decodeReadinessBody(t *testing.T, recorder *httptest.ResponseRecorder) readinessBody {
	t.Helper()

	var body readinessBody
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode readiness body: %v", err)
	}
	return body
}
