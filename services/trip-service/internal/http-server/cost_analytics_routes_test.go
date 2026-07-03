package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/analytics"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

func TestTripCostAnalyticsOwnerCalculatesBreakdownsAndInsights(t *testing.T) {
	router, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithBudgetConversion(routeTestExchangeRates{}, true, true),
	)
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)
	putCostAnalyticsItinerary(t, router, ownerToken, tripID, 0, "2026-08-10")
	putCostAnalyticsAccommodation(t, router, ownerToken, tripID)

	rec := getTripCostAnalytics(t, router, ownerToken, tripID, "currency=EUR")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	var result analytics.TripCostAnalytics
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode analytics: %v", err)
	}
	if result.Currency != "EUR" {
		t.Fatalf("expected EUR analytics, got %s", result.Currency)
	}
	if result.Summary.EstimatedTotal != 680 {
		t.Fatalf("expected estimated total 680, got %+v", result.Summary)
	}
	if result.Summary.BudgetAmount == nil || *result.Summary.BudgetAmount != 500 {
		t.Fatalf("expected budget 500, got %+v", result.Summary.BudgetAmount)
	}
	if result.Summary.OverBudgetAmount == nil || *result.Summary.OverBudgetAmount != 180 {
		t.Fatalf("expected over budget 180, got %+v", result.Summary.OverBudgetAmount)
	}
	if result.Summary.MissingEstimateCount != 1 {
		t.Fatalf("expected one missing estimate, got %+v", result.Summary)
	}
	if result.Summary.UncertainEstimateCount == 0 {
		t.Fatalf("expected uncertain estimates, got %+v", result.Summary)
	}
	if amountByAnalyticsCategory(result.ByCategory, "accommodation") != 180 {
		t.Fatalf("expected accommodation category total 180, got %+v", result.ByCategory)
	}
	if amountByAnalyticsSource(result.BySource, "provider") != 420 {
		t.Fatalf("expected provider source total 420, got %+v", result.BySource)
	}
	if len(result.ByDay) != 2 || result.ByDay[0].Date == nil || *result.ByDay[0].Date != "2026-08-10" {
		t.Fatalf("unexpected byDay: %+v", result.ByDay)
	}
	if len(result.ExpensiveItems) == 0 || result.ExpensiveItems[0].Name != "Boat tour" {
		t.Fatalf("expected Boat tour as top expensive item, got %+v", result.ExpensiveItems)
	}
	if !hasInsight(result.Insights, "trip_over_budget") || !hasInsight(result.Insights, "missing_estimates") {
		t.Fatalf("expected over-budget and missing-estimate insights, got %+v", result.Insights)
	}
}

func TestTripCostAnalyticsViewerCanReadAndStrangerCannot(t *testing.T) {
	router, repo := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	viewerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	strangerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	viewerToken := signAccessToken(t, viewerID, "viewer@example.com", testJWTSecret, time.Hour)
	strangerToken := signAccessToken(t, strangerID, "stranger@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)
	seedAcceptedCollaborator(repo, tripID, viewerID, "viewer")

	if rec := getTripCostAnalytics(t, router, viewerToken, tripID, ""); rec.Code != http.StatusOK {
		t.Fatalf("expected viewer HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	if rec := getTripCostAnalytics(t, router, strangerToken, tripID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected stranger HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceCostAnalyticsAggregatesTripsAndFiltersByDate(t *testing.T) {
	workspaceID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	memberID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	strangerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	workspaceProvider := routeTestWorkspaceProvider{
		access: map[uuid.UUID]map[uuid.UUID]workspaces.Role{
			workspaceID: {
				ownerID:  workspaces.RoleOwner,
				memberID: workspaces.RoleViewer,
			},
		},
	}
	router, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithWorkspaces(workspaceProvider, true),
		service.WithBudgetConversion(routeTestExchangeRates{}, true, true),
	)
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	memberToken := signAccessToken(t, memberID, "member@example.com", testJWTSecret, time.Hour)
	strangerToken := signAccessToken(t, strangerID, "stranger@example.com", testJWTSecret, time.Hour)

	firstTrip := createWorkspaceAnalyticsTrip(t, router, ownerToken, workspaceID, "Tokyo", "2026-09-10", 700)
	putCostAnalyticsItinerary(t, router, ownerToken, firstTrip, 0, "2026-09-10")
	secondTrip := createWorkspaceAnalyticsTrip(t, router, ownerToken, workspaceID, "Paris", "2026-11-05", 100)
	putSimpleCostItinerary(t, router, ownerToken, secondTrip, 0, 50)

	rec := getWorkspaceCostAnalytics(t, router, memberToken, workspaceID, "currency=EUR&from=2026-09-01&to=2026-09-30")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace viewer HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var result analytics.WorkspaceCostAnalytics
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode workspace analytics: %v", err)
	}
	if result.Summary.TripCount != 1 {
		t.Fatalf("expected one trip after date filter, got %+v", result.Summary)
	}
	if result.Summary.EstimatedTotal != 500 {
		t.Fatalf("expected filtered total 500, got %+v", result.Summary)
	}
	if len(result.ByMonth) != 1 || result.ByMonth[0].Month != "2026-09" {
		t.Fatalf("expected September month bucket, got %+v", result.ByMonth)
	}
	if len(result.ExpensiveItems) == 0 || result.ExpensiveItems[0].TripTitle != "Tokyo" {
		t.Fatalf("expected expensive items annotated with trip title, got %+v", result.ExpensiveItems)
	}

	rec = getWorkspaceCostAnalytics(t, router, strangerToken, workspaceID, "currency=EUR")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected non-member HTTP 403, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestCostAnalyticsValidatesQueryParams(t *testing.T) {
	router, _ := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)

	if rec := getTripCostAnalytics(t, router, ownerToken, tripID, "currency=EU"); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad currency HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}

	workspaceID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String()+"/analytics/costs?from=2026-13-01", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad date HTTP 400, got %d with %s", rec.Code, rec.Body.String())
	}
}

