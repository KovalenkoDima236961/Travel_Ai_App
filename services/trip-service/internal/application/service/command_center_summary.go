package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/groupreadiness"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triphealth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/verification"
)

type CommandCenterSummary struct {
	TripID             uuid.UUID                         `json:"tripId"`
	Trip               CommandCenterTripSummary          `json:"trip"`
	Health             *CommandCenterHealthSummary       `json:"health,omitempty"`
	Budget             *CommandCenterBudgetSummary       `json:"budget,omitempty"`
	GroupReadiness     *CommandCenterGroupSummary        `json:"groupReadiness,omitempty"`
	RealWorldReadiness *CommandCenterVerificationSummary `json:"realWorldReadiness,omitempty"`
	Route              CommandCenterRouteSummary         `json:"route"`
	Checklist          *CommandCenterChecklistSummary    `json:"checklist,omitempty"`
	Reminders          *CommandCenterReminderSummary     `json:"reminders,omitempty"`
	Expenses           *CommandCenterExpenseSummary      `json:"expenses,omitempty"`
	Activity           *CommandCenterActivitySummary     `json:"activity,omitempty"`
	SectionErrors      []CommandCenterSectionError       `json:"sectionErrors"`
	ComputedAt         time.Time                         `json:"computedAt"`
}

type CommandCenterTripSummary struct {
	Destination       string     `json:"destination"`
	StartDate         *string    `json:"startDate,omitempty"`
	Days              int32      `json:"days"`
	TripType          string     `json:"tripType"`
	ItineraryRevision int        `json:"itineraryRevision"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	WorkspaceID       *uuid.UUID `json:"workspaceId,omitempty"`
	Travelers         int32      `json:"travelers"`
	BudgetCurrency    string     `json:"budgetCurrency"`
	AccessRole        string     `json:"accessRole"`
	CanEdit           bool       `json:"canEdit"`
}

type CommandCenterTopFix struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Recommendation string              `json:"recommendation,omitempty"`
	Severity       triphealth.Severity `json:"severity"`
	Category       triphealth.Category `json:"category"`
	Label          string              `json:"label"`
	Href           string              `json:"href"`
}

type CommandCenterHealthSummary struct {
	Score              int                   `json:"score"`
	Level              triphealth.Level      `json:"level"`
	Summary            string                `json:"summary"`
	CriticalIssueCount int                   `json:"criticalIssueCount"`
	HighIssueCount     int                   `json:"highIssueCount"`
	WarningIssueCount  int                   `json:"warningIssueCount"`
	TopFixes           []CommandCenterTopFix `json:"topFixes"`
}

type CommandCenterBudgetSummary struct {
	ConfidenceScore int                              `json:"confidenceScore"`
	ConfidenceLevel budgetconfidence.ConfidenceLevel `json:"confidenceLevel"`
	RiskLevel       budgetconfidence.RiskLevel       `json:"riskLevel"`
	Summary         string                           `json:"summary"`
	Coverage        int                              `json:"coverage"`
	Currency        string                           `json:"currency"`
	EstimatedTotal  budgetconfidence.Money           `json:"estimatedTotal"`
	ActualTotal     budgetconfidence.Money           `json:"actualTotal"`
	TripBudget      *budgetconfidence.Money          `json:"tripBudget,omitempty"`
	BudgetExceeded  bool                             `json:"budgetExceeded"`
	MissingCount    int                              `json:"missingEstimateCount"`
}

type CommandCenterGroupSummary struct {
	Score                   int                  `json:"score"`
	Level                   groupreadiness.Level `json:"level"`
	Summary                 string               `json:"summary"`
	MemberCount             int                  `json:"memberCount"`
	MembersNeedingAttention int                  `json:"membersNeedingAttention"`
	TopActionLabel          string               `json:"topActionLabel,omitempty"`
	TopActionHref           string               `json:"topActionHref,omitempty"`
}

// CommandCenterVerificationSummary is deliberately compact. Detailed
// provider data remains private to GET /trips/{id}/verification.
type CommandCenterVerificationSummary struct {
	Score         int                `json:"score"`
	Level         verification.Level `json:"level"`
	TopIssueCount int                `json:"topIssueCount"`
	VerifiedCount int                `json:"verifiedCount"`
	StaleCount    int                `json:"staleCount"`
	MissingCount  int                `json:"missingCount"`
}

type CommandCenterRouteSummary struct {
	StopCount                 int `json:"stopCount"`
	LegCount                  int `json:"legCount"`
	SelectedTransportCoverage int `json:"selectedTransportCoverage"`
	MissingTransportCount     int `json:"missingTransportCount"`
}

type CommandCenterChecklistSummary struct {
	CompletedCount    int `json:"completedCount"`
	TotalCount        int `json:"totalCount"`
	OverdueCount      int `json:"overdueCount"`
	HighPriorityCount int `json:"highPriorityCount"`
}

type CommandCenterReminderSummary struct {
	TotalCount   int `json:"totalCount"`
	OverdueCount int `json:"overdueCount"`
	DueSoonCount int `json:"dueSoonCount"`
}

type CommandCenterExpenseSummary struct {
	ExpenseCount           int                `json:"expenseCount"`
	ActualTotal            appdto.MoneyAmount `json:"actualTotal"`
	PendingSettlementCount int                `json:"pendingSettlementCount"`
}

type CommandCenterActivitySummary struct {
	RecentCount int                 `json:"recentCount"`
	LatestAt    *time.Time          `json:"latestAt,omitempty"`
	Items       []activity.EventDTO `json:"items"`
}

type CommandCenterSectionError struct {
	Section string `json:"section"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *Service) GetCommandCenterSummary(ctx context.Context, tripID uuid.UUID) (CommandCenterSummary, error) {
	started := time.Now()
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return CommandCenterSummary{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return CommandCenterSummary{}, err
	}
	cacheKey := summaryCacheKey("command_center", trip, user.ID, access.Role())
	if cached, ok := s.summaryCache.get("command_center", cacheKey); ok {
		if response, valid := cached.(CommandCenterSummary); valid {
			return response, nil
		}
	}

	response := CommandCenterSummary{
		TripID:        trip.ID,
		Trip:          commandCenterTripSummary(trip, access),
		Route:         commandCenterRouteSummary(trip),
		SectionErrors: []CommandCenterSectionError{},
		ComputedAt:    time.Now().UTC(),
	}
	summaryCtx := ctx
	cancel := func() {}
	if s.summaryEndpointTimeout > 0 {
		summaryCtx, cancel = context.WithTimeout(ctx, s.summaryEndpointTimeout)
	}
	defer cancel()

	var wg sync.WaitGroup
	var errorMu sync.Mutex
	run := func(section string, load func(context.Context) error) {
		execute := func() {
			sectionCtx := summaryCtx
			cancelSection := func() {}
			if s.commandCenterSectionTimeout > 0 {
				sectionCtx, cancelSection = context.WithTimeout(summaryCtx, s.commandCenterSectionTimeout)
			}
			defer cancelSection()
			if loadErr := load(sectionCtx); loadErr != nil {
				errorMu.Lock()
				response.SectionErrors = append(response.SectionErrors, commandCenterSectionError(section, loadErr))
				errorMu.Unlock()
			}
		}
		if !s.commandCenterParallel {
			execute()
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			execute()
		}()
	}

	run("health", func(runCtx context.Context) error {
		health, loadErr := s.GetTripHealth(runCtx, tripID, triphealth.Options{})
		if loadErr != nil {
			return loadErr
		}
		response.Health = commandCenterHealthSummary(health)
		return nil
	})
	run("budget", func(runCtx context.Context) error {
		confidence, loadErr := s.GetBudgetConfidence(runCtx, tripID, budgetconfidence.Options{Currency: trip.BudgetCurrency})
		if loadErr != nil {
			return loadErr
		}
		response.Budget = commandCenterBudgetSummary(confidence)
		return nil
	})
	run("groupReadiness", func(runCtx context.Context) error {
		readiness, loadErr := s.GetGroupReadiness(runCtx, tripID, groupreadiness.Options{IncludeDetails: true})
		if loadErr != nil {
			return loadErr
		}
		response.GroupReadiness = commandCenterGroupReadinessSummary(readiness)
		return nil
	})
	run("verification", func(runCtx context.Context) error {
		readiness, loadErr := s.GetTripVerification(runCtx, tripID)
		if loadErr != nil {
			return loadErr
		}
		response.RealWorldReadiness = commandCenterVerificationSummary(readiness)
		return nil
	})
	run("checklist", func(runCtx context.Context) error {
		checklist, loadErr := s.GetTripChecklist(runCtx, tripID)
		if loadErr != nil {
			return loadErr
		}
		response.Checklist = commandCenterChecklistSummary(checklist)
		return nil
	})
	run("reminders", func(runCtx context.Context) error {
		reminders, loadErr := s.ListTripReminders(runCtx, tripID, appdto.ReminderListFilters{})
		if loadErr != nil {
			return loadErr
		}
		response.Reminders = &CommandCenterReminderSummary{
			TotalCount:   reminders.Summary.Total,
			OverdueCount: reminders.Summary.Overdue,
			DueSoonCount: reminders.Summary.DueToday,
		}
		return nil
	})
	run("expenses", func(runCtx context.Context) error {
		expenses, loadErr := s.GetTripExpenseSummary(runCtx, tripID, trip.BudgetCurrency)
		if loadErr != nil {
			return loadErr
		}
		response.Expenses = &CommandCenterExpenseSummary{
			ExpenseCount:           expenses.ExpenseCount,
			ActualTotal:            expenses.ActualTotal,
			PendingSettlementCount: expenses.SettlementSummary.PendingCount,
		}
		return nil
	})
	run("activity", func(runCtx context.Context) error {
		events, loadErr := s.ListActivity(runCtx, tripID, 5, "")
		if loadErr != nil {
			return loadErr
		}
		items := make([]activity.EventDTO, 0, len(events.Events))
		var latestAt *time.Time
		for i := range events.Events {
			items = append(items, activity.NewEventDTO(events.Events[i]))
			if latestAt == nil || events.Events[i].CreatedAt.After(*latestAt) {
				value := events.Events[i].CreatedAt
				latestAt = &value
			}
		}
		response.Activity = &CommandCenterActivitySummary{RecentCount: len(items), LatestAt: latestAt, Items: items}
		return nil
	})

	wg.Wait()
	tripobs.RecordSummaryCompute("command_center", time.Since(started))
	s.summaryCache.set("command_center", cacheKey, response)
	return response, nil
}

