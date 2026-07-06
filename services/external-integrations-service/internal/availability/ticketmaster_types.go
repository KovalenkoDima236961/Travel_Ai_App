package availability

// This file isolates the Ticketmaster Discovery API v2 response shapes. Only the
// fields the adapter consumes are modelled; everything else is ignored on
// decode. Keeping these types here means the rest of the service never sees a
// provider-specific payload — the mapper translates them into the canonical
// AvailabilitySearchResult. Shapes verified against the current Discovery API v2
// docs (Event Search, GET /discovery/v2/events.json).

const ticketmasterProviderName = "ticketmaster"

const ticketmasterDisplayName = "Ticketmaster"

// tmEventsResponse is the top-level Event Search payload.
type tmEventsResponse struct {
	Embedded tmEmbedded `json:"_embedded"`
	Page     tmPage     `json:"page"`
}

type tmEmbedded struct {
	Events []tmEvent `json:"events"`
}

type tmPage struct {
	TotalElements int `json:"totalElements"`
}

// tmEvent is one discovered event. Ticketmaster returns latitude/longitude as
// strings and localTime as an "HH:MM:SS" string, so both are parsed in the
// mapper rather than typed here.
type tmEvent struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	URL             string             `json:"url"`
	Dates           tmDates            `json:"dates"`
	PriceRanges     []tmPriceRange     `json:"priceRanges"`
	Classifications []tmClassification `json:"classifications"`
	Embedded        tmEventEmbedded    `json:"_embedded"`
}

type tmEventEmbedded struct {
	Venues []tmVenue `json:"venues"`
}

type tmDates struct {
	Start  tmDateStart  `json:"start"`
	Status tmDateStatus `json:"status"`
}

type tmDateStart struct {
	LocalDate string `json:"localDate"`
	LocalTime string `json:"localTime"`
	DateTime  string `json:"dateTime"`
}

// tmDateStatus.Code is one of: onsale, offsale, canceled, postponed, rescheduled.
type tmDateStatus struct {
	Code string `json:"code"`
}

type tmPriceRange struct {
	Type     string  `json:"type"`
	Currency string  `json:"currency"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
}

type tmClassification struct {
	Segment  tmNamed `json:"segment"`
	Genre    tmNamed `json:"genre"`
	SubGenre tmNamed `json:"subGenre"`
}

type tmNamed struct {
	Name string `json:"name"`
}

type tmVenue struct {
	Name     string        `json:"name"`
	City     tmNamed       `json:"city"`
	Address  tmAddress     `json:"address"`
	Location tmGeoLocation `json:"location"`
}

type tmAddress struct {
	Line1 string `json:"line1"`
}

// tmGeoLocation carries coordinates as strings ("48.2082"), matching the API.
type tmGeoLocation struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}
