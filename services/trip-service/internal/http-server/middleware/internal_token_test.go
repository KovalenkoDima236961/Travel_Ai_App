package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalServiceTokenSupportsRotation(t *testing.T) {
	handler := InternalServiceToken("old-token-value,new-token-value")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	)
	for _, token := range []string{"old-token-value", "new-token-value"} {
		req := httptest.NewRequest(http.MethodPost, "/internal/test", nil)
		req.Header.Set(InternalServiceTokenHeader, token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("token %q returned %d", token, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/internal/test", nil)
	req.Header.Set(InternalServiceTokenHeader, "invalid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token returned %d", rec.Code)
	}
}
