package response

import (
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
)

// TripActivityEvent is the JSON representation of one activity-feed event.
// actorUserId/entityType/entityId are nullable. The client renders actor labels
// ("You" / "Collaborator") from actorUserId; no author display names in v1.
type TripActivityEvent = activity.EventDTO

// TripActivity is the envelope returned by GET /trips/{id}/activity. Items is
// always a (possibly empty) slice so it serialises as [] rather than null;
// nextCursor is omitted when there are no older events.
type TripActivity struct {
	Items      []TripActivityEvent `json:"items"`
	NextCursor *string             `json:"nextCursor"`
}

// NewTripActivity maps an activity page to its API representation.
func NewTripActivity(result *activity.ListActivityResult) TripActivity {
	items := make([]TripActivityEvent, 0)
	var nextCursor *string
	if result != nil {
		items = make([]TripActivityEvent, 0, len(result.Events))
		for i := range result.Events {
			items = append(items, activity.NewEventDTO(result.Events[i]))
		}
		if result.NextCursor != "" {
			cursor := result.NextCursor
			nextCursor = &cursor
		}
	}
	return TripActivity{Items: items, NextCursor: nextCursor}
}
