package groupreadiness

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const (
	weightAvailability = 20
	weightPolls        = 20
	weightChecklist    = 20
	weightReminders    = 15
	weightNoOverdue    = 10
	weightSettlements  = 10
	weightActivity     = 5
)

type memberEvaluation struct {
	member             Member
	scoreInputs        map[string]scoreInput
	items              []Item
	completed          []CompletedItem
	nextAction         *Action
	categoryApplicable map[Category]bool
	categoryComplete   map[Category]bool
}

type scoreInput struct {
	weight     int
	applicable bool
	ratio      float64
}

// Evaluate computes explainable readiness from already-loaded trip signals.
// Callers own permission checks and fail-soft data loading.
func Evaluate(snapshot Snapshot, options Options) Response {
	now := snapshot.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tripID := uuid.Nil
	if snapshot.Trip != nil {
		tripID = snapshot.Trip.ID
	}

	evaluations := make([]memberEvaluation, 0, len(snapshot.Members))
	for _, member := range snapshot.Members {
		evaluation := newMemberEvaluation(member)
		evaluateAvailability(&evaluation, snapshot, now)
		evaluatePolls(&evaluation, snapshot)
		evaluateChecklist(&evaluation, snapshot, now)
		evaluateReminders(&evaluation, snapshot, now)
		evaluateSettlements(&evaluation, snapshot, now)
		evaluateApproval(&evaluation, snapshot)
		evaluateActivity(&evaluation)
		evaluation.nextAction = firstAction(evaluation.items)
		evaluations = append(evaluations, evaluation)
	}

	members := make([]CollaboratorReadiness, 0, len(evaluations))
	total := 0
	readyCount := 0
	for _, evaluation := range evaluations {
		score := scoreMember(evaluation.scoreInputs)
		level := levelFor(score, evaluation.items)
		if level == LevelReady {
			readyCount++
		}
		total += score
		items := evaluation.items
		completed := evaluation.completed
		if !options.IncludeDetails {
			items = summarizeItems(items)
			completed = summarizeCompleted(completed)
		}
		members = append(members, CollaboratorReadiness{
			UserID:         evaluation.member.UserID,
			DisplayName:    evaluation.member.DisplayName,
			Role:           evaluation.member.Role,
			Status:         string(level),
			Score:          score,
			Level:          level,
			IsCurrentUser:  evaluation.member.IsCurrentUser,
			Items:          items,
			CompletedItems: completed,
			NextAction:     evaluation.nextAction,
		})
	}

	score := 100
	if len(members) > 0 {
		score = int(math.Round(float64(total) / float64(len(members))))
	}
	level := groupLevel(score, members)
	response := Response{
		TripID:          tripID,
		Score:           score,
		Level:           level,
		Summary:         summary(level, readyCount, len(members), members),
		GeneratedAt:     now,
		Members:         members,
		CategorySummary: categorySummary(evaluations),
		TopActions:      topActions(evaluations, options.CanNudge, 6),
	}
	if options.IncludeDebug {
		response.Debug = map[string]any{
			"subsystemFailures": snapshot.SubsystemFailures,
			"memberCount":       len(snapshot.Members),
			"weights": map[string]int{
				"availability": weightAvailability,
				"polls":        weightPolls,
				"checklist":    weightChecklist,
				"reminders":    weightReminders,
				"noOverdue":    weightNoOverdue,
				"settlements":  weightSettlements,
				"activity":     weightActivity,
			},
		}
	}
	return response
}

func newMemberEvaluation(member Member) memberEvaluation {
	return memberEvaluation{
		member: member,
		scoreInputs: map[string]scoreInput{
			"availability": {weight: weightAvailability},
			"polls":        {weight: weightPolls},
			"checklist":    {weight: weightChecklist},
			"reminders":    {weight: weightReminders},
			"no_overdue":   {weight: weightNoOverdue},
			"settlements":  {weight: weightSettlements},
			"activity":     {weight: weightActivity},
		},
		items:              []Item{},
		completed:          []CompletedItem{},
		categoryApplicable: map[Category]bool{},
		categoryComplete:   map[Category]bool{},
	}
}

