package emailnotifications

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/email"
)

// appName is the product name shown in emails.
const appName = "AI Travel Planner"

// ErrNoTemplate is returned when a notification type has no email template. The
// orchestrator treats it as "skip", not a send failure.
var ErrNoTemplate = errors.New("no email template for notification type")

// BuildEmailInput carries everything templates need: the notification, the
// resolved recipient, and the public web base URL used to build safe links.
type BuildEmailInput struct {
	Notification     entity.Notification
	Recipient        users.UserProfile
	PublicWebBaseURL string
}

// BuildEmailForNotification renders the subject, text body, and HTML body for a
// notification. It returns ErrNoTemplate for any type without a template so the
// caller can skip it without treating it as an error.
//
// Templates are deliberately short and contain no secrets, tokens, share
// passwords, full itinerary payloads, or comment bodies.
func BuildEmailForNotification(input BuildEmailInput) (email.EmailMessage, error) {
	n := input.Notification
	meta := n.Metadata
	greeting := greetingLine(input.Recipient.DisplayName)
	destination := destinationOr(meta, "your trip")
	base := input.PublicWebBaseURL
	tripID := tripIDFor(n, meta)

	var c emailContent
	switch n.Type {
	case notifications.TypeCollaborationInvited:
		c = emailContent{
			subject:   "You were invited to collaborate on a trip",
			greeting:  greeting,
			body:      "You were invited to collaborate on " + destination + roleSuffix(meta, "role") + ".",
			linkLabel: "Open your trips to accept the invitation:",
			linkURL:   invitationsLink(base),
		}
	case notifications.TypeCommentCreated:
		c = emailContent{
			subject:   "New comment on a trip",
			greeting:  greeting,
			body:      "A collaborator commented on " + commentLocation(meta) + commentDestinationSuffix(meta) + ".",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeCollaboratorRoleChange:
		c = emailContent{
			subject:   "Your trip role changed",
			greeting:  greeting,
			body:      roleChangeSentence(meta, destination),
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeCollaboratorRemoved:
		// No link: the recipient no longer has access to the trip.
		c = emailContent{
			subject:  "You were removed from a trip",
			greeting: greeting,
			body:     "You no longer have access to " + destination + ".",
		}
	case notifications.TypeCollaborationAccepted:
		c = emailContent{
			subject:   "Invitation accepted",
			greeting:  greeting,
			body:      "A collaborator accepted your invitation for " + destination + ".",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeDayRegenerated:
		c = emailContent{
			subject:   "Trip day regenerated",
			greeting:  greeting,
			body:      dayWord(meta) + " of " + destination + " was regenerated.",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeItemRegenerated:
		c = emailContent{
			subject:   "Trip item regenerated",
			greeting:  greeting,
			body:      "An item on " + dayWord(meta) + " of " + destination + " was regenerated.",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeVersionRestored:
		c = emailContent{
			subject:   "Itinerary version restored",
			greeting:  greeting,
			body:      "An itinerary version for " + destination + " was restored.",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeItineraryUpdated:
		c = emailContent{
			subject:   "Itinerary updated",
			greeting:  greeting,
			body:      "The itinerary for " + destination + " was updated.",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeItineraryGenerated:
		c = emailContent{
			subject:   "Itinerary ready",
			greeting:  greeting,
			body:      "The itinerary for " + destination + " is ready.",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	case notifications.TypeWorkspaceInvited:
		workspaceName := workspaceNameOr(meta, "a workspace")
		c = emailContent{
			subject:   "Workspace invitation",
			greeting:  greeting,
			body:      "You were invited to join " + workspaceName + roleSuffix(meta, "role") + ".",
			linkLabel: "Open your workspace invitations:",
			linkURL:   metadataURLOr(base, meta, "url", workspaceInvitationsLink(base)),
		}
	case notifications.TypeWorkspaceInvitationAccepted:
		workspaceName := workspaceNameOr(meta, "your workspace")
		c = emailContent{
			subject:   "Workspace invitation accepted",
			greeting:  greeting,
			body:      "A teammate accepted your invitation to " + workspaceName + ".",
			linkLabel: "Open the workspace:",
			linkURL:   metadataURLOr(base, meta, "url", workspaceLink(base, workspaceIDFor(n, meta))),
		}
	case notifications.TypeWorkspaceInvitationDeclined:
		workspaceName := workspaceNameOr(meta, "your workspace")
		c = emailContent{
			subject:  "Workspace invitation declined",
			greeting: greeting,
			body:     "A teammate declined your invitation to " + workspaceName + ".",
		}
	case notifications.TypeWorkspaceMemberRemoved:
		workspaceName := workspaceNameOr(meta, "a workspace")
		c = emailContent{
			subject:  "Workspace access removed",
			greeting: greeting,
			body:     "You no longer have access to " + workspaceName + ".",
		}
	case notifications.TypeWorkspaceRoleChanged:
		workspaceName := workspaceNameOr(meta, "a workspace")
		c = emailContent{
			subject:   "Workspace role changed",
			greeting:  greeting,
			body:      workspaceRoleChangeSentence(meta, workspaceName),
			linkLabel: "Open the workspace:",
			linkURL:   metadataURLOr(base, meta, "url", workspaceLink(base, workspaceIDFor(n, meta))),
		}
	case notifications.TypeWorkspaceTripCreated:
		workspaceName := workspaceNameOr(meta, "a workspace")
		c = emailContent{
			subject:   "New workspace trip",
			greeting:  greeting,
			body:      "A new trip was created in " + workspaceName + ".",
			linkLabel: "Open the trip:",
			linkURL:   tripLink(base, tripID),
		}
	default:
		return email.EmailMessage{}, fmt.Errorf("%w: %q", ErrNoTemplate, n.Type)
	}

	return email.EmailMessage{
		ToEmail:  input.Recipient.Email,
		ToName:   input.Recipient.DisplayName,
		Subject:  c.subject,
		TextBody: c.text(),
		HTMLBody: c.htmlBody(),
	}, nil
}

// emailContent is a small, type-agnostic email body: a greeting, one body line,
// and an optional call-to-action link. text() and htmlBody() render it.
type emailContent struct {
	subject   string
	greeting  string
	body      string
	linkLabel string
	linkURL   string
}

func (c emailContent) text() string {
	var b strings.Builder
	if c.greeting != "" {
		b.WriteString(c.greeting)
		b.WriteString("\n\n")
	}
	b.WriteString(c.body)
	if c.linkURL != "" {
		b.WriteString("\n\n")
		if c.linkLabel != "" {
			b.WriteString(c.linkLabel)
			b.WriteString("\n")
		}
		b.WriteString(c.linkURL)
	}
	b.WriteString("\n\n")
	b.WriteString(appName)
	b.WriteString("\n")
	return b.String()
}

func (c emailContent) htmlBody() string {
	var b strings.Builder
	b.WriteString("<div>")
	if c.greeting != "" {
		b.WriteString("<p>" + html.EscapeString(c.greeting) + "</p>")
	}
	b.WriteString("<p>" + html.EscapeString(c.body) + "</p>")
	if c.linkURL != "" {
		label := c.linkLabel
		if label == "" {
			label = "Open"
		}
		b.WriteString(`<p>` + html.EscapeString(label) +
			` <a href="` + html.EscapeString(c.linkURL) + `">` + html.EscapeString(c.linkURL) + `</a></p>`)
	}
	b.WriteString("<hr><p>" + html.EscapeString(appName) + "</p>")
	b.WriteString("</div>")
	return b.String()
}

// --- link builders ---

// tripIDFor prefers the authoritative entity.TripID (always set by Trip Service
// for trip-related events) and falls back to the metadata "tripId" hint, so a
// deep link is built even if a future event omits the metadata key.
func tripIDFor(n entity.Notification, meta map[string]any) string {
	if n.TripID != nil && *n.TripID != uuid.Nil {
		return n.TripID.String()
	}
	return metaString(meta, "tripId")
}

func tripLink(base, tripID string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if tripID == "" {
		return base + "/trips"
	}
	return base + "/trips/" + tripID
}

func invitationsLink(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	return base + "/trips?tab=invitations"
}

func workspaceInvitationsLink(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	return base + "/workspace-invitations"
}

func workspaceIDFor(n entity.Notification, meta map[string]any) string {
	if n.EntityType != nil && *n.EntityType == notifications.EntityWorkspace && n.EntityID != nil && *n.EntityID != uuid.Nil {
		return n.EntityID.String()
	}
	return metaString(meta, "workspaceId")
}

func workspaceLink(base, workspaceID string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if workspaceID == "" {
		return base + "/workspaces"
	}
	return base + "/workspaces/" + workspaceID
}

func metadataURLOr(base string, meta map[string]any, key, fallback string) string {
	raw := metaString(meta, key)
	if raw == "" {
		return fallback
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		return strings.TrimRight(strings.TrimSpace(base), "/") + raw
	}
	return fallback
}

// --- sentence helpers ---

func greetingLine(displayName string) string {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = "there"
	}
	return "Hi " + name + ","
}

func destinationOr(meta map[string]any, fallback string) string {
	if d := metaString(meta, "destination"); d != "" {
		return d
	}
	return fallback
}

func workspaceNameOr(meta map[string]any, fallback string) string {
	if name := metaString(meta, "workspaceName"); name != "" {
		return name
	}
	return fallback
}

// roleSuffix renders " as {role}" when a role is present, else "".
func roleSuffix(meta map[string]any, key string) string {
	if role := metaString(meta, key); role != "" {
		return " as " + role
	}
	return ""
}

// roleChangeSentence handles missing old/new roles gracefully.
func roleChangeSentence(meta map[string]any, destination string) string {
	oldRole := metaString(meta, "oldRole")
	newRole := metaString(meta, "newRole")
	switch {
	case oldRole != "" && newRole != "":
		return "Your role for " + destination + " was changed from " + oldRole + " to " + newRole + "."
	case newRole != "":
		return "Your role for " + destination + " was changed to " + newRole + "."
	default:
		return "Your role for " + destination + " was changed."
	}
}

func workspaceRoleChangeSentence(meta map[string]any, workspaceName string) string {
	oldRole := metaString(meta, "oldRole")
	newRole := metaString(meta, "newRole")
	switch {
	case oldRole != "" && newRole != "":
		return "Your role for " + workspaceName + " was changed from " + oldRole + " to " + newRole + "."
	case newRole != "":
		return "Your role for " + workspaceName + " was changed to " + newRole + "."
	default:
		return "Your role for " + workspaceName + " was changed."
	}
}

// dayWord renders "Day N" when a day number is present, else "a day".
func dayWord(meta map[string]any) string {
	if day, ok := metaInt(meta, "dayNumber"); ok {
		return "Day " + strconv.Itoa(day)
	}
	return "a day"
}

// commentLocation renders "Day N · Item", "Day N", or "an item".
func commentLocation(meta map[string]any) string {
	day, hasDay := metaInt(meta, "dayNumber")
	item := metaString(meta, "itemName")
	switch {
	case hasDay && item != "":
		return "Day " + strconv.Itoa(day) + " · " + item
	case hasDay:
		return "Day " + strconv.Itoa(day)
	case item != "":
		return item
	default:
		return "an item"
	}
}

// commentDestinationSuffix appends " in {destination}" only when the comment
// metadata carries a destination (it often does not).
func commentDestinationSuffix(meta map[string]any) string {
	if d := metaString(meta, "destination"); d != "" {
		return " in " + d
	}
	return ""
}

// --- metadata coercion ---

func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// metaInt coerces a metadata value to int. JSON numbers arrive as float64 after
// a round-trip through the wire/JSONB; ints appear directly in unit tests.
func metaInt(meta map[string]any, key string) (int, bool) {
	if meta == nil {
		return 0, false
	}
	v, ok := meta[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return int(n), true
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(x)); err == nil {
			return n, true
		}
	}
	return 0, false
}
