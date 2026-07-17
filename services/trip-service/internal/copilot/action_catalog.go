package copilot

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
)

type actionDefinition struct {
	typeName   string
	label      string
	tab        string
	permission tripsecurity.TripPermission
	risk       ActionRisk
}

var actionDefinitions = []actionDefinition{
	{"open_command_center", "Open Command Center", "overview", tripsecurity.PermissionCommandCenterView, RiskSafeNavigation},
	{"open_trip_health", "Open Trip Health", "health", tripsecurity.PermissionHealthView, RiskSafeNavigation},
	{"open_route", "Open Route & Transport", "route", tripsecurity.PermissionRouteView, RiskSafeNavigation},
	{"open_route_leg", "Open Route & Transport", "route", tripsecurity.PermissionRouteView, RiskSafeNavigation},
	{"find_transport", "Find transport", "route", tripsecurity.PermissionRouteEdit, RiskMediumReview},
	{"open_budget", "Open Budget", "budget", tripsecurity.PermissionBudgetView, RiskSafeNavigation},
	{"open_budget_confidence", "Open Budget Confidence", "budget", tripsecurity.PermissionBudgetView, RiskSafeNavigation},
	{"open_expenses", "Open Expenses", "expenses", tripsecurity.PermissionExpensesView, RiskSafeNavigation},
	{"upload_receipt", "Upload receipt", "receipts", tripsecurity.PermissionReceiptsUpload, RiskLowRiskPrepare},
	{"add_expense", "Add expense", "expenses", tripsecurity.PermissionExpensesEdit, RiskLowRiskPrepare},
	{"open_checklist", "Open Checklist", "checklist", tripsecurity.PermissionTripView, RiskSafeNavigation},
	{"generate_checklist_screen", "Open checklist generator", "checklist", tripsecurity.PermissionTripEdit, RiskLowRiskPrepare},
	{"open_reminders", "Open Reminders", "reminders", tripsecurity.PermissionTripView, RiskSafeNavigation},
	{"open_group_readiness", "Open Group Readiness", "group-readiness", tripsecurity.PermissionGroupReadinessView, RiskSafeNavigation},
	{"request_availability_screen", "Open availability", "availability", tripsecurity.PermissionTripView, RiskSafeNavigation},
	{"open_polls", "Open Polls", "polls", tripsecurity.PermissionTripView, RiskSafeNavigation},
	{"open_approval", "Open Approval", "approval", tripsecurity.PermissionApprovalView, RiskSafeNavigation},
	{"open_policy", "Open Policy", "policy", tripsecurity.PermissionPolicyView, RiskSafeNavigation},
	{"open_itinerary", "Open Itinerary", "itinerary", tripsecurity.PermissionItineraryView, RiskSafeNavigation},
	{"open_itinerary_day", "Open Itinerary", "itinerary", tripsecurity.PermissionItineraryView, RiskSafeNavigation},
	{"open_generation_quality", "Open Generation Quality", "itinerary", tripsecurity.PermissionItineraryView, RiskSafeNavigation},
	{"open_version_history", "Open Version History", "versions", tripsecurity.PermissionItineraryView, RiskSafeNavigation},
	{"open_share_settings", "Open Share Settings", "sharing", tripsecurity.PermissionShareManage, RiskMediumReview},
	{"open_offline_settings", "Open Offline Settings", "offline", tripsecurity.PermissionTripView, RiskSafeNavigation},
	{"open_notification_settings", "Open Notification Settings", "settings", tripsecurity.PermissionTripView, RiskSafeNavigation},
	{"open_settings", "Open Settings", "settings", tripsecurity.PermissionTripView, RiskSafeNavigation},
	{"open_search", "Open Search", "overview", tripsecurity.PermissionTripView, RiskSafeNavigation},
}

func AvailableActions(tripID uuid.UUID, access service.TripAccess, client ClientContext) []Action {
	actions := make([]Action, 0, len(actionDefinitions))
	for _, definition := range actionDefinitions {
		if definition.risk == RiskHighMutation || !access.Allows(definition.permission) {
			continue
		}
		actions = append(actions, Action{
			Type:  definition.typeName,
			Label: definition.label,
			Href:  actionHref(tripID, definition.typeName, definition.tab, client),
			Style: ActionStyleSecondary,
		})
	}
	return actions
}

func actionHref(tripID uuid.UUID, typeName, tab string, client ClientContext) string {
	values := url.Values{}
	values.Set("tab", tab)
	if typeName == "open_route_leg" && safeIdentifier(client.SelectedRouteLegID) {
		values.Set("legId", client.SelectedRouteLegID)
	}
	if typeName == "open_itinerary_day" && client.SelectedDayNumber != nil && *client.SelectedDayNumber > 0 {
		values.Set("day", fmt.Sprintf("%d", *client.SelectedDayNumber))
	}
	return "/trips/" + tripID.String() + "?" + values.Encode()
}

func safeIdentifier(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func preferredActions(intent Intent, actions []Action) []Action {
	wanted := map[Intent][]string{
		IntentNextAction:            {"open_command_center", "open_trip_health", "open_route"},
		IntentExplainHealth:         {"open_trip_health", "open_command_center"},
		IntentExplainBudget:         {"open_budget_confidence", "open_budget"},
		IntentExplainRoute:          {"open_route_leg", "find_transport", "open_route"},
		IntentExplainGroupReadiness: {"open_group_readiness", "request_availability_screen"},
		IntentExplainChecklist:      {"open_checklist", "open_reminders", "generate_checklist_screen"},
		IntentExplainExpenses:       {"open_expenses", "upload_receipt", "add_expense"},
		IntentExplainApproval:       {"open_approval", "open_policy"},
		IntentHowTo:                 {"open_share_settings", "upload_receipt", "open_checklist", "open_search"},
		IntentExplainFeature:        {"open_share_settings", "open_offline_settings", "open_generation_quality"},
		IntentUnsafeMutationRequest: {"open_share_settings", "open_version_history", "open_command_center"},
	}
	byType := make(map[string]Action, len(actions))
	for _, action := range actions {
		byType[action.Type] = action
	}
	result := make([]Action, 0, 2)
	for _, typeName := range wanted[intent] {
		if action, ok := byType[typeName]; ok {
			if len(result) == 0 {
				action.Style = ActionStylePrimary
			}
			result = append(result, action)
			if len(result) == 2 {
				return result
			}
		}
	}
	if len(result) == 0 {
		for _, action := range actions {
			if strings.HasPrefix(action.Type, "open_") {
				action.Style = ActionStylePrimary
				return []Action{action}
			}
		}
	}
	return result
}

func actionByType(actions []Action, typeName string) (Action, bool) {
	for _, action := range actions {
		if action.Type == typeName {
			return action, true
		}
	}
	return Action{}, false
}

func actionRisk(typeName string) ActionRisk {
	for _, definition := range actionDefinitions {
		if definition.typeName == typeName {
			return definition.risk
		}
	}
	return RiskHighMutation
}