func evaluateAvailability(e *memberEvaluation, snapshot Snapshot, now time.Time) {
	applicable := len(snapshot.Members) > 1
	if !applicable {
		return
	}
	e.categoryApplicable[CategoryAvailability] = true
	submitted := false
	for _, response := range snapshot.AvailabilityResponses {
		if response.UserID == e.member.UserID {
			submitted = true
			break
		}
	}
	if submitted {
		e.scoreInputs["availability"] = scoreInput{weight: weightAvailability, applicable: true, ratio: 1}
		e.categoryComplete[CategoryAvailability] = true
		e.completed = append(e.completed, CompletedItem{Category: CategoryAvailability, Title: "Availability submitted"})
		return
	}
	severity := SeverityHigh
	if tripStartsWithin(snapshot.Trip, now, 3) {
		severity = SeverityCritical
	}
	e.scoreInputs["availability"] = scoreInput{weight: weightAvailability, applicable: true, ratio: 0}
	e.items = append(e.items, Item{
		ID:          "availability_missing",
		Category:    CategoryAvailability,
		Status:      ItemStatusMissing,
		Severity:    severity,
		Title:       "Availability not submitted",
		Description: fmt.Sprintf("%s has not submitted availability for this trip.", e.member.DisplayName),
		Action: &Action{
			Type:  "request_availability",
			Label: availabilityActionLabel(e.member),
			Href:  tripHref(snapshot.Trip, "dates"),
		},
	})
}

func evaluatePolls(e *memberEvaluation, snapshot Snapshot) {
	openPolls := make([]PollSnapshot, 0)
	requiredPolls := make([]PollSnapshot, 0)
	optionalMissing := 0
	votedRequired := 0
	for _, poll := range snapshot.Polls {
		if poll.Poll.Status != entity.PollStatusOpen {
			continue
		}
		openPolls = append(openPolls, poll)
		voted := userVoted(poll.Votes, e.member.UserID)
		required := pollRequired(poll.Poll)
		if required {
			requiredPolls = append(requiredPolls, poll)
			if voted {
				votedRequired++
			}
			continue
		}
		if !voted {
			optionalMissing++
		}
	}
	if len(openPolls) == 0 {
		return
	}
	e.categoryApplicable[CategoryPolls] = true
	if len(requiredPolls) > 0 {
		e.scoreInputs["polls"] = scoreInput{
			weight:     weightPolls,
			applicable: true,
			ratio:      float64(votedRequired) / float64(len(requiredPolls)),
		}
	}
	missingRequired := len(requiredPolls) - votedRequired
	switch {
	case missingRequired > 0:
		e.items = append(e.items, Item{
			ID:          "required_poll_not_voted",
			Category:    CategoryPolls,
			Status:      ItemStatusIncomplete,
			Severity:    SeverityHigh,
			Title:       fmt.Sprintf("%d required poll(s) need a vote", missingRequired),
			Description: fmt.Sprintf("%s has not voted on %d required open poll(s).", e.member.DisplayName, missingRequired),
			Action: &Action{
				Type:  "open_poll",
				Label: pollActionLabel(e.member),
				Href:  tripHref(snapshot.Trip, "polls"),
			},
		})
	case len(requiredPolls) > 0:
		e.categoryComplete[CategoryPolls] = true
		e.completed = append(e.completed, CompletedItem{Category: CategoryPolls, Title: "Required polls voted"})
	}
	if optionalMissing > 0 {
		e.items = append(e.items, Item{
			ID:          "optional_poll_not_voted",
			Category:    CategoryPolls,
			Status:      ItemStatusPending,
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf("%d optional poll(s) open", optionalMissing),
			Description: fmt.Sprintf("%s can still vote on %d optional poll(s).", e.member.DisplayName, optionalMissing),
			Action: &Action{
				Type:  "open_poll",
				Label: "Open polls",
				Href:  tripHref(snapshot.Trip, "polls"),
			},
		})
	}
}

