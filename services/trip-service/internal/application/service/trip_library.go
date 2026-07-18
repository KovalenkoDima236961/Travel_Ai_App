package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
)

const (
	defaultLibraryLimit = 30
	maxLibraryLimit     = 100
	maxArchiveReason    = 500
)

type tripArchiveRepository interface {
	ArchiveTrip(context.Context, uuid.UUID, uuid.UUID, string) (*entity.Trip, error)
	RestoreTrip(context.Context, uuid.UUID) (*entity.Trip, error)
}

type tripLibraryRepository interface {
	ListAccessibleForLibrary(context.Context, uuid.UUID, []uuid.UUID, *uuid.UUID) ([]entity.Trip, error)
	GetTripLibrarySummaries(context.Context, []uuid.UUID) (map[uuid.UUID]appdto.TripLibrarySummary, error)
}

// ArchiveTrip hides a trip from the active list without deleting or changing
// any of its planning, collaboration, recap, expense, or sharing data.
func (s *Service) ArchiveTrip(ctx context.Context, tripID uuid.UUID, input appdto.ArchiveTripInput) (appdto.ArchiveTripResult, error) {
	if !s.tripLibraryEnabled {
		return appdto.ArchiveTripResult{}, apperrs.NewDependencyError("trip library is disabled")
	}
	if len(strings.TrimSpace(input.Reason)) > maxArchiveReason {
		return appdto.ArchiveTripResult{}, apperrs.NewInvalidInput("archive reason must be at most %d characters", maxArchiveReason)
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ArchiveTripResult{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ArchiveTripResult{}, err
	}
	if !s.canManageTripArchive(ctx, trip, access, user.ID) {
		return appdto.ArchiveTripResult{}, apperrs.ErrForbidden
	}
	repository, ok := s.repo.(tripArchiveRepository)
	if !ok {
		return appdto.ArchiveTripResult{}, apperrs.NewDependencyError("trip archive storage is not configured")
	}
	archived, err := repository.ArchiveTrip(ctx, tripID, user.ID, strings.TrimSpace(input.Reason))
	if err != nil {
		tripobs.RecordTripArchive("error", tripWorkspaceScope(trip))
		return appdto.ArchiveTripResult{}, err
	}
	reason := archiveActivityReason(input.Reason)
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventTripArchived,
		EntityType: activityEntityType(activity.EntityTrip), EntityID: activityEntityID(tripID),
		Metadata: map[string]any{"reason": reason, "tripTitle": trip.Destination},
	})
	s.log.Info("trip archived", zap.String("trip_id", tripID.String()), zap.String("user_id", user.ID.String()), zap.String("scope", tripWorkspaceScope(trip)))
	s.summaryCache.clear("library_insights")
	tripobs.RecordTripArchive("success", tripWorkspaceScope(trip))
	return appdto.ArchiveTripResult{TripID: archived.ID, ArchivedAt: archived.ArchivedAt, Lifecycle: s.deriveLifecycle(archived)}, nil
}

