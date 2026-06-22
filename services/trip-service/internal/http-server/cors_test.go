package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
)

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	handler := corsMiddleware(config.CORSConfig{
		AllowedOrigins: "http://localhost:3000",
		AllowedMethods: "GET,POST,OPTIONS",
		AllowedHeaders: "Content-Type,Authorization",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/trips", nil)
	request.Header.Set("Origin", "http://localhost:3000")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Methods"); got != "GET,POST,OPTIONS" {
		t.Fatalf("expected allowed methods header, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type,Authorization" {
		t.Fatalf("expected allowed headers header, got %q", got)
	}
}

func TestCORSMiddlewareDoesNotAllowUnconfiguredOrigin(t *testing.T) {
	handler := corsMiddleware(config.CORSConfig{
		AllowedOrigins: "http://localhost:3000",
		AllowedMethods: "GET,POST,OPTIONS",
		AllowedHeaders: "Content-Type,Authorization",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/trips", nil)
	request.Header.Set("Origin", "http://example.com")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allowed origin header, got %q", got)
	}
}

func TestCORSMiddlewareHandlesPreflight(t *testing.T) {
	called := false
	handler := corsMiddleware(config.CORSConfig{
		AllowedOrigins: "http://localhost:3000",
		AllowedMethods: "GET,POST,PATCH,DELETE,OPTIONS",
		AllowedHeaders: "Content-Type,Authorization",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/trips", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", "POST")

	handler.ServeHTTP(recorder, request)

	if called {
		t.Fatal("expected preflight request not to call next handler")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected HTTP 204, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Methods"); got != "GET,POST,PATCH,DELETE,OPTIONS" {
		t.Fatalf("expected allowed methods header, got %q", got)
	}
}
