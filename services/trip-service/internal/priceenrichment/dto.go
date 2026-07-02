package priceenrichment

import (
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

const (
	StatusMatched = "matched"
	StatusNoMatch = "no_match"
	StatusSkipped = "skipped"
	StatusFailed  = "failed"

	ReviewStatusPending  = "pending"
	ReviewStatusAccepted = "accepted"
	ReviewStatusChanged  = "changed"
	ReviewStatusRemoved  = "removed"
)

type EnrichItineraryInput struct {
	Destination           string
	BudgetCurrency        string
	UserPreferredCurrency string
	StartDate             *time.Time
	Itinerary             aggregate.Itinerary
}

type EnrichItineraryResult struct {
	Itinerary aggregate.Itinerary
	Stats     Stats
}

type Stats struct {
	Candidates                 int
	Matched                    int
	Skipped                    int
	NoMatch                    int
	Failed                     int
	Overwritten                int
	NotOverwrittenExistingCost int
}