func (s *Service) RestoreTrip(ctx context.Context, tripID uuid.UUID) (appdto.ArchiveTripResult, error) {
	if !s.tripLibraryEnabled {
		return appdto.ArchiveTripResult{}, apperrs.NewDependencyError("trip library is disabled")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ArchiveTripResult{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ArchiveTripResult{}, err
	}
	if !s.canManageTripArchive(ctx, trip, access, user.ID) {
		return appdto.ArchiveTripResult{}, apperrs.ErrForbidden
	}
	repository, ok := s.repo.(tripArchiveRepository)
	if !ok {
		return appdto.ArchiveTripResult{}, apperrs.NewDependencyError("trip archive storage is not configured")
	}
	restored, err := repository.RestoreTrip(ctx, tripID)
	if err != nil {
		tripobs.RecordTripRestore("error", tripWorkspaceScope(trip))
		return appdto.ArchiveTripResult{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventTripRestored,
		EntityType: activityEntityType(activity.EntityTrip), EntityID: activityEntityID(tripID),
		Metadata: map[string]any{"tripTitle": trip.Destination},
	})
	s.log.Info("trip restored", zap.String("trip_id", tripID.String()), zap.String("user_id", user.ID.String()), zap.String("scope", tripWorkspaceScope(trip)))
	s.summaryCache.clear("library_insights")
	tripobs.RecordTripRestore("success", tripWorkspaceScope(trip))
	return appdto.ArchiveTripResult{TripID: restored.ID, ArchivedAt: nil, Lifecycle: s.deriveLifecycle(restored)}, nil
}

func (s *Service) canManageTripArchive(ctx context.Context, trip *entity.Trip, access TripAccess, userID uuid.UUID) bool {
	if trip == nil {
		return false
	}
	if trip.UserID != nil && *trip.UserID == userID {
		return true
	}
	if trip.WorkspaceID == nil {
		return false
	}
	return access.Source == "workspace" && (access.WorkspaceRole == "owner" || access.WorkspaceRole == "admin")
}

// GetTripLibrary returns compact, private history only. It intentionally keeps
// full itineraries and sensitive subordinate resources out of the response.
func (s *Service) GetTripLibrary(ctx context.Context, filters appdto.TripLibraryFilters) (appdto.TripLibraryResult, error) {
	started := time.Now()
	result, err := s.getTripLibrary(ctx, filters)
	status := "success"
	if err != nil {
		status = "error"
	}
	tripobs.RecordTripLibraryRead(status, time.Since(started), result.Total)
	return result, err
}

func (s *Service) getTripLibrary(ctx context.Context, filters appdto.TripLibraryFilters) (appdto.TripLibraryResult, error) {
	if !s.tripLibraryEnabled {
		return appdto.TripLibraryResult{}, apperrs.NewDependencyError("trip library is disabled")
	}
	normalized, err := normalizeTripLibraryFilters(filters)
	if err != nil {
		return appdto.TripLibraryResult{}, err
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripLibraryResult{}, err
	}
	workspaceIDs, err := s.accessibleWorkspaceIDs(ctx, user.ID)
	if err != nil {
		return appdto.TripLibraryResult{}, err
	}
	repository, ok := s.repo.(tripLibraryRepository)
	var trips []entity.Trip
	if ok {
		trips, err = repository.ListAccessibleForLibrary(ctx, user.ID, workspaceIDs, normalized.WorkspaceID)
	} else {
		trips, err = s.repo.ListAccessible(ctx, user.ID, workspaceIDs, appdto.TripListScopeAll, normalized.WorkspaceID, maxLimit, 0)
	}
	if err != nil {
		return appdto.TripLibraryResult{}, err
	}
	tripIDs := make([]uuid.UUID, 0, len(trips))
	for i := range trips {
		tripIDs = append(tripIDs, trips[i].ID)
	}
	summaries := map[uuid.UUID]appdto.TripLibrarySummary{}
	if ok {
		summaries, err = repository.GetTripLibrarySummaries(ctx, tripIDs)
		if err != nil {
			return appdto.TripLibraryResult{}, err
		}
	}

	workspaceRoles := s.libraryWorkspaceRoles(ctx, user.ID)
	items := make([]appdto.TripLibraryItem, 0, len(trips))
	availableYears := map[int]struct{}{}
	destinations := map[string]struct{}{}
	for i := range trips {
		trip := &trips[i]
		summary := summaries[trip.ID]
		item := s.newTripLibraryItem(trip, summary)
		item.Actions = libraryActions(item, libraryArchiveAllowed(trip, user.ID, workspaceRoles))
		if !tripLibraryMatches(item, normalized) {
			continue
		}
		items = append(items, item)
		if trip.StartDate != nil {
			availableYears[trip.StartDate.UTC().Year()] = struct{}{}
		}
		if strings.TrimSpace(trip.Destination) != "" {
			destinations[trip.Destination] = struct{}{}
		}
	}
	sortTripLibraryItems(items, normalized.Sort)
	result := summarizeTripLibrary(items, availableYears, destinations)
	result.Total = len(items)
	start, err := decodeLibraryCursor(normalized.Cursor)
	if err != nil {
		return appdto.TripLibraryResult{}, apperrs.NewInvalidInput("invalid library cursor")
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + normalized.Limit
	if end > len(items) {
		end = len(items)
	}
	result.Items = items[start:end]
	if end < len(items) {
		result.NextCursor = encodeLibraryCursor(end)
	}
	s.log.Info("trip library queried", zap.String("user_id", user.ID.String()), zap.Int("query_length", len(normalized.Query)), zap.String("scope", libraryScope(normalized.WorkspaceID)), zap.Int("result_count", len(items)))
	return result, nil
}

func (s *Service) GetTripLibraryInsights(ctx context.Context, workspaceID *uuid.UUID, year *int) (appdto.TripLibraryInsights, error) {
	started := time.Now()
	if !s.tripLibraryEnabled {
		return appdto.TripLibraryInsights{}, apperrs.NewDependencyError("trip library is disabled")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripLibraryInsights{}, err
	}
	workspaceScope := "all"
	if workspaceID != nil {
		workspaceScope = workspaceID.String()
	}
	yearScope := "all"
	if year != nil {
		yearScope = strconv.Itoa(*year)
	}
	cacheKey := summaryCacheKey("library_insights", nil, user.ID, workspaceScope, yearScope)
	if cached, ok := s.summaryCache.get("library_insights", cacheKey); ok {
		if result, valid := cached.(appdto.TripLibraryInsights); valid {
			tripobs.RecordTripLibraryInsights(libraryScope(workspaceID))
			return result, nil
		}
	}
	filters := appdto.TripLibraryFilters{WorkspaceID: workspaceID, Year: year, Limit: maxLibraryLimit, Sort: appdto.TripLibrarySortRecentlyUpdated}
	result, err := s.GetTripLibrary(ctx, filters)
	if err != nil {
		return appdto.TripLibraryInsights{}, err
	}
	insights := buildTripLibraryInsights(result.Items)
	tripobs.RecordSummaryCompute("library_insights", time.Since(started))
	s.summaryCache.setWithTTL("library_insights", cacheKey, insights, s.libraryInsightsCacheTTL)
	tripobs.RecordTripLibraryInsights(libraryScope(workspaceID))
	return insights, nil
}

func (s *Service) deriveLifecycle(trip *entity.Trip) entity.TripLifecycle {
	return entity.DeriveLifecycle(trip, entity.LifecycleOptions{Now: time.Now().UTC(), ReadyHealthScoreThreshold: s.readyHealthScoreThreshold, ReadyVerificationThreshold: s.readyVerificationScoreThreshold})
}

func (s *Service) newTripLibraryItem(trip *entity.Trip, summary appdto.TripLibrarySummary) appdto.TripLibraryItem {
	lifecycle := s.deriveLifecycle(trip)
	item := appdto.TripLibraryItem{Trip: trip, Lifecycle: lifecycle, Recap: appdto.TripLibraryRecap{HasRecap: summary.RecapStatus != nil, CreatedAt: summary.RecapCreatedAt}, Template: appdto.TripLibraryTemplate{HasTemplate: summary.TemplateID != nil, TemplateID: summary.TemplateID}, Route: libraryRoute(trip.Route)}
	if summary.RecapStatus != nil {
		item.Recap.Status = string(*summary.RecapStatus)
	}
	item.Completion = appdto.TripLibraryCompletion{PlannedItemCount: summary.PlannedCount, DoneItemCount: summary.DoneCount}
	if summary.PlannedCount > 0 {
		item.Completion.CompletionRate = float64(summary.DoneCount) / float64(summary.PlannedCount)
	}
	item.Budget = libraryBudget(trip, summary.ExpenseTotals)
	item.HasExpenses = summary.HasExpenses
	item.InsightLessons = summary.Lessons
	item.InsightMissedItems = summary.MissedItems
	return item
}

func normalizeTripLibraryFilters(in appdto.TripLibraryFilters) (appdto.TripLibraryFilters, error) {
	in.Query = strings.TrimSpace(in.Query)
	if len(in.Query) > 200 {
		return in, apperrs.NewInvalidInput("library search query must be at most 200 characters")
	}
	in.Lifecycle = strings.ToLower(strings.TrimSpace(in.Lifecycle))
	if in.Lifecycle == "" {
		in.Lifecycle = "all"
	}
	if in.Lifecycle != "all" {
		for _, lifecycle := range strings.Split(in.Lifecycle, ",") {
			if !isTripLifecycle(strings.TrimSpace(lifecycle)) {
				return in, apperrs.NewInvalidInput("invalid lifecycle")
			}
		}
	}
	in.Destination, in.Country = strings.TrimSpace(in.Destination), strings.TrimSpace(in.Country)
	in.TripType, in.TravelStyle, in.TransportMode = strings.TrimSpace(in.TripType), strings.TrimSpace(in.TravelStyle), strings.TrimSpace(in.TransportMode)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Currency != "" && len(in.Currency) != 3 {
		return in, apperrs.NewInvalidInput("currency must be a 3-letter code")
	}
	if in.Year != nil && (*in.Year < 1900 || *in.Year > 2200) {
		return in, apperrs.NewInvalidInput("year must be between 1900 and 2200")
	}
	if in.BudgetMin != nil && *in.BudgetMin < 0 || in.BudgetMax != nil && *in.BudgetMax < 0 {
		return in, apperrs.NewInvalidInput("budget filters must be non-negative")
	}
	if in.BudgetMin != nil && in.BudgetMax != nil && *in.BudgetMin > *in.BudgetMax {
		return in, apperrs.NewInvalidInput("budgetMin cannot exceed budgetMax")
	}
	if in.Limit == 0 {
		in.Limit = defaultLibraryLimit
	}
	if in.Limit < 1 || in.Limit > maxLibraryLimit {
		return in, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLibraryLimit)
	}
	if in.Sort == "" {
		in.Sort = appdto.TripLibrarySortRecentlyUpdated
	}
	if !isLibrarySort(in.Sort) {
		return in, apperrs.NewInvalidInput("invalid library sort")
	}
	return in, nil
}

func isTripLifecycle(value string) bool {
	switch entity.TripLifecycle(value) {
	case entity.TripLifecycleDraft, entity.TripLifecyclePlanning, entity.TripLifecycleReady, entity.TripLifecycleActive, entity.TripLifecycleCompleted, entity.TripLifecycleArchived:
		return true
	}
	return false
}
func isLibrarySort(value appdto.TripLibrarySort) bool {
	switch value {
	case appdto.TripLibrarySortRecentlyUpdated, appdto.TripLibrarySortTripDateDesc, appdto.TripLibrarySortTripDateAsc, appdto.TripLibrarySortDestination, appdto.TripLibrarySortBudgetDesc, appdto.TripLibrarySortBudgetAsc, appdto.TripLibrarySortCompletionRateDesc, appdto.TripLibrarySortRecapCreatedDesc:
		return true
	}
	return false
}

func tripLibraryMatches(item appdto.TripLibraryItem, filter appdto.TripLibraryFilters) bool {
	trip := item.Trip
	if filter.Lifecycle != "all" && !libraryLifecycleMatches(item.Lifecycle, filter.Lifecycle) {
		return false
	}
	if filter.Archived != nil && (item.Lifecycle == entity.TripLifecycleArchived) != *filter.Archived {
		return false
	}
	if filter.Query != "" && !strings.Contains(strings.ToLower(trip.Destination), strings.ToLower(filter.Query)) {
		return false
	}
	if filter.Year != nil && (trip.StartDate == nil || trip.StartDate.UTC().Year() != *filter.Year) {
		return false
	}
	if filter.Destination != "" && !strings.EqualFold(trip.Destination, filter.Destination) {
		return false
	}
	if filter.Country != "" && !tripHasCountry(trip, filter.Country) {
		return false
	}
	if filter.TripType != "" && trip.TripType != filter.TripType {
		return false
	}
	if filter.TravelStyle != "" && !tripHasStyle(trip, filter.TravelStyle) {
		return false
	}
	if filter.TransportMode != "" && !tripHasTransportMode(trip, filter.TransportMode) {
		return false
	}
	if filter.Currency != "" && !strings.EqualFold(trip.BudgetCurrency, filter.Currency) {
		return false
	}
	if filter.BudgetMin != nil && (trip.BudgetAmount == nil || *trip.BudgetAmount < *filter.BudgetMin) {
		return false
	}
	if filter.BudgetMax != nil && (trip.BudgetAmount == nil || *trip.BudgetAmount > *filter.BudgetMax) {
		return false
	}
	if filter.HasRecap != nil && item.Recap.HasRecap != *filter.HasRecap {
		return false
	}
	if filter.HasTemplate != nil && item.Template.HasTemplate != *filter.HasTemplate {
		return false
	}
	if filter.HasExpenses != nil && item.HasExpenses != *filter.HasExpenses {
		return false
	}
	return true
}

func libraryLifecycleMatches(lifecycle entity.TripLifecycle, filter string) bool {
	for _, value := range strings.Split(filter, ",") {
		if string(lifecycle) == strings.TrimSpace(value) {
			return true
		}
	}
	return false
}

func libraryRoute(route *aggregate.TripRoute) appdto.TripLibraryRoute {
	result := appdto.TripLibraryRoute{TransportModes: []string{}}
	if route == nil {
		return result
	}
	result.StopCount = len(route.Stops)
	seen := map[string]struct{}{}
	for _, leg := range route.Legs {
		mode := strings.TrimSpace(leg.Mode)
		if mode != "" {
			if _, exists := seen[mode]; !exists {
				seen[mode] = struct{}{}
				result.TransportModes = append(result.TransportModes, mode)
			}
		}
	}
	sort.Strings(result.TransportModes)
	return result
}

func libraryBudget(trip *entity.Trip, totals []appdto.LibraryMoney) appdto.TripLibraryBudget {
	result := appdto.TripLibraryBudget{}
	if trip.BudgetAmount != nil && trip.BudgetCurrency != "" {
		result.PlannedTotal = &appdto.LibraryMoney{Amount: *trip.BudgetAmount, Currency: trip.BudgetCurrency}
	}
	if len(totals) == 1 {
		value := totals[0]
		result.ActualTotal = &value
	} else if len(totals) > 1 {
		result.MixedCurrencies = true
	}
	if result.PlannedTotal != nil && result.ActualTotal != nil && strings.EqualFold(result.PlannedTotal.Currency, result.ActualTotal.Currency) {
		result.Variance = &appdto.LibraryMoney{Amount: result.ActualTotal.Amount - result.PlannedTotal.Amount, Currency: result.PlannedTotal.Currency}
	}
	return result
}

func libraryActions(item appdto.TripLibraryItem, canArchive bool) []string {
	actions := []string{"view_trip", "duplicate_trip", "plan_similar", "adapt_trip", "compare_budget", "view_expenses"}
	if item.Recap.HasRecap {
		actions = append(actions, "view_recap")
	} else {
		actions = append(actions, "create_recap")
	}
	if !item.Template.HasTemplate {
		actions = append(actions, "create_template")
	}
	if !canArchive {
		return actions
	}
	if item.Lifecycle == entity.TripLifecycleArchived {
		return append(actions, "restore_trip")
	}
	return append(actions, "archive_trip")
}

func (s *Service) libraryWorkspaceRoles(ctx context.Context, userID uuid.UUID) map[uuid.UUID]string {
	roles := map[uuid.UUID]string{}
	if !s.workspacesEnabled || s.workspaceProvider == nil {
		return roles
	}
	rows, err := s.workspaceProvider.ListForUser(ctx, userID)
	if err != nil {
		// Access filtering already succeeded. Fail closed for optional archive
		// actions if role information cannot be loaded.
		s.log.Warn("could not resolve workspace roles for library actions", zap.Error(err))
		return roles
	}
	for _, row := range rows {
		roles[row.ID] = string(row.Role)
	}
	return roles
}

func libraryArchiveAllowed(trip *entity.Trip, userID uuid.UUID, workspaceRoles map[uuid.UUID]string) bool {
	if trip == nil {
		return false
	}
	if trip.UserID != nil && *trip.UserID == userID {
		return true
	}
	if trip.WorkspaceID == nil {
		return false
	}
	role := workspaceRoles[*trip.WorkspaceID]
	return role == "owner" || role == "admin"
}

func tripHasCountry(trip *entity.Trip, country string) bool {
	if trip.Route == nil {
		return false
	}
	for _, stop := range trip.Route.Stops {
		if strings.EqualFold(strings.TrimSpace(stop.Country), country) {
			return true
		}
	}
	return false
}
func tripHasStyle(trip *entity.Trip, style string) bool {
	for _, interest := range trip.Interests {
		if strings.EqualFold(interest, style) {
			return true
		}
	}
	if trip.Route != nil {
		for _, value := range trip.Route.Preferences.TripStyles {
			if strings.EqualFold(value, style) {
				return true
			}
		}
	}
	return false
}
func tripHasTransportMode(trip *entity.Trip, mode string) bool {
	for _, value := range libraryRoute(trip.Route).TransportModes {
		if strings.EqualFold(value, mode) {
			return true
		}
	}
	return false
}

func sortTripLibraryItems(items []appdto.TripLibraryItem, sortBy appdto.TripLibrarySort) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		switch sortBy {
		case appdto.TripLibrarySortTripDateAsc:
			return libraryDate(left.Trip).Before(libraryDate(right.Trip))
		case appdto.TripLibrarySortTripDateDesc:
			return libraryDate(left.Trip).After(libraryDate(right.Trip))
		case appdto.TripLibrarySortDestination:
			return strings.ToLower(left.Trip.Destination) < strings.ToLower(right.Trip.Destination)
		case appdto.TripLibrarySortBudgetDesc:
			return libraryBudgetAmount(left.Trip) > libraryBudgetAmount(right.Trip)
		case appdto.TripLibrarySortBudgetAsc:
			return libraryBudgetAmount(left.Trip) < libraryBudgetAmount(right.Trip)
		case appdto.TripLibrarySortCompletionRateDesc:
			return left.Completion.CompletionRate > right.Completion.CompletionRate
		case appdto.TripLibrarySortRecapCreatedDesc:
			return libraryRecapDate(left).After(libraryRecapDate(right))
		default:
			return left.Trip.UpdatedAt.After(right.Trip.UpdatedAt)
		}
	})
}

