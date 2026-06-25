package presence

// PresenceSnapshot is the full SSE payload sent to trip viewers.
type PresenceSnapshot struct {
	TripID string         `json:"tripId"`
	Users  []PresenceUser `json:"users"`
}

// PresenceUser is one collapsed user presence row. Multiple tabs/devices for
// the same user are represented as one row.
type PresenceUser struct {
	UserID      string  `json:"userId"`
	DisplayName *string `json:"displayName,omitempty"`
	Role        string  `json:"role"`
	State       string  `json:"state"`
	ConnectedAt string  `json:"connectedAt"`
	LastSeenAt  string  `json:"lastSeenAt"`
}

type heartbeatPayload struct {
	Timestamp string `json:"ts"`
}