func commandCenterTripSummary(trip *entity.Trip, access TripAccess) CommandCenterTripSummary {
	var startDate *string
	if trip.StartDate != nil {
		value := trip.StartDate.Format("2006-01-02")
		startDate = &value
	}
	return CommandCenterTripSummary{
		Destination:       trip.Destination,
		StartDate:         startDate,
		Days:              trip.Days,
		TripType:          normalizeTripType(trip.TripType, trip.Route),
		ItineraryRevision: trip.ItineraryRevision,
		UpdatedAt:         trip.UpdatedAt,
		WorkspaceID:       trip.WorkspaceID,
		Travelers:         trip.Travelers,
		BudgetCurrency:    trip.BudgetCurrency,
		AccessRole:        access.Role(),
		CanEdit:           access.CanEdit(),
	}
}

func commandCenterRouteSummary(trip *entity.Trip) CommandCenterRouteSummary {
	if trip.Route == nil {
		return CommandCenterRouteSummary{}
	}
	selected := 0
	for _, leg := range trip.Route.Legs {
		if leg.SelectedTransportOption != nil {
			selected++
		}
	}
	coverage := 0
	if len(trip.Route.Legs) > 0 {
		coverage = int(float64(selected) / float64(len(trip.Route.Legs)) * 100)
	}
	return CommandCenterRouteSummary{
		StopCount:                 len(trip.Route.Stops),
		LegCount:                  len(trip.Route.Legs),
		SelectedTransportCoverage: coverage,
		MissingTransportCount:     len(trip.Route.Legs) - selected,
	}
}

