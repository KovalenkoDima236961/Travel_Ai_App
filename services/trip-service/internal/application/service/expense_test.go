package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestCalculateExpenseParticipantsSelectedEqualRoundingIsDeterministic(t *testing.T) {
	userA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userB := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	userC := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	expense := &entity.TripExpense{
		ID:           uuid.New(),
		TripID:       uuid.New(),
		Amount:       10,
		Currency:     "EUR",
		SplitType:    entity.ExpenseSplitSelectedEqual,
		PaidByUserID: userA,
		ExpenseDate:  time.Now(),
	}

	participants, err := calculateExpenseParticipants(
		expense,
		[]uuid.UUID{userC, userA, userB},
		nil,
		nil,
		map[uuid.UUID]expenseUser{
			userA: {ID: userA, DisplayName: "A"},
			userB: {ID: userB, DisplayName: "B"},
			userC: {ID: userC, DisplayName: "C"},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("calculate participants: %v", err)
	}
	if len(participants) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(participants))
	}

	got := map[uuid.UUID]float64{}
	for _, participant := range participants {
		if participant.ShareAmount == nil {
			t.Fatalf("participant %s missing share amount", participant.UserID)
		}
		got[participant.UserID] = *participant.ShareAmount
	}
	if got[userA] != 3.34 || got[userB] != 3.33 || got[userC] != 3.33 {
		t.Fatalf("unexpected deterministic split: %#v", got)
	}
}

func TestSettlementSuggestionsSimplifyBalances(t *testing.T) {
	tripID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	userA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userB := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	userC := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	suggestions := settlementSuggestions(
		tripID,
		"EUR",
		[]appdto.ExpenseBalance{
			{UserID: userA, NetOutstanding: money(50, "EUR")},
			{UserID: userB, NetOutstanding: money(-30, "EUR")},
			{UserID: userC, NetOutstanding: money(-20, "EUR")},
		},
		map[uuid.UUID]expenseUser{
			userA: {ID: userA, DisplayName: "A"},
			userB: {ID: userB, DisplayName: "B"},
			userC: {ID: userC, DisplayName: "C"},
		},
		"hash",
	)
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 settlement suggestions, got %d", len(suggestions))
	}
	if suggestions[0].FromUserID != userB || suggestions[0].ToUserID != userA ||
		suggestions[0].Amount.Amount != 30 {
		t.Fatalf("unexpected first suggestion: %#v", suggestions[0])
	}
	if suggestions[1].FromUserID != userC || suggestions[1].ToUserID != userA ||
		suggestions[1].Amount.Amount != 20 {
		t.Fatalf("unexpected second suggestion: %#v", suggestions[1])
	}
}
