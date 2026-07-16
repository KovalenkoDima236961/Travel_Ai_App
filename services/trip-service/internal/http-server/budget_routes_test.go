package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func budgetTestAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		Required:        true,
		JWTAccessSecret: testJWTSecret,
		HeaderName:      "Authorization",
		DevUserID:       "00000000-0000-0000-0000-000000000001",
	}
}

func createBudgetTestTrip(t *testing.T, router http.Handler, token string) uuid.UUID {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader([]byte(validCreateTripJSON())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	return uuid.MustParse(created.ID)
}

func putItinerary(t *testing.T, router http.Handler, token string, tripID uuid.UUID, revision int) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(validUpdateItineraryJSONWithRevision(revision))))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
}

func getBudgetSummary(t *testing.T, router http.Handler, token string, tripID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+tripID.String()+"/budget-summary", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func getBudgetConfidence(t *testing.T, router http.Handler, token string, tripID uuid.UUID, query string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	path := "/trips/" + tripID.String() + "/budget-confidence"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func putBudget(t *testing.T, router http.Handler, token string, tripID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/budget", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func fetchTripRevision(t *testing.T, router http.Handler, token string, tripID uuid.UUID) int {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trips/"+tripID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected get trip HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var trip struct {
		ItineraryRevision int `json:"itineraryRevision"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &trip); err != nil {
		t.Fatalf("decode trip: %v", err)
	}
	return trip.ItineraryRevision
}

func seedAcceptedCollaborator(repo *routeTestRepo, tripID, userID uuid.UUID, role entity.CollaboratorRole) {
	id := uuid.New()
	repo.collaboratorsByID[id] = entity.TripCollaborator{
		ID:              id,
		TripID:          tripID,
		UserID:          userID,
		Role:            role,
		Status:          entity.CollaboratorStatusAccepted,
		InvitedByUserID: userID,
		InvitedAt:       time.Now().UTC(),
	}
}

func TestBudgetSummaryRequiresAuth(t *testing.T) {
	router, _ := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)

	rec := getBudgetSummary(t, router, "", tripID)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 without token, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestBudgetSummaryOwnerReflectsItineraryCosts(t *testing.T) {
	router, _ := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)
	putItinerary(t, router, ownerToken, tripID, 0)

	rec := getBudgetSummary(t, router, ownerToken, tripID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var summary struct {
		Currency       string   `json:"currency"`
		TripBudget     *float64 `json:"tripBudget"`
		EstimatedTotal float64  `json:"estimatedTotal"`
		Remaining      *float64 `json:"remaining"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", summary.Currency)
	}
	if summary.EstimatedTotal != 12 {
		t.Fatalf("expected estimatedTotal 12 from itinerary item, got %v", summary.EstimatedTotal)
	}
	if summary.TripBudget == nil || *summary.TripBudget != 500 {
		t.Fatalf("expected tripBudget 500, got %v", summary.TripBudget)
	}
	if summary.Remaining == nil || *summary.Remaining != 488 {
		t.Fatalf("expected remaining 488, got %v", summary.Remaining)
	}
}

func TestBudgetConfidenceOwnerReflectsItineraryCosts(t *testing.T) {
	router, _ := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)
	putItinerary(t, router, ownerToken, tripID, 0)

	rec := getBudgetConfidence(t, router, ownerToken, tripID, "includeDebug=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var response budgetconfidence.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode budget confidence response: %v", err)
	}
	if response.TripID != tripID {
		t.Fatalf("expected trip ID %s, got %s", tripID, response.TripID)
	}
	if response.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", response.Currency)
	}
	if response.EstimatedTotal.Amount != 12 {
		t.Fatalf("expected estimated total 12, got %+v", response.EstimatedTotal)
	}
	if response.TripBudget == nil || response.TripBudget.Amount != 500 {
		t.Fatalf("expected trip budget 500, got %+v", response.TripBudget)
	}
	if response.Coverage.Overall <= 0 {
		t.Fatalf("expected positive coverage, got %+v", response.Coverage)
	}
	if response.Debug == nil {
		t.Fatalf("expected debug payload when includeDebug=true")
	}
}