func evaluateChecklist(e *memberEvaluation, snapshot Snapshot, now time.Time) {
	if snapshot.Checklist == nil {
		return
	}
	assigned := make([]entity.TripChecklistItem, 0)
	for _, item := range snapshot.Checklist.Items {
		if item.DeletedAt != nil || item.AssignedToUserID == nil || *item.AssignedToUserID != e.member.UserID {
			continue
		}
		assigned = append(assigned, item)
	}
	if len(assigned) == 0 {
		return
	}
	e.categoryApplicable[CategoryChecklist] = true
	checked := 0
	incomplete := 0
	overdue := 0
	highest := SeverityWarning
	for _, item := range assigned {
		if item.Checked {
			checked++
			continue
		}
		incomplete++
		if item.DueDate != nil && beforeToday(*item.DueDate, now) {
			overdue++
			highest = maxSeverity(highest, checklistSeverity(item.Priority, true, snapshot.Trip, now))
			continue
		}
		highest = maxSeverity(highest, checklistSeverity(item.Priority, false, snapshot.Trip, now))
	}
	e.scoreInputs["checklist"] = scoreInput{
		weight:     weightChecklist,
		applicable: true,
		ratio:      float64(checked) / float64(len(assigned)),
	}
	if overdue == 0 {
		e.scoreInputs["no_overdue"] = mergeNoOverdueInput(e.scoreInputs["no_overdue"], true)
	} else {
		e.scoreInputs["no_overdue"] = mergeNoOverdueInput(e.scoreInputs["no_overdue"], false)
	}
	if incomplete == 0 {
		e.categoryComplete[CategoryChecklist] = true
		e.completed = append(e.completed, CompletedItem{Category: CategoryChecklist, Title: "Assigned checklist items complete"})
		return
	}
	status := ItemStatusIncomplete
	title := fmt.Sprintf("%d assigned checklist item(s) incomplete", incomplete)
	if overdue > 0 {
		status = ItemStatusOverdue
		title = fmt.Sprintf("%d assigned checklist item(s) overdue", overdue)
	}
	e.items = append(e.items, Item{
		ID:          "assigned_checklist_items_incomplete",
		Category:    CategoryChecklist,
		Status:      status,
		Severity:    highest,
		Title:       title,
		Description: fmt.Sprintf("%s still has %d assigned checklist item(s) to complete.", e.member.DisplayName, incomplete),
		Action: &Action{
			Type:  "open_checklist",
			Label: checklistActionLabel(e.member),
			Href:  fmt.Sprintf("%s&assignedTo=%s", tripHref(snapshot.Trip, "checklist"), e.member.UserID.String()),
		},
	})
}

func evaluateReminders(e *memberEvaluation, snapshot Snapshot, now time.Time) {
	assigned := make([]entity.TripReminder, 0)
	for _, reminder := range snapshot.Reminders {
		if reminder.DeletedAt != nil || reminder.AssignedToUserID == nil || *reminder.AssignedToUserID != e.member.UserID {
			continue
		}
		if reminder.Status == entity.ReminderStatusCancelled || reminder.Status == entity.ReminderStatusDisabled {
			continue
		}
		assigned = append(assigned, reminder)
	}
	if len(assigned) == 0 {
		return
	}
	e.categoryApplicable[CategoryReminders] = true
	completed := 0
	incomplete := 0
	overdue := 0
	highest := SeverityWarning
	for _, reminder := range assigned {
		if reminder.Status == entity.ReminderStatusCompleted {
			completed++
			continue
		}
		incomplete++
		if beforeToday(reminder.TriggerDate, now) {
			overdue++
			highest = maxSeverity(highest, reminderSeverity(reminder.Priority, true, snapshot.Trip, now))
			continue
		}
		highest = maxSeverity(highest, reminderSeverity(reminder.Priority, false, snapshot.Trip, now))
	}
	e.scoreInputs["reminders"] = scoreInput{
		weight:     weightReminders,
		applicable: true,
		ratio:      float64(completed) / float64(len(assigned)),
	}
	if overdue == 0 {
		e.scoreInputs["no_overdue"] = mergeNoOverdueInput(e.scoreInputs["no_overdue"], true)
	} else {
		e.scoreInputs["no_overdue"] = mergeNoOverdueInput(e.scoreInputs["no_overdue"], false)
	}
	if incomplete == 0 {
		e.categoryComplete[CategoryReminders] = true
		e.completed = append(e.completed, CompletedItem{Category: CategoryReminders, Title: "Assigned reminders complete"})
		return
	}
	status := ItemStatusPending
	title := fmt.Sprintf("%d assigned reminder(s) pending", incomplete)
	if overdue > 0 {
		status = ItemStatusOverdue
		title = fmt.Sprintf("%d assigned reminder(s) overdue", overdue)
	}
	e.items = append(e.items, Item{
		ID:          "assigned_reminders_pending",
		Category:    CategoryReminders,
		Status:      status,
		Severity:    highest,
		Title:       title,
		Description: fmt.Sprintf("%s still has %d assigned reminder(s) to complete.", e.member.DisplayName, incomplete),
		Action: &Action{
			Type:  "open_reminders",
			Label: reminderActionLabel(e.member),
			Href:  fmt.Sprintf("%s&assignedTo=%s", tripHref(snapshot.Trip, "reminders"), e.member.UserID.String()),
		},
	})
}

