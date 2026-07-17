package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

type SearchRepository struct {
	db *postgres.DB
}

func NewRepository(db *postgres.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

func (r *SearchRepository) Search(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if params.Scope == ScopeCurrentTrip && params.TripID == nil {
		return nil, nil
	}
	results := make([]Result, 0, params.Limit*2)

	if includeTripScoped(params.Scope) {
		trips, err := r.searchTrips(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, trips...)
	}
	if params.Scope == ScopeAll || params.Scope == ScopeCurrentTrip || params.Scope == ScopeWorkspace {
		parsed, err := r.searchRouteAndItinerary(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, parsed...)

		expenses, err := r.searchExpenses(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, expenses...)

		receipts, err := r.searchReceipts(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, receipts...)

		checklistItems, err := r.searchChecklistItems(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, checklistItems...)

		reminders, err := r.searchReminders(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, reminders...)

		polls, err := r.searchPolls(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, polls...)

		collaborators, err := r.searchCollaborators(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, collaborators...)
	}
	if params.Scope == ScopeAll || params.Scope == ScopeTrips || params.Scope == ScopeWorkspace {
		templates, err := r.searchTemplates(ctx, params)
		if err != nil {
			return nil, err
		}
		results = append(results, templates...)
	}
	return results, nil
}

func (r *SearchRepository) searchTrips(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT t.id, t.destination, t.start_date, t.days, t.workspace_id, t.updated_at
FROM trips t
WHERE ` + accessibleTripPredicate + `
  AND t.destination ILIKE ANY($5::text[])
ORDER BY t.updated_at DESC
LIMIT $6`
	rows, err := r.db.Query(ctx, query, params.queryArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search trips: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id          uuid.UUID
			destination string
			startDate   pgtype.Date
			days        int
			workspaceID pgtype.UUID
			updatedAt   time.Time
		)
		if err := rows.Scan(&id, &destination, &startDate, &days, &workspaceID, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan trip search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		results = append(results, newResult(
			ResultTypeTrip,
			"trip:"+id.String(),
			destination,
			tripDescription(startDate, days),
			destination,
			params.workspaceName(workspacePtr),
			tripHref(id),
			idMetadata(map[string]string{"tripId": id.String()}),
			resultRefs{TripID: uuidValuePtr(id), WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchRouteAndItinerary(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT t.id, t.destination, t.workspace_id, t.itinerary, t.route_json, t.updated_at
FROM trips t
WHERE ` + accessibleTripPredicate + `
ORDER BY t.updated_at DESC
LIMIT $5`
	rows, err := r.db.Query(ctx, query, params.accessArgs(jsonTripLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search route itinerary: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			tripID      uuid.UUID
			destination string
			workspaceID pgtype.UUID
			itinerary   []byte
			route       []byte
			updatedAt   time.Time
		)
		if err := rows.Scan(&tripID, &destination, &workspaceID, &itinerary, &route, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan route itinerary search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		refs := resultRefs{TripID: uuidValuePtr(tripID), WorkspaceID: workspacePtr, UpdatedAt: updatedAt}
		workspaceName := params.workspaceName(workspacePtr)
		results = append(results, routeResults(params, tripID, destination, workspaceName, route, refs)...)
		results = append(results, itineraryResults(params, tripID, destination, workspaceName, itinerary, refs)...)
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchExpenses(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT e.id, e.trip_id, t.destination, t.workspace_id, e.title, COALESCE(e.description, ''),
       e.amount::float8, e.currency, e.category, e.expense_date, e.updated_at
FROM trip_expenses e
JOIN trips t ON t.id = e.trip_id
WHERE ` + accessibleTripPredicate + `
  AND e.status = 'active'
  AND e.deleted_at IS NULL
  AND (
    e.title ILIKE ANY($5::text[])
    OR COALESCE(e.description, '') ILIKE ANY($5::text[])
    OR e.category ILIKE ANY($5::text[])
    OR e.currency ILIKE ANY($5::text[])
  )
ORDER BY e.updated_at DESC
LIMIT $6`
	rows, err := r.db.Query(ctx, query, params.queryArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search expenses: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id, tripID  uuid.UUID
			destination string
			workspaceID pgtype.UUID
			title       string
			description string
			amount      float64
			currency    string
			category    string
			expenseDate pgtype.Date
			updatedAt   time.Time
		)
		if err := rows.Scan(&id, &tripID, &destination, &workspaceID, &title, &description, &amount, &currency, &category, &expenseDate, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan expense search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		results = append(results, newResult(
			ResultTypeExpense,
			"expense:"+id.String(),
			title,
			firstNonEmpty(shortText(description, 90), fmt.Sprintf("%s · %.2f %s", titleCase(category), amount, currency)),
			destination,
			params.workspaceName(workspacePtr),
			tripTabHref(tripID, "expenses", map[string]string{"expenseId": id.String()}),
			idMetadata(map[string]string{"tripId": tripID.String(), "expenseId": id.String(), "category": category, "date": dateString(expenseDate)}),
			resultRefs{TripID: uuidValuePtr(tripID), WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchReceipts(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT r.id, r.trip_id, t.destination, t.workspace_id, r.original_filename, r.status,
       COALESCE(e.title, ''), COALESCE(ocr.merchant, ''), COALESCE(ocr.suggested_title, ''),
       COALESCE(ocr.category, ''), r.updated_at
FROM trip_expense_receipts r
JOIN trips t ON t.id = r.trip_id
LEFT JOIN trip_expenses e ON e.id = r.expense_id AND e.status = 'active' AND e.deleted_at IS NULL
LEFT JOIN LATERAL (
  SELECT merchant, suggested_title, category
  FROM receipt_ocr_results
  WHERE receipt_id = r.id
  ORDER BY created_at DESC
  LIMIT 1
) ocr ON true
WHERE ` + accessibleTripPredicate + `
  AND r.deleted_at IS NULL
  AND r.status <> 'deleted'
  AND (
    r.original_filename ILIKE ANY($5::text[])
    OR COALESCE(e.title, '') ILIKE ANY($5::text[])
    OR COALESCE(ocr.merchant, '') ILIKE ANY($5::text[])
    OR COALESCE(ocr.suggested_title, '') ILIKE ANY($5::text[])
    OR COALESCE(ocr.category, '') ILIKE ANY($5::text[])
  )
ORDER BY r.updated_at DESC
LIMIT $6`
	rows, err := r.db.Query(ctx, query, params.queryArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search receipts: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id, tripID     uuid.UUID
			destination    string
			workspaceID    pgtype.UUID
			filename       string
			status         string
			expenseTitle   string
			merchant       string
			suggestedTitle string
			category       string
			updatedAt      time.Time
		)
		if err := rows.Scan(&id, &tripID, &destination, &workspaceID, &filename, &status, &expenseTitle, &merchant, &suggestedTitle, &category, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan receipt search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		description := strings.Join(nonEmpty([]string{firstNonEmpty(merchant, suggestedTitle, expenseTitle), titleCase(status), titleCase(category)}), " · ")
		results = append(results, newResult(
			ResultTypeReceipt,
			"receipt:"+id.String(),
			filename,
			description,
			destination,
			params.workspaceName(workspacePtr),
			tripTabHref(tripID, "receipts", map[string]string{"receiptId": id.String()}),
			idMetadata(map[string]string{"tripId": tripID.String(), "receiptId": id.String(), "status": status}),
			resultRefs{TripID: uuidValuePtr(tripID), WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchChecklistItems(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT item.id, item.trip_id, t.destination, t.workspace_id, item.title,
       COALESCE(item.description, ''), item.category, item.priority, item.due_date, item.updated_at
FROM trip_checklist_items item
JOIN trips t ON t.id = item.trip_id
WHERE ` + accessibleTripPredicate + `
  AND item.deleted_at IS NULL
  AND (
    item.title ILIKE ANY($5::text[])
    OR COALESCE(item.description, '') ILIKE ANY($5::text[])
    OR item.category ILIKE ANY($5::text[])
    OR item.priority ILIKE ANY($5::text[])
  )
ORDER BY item.updated_at DESC
LIMIT $6`
	rows, err := r.db.Query(ctx, query, params.queryArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search checklist items: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id, tripID  uuid.UUID
			destination string
			workspaceID pgtype.UUID
			title       string
			description string
			category    string
			priority    string
			dueDate     pgtype.Date
			updatedAt   time.Time
		)
		if err := rows.Scan(&id, &tripID, &destination, &workspaceID, &title, &description, &category, &priority, &dueDate, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan checklist search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		results = append(results, newResult(
			ResultTypeChecklistItem,
			"checklist_item:"+id.String(),
			title,
			firstNonEmpty(shortText(description, 90), strings.Join(nonEmpty([]string{titleCase(priority), titleCase(category), dateDescription("Due", dueDate)}), " · ")),
			destination,
			params.workspaceName(workspacePtr),
			tripTabHref(tripID, "checklist", map[string]string{"itemId": id.String()}),
			idMetadata(map[string]string{"tripId": tripID.String(), "itemId": id.String(), "category": category, "priority": priority}),
			resultRefs{TripID: uuidValuePtr(tripID), WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchReminders(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT rem.id, rem.trip_id, t.destination, t.workspace_id, rem.title,
       COALESCE(rem.description, ''), rem.category, rem.priority, rem.status, rem.trigger_date, rem.updated_at
FROM trip_reminders rem
JOIN trips t ON t.id = rem.trip_id
WHERE ` + accessibleTripPredicate + `
  AND rem.deleted_at IS NULL
  AND (
    rem.title ILIKE ANY($5::text[])
    OR COALESCE(rem.description, '') ILIKE ANY($5::text[])
    OR rem.category ILIKE ANY($5::text[])
    OR rem.priority ILIKE ANY($5::text[])
    OR rem.status ILIKE ANY($5::text[])
  )
ORDER BY rem.updated_at DESC
LIMIT $6`
	rows, err := r.db.Query(ctx, query, params.queryArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search reminders: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id, tripID  uuid.UUID
			destination string
			workspaceID pgtype.UUID
			title       string
			description string
			category    string
			priority    string
			status      string
			triggerDate pgtype.Date
			updatedAt   time.Time
		)
		if err := rows.Scan(&id, &tripID, &destination, &workspaceID, &title, &description, &category, &priority, &status, &triggerDate, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan reminder search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		results = append(results, newResult(
			ResultTypeReminder,
			"reminder:"+id.String(),
			title,
			firstNonEmpty(shortText(description, 90), strings.Join(nonEmpty([]string{titleCase(priority), titleCase(category), titleCase(status), dateDescription("Due", triggerDate)}), " · ")),
			destination,
			params.workspaceName(workspacePtr),
			tripTabHref(tripID, "reminders", map[string]string{"reminderId": id.String()}),
			idMetadata(map[string]string{"tripId": tripID.String(), "reminderId": id.String(), "category": category, "status": status}),
			resultRefs{TripID: uuidValuePtr(tripID), WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchPolls(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT poll.id, poll.trip_id, t.destination, t.workspace_id, poll.title,
       COALESCE(poll.description, ''), poll.poll_type, poll.status, poll.updated_at
FROM trip_polls poll
JOIN trips t ON t.id = poll.trip_id
WHERE ` + accessibleTripPredicate + `
  AND poll.status <> 'archived'
  AND (
    poll.title ILIKE ANY($5::text[])
    OR COALESCE(poll.description, '') ILIKE ANY($5::text[])
    OR poll.poll_type ILIKE ANY($5::text[])
    OR poll.status ILIKE ANY($5::text[])
  )
ORDER BY poll.updated_at DESC
LIMIT $6`
	rows, err := r.db.Query(ctx, query, params.queryArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search polls: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id, tripID  uuid.UUID
			destination string
			workspaceID pgtype.UUID
			title       string
			description string
			pollType    string
			status      string
			updatedAt   time.Time
		)
		if err := rows.Scan(&id, &tripID, &destination, &workspaceID, &title, &description, &pollType, &status, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan poll search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		results = append(results, newResult(
			ResultTypePoll,
			"poll:"+id.String(),
			title,
			firstNonEmpty(shortText(description, 90), strings.Join(nonEmpty([]string{titleCase(pollType), titleCase(status)}), " · ")),
			destination,
			params.workspaceName(workspacePtr),
			tripTabHref(tripID, "polls", map[string]string{"pollId": id.String()}),
			idMetadata(map[string]string{"tripId": tripID.String(), "pollId": id.String(), "status": status}),
			resultRefs{TripID: uuidValuePtr(tripID), WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchCollaborators(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT collab.id, collab.trip_id, t.destination, t.workspace_id, collab.user_id, collab.role, collab.status, collab.updated_at
FROM trip_collaborators collab
JOIN trips t ON t.id = collab.trip_id
WHERE ` + accessibleTripPredicate + `
  AND collab.status = 'accepted'
  AND (
    collab.user_id::text ILIKE ANY($5::text[])
    OR collab.role ILIKE ANY($5::text[])
  )
ORDER BY collab.updated_at DESC
LIMIT $6`
	rows, err := r.db.Query(ctx, query, params.queryArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search collaborators: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id, tripID, collaboratorUserID uuid.UUID
			destination                    string
			workspaceID                    pgtype.UUID
			role                           string
			status                         string
			updatedAt                      time.Time
		)
		if err := rows.Scan(&id, &tripID, &destination, &workspaceID, &collaboratorUserID, &role, &status, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan collaborator search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		results = append(results, newResult(
			ResultTypeCollaborator,
			"collaborator:"+id.String(),
			"Collaborator "+collaboratorUserID.String()[:8],
			strings.Join(nonEmpty([]string{titleCase(role), titleCase(status)}), " · "),
			destination,
			params.workspaceName(workspacePtr),
			tripTabHref(tripID, "collaborators", map[string]string{"collaboratorId": id.String()}),
			idMetadata(map[string]string{"tripId": tripID.String(), "collaboratorId": id.String(), "userId": collaboratorUserID.String(), "role": role}),
			resultRefs{TripID: uuidValuePtr(tripID), WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

func (r *SearchRepository) searchTemplates(ctx context.Context, params RepositorySearchParams) ([]Result, error) {
	const query = `
SELECT tmpl.id, tmpl.workspace_id, tmpl.title, COALESCE(tmpl.description, ''),
       COALESCE(tmpl.destination_hint, ''), tmpl.duration_days, tmpl.tags, tmpl.updated_at
FROM trip_templates tmpl
WHERE tmpl.status = 'active'
  AND (
    (tmpl.visibility = 'private' AND tmpl.created_by_user_id = $1 AND $3::uuid IS NULL)
    OR (tmpl.visibility = 'workspace' AND tmpl.workspace_id = ANY($2::uuid[]) AND ($3::uuid IS NULL OR tmpl.workspace_id = $3::uuid))
  )
  AND (
    tmpl.title ILIKE ANY($4::text[])
    OR COALESCE(tmpl.description, '') ILIKE ANY($4::text[])
    OR COALESCE(tmpl.destination_hint, '') ILIKE ANY($4::text[])
    OR array_to_string(tmpl.tags, ' ') ILIKE ANY($4::text[])
  )
ORDER BY tmpl.updated_at DESC
LIMIT $5`
	rows, err := r.db.Query(ctx, query, params.templateArgs(queryLimit(params))...)
	if err != nil {
		return nil, fmt.Errorf("search templates: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var (
			id              uuid.UUID
			workspaceID     pgtype.UUID
			title           string
			description     string
			destinationHint string
			durationDays    int
			tags            []string
			updatedAt       time.Time
		)
		if err := rows.Scan(&id, &workspaceID, &title, &description, &destinationHint, &durationDays, &tags, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan template search row: %w", err)
		}
		workspacePtr := uuidPtr(workspaceID)
		results = append(results, newResult(
			ResultTypeTemplate,
			"template:"+id.String(),
			title,
			firstNonEmpty(shortText(description, 90), strings.Join(nonEmpty([]string{destinationHint, fmt.Sprintf("%d days", durationDays), strings.Join(tags, ", ")}), " · ")),
			"",
			params.workspaceName(workspacePtr),
			templateHref(id),
			idMetadata(map[string]string{"templateId": id.String(), "workspaceId": uuidString(workspacePtr)}),
			resultRefs{WorkspaceID: workspacePtr, UpdatedAt: updatedAt},
		))
	}
	return results, rows.Err()
}

const accessibleTripPredicate = `(
    (t.user_id = $1 AND t.workspace_id IS NULL)
    OR t.workspace_id = ANY($2::uuid[])
    OR EXISTS (
      SELECT 1
      FROM trip_collaborators allowed_collab
      WHERE allowed_collab.trip_id = t.id
        AND allowed_collab.user_id = $1
        AND allowed_collab.status = 'accepted'
    )
  )
  AND ($3::uuid IS NULL OR t.id = $3::uuid)
  AND ($4::uuid IS NULL OR t.workspace_id = $4::uuid)`

func (params RepositorySearchParams) queryArgs(limit int) []any {
	return []any{
		params.UserID,
		params.WorkspaceIDs,
		uuidArg(params.TripID),
		uuidArg(params.WorkspaceID),
		params.Patterns,
		limit,
	}
}

func (params RepositorySearchParams) templateArgs(limit int) []any {
	return []any{
		params.UserID,
		params.WorkspaceIDs,
		uuidArg(params.WorkspaceID),
		params.Patterns,
		limit,
	}
}

func (params RepositorySearchParams) accessArgs(limit int) []any {
	return []any{
		params.UserID,
		params.WorkspaceIDs,
		uuidArg(params.TripID),
		uuidArg(params.WorkspaceID),
		limit,
	}
}

func (params RepositorySearchParams) workspaceName(workspaceID *uuid.UUID) string {
	if workspaceID == nil {
		return ""
	}
	return params.WorkspaceNames[*workspaceID]
}

func queryLimit(params RepositorySearchParams) int {
	limit := params.Limit * 3
	if limit < 20 {
		limit = 20
	}
	if limit > 150 {
		limit = 150
	}
	return limit
}

func jsonTripLimit(params RepositorySearchParams) int {
	if params.Scope == ScopeCurrentTrip {
		return 1
	}
	limit := params.Limit * 10
	if limit < 80 {
		limit = 80
	}
	if limit > 300 {
		limit = 300
	}
	return limit
}

func routeResults(params RepositorySearchParams, tripID uuid.UUID, contextName, workspaceName string, raw []byte, refs resultRefs) []Result {
	if len(raw) == 0 || strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil
	}
	var route aggregate.TripRoute
	if err := json.Unmarshal(raw, &route); err != nil {
		return nil
	}
	results := make([]Result, 0)
	for _, stop := range route.Stops {
		if !matchesTokens(params.Query, params.Tokens, stop.Destination, stop.City, stop.Country) {
			continue
		}
		stopID := firstNonEmpty(stop.ID, stop.Destination)
		results = append(results, newResult(
			ResultTypeRouteStop,
			"route_stop:"+tripID.String()+":"+stopID,
			firstNonEmpty(stop.Destination, stop.City, stop.Country),
			strings.Join(nonEmpty([]string{stop.City, stop.Country, dateRange(stop.ArrivalDate, stop.DepartureDate)}), " · "),
			contextName,
			workspaceName,
			tripTabHref(tripID, "route", map[string]string{"stopId": stopID}),
			idMetadata(map[string]string{"tripId": tripID.String(), "stopId": stopID}),
			refs,
		))
	}
	for _, leg := range route.Legs {
		selected := leg.SelectedTransportOption
		selectedFields := []string{}
		if selected != nil {
			selectedFields = append(selectedFields, selected.Provider, selected.OperatorName, selected.ServiceName, selected.OriginName, selected.DestinationName)
		}
		if matchesTokens(params.Query, params.Tokens, append([]string{leg.FromName, leg.ToName, leg.Mode}, selectedFields...)...) {
			legID := firstNonEmpty(leg.ID, leg.FromStopID+"-"+leg.ToStopID)
			results = append(results, newResult(
				ResultTypeRouteLeg,
				"route_leg:"+tripID.String()+":"+legID,
				firstNonEmpty(strings.TrimSpace(leg.FromName+" → "+leg.ToName), legID),
				routeLegDescription(leg),
				contextName,
				workspaceName,
				tripTabHref(tripID, "route", map[string]string{"legId": legID}),
				idMetadata(map[string]string{"tripId": tripID.String(), "legId": legID, "mode": leg.Mode}),
				refs,
			))
		}
		if selected != nil && matchesTokens(params.Query, params.Tokens, selected.Provider, selected.OperatorName, selected.ServiceName, selected.OriginName, selected.DestinationName, selected.Mode) {
			optionID := firstNonEmpty(selected.ID, leg.ID)
			results = append(results, newResult(
				ResultTypeTransportOption,
				"transport_option:"+tripID.String()+":"+optionID,
				firstNonEmpty(selected.OperatorName, selected.ServiceName, selected.Provider, leg.FromName+" → "+leg.ToName),
				transportOptionDescription(selected),
				contextName,
				workspaceName,
				tripTabHref(tripID, "route", map[string]string{"legId": leg.ID}),
				idMetadata(map[string]string{"tripId": tripID.String(), "legId": leg.ID, "transportOptionId": optionID}),
				refs,
			))
		}
	}
	return results
}

func itineraryResults(params RepositorySearchParams, tripID uuid.UUID, contextName, workspaceName string, raw []byte, refs resultRefs) []Result {
	if len(raw) == 0 || strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil
	}
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(raw, &itinerary); err != nil {
		return nil
	}
	results := make([]Result, 0)
	for _, day := range itinerary.Days {
		for index, item := range day.Items {
			placeName, placeAddress := "", ""
			if item.Place != nil {
				placeName = item.Place.Name
				placeAddress = item.Place.Address
			}
			if !matchesTokens(params.Query, params.Tokens, item.Name, item.Note, item.Type, item.Category, placeName, placeAddress, day.Title, day.LocationName) {
				continue
			}
			metadata := map[string]any{
				"tripId":    tripID.String(),
				"dayNumber": day.Day,
				"itemIndex": index,
			}
			results = append(results, newResult(
				ResultTypeItineraryItem,
				fmt.Sprintf("itinerary_item:%s:%d:%d", tripID.String(), day.Day, index),
				item.Name,
				itineraryItemDescription(day.Day, item, placeName),
				contextName,
				workspaceName,
				tripTabHref(tripID, "itinerary", map[string]string{"day": fmt.Sprint(day.Day), "itemIndex": fmt.Sprint(index)}),
				metadata,
				refs,
			))
		}
	}
	return results
}

func routeLegDescription(leg aggregate.RouteLeg) string {
	parts := []string{titleCase(leg.Mode)}
	if leg.EstimatedDurationMinutes != nil && *leg.EstimatedDurationMinutes > 0 {
		parts = append(parts, fmt.Sprintf("%d min", *leg.EstimatedDurationMinutes))
	}
	if leg.EstimatedCost != nil && leg.EstimatedCost.Amount != nil {
		parts = append(parts, fmt.Sprintf("%.2f %s estimate", *leg.EstimatedCost.Amount, leg.EstimatedCost.Currency))
	}
	if leg.SelectedTransportOption != nil {
		parts = append(parts, firstNonEmpty(leg.SelectedTransportOption.OperatorName, leg.SelectedTransportOption.Provider))
	}
	return strings.Join(nonEmpty(parts), " · ")
}

func transportOptionDescription(option *aggregate.SelectedTransportOption) string {
	if option == nil {
		return ""
	}
	parts := []string{titleCase(option.Mode), option.Provider, option.ServiceName}
	if option.DurationMinutes > 0 {
		parts = append(parts, fmt.Sprintf("%d min", option.DurationMinutes))
	}
	if option.EstimatedPrice != nil {
		parts = append(parts, fmt.Sprintf("%.2f %s", option.EstimatedPrice.Amount, option.EstimatedPrice.Currency))
	}
	return strings.Join(nonEmpty(parts), " · ")
}

func itineraryItemDescription(day int, item aggregate.ItineraryItem, placeName string) string {
	parts := []string{fmt.Sprintf("Day %d", day), item.Time, titleCase(firstNonEmpty(item.Category, item.Type)), placeName}
	if item.EstimatedCost != nil && item.EstimatedCost.Amount != nil {
		parts = append(parts, fmt.Sprintf("%.2f %s estimate", *item.EstimatedCost.Amount, item.EstimatedCost.Currency))
	}
	if item.Note != "" {
		parts = append(parts, shortText(item.Note, 80))
	}
	return strings.Join(nonEmpty(parts), " · ")
}

func tripDescription(startDate pgtype.Date, days int) string {
	parts := []string{}
	if startDate.Valid {
		parts = append(parts, startDate.Time.Format("Jan 2, 2006"))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d days", days))
	}
	return strings.Join(parts, " · ")
}

func dateDescription(label string, date pgtype.Date) string {
	if !date.Valid {
		return ""
	}
	return label + " " + date.Time.Format("Jan 2, 2006")
}

func dateString(date pgtype.Date) string {
	if !date.Valid {
		return ""
	}
	return date.Time.Format("2006-01-02")
}

func dateRange(start, end string) string {
	if start == "" && end == "" {
		return ""
	}
	if start == "" {
		return "Until " + end
	}
	if end == "" {
		return "From " + start
	}
	if start == end {
		return start
	}
	return start + " - " + end
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func titleCase(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return ""
	}
	words := strings.Fields(value)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}

func uuidArg(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return *id
}

func uuidValuePtr(id uuid.UUID) *uuid.UUID {
	value := id
	return &value
}

func uuidPtr(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