func TestBudgetConfidenceRequiresAuthAndHonorsPermissions(t *testing.T) {
	router, repo := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	viewerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	strangerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	viewerToken := signAccessToken(t, viewerID, "viewer@example.com", testJWTSecret, time.Hour)
	strangerToken := signAccessToken(t, strangerID, "stranger@example.com", testJWTSecret, time.Hour)

	tripID := createBudgetTestTrip(t, router, ownerToken)
	seedAcceptedCollaborator(repo, tripID, viewerID, entity.CollaboratorRoleViewer)

	if rec := getBudgetConfidence(t, router, "", tripID, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated HTTP 401, got %d with %s", rec.Code, rec.Body.String())
	}
	if rec := getBudgetConfidence(t, router, viewerToken, tripID, ""); rec.Code != http.StatusOK {
		t.Fatalf("expected viewer HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	if rec := getBudgetConfidence(t, router, strangerToken, tripID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected stranger HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestBudgetConfidenceDisabled(t *testing.T) {
	router, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithBudgetConfidenceConfig(budgetconfidence.Config{Enabled: false}),
	)
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)

	rec := getBudgetConfidence(t, router, ownerToken, tripID, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected disabled endpoint HTTP 503, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestBudgetSummaryConvertsForeignCurrencyCosts(t *testing.T) {
	router, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithBudgetConversion(routeTestExchangeRates{}, true, true),
	)
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)

	itineraryJSON := `{
		"expectedItineraryRevision": 0,
		"itinerary": {
			"currency": "EUR",
			"days": [
				{
					"day": 1,
					"title": "Tokyo",
					"items": [
						{"time":"09:00","type":"food","name":"Ramen","estimatedCost":{"amount":2500,"currency":"JPY","category":"food"}},
						{"time":"12:00","type":"ticket","name":"Museum","estimatedCost":{"amount":20,"currency":"EUR","category":"ticket"}}
					]
				}
			]
		}
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(itineraryJSON)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	revisionAfterItinerary := fetchTripRevision(t, router, ownerToken, tripID)

	accommodationJSON := `{
		"accommodation": {
			"name": "Tokyo Hotel",
			"type": "hotel",
			"estimatedCost": {"amount":17050,"currency":"JPY","category":"accommodation"}
		}
	}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/accommodation", bytes.NewReader([]byte(accommodationJSON)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accommodation update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	rec = getBudgetSummary(t, router, ownerToken, tripID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var summary budget.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.EstimatedTotal != 134.66 {
		t.Fatalf("expected converted total 134.66, got %+v", summary)
	}
	if summary.ConvertedItemCount != 2 {
		t.Fatalf("expected 2 converted costs, got %d", summary.ConvertedItemCount)
	}
	if summary.ExchangeRateInfo == nil || summary.ExchangeRateInfo.Provider != "route-test" {
		t.Fatalf("expected exchange rate info, got %+v", summary.ExchangeRateInfo)
	}
	if amountByOriginalCurrency(summary.OriginalCurrencyTotals, "JPY") != 19550 {
		t.Fatalf("expected original JPY total 19550, got %+v", summary.OriginalCurrencyTotals)
	}
	if revisionAfterSummary := fetchTripRevision(t, router, ownerToken, tripID); revisionAfterSummary != revisionAfterItinerary {
		t.Fatalf("expected budget summary not to mutate itineraryRevision %d, got %d", revisionAfterItinerary, revisionAfterSummary)
	}
}

