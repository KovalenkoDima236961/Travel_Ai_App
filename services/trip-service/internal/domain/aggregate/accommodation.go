package aggregate

// AccommodationType is the normalized stay category stored on a trip.
type AccommodationType string

const (
	AccommodationTypeHotel      AccommodationType = "hotel"
	AccommodationTypeHostel     AccommodationType = "hostel"
	AccommodationTypeApartment  AccommodationType = "apartment"
	AccommodationTypeGuesthouse AccommodationType = "guesthouse"
	AccommodationTypeHome       AccommodationType = "home"
	AccommodationTypeOther      AccommodationType = "other"
)

// Accommodation is the structured one-stay trip accommodation stored as JSONB
// on trips. It intentionally reuses PlaceRef and EstimatedCost so place and
// budget behavior stays compatible with itinerary items.
type Accommodation struct {
	Name          string            `json:"name"`
	Type          AccommodationType `json:"type"`
	Address       string            `json:"address,omitempty"`
	Place         *PlaceRef         `json:"place,omitempty"`
	CheckInDate   string            `json:"checkInDate,omitempty"`
	CheckOutDate  string            `json:"checkOutDate,omitempty"`
	EstimatedCost *EstimatedCost    `json:"estimatedCost,omitempty"`
	Notes         string            `json:"notes,omitempty"`
}
