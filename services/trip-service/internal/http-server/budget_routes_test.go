package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

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
