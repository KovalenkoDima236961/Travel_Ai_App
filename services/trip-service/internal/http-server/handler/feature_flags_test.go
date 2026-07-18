package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/featureflags"
)

func TestFeatureGateBlocksDisabledActionBeforeHandlerRuns(t *testing.T) {
	flags := featureflags.New(nil, featureflags.Config{Enabled: true, CacheTTLSeconds: 30}, "production", nil)
	h := New(nil, nil, nil).EnableFeatureFlags(flags)
	called := false
	router := chi.NewRouter()
	router.Post("/protected", h.gateFeature(featureflags.PolicyRepairEnabled, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden { t.Fatalf("status = %d, want 403", recorder.Code) }
	if called { t.Fatal("protected handler was called while flag was disabled") }
	if got := recorder.Body.String(); got == "" || !contains(got, "feature_disabled") { t.Fatalf("body = %s", got) }
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ { if value[i:i+len(part)] == part { return true } }
	return false
}
