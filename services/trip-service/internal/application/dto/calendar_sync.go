package dto

import "time"

type TripCalendarSyncStatus struct {
	Provider                 string     `json:"provider"`
	Connected                bool       `json:"connected"`
	ProviderAccountEmail     *string    `json:"providerAccountEmail,omitempty"`
	Synced                   bool       `json:"synced"`
	LastSyncedAt             *time.Time `json:"lastSyncedAt,omitempty"`
	SyncedItineraryRevision  int        `json:"syncedItineraryRevision,omitempty"`
	CurrentItineraryRevision int        `json:"currentItineraryRevision"`
	OutOfDate                bool       `json:"outOfDate"`
	EventCount               int        `json:"eventCount"`
}

type TripCalendarSyncResult struct {
	Provider          string     `json:"provider"`
	Status            string     `json:"status"`
	Created           int        `json:"created"`
	Updated           int        `json:"updated"`
	Deleted           int        `json:"deleted"`
	Failed            int        `json:"failed"`
	Skipped           int        `json:"skipped"`
	ItineraryRevision int        `json:"itineraryRevision"`
	LastSyncedAt      *time.Time `json:"lastSyncedAt,omitempty"`
}

type TripCalendarDeleteResult struct {
	Provider string `json:"provider"`
	Deleted  int    `json:"deleted"`
	Failed   int    `json:"failed"`
}
