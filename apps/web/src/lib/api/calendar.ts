import { apiFetch } from "@/lib/api/client";
import { getExternalIntegrationsApiBaseUrl } from "@/lib/config";
import type {
  CalendarConnectionStatus,
  TripCalendarDeleteResult,
  TripCalendarSyncResult,
  TripCalendarSyncStatus
} from "@/types/calendar-sync";

export const calendarKeys = {
  all: ["calendar-sync"] as const,
  googleConnection: () => [...calendarKeys.all, "google-connection"] as const,
  tripGoogleStatus: (tripId: string) =>
    [...calendarKeys.all, "trip", tripId, "google-status"] as const
};

export function getGoogleCalendarStatus() {
  return apiFetch<CalendarConnectionStatus>(
    "/calendar/google/status",
    {},
    {
      baseUrl: getExternalIntegrationsApiBaseUrl(),
      serviceName: "External Integrations Service"
    }
  );
}

export function startGoogleCalendarConnect(returnUrl: string) {
  return apiFetch<{ authUrl: string }>(
    "/calendar/google/connect",
    {
      method: "POST",
      body: JSON.stringify({ returnUrl })
    },
    {
      baseUrl: getExternalIntegrationsApiBaseUrl(),
      serviceName: "External Integrations Service"
    }
  );
}

export function disconnectGoogleCalendar() {
  return apiFetch<{ success: boolean }>(
    "/calendar/google/disconnect",
    { method: "DELETE" },
    {
      baseUrl: getExternalIntegrationsApiBaseUrl(),
      serviceName: "External Integrations Service"
    }
  );
}

export function getTripGoogleCalendarSyncStatus(tripId: string) {
  return apiFetch<TripCalendarSyncStatus>(`/trips/${tripId}/calendar-sync/google/status`);
}

export function syncTripToGoogleCalendar(
  tripId: string,
  expectedItineraryRevision: number
) {
  return apiFetch<TripCalendarSyncResult>(`/trips/${tripId}/calendar-sync/google/sync`, {
    method: "POST",
    body: JSON.stringify({ expectedItineraryRevision })
  });
}

export function removeTripGoogleCalendarSync(tripId: string) {
  return apiFetch<TripCalendarDeleteResult>(`/trips/${tripId}/calendar-sync/google`, {
    method: "DELETE"
  });
}
