package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/config"
)

func TestCORSMiddlewarePreflight(t *testing.T) {
	handler := corsMiddleware(config.CORSConfig{
		AllowedOrigins: "http://localhost:3000",
		AllowedMethods: "GET,POST,OPTIONS",
		AllowedHeaders: "Content-Type,Authorization",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/auth/register", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("unexpected allow origin %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type,Authorization" {
		t.Fatalf("unexpected allow headers %q", got)
	}
}
