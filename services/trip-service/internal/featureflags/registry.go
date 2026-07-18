// Package featureflags implements the small, central runtime-control registry
// owned by Trip Service. It intentionally supports only explicitly registered
// flags: runtime controls must never become a second configuration or secret
// store.
package featureflags

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type ValueType string

const (
	ValueTypeBoolean ValueType = "boolean"
	ValueTypeString  ValueType = "string"
	ValueTypeInt     ValueType = "int"
)

// Definition is the reviewed, source-controlled contract for a flag.
type Definition struct {
	Key                        string
	Type                       ValueType
	Default                    bool
	LocalDefault               bool
	ProductionDefault          bool
	SafeForFrontend            bool
	RequiresBackendEnforcement bool
	Category                   string
	Owner                      string
	Description                string
}

const (
	AIGenerationEnabled        = "ai_generation_enabled"
	AIRepairEnabled            = "ai_repair_enabled"
	CopilotEnabled             = "copilot_enabled"
	RouteAlternativesEnabled   = "route_alternatives_enabled"
	TemplateAdaptationEnabled  = "template_adaptation_enabled"
	PublicSharingEnabled       = "public_sharing_enabled"
	DataExportsEnabled         = "data_exports_enabled"
	RealProvidersEnabled       = "real_providers_enabled"
	CalendarSyncEnabled        = "calendar_sync_enabled"
	AvailabilitySearchEnabled  = "availability_search_enabled"
	TransportSearchEnabled     = "transport_search_enabled"
	ReceiptOCREnabled          = "receipt_ocr_enabled"
	WorkspaceApprovalsEnabled  = "workspace_approvals_enabled"
	PolicyRepairEnabled        = "policy_repair_enabled"
	WebPushEnabled             = "web_push_enabled"
	EmailNotificationsEnabled  = "email_notifications_enabled"
	NotificationDigestsEnabled = "notification_digests_enabled"
	OfflineModeEnabled         = "offline_mode_enabled"
	OpsDashboardEnabled        = "ops_dashboard_enabled"
)

var registry = map[string]Definition{
	AIGenerationEnabled:        boolean(AIGenerationEnabled, true, true, "ai", "Create or regenerate an itinerary."),
	AIRepairEnabled:            boolean(AIRepairEnabled, false, true, "ai", "Create or apply AI repair proposals."),
	CopilotEnabled:             boolean(CopilotEnabled, true, true, "ai", "Use the trip Copilot."),
	RouteAlternativesEnabled:   boolean(RouteAlternativesEnabled, true, true, "ai", "Create or apply route alternatives."),
	TemplateAdaptationEnabled:  boolean(TemplateAdaptationEnabled, true, true, "ai", "Create template-adaptation jobs."),
	PublicSharingEnabled:       boolean(PublicSharingEnabled, false, true, "sharing", "Create or change public trip shares."),
	DataExportsEnabled:         boolean(DataExportsEnabled, true, true, "sharing", "Create private data exports."),
	RealProvidersEnabled:       boolean(RealProvidersEnabled, false, true, "integrations", "Permit calls to real provider APIs."),
	CalendarSyncEnabled:        boolean(CalendarSyncEnabled, false, true, "integrations", "Connect or sync external calendars."),
	AvailabilitySearchEnabled:  boolean(AvailabilitySearchEnabled, true, true, "integrations", "Use availability-search providers."),
	TransportSearchEnabled:     boolean(TransportSearchEnabled, true, true, "integrations", "Use transport-search providers."),
	ReceiptOCREnabled:          boolean(ReceiptOCREnabled, false, true, "integrations", "Extract data from uploaded receipts."),
	WorkspaceApprovalsEnabled:  boolean(WorkspaceApprovalsEnabled, true, true, "collaboration", "Submit or decide workspace approvals."),
	PolicyRepairEnabled:        boolean(PolicyRepairEnabled, false, true, "collaboration", "Run automated policy repair."),
	WebPushEnabled:             boolean(WebPushEnabled, false, true, "notifications", "Register or send browser push notifications."),
	EmailNotificationsEnabled:  boolean(EmailNotificationsEnabled, true, true, "notifications", "Send email notifications."),
	NotificationDigestsEnabled: boolean(NotificationDigestsEnabled, true, true, "notifications", "Manage or process notification digests."),
	OfflineModeEnabled:         boolean(OfflineModeEnabled, true, false, "pwa", "Expose optional offline UI."),
	OpsDashboardEnabled:        boolean(OpsDashboardEnabled, false, true, "ops", "Expose allowlisted operations controls."),
}

func boolean(key string, productionDefault, localDefault bool, category, description string) Definition {
	return Definition{
		Key: key, Type: ValueTypeBoolean, Default: productionDefault,
		LocalDefault: localDefault, ProductionDefault: productionDefault,
		SafeForFrontend: true, RequiresBackendEnforcement: key != OfflineModeEnabled,
		Category: category, Owner: "trip-service", Description: description,
	}
}

func DefinitionFor(key string) (Definition, bool) {
	definition, ok := registry[strings.TrimSpace(key)]
	return definition, ok
}

func Definitions() []Definition {
	definitions := make([]Definition, 0, len(registry))
	for _, definition := range registry {
		definitions = append(definitions, definition)
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Key < definitions[j].Key })
	return definitions
}

// EnvironmentDefault resolves the checked-in default and a validated env
// override. FEATURE_<FLAG_KEY_UPPER> is intentionally the only env spelling.
func EnvironmentDefault(definition Definition, environment string) (bool, string, error) {
	value := definition.ProductionDefault
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "local", "development", "test":
		value = definition.LocalDefault
	}

	envName := "FEATURE_" + strings.ToUpper(definition.Key)
	raw, exists := os.LookupEnv(envName)
	if !exists || strings.TrimSpace(raw) == "" {
		return value, "default", nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return value, "default", fmt.Errorf("%s must be boolean: %w", envName, err)
	}
	return parsed, "env", nil
}
