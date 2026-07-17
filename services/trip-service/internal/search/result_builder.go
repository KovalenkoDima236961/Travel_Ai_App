package search

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	categoryTrips          = "Trips"
	categoryRouteTransport = "Route & transport"
	categoryItinerary      = "Itinerary"
	categoryMoney          = "Money"
	categoryPrepare        = "Prepare"
	categoryTeam           = "Team"
	categoryTemplates      = "Templates"
	categoryWorkspaces     = "Workspaces"
	categorySettings       = "Settings"
	categoryOps            = "Ops"
)

func newResult(
	resultType ResultType,
	id string,
	title string,
	description string,
	context string,
	workspaceName string,
	href string,
	metadata map[string]any,
	refs resultRefs,
) Result {
	return Result{
		ID:            id,
		Type:          resultType,
		Title:         strings.TrimSpace(title),
		Description:   strings.TrimSpace(description),
		Context:       strings.TrimSpace(context),
		WorkspaceName: strings.TrimSpace(workspaceName),
		Href:          href,
		Icon:          iconForType(resultType),
		Category:      categoryForType(resultType),
		Metadata:      metadata,
		TripID:        refs.TripID,
		WorkspaceID:   refs.WorkspaceID,
		UpdatedAt:     refs.UpdatedAt,
	}
}

type resultRefs struct {
	TripID      *uuid.UUID
	WorkspaceID *uuid.UUID
	UpdatedAt   time.Time
}

func categoryForType(resultType ResultType) string {
	switch resultType {
	case ResultTypeTrip:
		return categoryTrips
	case ResultTypeRouteStop, ResultTypeRouteLeg, ResultTypeTransportOption:
		return categoryRouteTransport
	case ResultTypeItineraryItem:
		return categoryItinerary
	case ResultTypeExpense, ResultTypeReceipt:
		return categoryMoney
	case ResultTypeChecklistItem, ResultTypeReminder:
		return categoryPrepare
	case ResultTypePoll, ResultTypeCollaborator:
		return categoryTeam
	case ResultTypeTemplate:
		return categoryTemplates
	case ResultTypeWorkspace:
		return categoryWorkspaces
	case ResultTypeSetting, ResultTypeCommand:
		return categorySettings
	case ResultTypeOpsPage:
		return categoryOps
	default:
		return "Other"
	}
}

func iconForType(resultType ResultType) string {
	switch resultType {
	case ResultTypeTrip:
		return "map"
	case ResultTypeWorkspace:
		return "workspace"
	case ResultTypeTemplate:
		return "template"
	case ResultTypeItineraryItem:
		return "calendar"
	case ResultTypeRouteStop:
		return "map-pin"
	case ResultTypeRouteLeg, ResultTypeTransportOption:
		return "route"
	case ResultTypeExpense:
		return "receipt-text"
	case ResultTypeReceipt:
		return "receipt"
	case ResultTypeChecklistItem:
		return "check-square"
	case ResultTypeReminder:
		return "bell"
	case ResultTypePoll:
		return "vote"
	case ResultTypeCollaborator:
		return "users"
	case ResultTypeNotification:
		return "inbox"
	case ResultTypeSetting:
		return "settings"
	case ResultTypeCommand:
		return "command"
	case ResultTypeOpsPage:
		return "activity"
	default:
		return "search"
	}
}

func tripHref(tripID uuid.UUID) string {
	return "/trips/" + tripID.String()
}

func tripTabHref(tripID uuid.UUID, tab string, values map[string]string) string {
	query := url.Values{}
	query.Set("tab", tab)
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			query.Set(key, value)
		}
	}
	return fmt.Sprintf("/trips/%s?%s", tripID.String(), query.Encode())
}

func templateHref(templateID uuid.UUID) string {
	return "/templates/" + templateID.String()
}

func workspaceHref(workspaceID uuid.UUID) string {
	return "/workspaces/" + workspaceID.String()
}

func idMetadata(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			out[key] = value
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func shortText(value string, max int) string {
	value = strings.TrimSpace(strings.Join(strings.Fields(value), " "))
	if max <= 0 || len(value) <= max {
		return value
	}
	if max < 4 {
		return value[:max]
	}
	return strings.TrimSpace(value[:max-1]) + "…"
}
