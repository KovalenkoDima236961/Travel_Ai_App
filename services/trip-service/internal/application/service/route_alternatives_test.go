package service

import (
	"testing"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/routealternatives"
)

func TestSuggestRouteAlternativesPersistsSession(t *testing.T) {
	repo := &mockRepo{}
	gen := &mockGenerator{routeAlternativesResult: routeAlternativeTestResponse()}
	svc := newTestService(repo, gen)

	amount := 700.0
	view, err := svc.SuggestRouteAlternatives(authContext(), routealternatives.SuggestInput{
		Prompt:          "A 5-day Austria train trip from Bratislava",
		DurationDays:    5,
		Budget:          &routealternatives.BudgetEstimate{Amount: &amount, Currency: "EUR"},
		Travelers:       2,
		OutputLanguage:  "en",
		SuggestionCount: 3,
		Transport: routealternatives.TransportInput{
			PreferredModes: []string{aggregate.TransportModeTrain},
			AvoidModes:     []string{aggregate.TransportModeFlight},
		},
		TripStyles: []string{"nature", "culture", "train_trip"},
	})
	if err != nil {
		t.Fatalf("SuggestRouteAlternatives returned error: %v", err)
	}
	if !gen.routeAlternativesCalled {
		t.Fatal("expected generator to be called")
	}
	if len(view.Alternatives) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(view.Alternatives))
	}
	if len(repo.routeAlternativeSessions) != 1 {
		t.Fatalf("expected one persisted session, got %d", len(repo.routeAlternativeSessions))
	}
	if repo.routeAlternativeSessions[0].Source != routealternatives.SourcePreTrip {
		t.Fatalf("expected pre-trip source, got %q", repo.routeAlternativeSessions[0].Source)
	}
	if view.ComparisonSummary.BestOverallAlternativeID == "" {
		t.Fatal("expected comparison summary to be filled")
	}
}

func TestRefineRouteAlternativesCreatesChildSession(t *testing.T) {
	repo := &mockRepo{}
	gen := &mockGenerator{routeAlternativesResult: routeAlternativeTestResponse()}
	svc := newTestService(repo, gen)

	parent, err := svc.SuggestRouteAlternatives(authContext(), routealternatives.SuggestInput{
		Prompt:         "A 5-day Austria train trip from Bratislava",
		DurationDays:   5,
		Travelers:      2,
		OutputLanguage: "en",
	})
	if err != nil {
		t.Fatalf("SuggestRouteAlternatives returned error: %v", err)
	}
	child, err := svc.RefineRouteAlternativeSession(authContext(), parent.ID, routealternatives.RefineInput{
		Instruction:           "Make it cheaper and use fewer stops.",
		SelectedAlternativeID: "classic-austria-train-route",
	})
	if err != nil {
		t.Fatalf("RefineRouteAlternativeSession returned error: %v", err)
	}
	if child.ParentSessionID == nil || *child.ParentSessionID != parent.ID {
		t.Fatalf("expected child session to reference parent %s, got %v", parent.ID, child.ParentSessionID)
	}
	if len(repo.routeAlternativeSessions) != 2 {
		t.Fatalf("expected two persisted sessions, got %d", len(repo.routeAlternativeSessions))
	}
	if got := gen.capturedRouteAlternativesInput.Refinement.Instruction; got != "Make it cheaper and use fewer stops." {
		t.Fatalf("expected refinement instruction to be forwarded, got %q", got)
	}
}

func TestCreateTripFromRouteAlternativeCreatesMultiDestinationTrip(t *testing.T) {
	repo := &mockRepo{}
	gen := &mockGenerator{routeAlternativesResult: routeAlternativeTestResponse()}
	svc := newTestService(repo, gen)

	session, err := svc.SuggestRouteAlternatives(authContext(), routealternatives.SuggestInput{
		Prompt:         "A 5-day Austria train trip from Bratislava",
		DurationDays:   5,
		Travelers:      2,
		OutputLanguage: "en",
	})
	if err != nil {
		t.Fatalf("SuggestRouteAlternatives returned error: %v", err)
	}
	travelers := int32(2)
	trip, err := svc.CreateTripFromRouteAlternative(
		authContext(),
		session.ID,
		"classic-austria-train-route",
		routealternatives.CreateTripInput{Title: "Austria by train", Travelers: &travelers},
	)
	if err != nil {
		t.Fatalf("CreateTripFromRouteAlternative returned error: %v", err)
	}
	if trip.TripType != entity.TripTypeMultiDestination {
		t.Fatalf("expected multi-destination trip, got %q", trip.TripType)
	}
	if trip.Route == nil || len(trip.Route.Stops) != 2 {
		t.Fatalf("expected selected route with two stops, got %#v", trip.Route)
	}
	if !repo.creationMetadataCalled {
		t.Fatal("expected creation metadata to be updated")
	}
	if repo.creationMetadata["creationSource"] != "route_alternative" {
		t.Fatalf("expected route alternative creation source, got %#v", repo.creationMetadata["creationSource"])
	}
	stored, err := repo.GetRouteAlternativeSessionByID(authContext(), session.ID)
	if err != nil {
		t.Fatalf("expected stored session: %v", err)
	}
	if stored.Status != routealternatives.StatusCreatedTrip {
		t.Fatalf("expected session status created_trip, got %q", stored.Status)
	}
	if stored.SelectedAlternativeID != "classic-austria-train-route" {
		t.Fatalf("expected selected alternative to be recorded, got %q", stored.SelectedAlternativeID)
	}
}

