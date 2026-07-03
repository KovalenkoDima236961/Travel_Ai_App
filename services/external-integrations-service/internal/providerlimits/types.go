package providerlimits

// Provider categories managed by the guard. Each category has exactly one
// active provider at a time, selected by service configuration.
const (
	CategoryPlaces       = "places"
	CategoryRoutes       = "routes"
	CategoryWeather      = "weather"
	CategoryCalendar     = "calendar"
	CategoryExchangeRate = "exchange_rate"
	CategoryPrice        = "price"
)

// Bounded operation names. Operations are globally unique, so the category can
// always be derived from the operation alone (see CategoryForOperation).
const (
	OpPlaceSearch         = "place_search"
	OpPlaceDetails        = "place_details"
	OpRouteEstimate       = "route_estimate"
	OpWeatherForecast     = "weather_forecast"
	OpCalendarEventCreate = "calendar_event_create"
	OpCalendarEventUpdate = "calendar_event_update"
	OpCalendarEventDelete = "calendar_event_delete"
	OpCalendarSync        = "calendar_sync"
	OpExchangeRateLatest  = "exchange_rate_latest"
	OpExchangeRateConvert = "exchange_rate_convert"
	OpPriceEstimate       = "price_estimate"
)

// operationCategory maps every bounded operation to its provider category.
var operationCategory = map[string]string{
	OpPlaceSearch:         CategoryPlaces,
	OpPlaceDetails:        CategoryPlaces,
	OpRouteEstimate:       CategoryRoutes,
	OpWeatherForecast:     CategoryWeather,
	OpCalendarEventCreate: CategoryCalendar,
	OpCalendarEventUpdate: CategoryCalendar,
	OpCalendarEventDelete: CategoryCalendar,
	OpCalendarSync:        CategoryCalendar,
	OpExchangeRateLatest:  CategoryExchangeRate,
	OpExchangeRateConvert: CategoryExchangeRate,
	OpPriceEstimate:       CategoryPrice,
}

// categoryOperations lists the operations that belong to each category, used to
// aggregate operation-level usage into a per-category total for the Ops view.
var categoryOperations = map[string][]string{
	CategoryPlaces:       {OpPlaceSearch, OpPlaceDetails},
	CategoryRoutes:       {OpRouteEstimate},
	CategoryWeather:      {OpWeatherForecast},
	CategoryCalendar:     {OpCalendarEventCreate, OpCalendarEventUpdate, OpCalendarEventDelete, OpCalendarSync},
	CategoryExchangeRate: {OpExchangeRateLatest, OpExchangeRateConvert},
	CategoryPrice:        {OpPriceEstimate},
}

// CategoryForOperation returns the provider category for a bounded operation, or
// "" when the operation is unknown.
func CategoryForOperation(operation string) string {
	return operationCategory[operation]
}

// OperationsForCategory returns the bounded operations belonging to a category,
// used by the Ops layer to aggregate operation-level usage per category.
func OperationsForCategory(category string) []string {
	ops := categoryOperations[category]
	out := make([]string, len(ops))
	copy(out, ops)
	return out
}

// ProviderCall describes a single guarded provider request.
type ProviderCall struct {
	// Provider is the active provider name (e.g. "ors", "mock"). Used for
	// metrics and Ops display only.
	Provider string
	// Operation is one of the bounded operation names above and determines the
	// category, rate limit, and daily quota that apply.
	Operation string
	// Cost is the number of quota units this call consumes. v1 uses 1.
	Cost int64
	// AllowFallback records whether the caller can fall back to mock/cache when
	// limited. It is advisory metadata for logs; the caller performs the
	// fallback itself based on the returned Decision.
	AllowFallback bool
}

// Decision is the guard's verdict for a ProviderCall.
type Decision struct {
	// Allowed is true when the real provider call may proceed.
	Allowed bool
	// Limited is true when the in-memory per-minute rate limit blocked the call.
	Limited bool
	// QuotaExceeded is true when the daily quota blocked the call.
	QuotaExceeded bool
	// Unavailable is true when the quota store failed and fail-open is off.
	Unavailable bool
	// RetryAfterSeconds is a safe hint for when the caller may retry.
	RetryAfterSeconds int
	// Reason is a short machine token: allowed, rate_limited, quota_exceeded,
	// limits_unavailable, disabled, fail_open.
	Reason    string
	Provider  string
	Operation string
	Category  string
	// DailyQuota is 0 when the provider is unlimited.
	DailyQuota     int64
	DailyUsed      int64
	DailyRemaining int64
}

// Result reasons.
const (
	ReasonAllowed           = "allowed"
	ReasonRateLimited       = "rate_limited"
	ReasonQuotaExceeded     = "quota_exceeded"
	ReasonLimitsUnavailable = "limits_unavailable"
	ReasonDisabled          = "disabled"
	ReasonFailOpen          = "fail_open"
)