func libraryDate(trip *entity.Trip) time.Time {
	if trip.StartDate == nil {
		return time.Time{}
	}
	return trip.StartDate.UTC()
}
func libraryBudgetAmount(trip *entity.Trip) float64 {
	if trip.BudgetAmount == nil {
		return math.Inf(-1)
	}
	return *trip.BudgetAmount
}
func libraryRecapDate(item appdto.TripLibraryItem) time.Time {
	if item.Recap.CreatedAt == nil {
		return time.Time{}
	}
	return item.Recap.CreatedAt.UTC()
}

func summarizeTripLibrary(items []appdto.TripLibraryItem, years map[int]struct{}, destinations map[string]struct{}) appdto.TripLibraryResult {
	result := appdto.TripLibraryResult{Items: []appdto.TripLibraryItem{}, AvailableYears: sortedLibraryYears(years), Destinations: sortedLibraryStrings(destinations)}
	for _, item := range items {
		if item.Lifecycle == entity.TripLifecycleCompleted {
			result.Completed++
		}
		if item.Lifecycle == entity.TripLifecycleArchived {
			result.Archived++
		}
		if item.Recap.HasRecap {
			result.WithRecaps++
		}
		if item.Template.HasTemplate {
			result.WithTemplates++
		}
	}
	return result
}

