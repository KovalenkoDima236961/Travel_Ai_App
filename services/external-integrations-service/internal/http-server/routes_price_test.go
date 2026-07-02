package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/prices"
)

func TestPriceEstimateEndpointReturnsMatchedResult(t *testing.T) {
	resp := performInternalPriceRequest(newTestRouter(), `{
		"destination":"Rome",
		"currency":"EUR",
		"date":"2026-08-10",
		"place":{"name":"Colosseum","category":"landmark","lat":41.8902,"lng":12.4922},
		"itemContext":{"name":"Visit the Colosseum","type":"attraction"}
	}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body prices.PriceEstimateResult
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Matched || body.EstimatedCost == nil {
		t.Fatalf("expected matched result, got %+v", body)
	}
	if body.EstimatedCost.Source != "provider" || body.Provider != "mock" {
		t.Fatalf("unexpected response shape: %+v", body)
	}
}

func TestPriceEstimateEndpointReturnsNoMatch(t *testing.T) {
	resp := performInternalPriceRequest(newTestRouter(), `{
		"destination":"Paris",
		"currency":"EUR",
		"place":{"name":"Luxembourg Gardens","category":"park"},
		"itemContext":{"name":"Walk through Luxembourg Gardens","type":"walk"}
	}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var body prices.PriceEstimateResult
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Matched || body.EstimatedCost != nil {
		t.Fatalf("expected no_match result, got %+v", body)
	}
}

func TestPriceEstimateEndpointValidatesMissingPlace(t *testing.T) {
	resp := performInternalPriceRequest(newTestRouter(), `{"destination":"Rome","currency":"EUR"}`)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPriceEstimateEndpointValidatesInvalidCurrency(t *testing.T) {
	resp := performInternalPriceRequest(newTestRouter(), `{
		"destination":"Rome",
		"currency":"EU",
		"place":{"name":"Colosseum","category":"landmark"}
	}`)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPriceEstimateEndpointRequiresInternalToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/prices/estimate", strings.NewReader(`{
		"destination":"Rome",
		"place":{"name":"Colosseum","category":"landmark"}
	}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	newTestRouter().ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func performInternalPriceRequest(router http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/prices/estimate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service-Token", "dev-internal-service-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
