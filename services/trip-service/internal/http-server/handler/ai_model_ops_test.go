package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func TestRegisterOpsRoutesIncludesAIModelRoutes(t *testing.T) {
	handler := New(nil, nil, zap.NewNop())
	router := chi.NewRouter()
	handler.RegisterOpsRoutes(router, time.Minute)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ops/ai/model-deployments/9c44ee32-45c2-4402-a1b4-e6ec83fd4739/online-summary", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("RegisterOpsRoutes did not mount /ops/ai/model-deployments routes")
	}
}
