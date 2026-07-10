package entity

import (
	"time"

	"github.com/google/uuid"
)

type PollType string

const (
	PollTypeSingleChoice   PollType = "single_choice"
	PollTypeMultipleChoice PollType = "multiple_choice"
	PollTypeRating         PollType = "rating"
	PollTypeYesNo          PollType = "yes_no"
	PollTypeDateChoice     PollType = "date_choice"
)

func (t PollType) Valid() bool {
	switch t {
	case PollTypeSingleChoice, PollTypeMultipleChoice, PollTypeRating, PollTypeYesNo, PollTypeDateChoice:
		return true
	default:
		return false
	}
}

type PollStatus string

const (
	PollStatusOpen     PollStatus = "open"
	PollStatusClosed   PollStatus = "closed"
	PollStatusArchived PollStatus = "archived"
)

type TripPoll struct {
	ID                 uuid.UUID
	TripID             uuid.UUID
	CreatedByUserID    uuid.UUID
	Title              string
	Description        string
	PollType           PollType
	Status             PollStatus
	AllowMultipleVotes bool
	ClosesAt           *time.Time
	Metadata           map[string]any
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ClosedAt           *time.Time
	ClosedByUserID     *uuid.UUID
}

type TripPollOption struct {
	ID          uuid.UUID
	PollID      uuid.UUID
	OptionKey   string
	Label       string
	Description string
	SortOrder   int
	Metadata    map[string]any
	CreatedAt   time.Time
}

type TripPollVote struct {
	ID          uuid.UUID
	PollID      uuid.UUID
	OptionID    *uuid.UUID
	UserID      uuid.UUID
	VoteValue   string
	RatingValue *int
	Metadata    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ItineraryReaction string

const (
	ItineraryReactionWantToDo ItineraryReaction = "want_to_do"
	ItineraryReactionNeutral  ItineraryReaction = "neutral"
	ItineraryReactionSkip     ItineraryReaction = "skip"
	ItineraryReactionMustHave ItineraryReaction = "must_have"
)

func (r ItineraryReaction) Valid() bool {
	switch r {
	case ItineraryReactionWantToDo, ItineraryReactionNeutral, ItineraryReactionSkip, ItineraryReactionMustHave:
		return true
	default:
		return false
	}
}

type ItineraryItemReaction struct {
	ID        uuid.UUID
	TripID    uuid.UUID
	DayNumber int
	ItemIndex int
	ItemID    string
	UserID    uuid.UUID
	Reaction  ItineraryReaction
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DiscoverySuggestionVoteValue string

const (
	DiscoverySuggestionVoteLike          DiscoverySuggestionVoteValue = "like"
	DiscoverySuggestionVoteDislike       DiscoverySuggestionVoteValue = "dislike"
	DiscoverySuggestionVoteFavorite      DiscoverySuggestionVoteValue = "favorite"
	DiscoverySuggestionVoteNotInterested DiscoverySuggestionVoteValue = "not_interested"
)

func (v DiscoverySuggestionVoteValue) Valid() bool {
	switch v {
	case DiscoverySuggestionVoteLike,
		DiscoverySuggestionVoteDislike,
		DiscoverySuggestionVoteFavorite,
		DiscoverySuggestionVoteNotInterested:
		return true
	default:
		return false
	}
}

type DiscoverySuggestionVote struct {
	ID           uuid.UUID
	SessionID    uuid.UUID
	SuggestionID string
	TripID       *uuid.UUID
	UserID       uuid.UUID
	Vote         DiscoverySuggestionVoteValue
	Metadata     map[string]any
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
