package request

import (
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateTripPoll struct {
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	PollType           entity.PollType        `json:"pollType"`
	AllowMultipleVotes bool                   `json:"allowMultipleVotes"`
	ClosesAt           *time.Time             `json:"closesAt"`
	Metadata           map[string]any         `json:"metadata"`
	Options            []CreateTripPollOption `json:"options"`
}

type CreateTripPollOption struct {
	OptionKey   string         `json:"optionKey"`
	Label       string         `json:"label"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

func (r CreateTripPoll) ToInput() appdto.CreateTripPollInput {
	options := make([]appdto.CreateTripPollOptionInput, 0, len(r.Options))
	for _, option := range r.Options {
		options = append(options, appdto.CreateTripPollOptionInput{
			OptionKey:   option.OptionKey,
			Label:       option.Label,
			Description: option.Description,
			Metadata:    option.Metadata,
		})
	}
	return appdto.CreateTripPollInput{
		Title:              r.Title,
		Description:        r.Description,
		PollType:           r.PollType,
		AllowMultipleVotes: r.AllowMultipleVotes,
		ClosesAt:           r.ClosesAt,
		Metadata:           r.Metadata,
		Options:            options,
	}
}

type VoteTripPoll struct {
	OptionIDs   []uuid.UUID    `json:"optionIds"`
	VoteValue   string         `json:"voteValue"`
	RatingValue *int           `json:"ratingValue"`
	Metadata    map[string]any `json:"metadata"`
}

func (r VoteTripPoll) ToInput() appdto.VoteTripPollInput {
	return appdto.VoteTripPollInput{
		OptionIDs:   r.OptionIDs,
		VoteValue:   r.VoteValue,
		RatingValue: r.RatingValue,
		Metadata:    r.Metadata,
	}
}

type SetItineraryItemReaction struct {
	DayNumber int                      `json:"dayNumber"`
	ItemIndex int                      `json:"itemIndex"`
	ItemID    string                   `json:"itemId"`
	Reaction  entity.ItineraryReaction `json:"reaction"`
	Metadata  map[string]any           `json:"metadata"`
}

func (r SetItineraryItemReaction) ToInput() appdto.SetItineraryItemReactionInput {
	return appdto.SetItineraryItemReactionInput{
		DayNumber: r.DayNumber,
		ItemIndex: r.ItemIndex,
		ItemID:    r.ItemID,
		Reaction:  r.Reaction,
		Metadata:  r.Metadata,
	}
}