func TestApplyRouteAlternativeUpdatesExistingTrip(t *testing.T) {
	userID := testUserID()
	tripID := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{
			ID:                tripID,
			UserID:            &userID,
			Destination:       "Austria",
			Days:              5,
			Travelers:         2,
			Pace:              "balanced",
			ItineraryRevision: 0,
		},
	}
	gen := &mockGenerator{routeAlternativesResult: routeAlternativeTestResponse()}
	svc := newTestService(repo, gen)

	session, err := svc.SuggestTripRouteAlternatives(authContext(), tripID, routealternatives.ExistingTripSuggestInput{
		Prompt:                    "Find a better train route.",
		SuggestionCount:           2,
		UseCurrentRouteAsBaseline: true,
		OutputLanguage:            "en",
	})
	if err != nil {
		t.Fatalf("SuggestTripRouteAlternatives returned error: %v", err)
	}
	updated, err := svc.ApplyRouteAlternative(
		authContext(),
		tripID,
		session.ID,
		"classic-austria-train-route",
		routealternatives.ApplyInput{ExpectedItineraryRevision: intPtr(0)},
	)
	if err != nil {
		t.Fatalf("ApplyRouteAlternative returned error: %v", err)
	}
	if !repo.routeUpdateCalled {
		t.Fatal("expected route update to be called")
	}
	if updated.Route == nil || len(updated.Route.Stops) != 2 {
		t.Fatalf("expected applied route with two stops, got %#v", updated.Route)
	}
	stored, err := repo.GetRouteAlternativeSessionByID(authContext(), session.ID)
	if err != nil {
		t.Fatalf("expected stored session: %v", err)
	}
	if stored.Status != routealternatives.StatusApplied {
		t.Fatalf("expected session status applied, got %q", stored.Status)
	}
}

func TestCreateRouteAlternativesPollFeedsGroupPreferences(t *testing.T) {
	userID := testUserID()
	tripID := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{
			ID:          tripID,
			UserID:      &userID,
			Destination: "Austria",
			Days:        5,
			Travelers:   2,
			Pace:        "balanced",
		},
	}
	gen := &mockGenerator{routeAlternativesResult: routeAlternativeTestResponse()}
	svc := newTestService(repo, gen)

	session, err := svc.SuggestTripRouteAlternatives(authContext(), tripID, routealternatives.ExistingTripSuggestInput{
		Prompt:          "Find a better train route.",
		SuggestionCount: 2,
		OutputLanguage:  "en",
	})
	if err != nil {
		t.Fatalf("SuggestTripRouteAlternatives returned error: %v", err)
	}
	pollInfo, err := svc.CreateRouteAlternativesPoll(authContext(), tripID, session.ID, routealternatives.CreatePollInput{})
	if err != nil {
		t.Fatalf("CreateRouteAlternativesPoll returned error: %v", err)
	}
	if len(pollInfo.Options) != 2 {
		t.Fatalf("expected two poll options, got %d", len(pollInfo.Options))
	}
	voted, err := svc.VoteTripPoll(authContext(), tripID, pollInfo.Poll.ID, appdto.VoteTripPollInput{
		OptionIDs: []uuid.UUID{pollInfo.Options[0].ID},
	})
	if err != nil {
		t.Fatalf("VoteTripPoll returned error: %v", err)
	}
	if len(voted.Results.WinningOptionIDs) != 1 {
		t.Fatalf("expected one winning option, got %d", len(voted.Results.WinningOptionIDs))
	}
	summary, err := svc.GetGroupPreferences(authContext(), tripID)
	if err != nil {
		t.Fatalf("GetGroupPreferences returned error: %v", err)
	}
	if len(summary.RouteAlternativeVotes) != 1 {
		t.Fatalf("expected one route alternative vote summary, got %d", len(summary.RouteAlternativeVotes))
	}
	if summary.AIConstraints.PreferredRouteAlternativeID != pollInfo.Options[0].OptionKey {
		t.Fatalf("expected preferred alternative %q, got %q", pollInfo.Options[0].OptionKey, summary.AIConstraints.PreferredRouteAlternativeID)
	}
	if summary.AIConstraints.PreferredRouteSessionID != session.ID.String() {
		t.Fatalf("expected preferred session %q, got %q", session.ID, summary.AIConstraints.PreferredRouteSessionID)
	}
}