func evaluateSettlements(e *memberEvaluation, snapshot Snapshot, now time.Time) {
	if len(snapshot.Settlements) == 0 {
		return
	}
	e.categoryApplicable[CategorySettlements] = true
	pending := 0
	for _, settlement := range snapshot.Settlements {
		if settlement.Status != entity.SettlementStatusPending {
			continue
		}
		if settlement.FromUserID == e.member.UserID || settlement.ToUserID == e.member.UserID {
			pending++
		}
	}
	if pending == 0 {
		e.scoreInputs["settlements"] = scoreInput{weight: weightSettlements, applicable: true, ratio: 1}
		e.categoryComplete[CategorySettlements] = true
		e.completed = append(e.completed, CompletedItem{Category: CategorySettlements, Title: "Settlements handled"})
		return
	}
	severity := settlementSeverity(snapshot.Trip, now)
	e.scoreInputs["settlements"] = scoreInput{weight: weightSettlements, applicable: true, ratio: 0}
	e.items = append(e.items, Item{
		ID:          "pending_settlements",
		Category:    CategorySettlements,
		Status:      ItemStatusPending,
		Severity:    severity,
		Title:       fmt.Sprintf("%d pending settlement(s)", pending),
		Description: fmt.Sprintf("%s is involved in %d pending settlement(s).", e.member.DisplayName, pending),
		Action: &Action{
			Type:  "open_settlements",
			Label: settlementActionLabel(e.member),
			Href:  tripHref(snapshot.Trip, "settlements"),
		},
	})
}

func evaluateApproval(e *memberEvaluation, snapshot Snapshot) {
	if snapshot.Trip == nil || snapshot.Trip.WorkspaceID == nil || snapshot.Approval == nil {
		return
	}
	status := approvals.Status(snapshot.Approval.Status)
	if status != approvals.StatusChangesRequested {
		return
	}
	isOwner := e.member.Role == "owner"
	if !isOwner && snapshot.Trip.UserID != nil {
		isOwner = *snapshot.Trip.UserID == e.member.UserID
	}
	if !isOwner {
		return
	}
	e.categoryApplicable[CategoryApproval] = true
	e.items = append(e.items, Item{
		ID:          "changes_requested_action_required",
		Category:    CategoryApproval,
		Status:      ItemStatusIncomplete,
		Severity:    SeverityHigh,
		Title:       "Approval changes requested",
		Description: "Workspace approval requested changes before the trip is ready.",
		Action: &Action{
			Type:  "open_approval",
			Label: "Open approval",
			Href:  tripHref(snapshot.Trip, "approval"),
		},
	})
}

func evaluateActivity(e *memberEvaluation) {
	// Activity read markers are not available server-side in v1.
	e.scoreInputs["activity"] = scoreInput{weight: weightActivity, applicable: false, ratio: 0}
}

func scoreMember(inputs map[string]scoreInput) int {
	totalWeight := 0
	earned := 0.0
	for _, input := range inputs {
		if !input.applicable {
			continue
		}
		totalWeight += input.weight
		earned += float64(input.weight) * clampRatio(input.ratio)
	}
	if totalWeight == 0 {
		return 100
	}
	return clampInt(int(math.Round((earned/float64(totalWeight))*100)), 0, 100)
}

func levelFor(score int, items []Item) Level {
	hasHigh := false
	hasCritical := false
	for _, item := range items {
		switch item.Severity {
		case SeverityCritical:
			hasCritical = true
		case SeverityHigh:
			hasHigh = true
		}
	}
	switch {
	case score >= 90 && !hasHigh && !hasCritical:
		return LevelReady
	case score >= 75 && !hasCritical:
		return LevelAlmostReady
	case score >= 50:
		return LevelNeedsAttention
	default:
		return LevelNotReady
	}
}