func commandCenterHealthSummary(health triphealth.Response) *CommandCenterHealthSummary {
	out := &CommandCenterHealthSummary{
		Score:    health.Score,
		Level:    health.Level,
		Summary:  health.Summary,
		TopFixes: []CommandCenterTopFix{},
	}
	issueByID := make(map[string]triphealth.Issue, len(health.Issues))
	for _, issue := range health.Issues {
		issueByID[issue.ID] = issue
		if issue.Status != triphealth.StatusOpen {
			continue
		}
		switch issue.Severity {
		case triphealth.SeverityCritical:
			out.CriticalIssueCount++
		case triphealth.SeverityHigh:
			out.HighIssueCount++
		case triphealth.SeverityWarning:
			out.WarningIssueCount++
		}
	}
	for _, fix := range health.TopFixes {
		issue := issueByID[fix.IssueID]
		out.TopFixes = append(out.TopFixes, CommandCenterTopFix{
			ID:             fix.IssueID,
			Title:          firstNonEmptySummary(issue.Title, fix.Label),
			Description:    issue.Description,
			Recommendation: issue.Recommendation,
			Severity:       issue.Severity,
			Category:       issue.Category,
			Label:          fix.Label,
			Href:           fix.Href,
		})
		if len(out.TopFixes) == 5 {
			break
		}
	}
	return out
}

