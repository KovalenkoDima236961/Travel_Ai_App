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

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

func TestWorkspaceBudgetOwnerCreatesViewerReadsSummaryAndCannotPatch(t *testing.T) {
	workspaceID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	viewerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	strangerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	workspaceProvider := routeTestWorkspaceProvider{
		access: map[uuid.UUID]map[uuid.UUID]workspaces.Role{
			workspaceID: {
				ownerID:  workspaces.RoleOwner,
				viewerID: workspaces.RoleViewer,
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
	viewerToken := signAccessToken(t, viewerID, "viewer@example.com", testJWTSecret, time.Hour)
	strangerToken := signAccessToken(t, strangerID, "stranger@example.com", testJWTSecret, time.Hour)

	tripID := createWorkspaceAnalyticsTrip(t, router, ownerToken, workspaceID, "Tokyo", "2026-09-10", 700)
	putCostAnalyticsItinerary(t, router, ownerToken, tripID, 0, "2026-09-10")

	createRec := postWorkspaceBudget(t, router, ownerToken, workspaceID, `{
		"name":"Japan group budget",
		"amount":1000,
		"currency":"EUR",
		"periodStart":"2026-09-01",
		"periodEnd":"2026-09-30"
	}`)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create HTTP 201, got %d with %s", createRec.Code, createRec.Body.String())
	}
	var created appdto.WorkspaceBudgetEnvelope
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created budget: %v", err)
	}
	if !created.Budget.IsPrimary {
		t.Fatalf("expected first active budget to become primary: %+v", created.Budget)
	}

	summaryRec := getWorkspaceBudgetSummary(t, router, viewerToken, workspaceID, created.Budget.ID)
	if summaryRec.Code != http.StatusOK {
		t.Fatalf("expected viewer summary HTTP 200, got %d with %s", summaryRec.Code, summaryRec.Body.String())
	}
	var summary appdto.WorkspaceBudgetSummaryResponse
	if err := json.Unmarshal(summaryRec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode budget summary: %v", err)
	}
	if summary.Summary.TripCount != 1 || summary.Summary.EstimatedTotal != 500 {
		t.Fatalf("expected one included trip and total 500, got %+v", summary.Summary)
	}
	if summary.Summary.UtilizationPercent != 50 {
		t.Fatalf("expected 50%% utilization, got %+v", summary.Summary)
	}

	patchReq := httptest.NewRequest(
		http.MethodPatch,
		"/workspaces/"+workspaceID.String()+"/budgets/"+created.Budget.ID.String(),
		bytes.NewReader([]byte(`{"amount":900}`)),
	)
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+viewerToken)
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusForbidden {
		t.Fatalf("expected viewer patch HTTP 403, got %d with %s", patchRec.Code, patchRec.Body.String())
	}

	strangerRec := getWorkspaceBudgetSummary(t, router, strangerToken, workspaceID, created.Budget.ID)
	if strangerRec.Code != http.StatusForbidden {
		t.Fatalf("expected stranger summary HTTP 403, got %d with %s", strangerRec.Code, strangerRec.Body.String())
	}
}

func TestWorkspaceBudgetMakePrimaryClearsPreviousPrimary(t *testing.T) {
	workspaceID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	workspaceProvider := routeTestWorkspaceProvider{
		access: map[uuid.UUID]map[uuid.UUID]workspaces.Role{
			workspaceID: {ownerID: workspaces.RoleAdmin},
		},
	}
	router, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithWorkspaces(workspaceProvider, true),
	)
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)

	first := decodeWorkspaceBudgetEnvelope(t, postWorkspaceBudget(t, router, ownerToken, workspaceID, `{
		"name":"First budget",
		"amount":1000,
		"currency":"EUR"
	}`))
	second := decodeWorkspaceBudgetEnvelope(t, postWorkspaceBudget(t, router, ownerToken, workspaceID, `{
		"name":"Second budget",
		"amount":2000,
		"currency":"EUR",
		"isPrimary":true
	}`))
	if !second.Budget.IsPrimary {
		t.Fatalf("expected second budget primary: %+v", second.Budget)
	}

	listRec := getWorkspaceBudgets(t, router, ownerToken, workspaceID, "status=active")
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list HTTP 200, got %d with %s", listRec.Code, listRec.Body.String())
	}
	var list appdto.WorkspaceBudgetsEnvelope
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode budget list: %v", err)
	}
	primaryCount := 0
	for _, budget := range list.Budgets {
		if budget.IsPrimary {
			primaryCount++
			if budget.ID != second.Budget.ID {
				t.Fatalf("expected second budget as only primary, got %+v", budget)
			}
		}
		if budget.ID == first.Budget.ID && budget.IsPrimary {
			t.Fatalf("expected first budget primary flag cleared")
		}
	}
	if primaryCount != 1 {
		t.Fatalf("expected one primary budget, got %d in %+v", primaryCount, list.Budgets)
	}
}