func groupLevel(score int, members []CollaboratorReadiness) Level {
	hasHigh := false
	hasCritical := false
	for _, member := range members {
		for _, item := range member.Items {
			switch item.Severity {
			case SeverityCritical:
				hasCritical = true
			case SeverityHigh:
				hasHigh = true
			}
		}
	}
	switch {
	case score >= 90 && !hasHigh && !hasCritical:
		return LevelReady
	case score >= 75 && !hasCritical:
		return LevelAlmostReady
	case score >= 50:
		return LevelNeedsAttention
	default:
		return LevelNotReady
	}
}

func summary(level Level, readyCount, total int, members []CollaboratorReadiness) string {
	if total == 0 {
		return "No collaborators are available for readiness yet."
	}
	if readyCount == total && level == LevelReady {
		return "Everyone is ready."
	}
	categoryCounts := map[Category]int{}
	for _, member := range members {
		for _, item := range member.Items {
			if item.Severity == SeverityInfo {
				continue
			}
			categoryCounts[item.Category]++
		}
	}
	topCategories := sortedCategoriesByCount(categoryCounts)
	if len(topCategories) == 0 {
		return fmt.Sprintf("%d of %d collaborators are ready. No urgent group actions are open.", readyCount, total)
	}
	labels := make([]string, 0, minInt(2, len(topCategories)))
	for _, category := range topCategories {
		if len(labels) >= 2 {
			break
		}
		labels = append(labels, categoryLabel(category))
	}
	return fmt.Sprintf("%d of %d collaborators are ready. %s still need attention.", readyCount, total, strings.Join(labels, " and "))
}

func categorySummary(evaluations []memberEvaluation) []CategorySummary {
	type bucket struct {
		total   int
		ready   int
		issues  int
		highest Severity
	}
	byCategory := map[Category]*bucket{}
	for _, evaluation := range evaluations {
		categories := map[Category]bool{}
		for category := range evaluation.categoryApplicable {
			categories[category] = true
		}
		for _, item := range evaluation.items {
			categories[item.Category] = true
		}
		for category := range categories {
			if _, ok := byCategory[category]; !ok {
				byCategory[category] = &bucket{}
			}
			byCategory[category].total++
			if evaluation.categoryComplete[category] {
				byCategory[category].ready++
			}
		}
		for _, item := range evaluation.items {
			b := byCategory[item.Category]
			if b == nil {
				b = &bucket{}
				byCategory[item.Category] = b
			}
			b.issues++
			b.highest = maxSeverity(b.highest, item.Severity)
		}
	}
	categories := make([]Category, 0, len(byCategory))
	for category := range byCategory {
		categories = append(categories, category)
	}
	sort.SliceStable(categories, func(i, j int) bool {
		return categoryRank(categories[i]) < categoryRank(categories[j])
	})
	out := make([]CategorySummary, 0, len(categories))
	for _, category := range categories {
		b := byCategory[category]
		out = append(out, CategorySummary{
			Category:        category,
			ReadyCount:      b.ready,
			TotalCount:      b.total,
			OpenIssueCount:  b.issues,
			HighestSeverity: b.highest,
		})
	}
	return out
}

