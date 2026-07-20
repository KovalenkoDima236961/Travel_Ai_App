package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
)

// newKnowledgeOpsRouter mounts the ops routes the way RegisterOpsRoutes does,
// without the feature-flag middleware, so these tests exercise routing and
// request validation rather than flag configuration.
func newKnowledgeOpsRouter(t *testing.T, store *knowledge.Store, ingestor *knowledge.Ingestor) http.Handler {
	t.Helper()
	handler := New(nil, nil, zap.NewNop()).EnableKnowledgeOps(store, ingestor)
	router := chi.NewRouter()
	router.Route("/ops", func(r chi.Router) {
		handler.registerOpsKnowledgeRoutes(r)
	})
	return router
}

// Without a knowledge store the routes must not exist at all, rather than
// existing and failing at request time.
func TestOpsKnowledgeRoutesAreSkippedWithoutStore(t *testing.T) {
	router := newKnowledgeOpsRouter(t, nil, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ops/ai/knowledge/quality-summary", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when no knowledge store is wired, got %d", recorder.Code)
	}
}

// This is the regression guard for the wiring itself: with a store present the
// routes must be registered.
func TestOpsKnowledgeRoutesAreRegisteredWithStore(t *testing.T) {
	router := newKnowledgeOpsRouter(t, &knowledge.Store{}, nil)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/ops/ai/knowledge/quality-summary"},
		{http.MethodGet, "/ops/ai/knowledge/provider-ingestion/status"},
		{http.MethodPost, "/ops/ai/knowledge/provider-ingestion/run"},
		{http.MethodGet, "/ops/ai/knowledge/places"},
		{http.MethodGet, "/ops/ai/knowledge/duplicates"},
		{http.MethodGet, "/ops/ai/knowledge/provider-observations"},
	}
	for _, route := range routes {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(route.method, route.path, strings.NewReader("{}"))
		router.ServeHTTP(recorder, request)
		if recorder.Code == http.StatusNotFound || recorder.Code == http.StatusMethodNotAllowed {
			t.Errorf("%s %s is not registered (status %d)", route.method, route.path, recorder.Code)
		}
	}
}

// Ingestion actions must report unavailability rather than pretending a run
// started when no provider is configured.
func TestOpsKnowledgeIngestionRunRequiresProvider(t *testing.T) {
	router := newKnowledgeOpsRouter(t, &knowledge.Store{}, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/ops/ai/knowledge/provider-ingestion/run",
		strings.NewReader(`{"destinationName":"Rome"}`))
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without a provider, got %d", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "knowledge_provider_unavailable" {
		t.Fatalf("expected knowledge_provider_unavailable, got %v", body["error"])
	}
}

func TestOpsKnowledgeReviewValidatesAction(t *testing.T) {
	router := newKnowledgeOpsRouter(t, &knowledge.Store{}, nil)
	placeID := "11111111-1111-1111-1111-111111111111"

	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"unsupported action", `{"action":"deleted"}`, http.StatusBadRequest},
		{"rejection without reason", `{"action":"rejected"}`, http.StatusBadRequest},
		{"malformed body", `not json`, http.StatusBadRequest},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPatch,
				"/ops/ai/knowledge/places/"+placeID+"/review", strings.NewReader(testCase.body))
			router.ServeHTTP(recorder, request)
			if recorder.Code != testCase.wantCode {
				t.Fatalf("expected %d, got %d (%s)", testCase.wantCode, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestOpsKnowledgeInvalidIdentifiersAreRejected(t *testing.T) {
	router := newKnowledgeOpsRouter(t, &knowledge.Store{}, nil)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/ops/ai/knowledge/places?destinationId=not-a-uuid", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an invalid destinationId, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/ops/ai/knowledge/places?filter=drop_tables", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown filter, got %d", recorder.Code)
	}
}

func TestOpsKnowledgeMergeRequiresCanonicalPlace(t *testing.T) {
	router := newKnowledgeOpsRouter(t, &knowledge.Store{}, nil)
	groupID := "22222222-2222-2222-2222-222222222222"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost,
		"/ops/ai/knowledge/duplicates/"+groupID+"/merge", strings.NewReader(`{"reason":"same place"}`))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when canonicalPlaceId is missing, got %d", recorder.Code)
	}
}

// RegisterOpsRoutes must include the knowledge routes; this catches a
// regression where the sub-router stops being mounted.
func TestRegisterOpsRoutesIncludesKnowledge(t *testing.T) {
	handler := New(nil, nil, zap.NewNop()).EnableKnowledgeOps(&knowledge.Store{}, nil)
	router := chi.NewRouter()
	handler.RegisterOpsRoutes(router, time.Minute)

	found := false
	_ = chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.Contains(route, "/ai/knowledge/") {
			found = true
		}
		return nil
	})
	if !found {
		t.Fatal("RegisterOpsRoutes did not mount any /ai/knowledge routes")
	}
}