func TestBudgetSummaryConversionWarningAndFailClosed(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	body := `{
		"expectedItineraryRevision": 0,
		"itinerary": {
			"days": [
				{
					"day": 1,
					"title": "Day",
					"items": [
						{"time":"09:00","type":"ticket","name":"Show","estimatedCost":{"amount":99,"currency":"XXX","category":"ticket"}}
					]
				}
			]
		}
	}`

	router, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithBudgetConversion(routeTestExchangeRates{}, true, true),
	)
	tripID := createBudgetTestTrip(t, router, ownerToken)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	rec = getBudgetSummary(t, router, ownerToken, tripID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected fail-open HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var summary budget.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.UnconvertedItemCount != 1 || len(summary.ConversionWarnings) != 1 {
		t.Fatalf("expected one conversion warning, got %+v", summary)
	}
	if summary.ConversionWarnings[0].Reason != "unsupported_currency" {
		t.Fatalf("expected unsupported_currency warning, got %+v", summary.ConversionWarnings)
	}

	failClosedRouter, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithBudgetConversion(routeTestExchangeRates{}, true, false),
	)
	failClosedTripID := createBudgetTestTrip(t, failClosedRouter, ownerToken)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/trips/"+failClosedTripID.String()+"/itinerary", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	failClosedRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	rec = getBudgetSummary(t, failClosedRouter, ownerToken, failClosedTripID)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected fail-closed HTTP 502, got %d with %s", rec.Code, rec.Body.String())
	}
	var errBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errBody["error"] != "budget_conversion_failed" {
		t.Fatalf("expected budget_conversion_failed, got %+v", errBody)
	}
}

func TestUpdateBudgetOwnerSucceedsWithoutRevisionBump(t *testing.T) {
	router, _ := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)
	putItinerary(t, router, ownerToken, tripID, 0)

	revisionBefore := fetchTripRevision(t, router, ownerToken, tripID)

	rec := putBudget(t, router, ownerToken, tripID, `{"budget":{"amount":700,"currency":"EUR"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Budget *struct {
			Amount   float64 `json:"amount"`
			Currency string  `json:"currency"`
		} `json:"budget"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode budget envelope: %v", err)
	}
	if envelope.Budget == nil || envelope.Budget.Amount != 700 || envelope.Budget.Currency != "EUR" {
		t.Fatalf("unexpected budget envelope: %+v", envelope.Budget)
	}

	revisionAfter := fetchTripRevision(t, router, ownerToken, tripID)
	if revisionAfter != revisionBefore {
		t.Fatalf("expected itineraryRevision unchanged (%d), got %d", revisionBefore, revisionAfter)
	}

	// Summary should reflect the new budget.
	summaryRec := getBudgetSummary(t, router, ownerToken, tripID)
	var summary struct {
		TripBudget *float64 `json:"tripBudget"`
		Remaining  *float64 `json:"remaining"`
	}
	_ = json.Unmarshal(summaryRec.Body.Bytes(), &summary)
	if summary.TripBudget == nil || *summary.TripBudget != 700 {
		t.Fatalf("expected updated tripBudget 700, got %v", summary.TripBudget)
	}
}

func TestUpdateBudgetClearAndValidation(t *testing.T) {
	router, _ := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)

	// Invalid currency.
	if rec := putBudget(t, router, ownerToken, tripID, `{"budget":{"amount":100,"currency":"EU"}}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400 for invalid currency, got %d with %s", rec.Code, rec.Body.String())
	}
	// Negative amount.
	if rec := putBudget(t, router, ownerToken, tripID, `{"budget":{"amount":-5,"currency":"EUR"}}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400 for negative amount, got %d with %s", rec.Code, rec.Body.String())
	}
	// Clear budget.
	rec := putBudget(t, router, ownerToken, tripID, `{"budget":null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for clear, got %d with %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Budget *struct{} `json:"budget"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode budget envelope: %v", err)
	}
	if envelope.Budget != nil {
		t.Fatalf("expected null budget after clear, got %+v", envelope.Budget)
	}
}

