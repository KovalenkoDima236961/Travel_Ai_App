package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalServiceTokenRotationAndDenials(t *testing.T) {
	handler := InternalServiceToken("old-token,new-token")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	for _, test := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "invalid", token: "wrong", want: http.StatusUnauthorized},
		{name: "old active token", token: "old-token", want: http.StatusNoContent},
		{name: "new active token", token: "new-token", want: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/internal/test", nil)
			if test.token != "" {
				request.Header.Set(InternalServiceTokenHeader, test.token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d, want %d", response.Code, test.want)
			}
		})
	}
}
