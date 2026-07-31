package search

import "github.com/google/uuid"

type navigationCommand struct {
	id          string
	title       string
	description string
	href        string
}

func commandResults(
	query string,
	tokens []string,
	currentTripID *uuid.UUID,
	typeFilters map[ResultType]struct{},
) []Result {
	if !typeAllowed(typeFilters, ResultTypeCommand) {
		return nil
	}
	commands := []navigationCommand{
		{id: "trip.create", title: "Create new trip", description: "Start a new AI travel plan.", href: "/trips/new"},
		{id: "trip.list", title: "Open my trips", description: "Go to your trip list.", href: "/trips"},
		{id: "trip.shared", title: "Open shared with me", description: "View trips shared with you.", href: "/trips?filter=shared"},
		{id: "library.open", title: "Open trip library", description: "Browse completed and archived trips.", href: "/library"},
		{id: "templates.open", title: "Open templates", description: "Browse reusable trip templates.", href: "/templates"},
		{id: "workspaces.open", title: "Open workspaces", description: "Switch or manage workspaces.", href: "/workspaces"},
		{id: "notifications.open", title: "Open notifications", description: "Review your notification center.", href: "/notifications"},
		{id: "settings.open", title: "Open settings", description: "Manage account and app preferences.", href: "/settings"},
		{id: "offline.open", title: "Open offline trips", description: "Review trips saved on this device.", href: "/offline-trips"},
	}
	if currentTripID != nil {
		commands = append(commands,
			navigationCommand{id: "trip.commandCenter", title: "Open Command Center", description: "Open the current trip overview.", href: tripTabHref(*currentTripID, "command_center", nil)},
			navigationCommand{id: "trip.timeline", title: "Open current trip timeline", description: "Review the current trip timeline.", href: tripTabHref(*currentTripID, "timeline", nil)},
			navigationCommand{id: "trip.budget", title: "Open current trip budget", description: "Review the current trip budget.", href: tripTabHref(*currentTripID, "budget", nil)},
			navigationCommand{id: "trip.checklist", title: "Open current trip checklist", description: "Review packing and preparation items.", href: tripTabHref(*currentTripID, "checklist", nil)},
			navigationCommand{id: "trip.expenses", title: "Open current trip expenses", description: "Review spending and receipts.", href: tripTabHref(*currentTripID, "expenses", nil)},
		)
	}

	results := make([]Result, 0, len(commands))
	for _, command := range commands {
		if query != "" && !matchesTokens(query, tokens, command.title, command.description) {
			continue
		}
		results = append(results, newResult(
			ResultTypeCommand,
			"command:"+command.id,
			command.title,
			command.description,
			"",
			"",
			command.href,
			idMetadata(map[string]string{"commandId": command.id}),
			resultRefs{},
		))
	}
	return results
}