func TestPublicShareStripsItineraryTotalBudget(t *testing.T) {
	router, _ := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	tripID := createBudgetTestTrip(t, router, ownerToken)

	// Store an itinerary that carries a top-level totalBudget and an item cost.
	itineraryJSON := `{
		"expectedItineraryRevision": 0,
		"itinerary": {
			"currency": "EUR",
			"totalBudget": 999,
			"days": [
				{
					"day": 1,
					"title": "Day",
					"items": [
						{"time":"09:00","type":"ticket","name":"Museum","estimatedCost":{"amount":18,"currency":"EUR","category":"ticket"}}
					]
				}
			]
		}
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/trips/"+tripID.String()+"/itinerary", bytes.NewReader([]byte(itineraryJSON)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected itinerary update HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}

	// Enable the public share.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trips/"+tripID.String()+"/share", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected create share HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var share struct {
		ShareToken string `json:"shareToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatalf("decode share: %v", err)
	}

	// Fetch the public trip and assert totalBudget is stripped but item costs remain.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/public/trips/"+share.ShareToken, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected public trip HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	var publicBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &publicBody); err != nil {
		t.Fatalf("decode public trip: %v", err)
	}
	itinerary, ok := publicBody["itinerary"].(map[string]any)
	if !ok {
		t.Fatalf("expected public itinerary object, got %+v", publicBody["itinerary"])
	}
	if _, ok := itinerary["totalBudget"]; ok {
		t.Fatalf("public itinerary must not include totalBudget: %+v", itinerary)
	}
	days, ok := itinerary["days"].([]any)
	if !ok || len(days) != 1 {
		t.Fatalf("expected one public itinerary day, got %+v", itinerary["days"])
	}
	day := days[0].(map[string]any)
	items := day["items"].([]any)
	item := items[0].(map[string]any)
	if _, ok := item["estimatedCost"]; !ok {
		t.Fatalf("expected item estimatedCost to remain on public share, got %+v", item)
	}
}

func TestBudgetPermissionsViewerAndNonCollaborator(t *testing.T) {
	router, repo := newAuthTestRouter(t, budgetTestAuthConfig())
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	viewerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	strangerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	viewerToken := signAccessToken(t, viewerID, "viewer@example.com", testJWTSecret, time.Hour)
	strangerToken := signAccessToken(t, strangerID, "stranger@example.com", testJWTSecret, time.Hour)

	tripID := createBudgetTestTrip(t, router, ownerToken)
	seedAcceptedCollaborator(repo, tripID, viewerID, entity.CollaboratorRoleViewer)

	// Viewer can read the summary.
	if rec := getBudgetSummary(t, router, viewerToken, tripID); rec.Code != http.StatusOK {
		t.Fatalf("expected viewer summary HTTP 200, got %d with %s", rec.Code, rec.Body.String())
	}
	// Viewer cannot update the budget.
	if rec := putBudget(t, router, viewerToken, tripID, `{"budget":{"amount":300,"currency":"EUR"}}`); rec.Code != http.StatusForbidden {
		t.Fatalf("expected viewer update HTTP 403, got %d with %s", rec.Code, rec.Body.String())
	}
	// Non-collaborator cannot read the summary.
	if rec := getBudgetSummary(t, router, strangerToken, tripID); rec.Code != http.StatusNotFound {
		t.Fatalf("expected non-collaborator summary HTTP 404, got %d with %s", rec.Code, rec.Body.String())
	}
}

type routeTestExchangeRates struct{}

func (routeTestExchangeRates) Convert(_ context.Context, amount float64, from string, to string) (*budget.CurrencyConversionResult, error) {
	if from == "JPY" && to == "EUR" {
		return &budget.CurrencyConversionResult{
			Provider:        "route-test",
			From:            from,
			To:              to,
			Amount:          amount,
			ConvertedAmount: mathRound2(amount / 170.5),
			Rate:            1 / 170.5,
			AsOf:            time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
		}, nil
	}
	return nil, routeTestExchangeRateError{reason: "unsupported_currency"}
}

type routeTestExchangeRateError struct {
	reason string
}

func (e routeTestExchangeRateError) Error() string  { return e.reason }
func (e routeTestExchangeRateError) Reason() string { return e.reason }

func amountByOriginalCurrency(totals []budget.OriginalCurrencyTotal, currency string) float64 {
	for _, total := range totals {
		if total.Currency == currency {
			return total.Amount
		}
	}
	return 0
}

func mathRound2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