func routeAlternativeTestResponse() *routealternatives.Response {
	viennaNights := 2
	salzburgNights := 3
	trainMinutes := 75
	trainDistance := 80.0
	transferCostAmount := 32.0
	totalBudget := 650.0
	cheaperBudget := 520.0
	return &routealternatives.Response{
		SessionTitle: "Austria train route options",
		Alternatives: []routealternatives.Alternative{
			{
				ID:      "classic-austria-train-route",
				Title:   "Classic Austria Train Route",
				Summary: "A balanced route through Vienna and Salzburg.",
				Route: aggregate.TripRoute{
					Origin: &aggregate.RoutePlace{Name: "Bratislava", Country: "Slovakia"},
					Stops: []aggregate.RouteStop{
						{ID: "vienna", Destination: "Vienna", City: "Vienna", Country: "Austria", Nights: &viennaNights},
						{ID: "salzburg", Destination: "Salzburg", City: "Salzburg", Country: "Austria", Nights: &salzburgNights},
					},
					Legs: []aggregate.RouteLeg{
						{
							ID:                       "leg-bratislava-vienna",
							FromStopID:               "origin",
							ToStopID:                 "vienna",
							FromName:                 "Bratislava",
							ToName:                   "Vienna",
							Mode:                     aggregate.TransportModeTrain,
							EstimatedDurationMinutes: &trainMinutes,
							EstimatedDistanceKm:      &trainDistance,
							EstimatedCost: &aggregate.EstimatedCost{
								Amount:     &transferCostAmount,
								Currency:   "EUR",
								Category:   "transport",
								Confidence: "medium",
								Source:     "ai",
							},
						},
						{
							ID:                       "leg-vienna-salzburg",
							FromStopID:               "vienna",
							ToStopID:                 "salzburg",
							FromName:                 "Vienna",
							ToName:                   "Salzburg",
							Mode:                     aggregate.TransportModeTrain,
							EstimatedDurationMinutes: &trainMinutes,
							EstimatedDistanceKm:      &trainDistance,
							EstimatedCost: &aggregate.EstimatedCost{
								Amount:     &transferCostAmount,
								Currency:   "EUR",
								Category:   "transport",
								Confidence: "medium",
								Source:     "ai",
							},
						},
					},
					Preferences: aggregate.RoutePreferences{PreferredModes: []string{aggregate.TransportModeTrain}},
				},
				EstimatedBudget: &routealternatives.BudgetEstimate{Amount: &totalBudget, Currency: "EUR", Confidence: "medium"},
				Difficulty:      "balanced",
				BestFor:         []string{"culture", "nature", "train_trip"},
				Pros:            []string{"Mostly train-friendly"},
				Cons:            []string{"Popular cities can be busy"},
				Warnings:        []string{"Train times and prices are estimates."},
			},
			{
				ID:      "relaxed-two-city-route",
				Title:   "Relaxed Two-City Route",
				Summary: "A cheaper, simpler route with fewer transfers.",
				Route: aggregate.TripRoute{
					Origin: &aggregate.RoutePlace{Name: "Bratislava", Country: "Slovakia"},
					Stops: []aggregate.RouteStop{
						{ID: "vienna", Destination: "Vienna", City: "Vienna", Country: "Austria", Nights: &viennaNights},
						{ID: "graz", Destination: "Graz", City: "Graz", Country: "Austria", Nights: &salzburgNights},
					},
					Legs: []aggregate.RouteLeg{
						{
							ID:                       "leg-bratislava-vienna",
							FromStopID:               "origin",
							ToStopID:                 "vienna",
							FromName:                 "Bratislava",
							ToName:                   "Vienna",
							Mode:                     aggregate.TransportModeTrain,
							EstimatedDurationMinutes: &trainMinutes,
							EstimatedDistanceKm:      &trainDistance,
						},
						{
							ID:                       "leg-vienna-graz",
							FromStopID:               "vienna",
							ToStopID:                 "graz",
							FromName:                 "Vienna",
							ToName:                   "Graz",
							Mode:                     aggregate.TransportModeTrain,
							EstimatedDurationMinutes: &trainMinutes,
							EstimatedDistanceKm:      &trainDistance,
						},
					},
					Preferences: aggregate.RoutePreferences{PreferredModes: []string{aggregate.TransportModeTrain}},
				},
				EstimatedBudget: &routealternatives.BudgetEstimate{Amount: &cheaperBudget, Currency: "EUR", Confidence: "medium"},
				Difficulty:      "relaxed",
				BestFor:         []string{"low_budget", "train_trip"},
				Pros:            []string{"Fewer transfers"},
				Cons:            []string{"Less alpine scenery"},
				Warnings:        []string{"Estimates are approximate."},
			},
		},
		Warnings: []string{"Route estimates are approximate."},
	}
}
