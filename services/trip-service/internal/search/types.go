package search

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type ResultType string

const (
	ResultTypeTrip            ResultType = "trip"
	ResultTypeWorkspace       ResultType = "workspace"
	ResultTypeTemplate        ResultType = "template"
	ResultTypeItineraryItem   ResultType = "itinerary_item"
	ResultTypeRouteStop       ResultType = "route_stop"
	ResultTypeRouteLeg        ResultType = "route_leg"
	ResultTypeTransportOption ResultType = "transport_option"
	ResultTypeExpense         ResultType = "expense"
	ResultTypeReceipt         ResultType = "receipt"
	ResultTypeChecklistItem   ResultType = "checklist_item"
	ResultTypeReminder        ResultType = "reminder"
	ResultTypePoll            ResultType = "poll"
	ResultTypeCollaborator    ResultType = "collaborator"
	ResultTypeNotification    ResultType = "notification"
	ResultTypeSetting         ResultType = "setting"
	ResultTypeCommand         ResultType = "command"
	ResultTypeOpsPage         ResultType = "ops_page"
)

type Scope string

const (
	ScopeAll         Scope = "all"
	ScopeTrips       Scope = "trips"
	ScopeCurrentTrip Scope = "current_trip"
	ScopeWorkspace   Scope = "workspace"
	ScopeOps         Scope = "ops"
)

func ParseScope(raw string) (Scope, bool) {
	switch Scope(strings.TrimSpace(raw)) {
	case "", ScopeAll:
		return ScopeAll, true
	case ScopeTrips:
		return ScopeTrips, true
	case ScopeCurrentTrip:
		return ScopeCurrentTrip, true
	case ScopeWorkspace:
		return ScopeWorkspace, true
	case ScopeOps:
		return ScopeOps, true
	default:
		return ScopeAll, false
	}
}

type Config struct {
	Enabled          bool
	DefaultLimit     int
	MaxLimit         int
	PerCategoryLimit int
	MinQueryLength   int
	QueryTimeout     time.Duration
}

func NormalizeConfig(cfg Config) Config {
	if cfg.DefaultLimit <= 0 {
		cfg.DefaultLimit = 20
	}
	if cfg.MaxLimit <= 0 {
		cfg.MaxLimit = 50
	}
	if cfg.MaxLimit < cfg.DefaultLimit {
		cfg.MaxLimit = cfg.DefaultLimit
	}
	if cfg.PerCategoryLimit <= 0 {
		cfg.PerCategoryLimit = 5
	}
	if cfg.MinQueryLength <= 0 {
		cfg.MinQueryLength = 2
	}
	if cfg.QueryTimeout <= 0 {
		cfg.QueryTimeout = 3 * time.Second
	}
	return cfg
}

type Params struct {
	Query           string
	Scope           Scope
	TripID          *uuid.UUID
	WorkspaceID     *uuid.UUID
	Limit           int
	IncludeCommands bool
}

type RepositorySearchParams struct {
	UserID           uuid.UUID
	Query            string
	Tokens           []string
	Patterns         []string
	Scope            Scope
	TripID           *uuid.UUID
	WorkspaceID      *uuid.UUID
	WorkspaceIDs     []uuid.UUID
	WorkspaceNames   map[uuid.UUID]string
	CurrentTripID    *uuid.UUID
	Limit            int
	PerCategoryLimit int
}
