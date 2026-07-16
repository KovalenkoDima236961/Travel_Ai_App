package groupreadiness

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestEvaluate_AllCompleteReady(t *testing.T) {
	tripID := uuid.New()
	ownerID := uuid.New()
	collaboratorID := uuid.New()
	pollID := uuid.New()
	checklistID := uuid.New()
	reminderID := uuid.New()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	response := Evaluate(Snapshot{
		Trip: &entity.Trip{ID: tripID, UserID: &ownerID, StartDate: ptrTime(now.AddDate(0, 0, 30)), Days: 4},
		Members: []Member{
			{UserID: ownerID, DisplayName: "Owner", Role: "owner", IsCurrentUser: true},
			{UserID: collaboratorID, DisplayName: "Adam", Role: "editor"},
		},
		AvailabilityResponses: []entity.TripAvailabilityResponse{
			{ID: uuid.New(), TripID: tripID, UserID: ownerID},
			{ID: uuid.New(), TripID: tripID, UserID: collaboratorID},
		},
		Polls: []PollSnapshot{{
			Poll: entity.TripPoll{ID: pollID, TripID: tripID, PollType: entity.PollTypeDateChoice, Status: entity.PollStatusOpen},
			Votes: []entity.TripPollVote{
				{ID: uuid.New(), PollID: pollID, UserID: ownerID},
				{ID: uuid.New(), PollID: pollID, UserID: collaboratorID},
			},
		}},
		Checklist: &entity.TripChecklist{
			ID:     checklistID,
			TripID: tripID,
			Items: []entity.TripChecklistItem{
				{ID: uuid.New(), ChecklistID: checklistID, TripID: tripID, AssignedToUserID: &ownerID, Checked: true},
				{ID: uuid.New(), ChecklistID: checklistID, TripID: tripID, AssignedToUserID: &collaboratorID, Checked: true},
			},
		},
		Reminders: []entity.TripReminder{
			{ID: reminderID, TripID: tripID, AssignedToUserID: &ownerID, Status: entity.ReminderStatusCompleted, TriggerDate: now.AddDate(0, 0, 1)},
			{ID: uuid.New(), TripID: tripID, AssignedToUserID: &collaboratorID, Status: entity.ReminderStatusCompleted, TriggerDate: now.AddDate(0, 0, 1)},
		},
		Settlements: []entity.TripSettlement{
			{ID: uuid.New(), TripID: tripID, FromUserID: collaboratorID, ToUserID: ownerID, Status: entity.SettlementStatusPaid},
		},
		Now: now,
	}, Options{IncludeDetails: true, CanNudge: true})

	if response.Score != 100 || response.Level != LevelReady {
		t.Fatalf("expected ready 100, got score=%d level=%s", response.Score, response.Level)
	}
	for _, member := range response.Members {
		if member.Score != 100 || len(member.Items) != 0 {
			t.Fatalf("expected member ready with no issues, got %+v", member)
		}
	}
}

func TestEvaluate_MissingAvailabilityLowersScore(t *testing.T) {
	tripID := uuid.New()
	ownerID := uuid.New()
	collaboratorID := uuid.New()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	response := Evaluate(Snapshot{
		Trip: &entity.Trip{ID: tripID, UserID: &ownerID, StartDate: ptrTime(now.AddDate(0, 0, 10)), Days: 3},
		Members: []Member{
			{UserID: ownerID, DisplayName: "Owner", Role: "owner"},
			{UserID: collaboratorID, DisplayName: "Adam", Role: "editor"},
		},
		AvailabilityResponses: []entity.TripAvailabilityResponse{
			{ID: uuid.New(), TripID: tripID, UserID: ownerID},
		},
		Now: now,
	}, Options{IncludeDetails: true, CanNudge: true})

	if response.Score >= 100 {
		t.Fatalf("expected missing availability to lower group score, got %d", response.Score)
	}
	member := findMember(t, response, collaboratorID)
	if member.Score != 0 || member.Level != LevelNotReady {
		t.Fatalf("expected collaborator not ready from missing availability, got score=%d level=%s", member.Score, member.Level)
	}
	if len(member.Items) != 1 || member.Items[0].Category != CategoryAvailability || member.Items[0].Severity != SeverityHigh {
		t.Fatalf("expected high availability issue, got %+v", member.Items)
	}
}