func sortedLibraryYears(values map[int]struct{}) []int {
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(result)))
	return result
}
func sortedLibraryStrings(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
func encodeLibraryCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}
func decodeLibraryCursor(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, err
	}
	offset, err := strconv.Atoi(string(decoded))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return offset, nil
}
func archiveActivityReason(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "completed", "abandoned", "duplicate":
		return strings.ToLower(strings.TrimSpace(reason))
	default:
		return "other"
	}
}
func tripWorkspaceScope(trip *entity.Trip) string {
	if trip != nil && trip.WorkspaceID != nil {
		return "workspace"
	}
	return "personal"
}
func libraryScope(workspaceID *uuid.UUID) string {
	if workspaceID != nil {
		return "workspace"
	}
	return "all"
}

func buildTripLibraryInsights(items []appdto.TripLibraryItem) appdto.TripLibraryInsights {
	result := appdto.TripLibraryInsights{}
	destinations, countries, modes, styles, missed := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	lessons := map[string]int{}
	planned, actual := []appdto.LibraryMoney{}, []appdto.LibraryMoney{}
	for _, item := range items {
		result.TripCount++
		result.TotalTravelDays += int(item.Trip.Days)
		if item.Lifecycle == entity.TripLifecycleCompleted {
			result.CompletedTripCount++
		}
		if item.Lifecycle == entity.TripLifecycleArchived {
			result.ArchivedTripCount++
		}
		if item.Recap.HasRecap {
			result.TripRecapCount++
		}
		if item.Template.HasTemplate {
			result.TemplatesCreated++
		}
		destinations[item.Trip.Destination]++
		for _, country := range tripCountries(item.Trip) {
			countries[country]++
		}
		for _, mode := range item.Route.TransportModes {
			modes[mode]++
		}
		for _, style := range tripStyles(item.Trip) {
			styles[style]++
		}
		for _, label := range item.InsightMissedItems {
			if label = strings.TrimSpace(label); label != "" {
				missed[label]++
			}
		}
		for _, lesson := range item.InsightLessons {
			if lesson = strings.TrimSpace(lesson); lesson != "" {
				lessons[lesson]++
			}
		}
		if item.Budget.PlannedTotal != nil {
			planned = append(planned, *item.Budget.PlannedTotal)
		}
		if item.Budget.ActualTotal != nil {
			actual = append(actual, *item.Budget.ActualTotal)
		}
		if item.Budget.Variance != nil {
			if item.Budget.Variance.Amount <= 0 {
				result.UnderBudgetTrips++
			} else {
				result.OverBudgetTrips++
			}
		}
	}
	result.CountriesVisited = len(countries)
	result.TopDestinations = topLibraryCounts(destinations, 5)
	result.TopCountries = topLibraryCounts(countries, 5)
	result.TransportModes = topLibraryCounts(modes, 5)
	result.TravelStyles = topLibraryCounts(styles, 5)
	result.CommonlyMissed = topLibraryCounts(missed, 5)
	result.AverageTripBudget, result.MixedCurrencies = averageLibraryMoney(planned)
	averageActual, actualMixed := averageLibraryMoney(actual)
	result.AverageActualSpend = averageActual
	result.MixedCurrencies = result.MixedCurrencies || actualMixed
	for _, item := range topLibraryCounts(lessons, 5) {
		result.CommonLessons = append(result.CommonLessons, item.Label)
	}
	return result
}

