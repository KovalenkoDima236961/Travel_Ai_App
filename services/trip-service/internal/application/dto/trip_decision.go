package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateTripPollInput struct {
	Title              string
	Description        string
	PollType           entity.PollType
	AllowMultipleVotes bool
	ClosesAt           *time.Time
	Metadata           map[string]any
	Options            []CreateTripPollOptionInput
}

type CreateTripPollOptionInput struct {
	OptionKey   string
	Label       string
	Description string
	Metadata    map[string]any
}

type VoteTripPollInput struct {
	OptionIDs   []uuid.UUID
	VoteValue   string
	RatingValue *int
	Metadata    map[string]any
}

type TripPollInfo struct {
	Poll      entity.TripPoll
	Options   []entity.TripPollOption
	Results   PollResults
	UserVotes []entity.TripPollVote
	CanManage bool
	CanVote   bool
}

type PollResults struct {
	TotalVoters      int
	TotalVotes       int
	Options          []PollOptionResult
	WinningOptionIDs []uuid.UUID
}

type PollOptionResult struct {
	OptionID      uuid.UUID
	OptionKey     string
	Label         string
	VoteCount     int
	Percentage    int
	AverageRating *float64
}

type SetItineraryItemReactionInput struct {
	DayNumber int
	ItemIndex int
	ItemID    string
	Reaction  entity.ItineraryReaction
	Metadata  map[string]any
}

type ItineraryItemReactionSummary struct {
	DayNumber           int
	ItemIndex           int
	ItemID              string
	ItemName            string
	Counts              map[entity.ItineraryReaction]int
	CurrentUserReaction *entity.ItineraryReaction
	Score               int
}

type GroupPreferencesSummary struct {
	TripID                 uuid.UUID                     `json:"tripId"`
	GeneratedAt            time.Time                     `json:"generatedAt"`
	Summary                GroupPreferencesCounts        `json:"summary"`
	TopPollChoices         []GroupPreferencePollChoice   `json:"topPollChoices"`
	ItineraryPreferences   GroupItineraryPreferences     `json:"itineraryPreferences"`
	TransportPreferences   []GroupPreferenceScore        `json:"transportPreferences"`
	DestinationPreferences []GroupPreferenceScore        `json:"destinationPreferences"`
	DatePreferences        []GroupPreferenceScore        `json:"datePreferences"`
	AIConstraintSummary    string                        `json:"aiConstraintSummary"`
	AIConstraints          GroupPreferencesAIConstraints `json:"aiConstraints"`
}

type GroupPreferencesCounts struct {
	CollaboratorCount int `json:"collaboratorCount"`
	PollCount         int `json:"pollCount"`
	OpenPollCount     int `json:"openPollCount"`
	ReactionCount     int `json:"reactionCount"`
	MustHaveItemCount int `json:"mustHaveItemCount"`
	SkipItemCount     int `json:"skipItemCount"`
	OpenDecisionCount int `json:"openDecisionCount"`
}

type GroupPreferencePollChoice struct {
	PollID         uuid.UUID                     `json:"pollId"`
	Title          string                        `json:"title"`
	PollType       entity.PollType               `json:"pollType"`
	WinningOptions []GroupPreferenceOptionChoice `json:"winningOptions"`
}

type GroupPreferenceOptionChoice struct {
	OptionID   uuid.UUID `json:"optionId"`
	OptionKey  string    `json:"optionKey"`
	Label      string    `json:"label"`
	VoteCount  int       `json:"voteCount"`
	Percentage int       `json:"percentage"`
}

type GroupItineraryPreferences struct {
	MustHaveItems    []GroupPreferenceItineraryItem `json:"mustHaveItems"`
	MostSkippedItems []GroupPreferenceItineraryItem `json:"mostSkippedItems"`
	Controversial    []GroupPreferenceItineraryItem `json:"controversial"`
}

type GroupPreferenceItineraryItem struct {
	DayNumber int    `json:"dayNumber"`
	ItemIndex int    `json:"itemIndex"`
	ItemID    string `json:"itemId,omitempty"`
	Name      string `json:"name"`
	Count     int    `json:"count"`
	Score     int    `json:"score"`
}

type GroupPreferenceScore struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Score int    `json:"score"`
	Votes int    `json:"votes"`
}

type GroupPreferencesAIConstraints struct {
	Summary                 string                         `json:"summary"`
	MustHaveItems           []GroupPreferenceItineraryItem `json:"mustHaveItems"`
	SkipCandidates          []GroupPreferenceItineraryItem `json:"skipCandidates"`
	PreferredDestinations   []string                       `json:"preferredDestinations"`
	PreferredTransportModes []string                       `json:"preferredTransportModes"`
	PreferredDates          []string                       `json:"preferredDates"`
	OpenDecisionCount       int                            `json:"openDecisionCount"`
}
