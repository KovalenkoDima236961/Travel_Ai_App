package transport

import "time"

const (
	ModeTrain           = "train"
	ModeBus             = "bus"
	ModeFlight          = "flight"
	ModeFerry           = "ferry"
	ModeCar             = "car"
	ModeRentalCar       = "rental_car"
	ModePublicTransport = "public_transport"
	ModeWalk            = "walk"
	ModeBike            = "bike"
	ModeHiking          = "hiking"
	ModeBoat            = "boat"
	ModeOther           = "other"
)

var supportedModes = map[string]struct{}{
	ModeTrain:           {},
	ModeBus:             {},
	ModeFlight:          {},
	ModeFerry:           {},
	ModeCar:             {},
	ModeRentalCar:       {},
	ModePublicTransport: {},
	ModeWalk:            {},
	ModeBike:            {},
	ModeHiking:          {},
	ModeBoat:            {},
	ModeOther:           {},
}

const (
	ProviderMock          = "mock"
	ProviderRouteEstimate = "route_estimate"
	ProviderGTFSStatic    = "gtfs_static"
	ProviderAmadeus       = "amadeus"
	ProviderSkyscanner    = "skyscanner"
	ProviderRome2Rio      = "rome2rio"
	ProviderNationalRail  = "national_rail"
	ProviderFerry         = "ferry_provider"
	ProviderManual        = "manual"
)

const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

const (
	StatusAvailable   = "available"
	StatusLimited     = "limited"
	StatusUnknown     = "unknown"
	StatusUnavailable = "unavailable"
)

const (
	TimePreferenceDepartAfter  = "depart_after"
	TimePreferenceArriveBefore = "arrive_before"
	TimePreferenceFlexible     = "flexible"
)

type Location struct {
	Name    string   `json:"name"`
	Lat     *float64 `json:"lat,omitempty"`
	Lng     *float64 `json:"lng,omitempty"`
	Country string   `json:"country,omitempty"`
	StopID  string   `json:"stopId,omitempty"`
}

type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type MoneyRange struct {
	Min Money `json:"min"`
	Max Money `json:"max"`
}

type SearchConstraints struct {
	MaxDurationMinutes *int     `json:"maxDurationMinutes,omitempty"`
	MaxPriceAmount     *float64 `json:"maxPriceAmount,omitempty"`
	AvoidFlights       bool     `json:"avoidFlights"`
	PreferredModes     []string `json:"preferredModes,omitempty"`
	AccessibilityNotes *string  `json:"accessibilityNotes,omitempty"`
}

type TransportSearchRequest struct {
	Origin         Location          `json:"origin"`
	Destination    Location          `json:"destination"`
	Date           string            `json:"date"`
	Time           string            `json:"time,omitempty"`
	TimePreference string            `json:"timePreference,omitempty"`
	Travelers      int               `json:"travelers,omitempty"`
	Modes          []string          `json:"modes,omitempty"`
	Currency       string            `json:"currency,omitempty"`
	Locale         string            `json:"locale,omitempty"`
	Constraints    SearchConstraints `json:"constraints,omitempty"`
}

type TransportOption struct {
	ID                  string         `json:"id"`
	Mode                string         `json:"mode"`
	Provider            string         `json:"provider"`
	OperatorName        string         `json:"operatorName,omitempty"`
	ServiceName         string         `json:"serviceName,omitempty"`
	OriginName          string         `json:"originName,omitempty"`
	DestinationName     string         `json:"destinationName,omitempty"`
	DepartureDate       string         `json:"departureDate,omitempty"`
	DepartureTime       string         `json:"departureTime,omitempty"`
	ArrivalDate         string         `json:"arrivalDate,omitempty"`
	ArrivalTime         string         `json:"arrivalTime,omitempty"`
	DurationMinutes     int            `json:"durationMinutes"`
	Transfers           int            `json:"transfers"`
	EstimatedPrice      *Money         `json:"estimatedPrice,omitempty"`
	PriceRange          *MoneyRange    `json:"priceRange,omitempty"`
	BookingURL          *string        `json:"bookingUrl,omitempty"`
	ProviderURL         *string        `json:"providerUrl,omitempty"`
	Status              string         `json:"status"`
	Confidence          string         `json:"confidence"`
	EmissionsEstimateKg *float64       `json:"emissionsEstimateKg,omitempty"`
	BaggageNotes        *string        `json:"baggageNotes,omitempty"`
	AccessibilityNotes  *string        `json:"accessibilityNotes,omitempty"`
	Warnings            []string       `json:"warnings,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

type SearchSummary struct {
	Origin        string   `json:"origin"`
	Destination   string   `json:"destination"`
	Date          string   `json:"date"`
	SearchedModes []string `json:"searchedModes"`
	Provider      string   `json:"provider"`
	FallbackUsed  bool     `json:"fallbackUsed"`
	Cached        bool     `json:"cached"`
	Warnings      []string `json:"warnings,omitempty"`
}

type TransportSearchResponse struct {
	Options []TransportOption `json:"options"`
	Summary SearchSummary     `json:"summary"`
}

func (r TransportSearchResponse) markCached() TransportSearchResponse {
	out := cloneResponse(r)
	out.Summary.Cached = true
	return out
}

func cloneResponse(in TransportSearchResponse) TransportSearchResponse {
	out := in
	out.Options = make([]TransportOption, len(in.Options))
	for i := range in.Options {
		out.Options[i] = cloneOption(in.Options[i])
	}
	out.Summary.SearchedModes = append([]string(nil), in.Summary.SearchedModes...)
	out.Summary.Warnings = append([]string(nil), in.Summary.Warnings...)
	return out
}

func cloneOption(in TransportOption) TransportOption {
	out := in
	if in.EstimatedPrice != nil {
		price := *in.EstimatedPrice
		out.EstimatedPrice = &price
	}
	if in.PriceRange != nil {
		priceRange := *in.PriceRange
		out.PriceRange = &priceRange
	}
	if in.BookingURL != nil {
		value := *in.BookingURL
		out.BookingURL = &value
	}
	if in.ProviderURL != nil {
		value := *in.ProviderURL
		out.ProviderURL = &value
	}
	if in.EmissionsEstimateKg != nil {
		value := *in.EmissionsEstimateKg
		out.EmissionsEstimateKg = &value
	}
	if in.BaggageNotes != nil {
		value := *in.BaggageNotes
		out.BaggageNotes = &value
	}
	if in.AccessibilityNotes != nil {
		value := *in.AccessibilityNotes
		out.AccessibilityNotes = &value
	}
	out.Warnings = append([]string(nil), in.Warnings...)
	if in.Metadata != nil {
		out.Metadata = make(map[string]any, len(in.Metadata))
		for key, value := range in.Metadata {
			out.Metadata[key] = value
		}
	}
	return out
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
