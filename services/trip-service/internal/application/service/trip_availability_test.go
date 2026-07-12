package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestGetTripDateOptionsPrefersSharedWeekendOverlap(t *testing.T) {
	tripID := uuid.New()
	ownerID := testUserID()
	collaboratorID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := availabilityTestRepo(tripID, ownerID, collaboratorID)
	repo.availabilityResponses = []entity.TripAvailabilityResponse{
		{
			ID:              uuid.New(),
			TripID:          tripID,
			UserID:          ownerID,
			AvailableRanges: []entity.AvailabilityDateRange{{StartDate: "2026-09-10", EndDate: "2026-09-15"}},
			PreferredRanges: []entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-13"}},
		},
		{
			ID:                uuid.New(),
			TripID:            tripID,
			UserID:            collaboratorID,
			AvailableRanges:   []entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-16"}},
			UnavailableRanges: []entity.AvailabilityDateRange{{StartDate: "2026-09-14", EndDate: "2026-09-14"}},
			PreferredRanges:   []entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-13"}},
		},
	}

	got, err := newTestService(repo, nil).GetTripDateOptions(authContext(), tripID, appdto.DateOptionsInput{
		MinDays:         intPtr(2),
		MaxDays:         intPtr(2),
		SearchStartDate: "2026-09-12",
		SearchEndDate:   "2026-09-16",
		PreferWeekends:  availabilityBoolPtr(true),
		Limit:           5,
	})
	if err != nil {
		t.Fatalf("GetTripDateOptions returned error: %v", err)
	}
	if len(got.Options) == 0 {
		t.Fatal("expected at least one date option")
	}
	best := got.Options[0]
	if best.StartDate != "2026-09-12" || best.EndDate != "2026-09-13" {
		t.Fatalf("expected shared weekend overlap first, got %s to %s", best.StartDate, best.EndDate)
	}
	if best.AvailableUserCount != 2 || best.PreferredUserCount != 2 || best.ConflictUserCount != 0 {
		t.Fatalf("unexpected best option counts: %+v", best)
	}
}

func TestApplyTripDateOptionUpdatesTripAndMetadata(t *testing.T) {
	tripID := uuid.New()
	ownerID := testUserID()
	collaboratorID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	repo := availabilityTestRepo(tripID, ownerID, collaboratorID)
	repo.getByIDResult.StartDate = &startDate
	repo.getByIDResult.ItineraryRevision = 7
	repo.availabilityResponses = []entity.TripAvailabilityResponse{
		{
			ID:              uuid.New(),
			TripID:          tripID,
			UserID:          ownerID,
			AvailableRanges: []entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-13"}},
			PreferredRanges: []entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-13"}},
		},
		{
			ID:              uuid.New(),
			TripID:          tripID,
			UserID:          collaboratorID,
			AvailableRanges: []entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-13"}},
		},
	}
	svc := newTestService(repo, nil)
	options, err := svc.GetTripDateOptions(authContext(), tripID, appdto.DateOptionsInput{
		MinDays: intPtr(2),
		MaxDays: intPtr(2),
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("GetTripDateOptions returned error: %v", err)
	}
	if len(options.Options) != 1 {
		t.Fatalf("expected one date option, got %d", len(options.Options))
	}

	applied, err := svc.ApplyTripDateOption(authContext(), tripID, options.Options[0].ID, appdto.ApplyDateOptionInput{
		ExpectedItineraryRevision: intPtr(7),
	})
	if err != nil {
		t.Fatalf("ApplyTripDateOption returned error: %v", err)
	}
	if applied.Trip.StartDate == nil || applied.Trip.StartDate.Format("2006-01-02") != "2026-09-12" {
		t.Fatalf("expected trip start date to be applied, got %v", applied.Trip.StartDate)
	}
	if applied.Trip.Days != 2 {
		t.Fatalf("expected trip duration 2, got %d", applied.Trip.Days)
	}
	if repo.creationMetadata["selectedDateOption"] == nil {
		t.Fatal("expected selected date option metadata to be stored")
	}
}

func availabilityTestRepo(tripID, ownerID, collaboratorID uuid.UUID) *mockRepo {
	return &mockRepo{
		getByIDResult: &entity.Trip{
			ID:                tripID,
			UserID:            &ownerID,
			Destination:       "Rome",
			Days:              2,
			Travelers:         2,
			Pace:              "balanced",
			CreationMetadata:  map[string]any{},
			ItineraryRevision: 7,
		},
		listCollaborators: []entity.TripCollaborator{
			{
				ID:     uuid.New(),
				TripID: tripID,
				UserID: collaboratorID,
				Role:   entity.CollaboratorRoleViewer,
				Status: entity.CollaboratorStatusAccepted,
			},
		},
	}
}

func availabilityBoolPtr(value bool) *bool {
	return &value
}