func topActions(evaluations []memberEvaluation, canNudge bool, limit int) []TopAction {
	type candidate struct {
		member Member
		item   Item
	}
	candidates := make([]candidate, 0)
	for _, evaluation := range evaluations {
		for _, item := range evaluation.items {
			if item.Action == nil {
				continue
			}
			if !canNudge && !evaluation.member.IsCurrentUser {
				continue
			}
			candidates = append(candidates, candidate{member: evaluation.member, item: item})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if severityRank(left.item.Severity) != severityRank(right.item.Severity) {
			return severityRank(left.item.Severity) > severityRank(right.item.Severity)
		}
		if left.member.IsCurrentUser != right.member.IsCurrentUser {
			return left.member.IsCurrentUser
		}
		if categoryRank(left.item.Category) != categoryRank(right.item.Category) {
			return categoryRank(left.item.Category) < categoryRank(right.item.Category)
		}
		return left.member.DisplayName < right.member.DisplayName
	})
	seen := map[string]struct{}{}
	out := make([]TopAction, 0, limit)
	for _, candidate := range candidates {
		if len(out) >= limit {
			break
		}
		action := candidate.item.Action
		key := fmt.Sprintf("%s:%s:%s", action.Type, candidate.member.UserID, candidate.item.Category)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		target := candidate.member.UserID
		label := action.Label
		if canNudge && !candidate.member.IsCurrentUser {
			label = nudgeLabel(candidate.item.Category, candidate.member.DisplayName)
		}
		out = append(out, TopAction{
			ID:           key,
			Label:        label,
			Description:  candidate.item.Description,
			Href:         action.Href,
			ActionType:   action.Type,
			TargetUserID: &target,
		})
	}
	return out
}

func firstAction(items []Item) *Action {
	if len(items) == 0 {
		return nil
	}
	candidates := append([]Item(nil), items...)
	sort.SliceStable(candidates, func(i, j int) bool {
		if severityRank(candidates[i].Severity) != severityRank(candidates[j].Severity) {
			return severityRank(candidates[i].Severity) > severityRank(candidates[j].Severity)
		}
		return categoryRank(candidates[i].Category) < categoryRank(candidates[j].Category)
	})
	for _, item := range candidates {
		if item.Action != nil {
			return item.Action
		}
	}
	return nil
}

func summarizeItems(items []Item) []Item {
	if len(items) <= 3 {
		return items
	}
	sort.SliceStable(items, func(i, j int) bool {
		if severityRank(items[i].Severity) != severityRank(items[j].Severity) {
			return severityRank(items[i].Severity) > severityRank(items[j].Severity)
		}
		return categoryRank(items[i].Category) < categoryRank(items[j].Category)
	})
	return append([]Item(nil), items[:3]...)
}

func summarizeCompleted(items []CompletedItem) []CompletedItem {
	if len(items) <= 4 {
		return items
	}
	return append([]CompletedItem(nil), items[:4]...)
}

func pollRequired(poll entity.TripPoll) bool {
	if poll.PollType == entity.PollTypeDateChoice {
		return true
	}
	if metadataBool(poll.Metadata, "required") {
		return true
	}
	category := metadataString(poll.Metadata, "category")
	return category == "date_options" || category == "route_alternatives" || category == "route_decision"
}

func userVoted(votes []entity.TripPollVote, userID uuid.UUID) bool {
	for _, vote := range votes {
		if vote.UserID == userID {
			return true
		}
	}
	return false
}

func metadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	switch value := metadata[key].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true") || value == "1"
	default:
		return false
	}
}