type routeTestWorkspaceProvider struct {
	access map[uuid.UUID]map[uuid.UUID]workspaces.Role
}

func (p routeTestWorkspaceProvider) AccessCheck(_ context.Context, userID, workspaceID uuid.UUID) (*workspaces.Access, error) {
	role, ok := p.access[workspaceID][userID]
	if !ok {
		return &workspaces.Access{HasAccess: false}, nil
	}
	return &workspaces.Access{HasAccess: true, Role: role, Status: "active"}, nil
}

func (p routeTestWorkspaceProvider) ListForUser(_ context.Context, userID uuid.UUID) ([]workspaces.UserWorkspace, error) {
	out := make([]workspaces.UserWorkspace, 0)
	for workspaceID, users := range p.access {
		if role, ok := users[userID]; ok {
			out = append(out, workspaces.UserWorkspace{ID: workspaceID, Role: role})
		}
	}
	return out, nil
}

func getTripCostAnalytics(t *testing.T, router http.Handler, token string, tripID uuid.UUID, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/trips/" + tripID.String() + "/analytics/costs"
	if strings.TrimSpace(query) != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getWorkspaceCostAnalytics(t *testing.T, router http.Handler, token string, workspaceID uuid.UUID, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/workspaces/" + workspaceID.String() + "/analytics/costs"
	if strings.TrimSpace(query) != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func putCostAnalyticsItinerary(t *testing.T, router http.Handler, token string, tripID uuid.UUID, revision int, startDate string) {
	t.Helper()
	body := fmt.Sprintf(`{
		"expectedItineraryRevision": %d,
		"itinerary": {
			"currency": "EUR",
			"days": [
				{
					"day": 1,
					"title": "Arrival",
					"items": [
						{"time":"09:00","type":"ticket","name":"Museum","estimatedCost":{"amount":120,"currency":"EUR","category":"ticket","source":"provider","confidence":"high"}},
						{"time":"12:00","type":"food","name":"Lunch","estimatedCost":{"amount":80,"currency":"EUR","category":"food","source":"ai","confidence":"low"}},
						{"time":"16:00","type":"transport","name":"Metro"}
					]
				},
				{
					"day": 2,
					"title": "Water",
					"items": [
						{"time":"10:00","type":"tour","name":"Boat tour","estimatedCost":{"amount":300,"currency":"EUR","category":"activity","source":"provider","confidence":"high"}}
					]
				}
			],
			"generatedAt": "%sT00:00:00Z"
		}
	}`, revision, startDate)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
}

func putSimpleCostItinerary(t *testing.T, router http.Handler, token string, tripID uuid.UUID, revision int, amount float64) {
	t.Helper()
	body := fmt.Sprintf(`{
		"expectedItineraryRevision": %d,
		"itinerary": {
			"currency": "EUR",
			"days": [
				{
					"day": 1,
					"title": "Simple",
					"items": [
						{"time":"09:00","type":"activity","name":"Walk","estimatedCost":{"amount":%v,"currency":"EUR","category":"activity","source":"manual","confidence":"medium"}}
					]
				}
			]
		}
	}`, revision, amount)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected simple itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
}

func putCostAnalyticsAccommodation(t *testing.T, router http.Handler, token string, tripID uuid.UUID) {
	t.Helper()
	body := `{
		"accommodation": {
			"name": "Central Hotel",
			"type": "hotel",
			"estimatedCost": {"amount":180,"currency":"EUR","category":"accommodation","source":"manual","confidence":"medium"}
		}
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/accommodation", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accommodation update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
}

func createWorkspaceAnalyticsTrip(
	t *testing.T,
	router http.Handler,
	token string,
	workspaceID uuid.UUID,
	destination string,
	startDate string,
	budgetAmount float64,
) uuid.UUID {
	t.Helper()
	body := fmt.Sprintf(`{
		"destination": %q,
		"workspaceId": %q,
		"startDate": %q,
		"days": 2,
		"budgetAmount": %v,
		"budgetCurrency": "EUR",
		"travelers": 2,
		"interests": ["food"],
		"pace": "balanced"
	}`, destination, workspaceID.String(), startDate, budgetAmount)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create workspace trip HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode workspace trip: %v", err)
	}
	return uuid.MustParse(created.ID)
}

func amountByAnalyticsCategory(entries []analytics.CostAmountBreakdown, category string) float64 {
	for _, entry := range entries {
		if entry.Category == category {
			return entry.Amount
		}
	}
	return 0
}

func amountByAnalyticsSource(entries []analytics.CostAmountBreakdown, source string) float64 {
	for _, entry := range entries {
		if entry.Source == source {
			return entry.Amount
		}
	}
	return 0
}

func hasInsight(insights []analytics.CostInsight, insightType string) bool {
	for _, insight := range insights {
		if insight.Type == insightType {
			return true
		}
	}
	return false
}