func commandCenterBudgetSummary(confidence budgetconfidence.Response) *CommandCenterBudgetSummary {
	exceeded := confidence.TripBudget != nil && confidence.ActualTotal.Amount > confidence.TripBudget.Amount
	missing := 0
	for _, source := range confidence.SourceQuality {
		if source.Source == budgetconfidence.SourceMissingCost {
			missing += source.ItemCount
		}
	}
	return &CommandCenterBudgetSummary{
		ConfidenceScore: confidence.Score,
		ConfidenceLevel: confidence.Level,
		RiskLevel:       confidence.RiskLevel,
		Summary:         confidence.Summary,
		Coverage:        confidence.Coverage.Overall,
		Currency:        confidence.Currency,
		EstimatedTotal:  confidence.EstimatedTotal,
		ActualTotal:     confidence.ActualTotal,
		TripBudget:      confidence.TripBudget,
		BudgetExceeded:  exceeded,
		MissingCount:    missing,
	}
}

func commandCenterGroupReadinessSummary(readiness groupreadiness.Response) *CommandCenterGroupSummary {
	attention := 0
	for _, member := range readiness.Members {
		if member.Level != groupreadiness.LevelReady {
			attention++
		}
	}
	out := &CommandCenterGroupSummary{
		Score:                   readiness.Score,
		Level:                   readiness.Level,
		Summary:                 readiness.Summary,
		MemberCount:             len(readiness.Members),
		MembersNeedingAttention: attention,
	}
	if len(readiness.TopActions) > 0 {
		out.TopActionLabel = readiness.TopActions[0].Label
		out.TopActionHref = readiness.TopActions[0].Href
	}
	return out
}

func commandCenterVerificationSummary(readiness verification.Response) *CommandCenterVerificationSummary {
	return &CommandCenterVerificationSummary{
		Score:         readiness.Score,
		Level:         readiness.Level,
		TopIssueCount: len(readiness.TopIssues),
		VerifiedCount: readiness.Summary.VerifiedCount,
		StaleCount:    readiness.Summary.StaleCount,
		MissingCount:  readiness.Summary.MissingCount,
	}
}

func commandCenterChecklistSummary(checklist *appdto.ChecklistViewResponse) *CommandCenterChecklistSummary {
	if checklist == nil || checklist.Summary == nil {
		return &CommandCenterChecklistSummary{}
	}
	overdue := 0
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if checklist.Checklist != nil {
		for _, item := range checklist.Checklist.Items {
			if item.Checked || item.DueDate == nil {
				continue
			}
			if due, err := time.Parse("2006-01-02", *item.DueDate); err == nil && due.Before(today) {
				overdue++
			}
		}
	}
	return &CommandCenterChecklistSummary{
		CompletedCount:    checklist.Summary.CheckedItems,
		TotalCount:        checklist.Summary.TotalItems,
		OverdueCount:      overdue,
		HighPriorityCount: checklist.Summary.HighPriorityUnchecked,
	}
}

func commandCenterSectionError(section string, err error) CommandCenterSectionError {
	code := strings.ToLower(strings.ReplaceAll(section, "Readiness", "_readiness")) + "_summary_unavailable"
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		code = strings.ToLower(section) + "_summary_timeout"
	}
	return CommandCenterSectionError{
		Section: section,
		Code:    code,
		Message: fmt.Sprintf("%s summary is temporarily unavailable.", section),
	}
}

func firstNonEmptySummary(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Review trip readiness"
}
