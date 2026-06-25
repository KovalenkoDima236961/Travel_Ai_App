// Package notifications holds the Notification Service use cases: creating
// notifications (via the internal batch endpoint) and serving a user's own
// notification list, unread count, and read-state changes.
package notifications

// Notification type constants. Use these instead of scattered string literals so
// the vocabulary stays consistent across the create path, the API, and tests.
// They mirror the Trip Service activity vocabulary where the two overlap.
const (
	// Collaboration.
	TypeCollaborationInvited   = "collaboration_invited"
	TypeCollaborationAccepted  = "collaboration_accepted"
	TypeCollaboratorRoleChange = "collaborator_role_changed"
	TypeCollaboratorRemoved    = "collaborator_removed"

	// Comments.
	TypeCommentCreated = "comment_created"

	// Itinerary.
	TypeItineraryUpdated   = "itinerary_updated"
	TypeItineraryGenerated = "itinerary_generated"
	TypeDayRegenerated     = "day_regenerated"
	TypeItemRegenerated    = "item_regenerated"
	TypeVersionRestored    = "version_restored"
)

// Entity type constants describe the kind of object a notification refers to.
const (
	EntityTrip             = "trip"
	EntityComment          = "comment"
	EntityCollaborator     = "collaborator"
	EntityItinerary        = "itinerary"
	EntityItineraryDay     = "itinerary_day"
	EntityItineraryItem    = "itinerary_item"
	EntityItineraryVersion = "itinerary_version"
)

// knownTypes is the set of notification types this version recognises. The
// internal create endpoint rejects unknown types so a typo at a caller never
// silently lands an un-renderable notification in a user's inbox.
var knownTypes = map[string]struct{}{
	TypeCollaborationInvited:   {},
	TypeCollaborationAccepted:  {},
	TypeCollaboratorRoleChange: {},
	TypeCollaboratorRemoved:    {},
	TypeCommentCreated:         {},
	TypeItineraryUpdated:       {},
	TypeItineraryGenerated:     {},
	TypeDayRegenerated:         {},
	TypeItemRegenerated:        {},
	TypeVersionRestored:        {},
}

// IsKnownType reports whether the notification type is part of the recognised
// vocabulary.
func IsKnownType(notificationType string) bool {
	_, ok := knownTypes[notificationType]
	return ok
}