func tripCountries(trip *entity.Trip) []string {
	values := map[string]struct{}{}
	if trip.Route != nil {
		for _, stop := range trip.Route.Stops {
			if value := strings.TrimSpace(stop.Country); value != "" {
				values[value] = struct{}{}
			}
		}
	}
	return sortedLibraryStrings(values)
}
func tripStyles(trip *entity.Trip) []string {
	values := map[string]struct{}{}
	for _, style := range trip.Interests {
		if value := strings.TrimSpace(style); value != "" {
			values[value] = struct{}{}
		}
	}
	if trip.Route != nil {
		for _, style := range trip.Route.Preferences.TripStyles {
			if value := strings.TrimSpace(style); value != "" {
				values[value] = struct{}{}
			}
		}
	}
	return sortedLibraryStrings(values)
}
func topLibraryCounts(values map[string]int, limit int) []appdto.TripLibraryCount {
	result := make([]appdto.TripLibraryCount, 0, len(values))
	for label, count := range values {
		if label != "" && count > 0 {
			result = append(result, appdto.TripLibraryCount{Label: label, Count: count})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Label < result[j].Label
		}
		return result[i].Count > result[j].Count
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}
func averageLibraryMoney(values []appdto.LibraryMoney) (*appdto.LibraryMoney, bool) {
	if len(values) == 0 {
		return nil, false
	}
	currency := values[0].Currency
	sum := 0.0
	for _, value := range values {
		if !strings.EqualFold(value.Currency, currency) {
			return nil, true
		}
		sum += value.Amount
	}
	return &appdto.LibraryMoney{Amount: sum / float64(len(values)), Currency: currency}, false
}
