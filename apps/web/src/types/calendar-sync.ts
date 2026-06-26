export type CalendarConnectionStatus = {
  connected: boolean;
  provider: "google";
  providerAccountEmail?: string | null;
  connectedAt?: string | null;
  scopes?: string | null;
};

export type TripCalendarSyncStatus = {
  provider: "google";
  connected: boolean;
  providerAccountEmail?: string | null;
  synced: boolean;
  lastSyncedAt?: string | null;
  syncedItineraryRevision?: number;
  currentItineraryRevision: number;
  outOfDate: boolean;
  eventCount: number;
};

export type TripCalendarSyncResult = {
  provider: "google";
  status: "synced" | "no_timed_items";
  created: number;
  updated: number;
  deleted: number;
  failed: number;
  skipped?: number;
  itineraryRevision: number;
  lastSyncedAt?: string | null;
};

export type TripCalendarDeleteResult = {
  provider: "google";
  deleted: number;
  failed: number;
};