func TestWorkspaceBudgetNotificationsGoToOwnerAdminsOnly(t *testing.T) {
	workspaceID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	adminID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	memberID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	viewerID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	workspaceProvider := routeTestWorkspaceProvider{
		access: map[uuid.UUID]map[uuid.UUID]workspaces.Role{
			workspaceID: {
				ownerID:  workspaces.RoleOwner,
				adminID:  workspaces.RoleAdmin,
				memberID: workspaces.RoleMember,
				viewerID: workspaces.RoleViewer,
			},
		},
	}
	notifier := &routeRecordingNotifier{}
	router, _ := newAuthTestRouterWithOptions(
		t,
		budgetTestAuthConfig(),
		service.WithWorkspaces(workspaceProvider, true),
		service.WithNotifications(notifier, true, true),
	)
	ownerToken := signAccessToken(t, ownerID, "owner@example.com", testJWTSecret, time.Hour)
	adminToken := signAccessToken(t, adminID, "admin@example.com", testJWTSecret, time.Hour)

	created := decodeWorkspaceBudgetEnvelope(t, postWorkspaceBudget(t, router, ownerToken, workspaceID, `{
		"name":"Japan group budget",
		"amount":1000,
		"currency":"EUR"
	}`))
	assertLatestBudgetNotification(t, notifier, notifications.TypeWorkspaceBudgetCreated, adminID, ownerID, created.Budget.ID)

	patchReq := httptest.NewRequest(
		http.MethodPatch,
		"/workspaces/"+workspaceID.String()+"/budgets/"+created.Budget.ID.String(),
		bytes.NewReader([]byte(`{"amount":1200}`)),
	)
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+adminToken)
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected update HTTP 200, got %d with %s", patchRec.Code, patchRec.Body.String())
	}
	assertLatestBudgetNotification(t, notifier, notifications.TypeWorkspaceBudgetUpdated, ownerID, adminID, created.Budget.ID)

	archiveReq := httptest.NewRequest(
		http.MethodPost,
		"/workspaces/"+workspaceID.String()+"/budgets/"+created.Budget.ID.String()+"/archive",
		bytes.NewReader([]byte(`{"reason":"replaced"}`)),
	)
	archiveReq.Header.Set("Content-Type", "application/json")
	archiveReq.Header.Set("Authorization", "Bearer "+ownerToken)
	archiveRec := httptest.NewRecorder()
	router.ServeHTTP(archiveRec, archiveReq)
	if archiveRec.Code != http.StatusOK {
		t.Fatalf("expected archive HTTP 200, got %d with %s", archiveRec.Code, archiveRec.Body.String())
	}
	assertLatestBudgetNotification(t, notifier, notifications.TypeWorkspaceBudgetArchived, adminID, ownerID, created.Budget.ID)

	if len(notifier.sent) != 3 {
		t.Fatalf("expected exactly 3 notifications, got %+v", notifier.sent)
	}
	for _, sent := range notifier.sent {
		if sent.UserID == memberID || sent.UserID == viewerID {
			t.Fatalf("expected member/viewer not to be notified, got %+v", sent)
		}
	}
}

type routeRecordingNotifier struct {
	sent []notifications.NotificationCreateInput
}

func (n *routeRecordingNotifier) CreateNotifications(_ context.Context, batch []notifications.NotificationCreateInput) error {
	n.sent = append(n.sent, batch...)
	return nil
}

func assertLatestBudgetNotification(
	t *testing.T,
	notifier *routeRecordingNotifier,
	notificationType string,
	recipientID uuid.UUID,
	actorID uuid.UUID,
	budgetID uuid.UUID,
) {
	t.Helper()
	if len(notifier.sent) == 0 {
		t.Fatal("expected notification to be sent")
	}
	got := notifier.sent[len(notifier.sent)-1]
	if got.Type != notificationType {
		t.Fatalf("expected notification type %q, got %+v", notificationType, got)
	}
	if got.UserID != recipientID {
		t.Fatalf("expected recipient %s, got %+v", recipientID, got)
	}
	if got.ActorUserID == nil || *got.ActorUserID != actorID {
		t.Fatalf("expected actor %s, got %+v", actorID, got)
	}
	if got.EntityType == nil || *got.EntityType != notifications.EntityWorkspaceBudget {
		t.Fatalf("expected workspace budget entity type, got %+v", got)
	}
	if got.EntityID == nil || *got.EntityID != budgetID {
		t.Fatalf("expected entity id %s, got %+v", budgetID, got)
	}
}

func postWorkspaceBudget(t *testing.T, router http.Handler, token string, workspaceID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/workspaces/"+workspaceID.String()+"/budgets", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getWorkspaceBudgets(t *testing.T, router http.Handler, token string, workspaceID uuid.UUID, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/workspaces/" + workspaceID.String() + "/budgets"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getWorkspaceBudgetSummary(t *testing.T, router http.Handler, token string, workspaceID, budgetID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String()+"/budgets/"+budgetID.String()+"/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeWorkspaceBudgetEnvelope(t *testing.T, rec *httptest.ResponseRecorder) appdto.WorkspaceBudgetEnvelope {
	t.Helper()
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create HTTP 201, got %d with %s", rec.Code, rec.Body.String())
	}
	var out appdto.WorkspaceBudgetEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode budget envelope: %v", err)
	}
	return out
}
