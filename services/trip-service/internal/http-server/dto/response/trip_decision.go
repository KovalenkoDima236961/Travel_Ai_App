package response

import (
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type TripPoll struct {
	ID                 uuid.UUID         `json:"id"`
	TripID             uuid.UUID         `json:"tripId"`
	Title              string            `json:"title"`
	Description        string            `json:"description,omitempty"`
	PollType           entity.PollType   `json:"pollType"`
	Status             entity.PollStatus `json:"status"`
	AllowMultipleVotes bool              `json:"allowMultipleVotes"`
	Options            []TripPollOption  `json:"options"`
	Results            PollResults       `json:"results"`
	UserVotes          []TripPollVote    `json:"userVotes"`
	CreatedByUserID    uuid.UUID         `json:"createdByUserId"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	ClosesAt           *time.Time        `json:"closesAt,omitempty"`
	ClosedAt           *time.Time        `json:"closedAt,omitempty"`
	ClosedByUserID     *uuid.UUID        `json:"closedByUserId,omitempty"`
	CanManage          bool              `json:"canManage"`
	CanVote            bool              `json:"canVote"`
	Metadata           map[string]any    `json:"metadata,omitempty"`
}

type TripPollOption struct {
	ID          uuid.UUID      `json:"id"`
	OptionKey   string         `json:"optionKey"`
	Label       string         `json:"label"`
	Description string         `json:"description,omitempty"`
	SortOrder   int            `json:"sortOrder"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type TripPollVote struct {
	ID          uuid.UUID      `json:"id"`
	OptionID    *uuid.UUID     `json:"optionId,omitempty"`
	VoteValue   string         `json:"voteValue,omitempty"`
	RatingValue *int           `json:"ratingValue,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type PollResults struct {
	TotalVoters      int                `json:"totalVoters"`
	TotalVotes       int                `json:"totalVotes"`
	Options          []PollOptionResult `json:"options"`
	WinningOptionIDs []uuid.UUID        `json:"winningOptionIds"`
}

type PollOptionResult struct {
	OptionID      uuid.UUID `json:"optionId"`
	OptionKey     string    `json:"optionKey"`
	Label         string    `json:"label"`
	VoteCount     int       `json:"voteCount"`
	Percentage    int       `json:"percentage"`
	AverageRating *float64  `json:"averageRating,omitempty"`
}

type ListTripPolls struct {
	Items []TripPoll `json:"items"`
}

type ItineraryItemReactionSummary struct {
	DayNumber           int            `json:"dayNumber"`
	ItemIndex           int            `json:"itemIndex"`
	ItemID              string         `json:"itemId,omitempty"`
	ItemName            string         `json:"itemName,omitempty"`
	Counts              map[string]int `json:"counts"`
	CurrentUserReaction *string        `json:"currentUserReaction,omitempty"`
	Score               int            `json:"score"`
}

type ListItineraryItemReactionSummaries struct {
	Items []ItineraryItemReactionSummary `json:"items"`
}

type GroupPreferencesSummary = appdto.GroupPreferencesSummary

func NewTripPoll(info appdto.TripPollInfo) TripPoll {
	poll := info.Poll
	options := make([]TripPollOption, 0, len(info.Options))
	for _, option := range info.Options {
		options = append(options, TripPollOption{
			ID:          option.ID,
			OptionKey:   option.OptionKey,
			Label:       option.Label,
			Description: option.Description,
			SortOrder:   option.SortOrder,
			Metadata:    option.Metadata,
			CreatedAt:   option.CreatedAt,
		})
	}
	userVotes := make([]TripPollVote, 0, len(info.UserVotes))
	for _, vote := range info.UserVotes {
		userVotes = append(userVotes, TripPollVote{
			ID:          vote.ID,
			OptionID:    vote.OptionID,
			VoteValue:   vote.VoteValue,
			RatingValue: vote.RatingValue,
			Metadata:    vote.Metadata,
			CreatedAt:   vote.CreatedAt,
			UpdatedAt:   vote.UpdatedAt,
		})
	}
	return TripPoll{
		ID:                 poll.ID,
		TripID:             poll.TripID,
		Title:              poll.Title,
		Description:        poll.Description,
		PollType:           poll.PollType,
		Status:             poll.Status,
		AllowMultipleVotes: poll.AllowMultipleVotes,
		Options:            options,
		Results:            NewPollResults(info.Results),
		UserVotes:          userVotes,
		CreatedByUserID:    poll.CreatedByUserID,
		CreatedAt:          poll.CreatedAt,
		UpdatedAt:          poll.UpdatedAt,
		ClosesAt:           poll.ClosesAt,
		ClosedAt:           poll.ClosedAt,
		ClosedByUserID:     poll.ClosedByUserID,
		CanManage:          info.CanManage,
		CanVote:            info.CanVote,
		Metadata:           poll.Metadata,
	}
}

func NewListTripPolls(infos []appdto.TripPollInfo) ListTripPolls {
	items := make([]TripPoll, 0, len(infos))
	for _, info := range infos {
		items = append(items, NewTripPoll(info))
	}
	return ListTripPolls{Items: items}
}

func NewPollResults(results appdto.PollResults) PollResults {
	options := make([]PollOptionResult, 0, len(results.Options))
	for _, option := range results.Options {
		options = append(options, PollOptionResult{
			OptionID:      option.OptionID,
			OptionKey:     option.OptionKey,
			Label:         option.Label,
			VoteCount:     option.VoteCount,
			Percentage:    option.Percentage,
			AverageRating: option.AverageRating,
		})
	}
	return PollResults{
		TotalVoters:      results.TotalVoters,
		TotalVotes:       results.TotalVotes,
		Options:          options,
		WinningOptionIDs: append([]uuid.UUID(nil), results.WinningOptionIDs...),
	}
}

func NewItineraryItemReactionSummary(summary appdto.ItineraryItemReactionSummary) ItineraryItemReactionSummary {
	counts := map[string]int{
		string(entity.ItineraryReactionMustHave): summary.Counts[entity.ItineraryReactionMustHave],
		string(entity.ItineraryReactionWantToDo): summary.Counts[entity.ItineraryReactionWantToDo],
		string(entity.ItineraryReactionNeutral):  summary.Counts[entity.ItineraryReactionNeutral],
		string(entity.ItineraryReactionSkip):     summary.Counts[entity.ItineraryReactionSkip],
	}
	var current *string
	if summary.CurrentUserReaction != nil {
		value := string(*summary.CurrentUserReaction)
		current = &value
	}
	return ItineraryItemReactionSummary{
		DayNumber:           summary.DayNumber,
		ItemIndex:           summary.ItemIndex,
		ItemID:              summary.ItemID,
		ItemName:            summary.ItemName,
		Counts:              counts,
		CurrentUserReaction: current,
		Score:               summary.Score,
	}
}

func NewListItineraryItemReactionSummaries(infos []appdto.ItineraryItemReactionSummary) ListItineraryItemReactionSummaries {
	items := make([]ItineraryItemReactionSummary, 0, len(infos))
	for _, info := range infos {
		items = append(items, NewItineraryItemReactionSummary(info))
	}
	return ListItineraryItemReactionSummaries{Items: items}
}