func mergeNoOverdueInput(existing scoreInput, clear bool) scoreInput {
	if !existing.applicable {
		if clear {
			return scoreInput{weight: weightNoOverdue, applicable: true, ratio: 1}
		}
		return scoreInput{weight: weightNoOverdue, applicable: true, ratio: 0}
	}
	if existing.ratio == 0 || !clear {
		return scoreInput{weight: weightNoOverdue, applicable: true, ratio: 0}
	}
	return scoreInput{weight: weightNoOverdue, applicable: true, ratio: 1}
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func checklistSeverity(priority entity.ChecklistPriority, overdue bool, trip *entity.Trip, now time.Time) Severity {
	if overdue && priority == entity.ChecklistPriorityCritical {
		return SeverityCritical
	}
	if overdue && priority == entity.ChecklistPriorityHigh {
		if tripStartsWithin(trip, now, 3) {
			return SeverityCritical
		}
		return SeverityHigh
	}
	if overdue {
		return SeverityWarning
	}
	if priority == entity.ChecklistPriorityCritical || priority == entity.ChecklistPriorityHigh {
		return SeverityWarning
	}
	return SeverityWarning
}

func reminderSeverity(priority entity.ReminderPriority, overdue bool, trip *entity.Trip, now time.Time) Severity {
	if overdue && priority == entity.ReminderPriorityCritical {
		return SeverityCritical
	}
	if overdue && priority == entity.ReminderPriorityHigh {
		if tripStartsWithin(trip, now, 3) {
			return SeverityCritical
		}
		return SeverityHigh
	}
	if overdue {
		return SeverityWarning
	}
	if priority == entity.ReminderPriorityCritical || priority == entity.ReminderPriorityHigh {
		return SeverityWarning
	}
	return SeverityWarning
}

func settlementSeverity(trip *entity.Trip, now time.Time) Severity {
	if trip == nil || trip.StartDate == nil {
		return SeverityInfo
	}
	start := truncateDate(*trip.StartDate)
	end := start.AddDate(0, 0, int(maxInt32(trip.Days, 1)))
	today := truncateDate(now)
	if today.After(end) {
		return SeverityHigh
	}
	if !today.Before(start) {
		return SeverityWarning
	}
	return SeverityInfo
}

func tripStartsWithin(trip *entity.Trip, now time.Time, days int) bool {
	if trip == nil || trip.StartDate == nil {
		return false
	}
	start := truncateDate(*trip.StartDate)
	today := truncateDate(now)
	if start.Before(today) {
		return false
	}
	return !start.After(today.AddDate(0, 0, days))
}

func beforeToday(value time.Time, now time.Time) bool {
	return truncateDate(value).Before(truncateDate(now))
}

func truncateDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func tripHref(trip *entity.Trip, tab string) string {
	if trip == nil {
		return "#"
	}
	return fmt.Sprintf("/trips/%s?tab=%s", trip.ID.String(), tab)
}

func availabilityActionLabel(member Member) string {
	if member.IsCurrentUser {
		return "Submit availability"
	}
	return "Request availability"
}

func pollActionLabel(member Member) string {
	if member.IsCurrentUser {
		return "Vote now"
	}
	return "Open polls"
}

func checklistActionLabel(member Member) string {
	if member.IsCurrentUser {
		return "Open checklist"
	}
	return "Open checklist"
}

func reminderActionLabel(member Member) string {
	if member.IsCurrentUser {
		return "Complete reminders"
	}
	return "Open reminders"
}

func settlementActionLabel(member Member) string {
	if member.IsCurrentUser {
		return "Review settlements"
	}
	return "Open settlements"
}

func nudgeLabel(category Category, displayName string) string {
	switch category {
	case CategoryAvailability:
		return fmt.Sprintf("Request availability from %s", displayName)
	case CategoryPolls:
		return fmt.Sprintf("Remind %s to vote", displayName)
	case CategoryChecklist, CategoryReminders:
		return fmt.Sprintf("Remind %s about assigned items", displayName)
	case CategorySettlements:
		return fmt.Sprintf("Ask %s to review settlements", displayName)
	default:
		return fmt.Sprintf("Remind %s", displayName)
	}
}

func categoryLabel(category Category) string {
	switch category {
	case CategoryAvailability:
		return "availability"
	case CategoryPolls:
		return "polls"
	case CategoryChecklist:
		return "checklist assignments"
	case CategoryReminders:
		return "reminders"
	case CategorySettlements:
		return "settlements"
	case CategoryApproval:
		return "approval"
	default:
		return strings.ReplaceAll(string(category), "_", " ")
	}
}

func sortedCategoriesByCount(counts map[Category]int) []Category {
	categories := make([]Category, 0, len(counts))
	for category, count := range counts {
		if count > 0 {
			categories = append(categories, category)
		}
	}
	sort.SliceStable(categories, func(i, j int) bool {
		if counts[categories[i]] != counts[categories[j]] {
			return counts[categories[i]] > counts[categories[j]]
		}
		return categoryRank(categories[i]) < categoryRank(categories[j])
	})
	return categories
}

func categoryRank(category Category) int {
	switch category {
	case CategoryApproval:
		return 1
	case CategoryAvailability:
		return 2
	case CategoryPolls:
		return 3
	case CategoryChecklist:
		return 4
	case CategoryReminders:
		return 5
	case CategorySettlements:
		return 6
	case CategoryExpenses:
		return 7
	case CategoryComments:
		return 8
	case CategoryActivity:
		return 9
	default:
		return 20
	}
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func maxSeverity(a, b Severity) Severity {
	if severityRank(b) > severityRank(a) {
		return b
	}
	return a
}

func clampRatio(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt32(value int32, fallback int32) int32 {
	if value < fallback {
		return fallback
	}
	return value
}