func TestEvaluate_NotApplicableCategoriesRedistribute(t *testing.T) {
	userID := uuid.New()
	response := Evaluate(Snapshot{
		Trip:    &entity.Trip{ID: uuid.New(), UserID: &userID, Days: 2},
		Members: []Member{{UserID: userID, DisplayName: "Solo", Role: "owner", IsCurrentUser: true}},
		Now:     time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
	}, Options{IncludeDetails: true})

	if response.Score != 100 || response.Level != LevelReady {
		t.Fatalf("expected solo/no-signal trip to be ready, got score=%d level=%s", response.Score, response.Level)
	}
}

func TestEvaluate_IncompleteCollaborationItems(t *testing.T) {
	tripID := uuid.New()
	ownerID := uuid.New()
	collaboratorID := uuid.New()
	pollID := uuid.New()
	checklistID := uuid.New()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	yesterday := now.AddDate(0, 0, -1)
	tripStart := now.AddDate(0, 0, -7)

	response := Evaluate(Snapshot{
		Trip: &entity.Trip{ID: tripID, UserID: &ownerID, StartDate: &tripStart, Days: 3},
		Members: []Member{
			{UserID: ownerID, DisplayName: "Owner", Role: "owner"},
			{UserID: collaboratorID, DisplayName: "Adam", Role: "editor"},
		},
		AvailabilityResponses: []entity.TripAvailabilityResponse{
			{ID: uuid.New(), TripID: tripID, UserID: ownerID},
			{ID: uuid.New(), TripID: tripID, UserID: collaboratorID},
		},
		Polls: []PollSnapshot{{
			Poll:  entity.TripPoll{ID: pollID, TripID: tripID, PollType: entity.PollTypeDateChoice, Status: entity.PollStatusOpen},
			Votes: []entity.TripPollVote{{ID: uuid.New(), PollID: pollID, UserID: ownerID}},
		}},
		Checklist: &entity.TripChecklist{
			ID:     checklistID,
			TripID: tripID,
			Items: []entity.TripChecklistItem{{
				ID:               uuid.New(),
				ChecklistID:      checklistID,
				TripID:           tripID,
				AssignedToUserID: &collaboratorID,
				Priority:         entity.ChecklistPriorityHigh,
				DueDate:          &yesterday,
			}},
		},
		Reminders: []entity.TripReminder{{
			ID:               uuid.New(),
			TripID:           tripID,
			AssignedToUserID: &collaboratorID,
			Priority:         entity.ReminderPriorityHigh,
			Status:           entity.ReminderStatusPending,
			TriggerDate:      yesterday,
		}},
		Settlements: []entity.TripSettlement{{
			ID:         uuid.New(),
			TripID:     tripID,
			FromUserID: collaboratorID,
			ToUserID:   ownerID,
			Status:     entity.SettlementStatusPending,
		}},
		Now: now,
	}, Options{IncludeDetails: true, CanNudge: true})

	member := findMember(t, response, collaboratorID)
	if member.Score >= 50 {
		t.Fatalf("expected multiple incomplete items to push member below 50, got %d", member.Score)
	}
	categories := map[Category]bool{}
	for _, item := range member.Items {
		categories[item.Category] = true
	}
	for _, category := range []Category{CategoryPolls, CategoryChecklist, CategoryReminders, CategorySettlements} {
		if !categories[category] {
			t.Fatalf("expected category %s in issues, got %+v", category, member.Items)
		}
	}
}

func findMember(t *testing.T, response Response, userID uuid.UUID) CollaboratorReadiness {
	t.Helper()
	for _, member := range response.Members {
		if member.UserID == userID {
			return member
		}
	}
	t.Fatalf("member %s not found", userID)
	return CollaboratorReadiness{}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
