import { apiFetch, apiFetchPublic } from "@/lib/api/client";
import type {
  ItineraryVersionDetail,
  ListItineraryVersionsResponse
} from "@/types/itinerary-version";
import type {
  PublicShareStatus,
  PublicShareUnlockResponse,
  PublicTrip,
  TripShareInfo,
  UpdateTripShareRequest
} from "@/types/share";
import type {
  CollaborationInvitation,
  CollaboratorRole,
  SharedTripSummary,
  TripCollaborator
} from "@/types/collaboration";
import type { CreateTripInput, Itinerary, Trip, TripsListResponse } from "@/types/trip";

type ListTripsParams = {
  limit?: number;
  offset?: number;
};

export const tripKeys = {
  all: ["trips"] as const,
  lists: () => [...tripKeys.all, "list"] as const,
  list: (params: ListTripsParams) => [...tripKeys.lists(), params] as const,
  shared: () => [...tripKeys.all, "shared-with-me"] as const,
  invitations: () => ["collaboration", "invitations"] as const,
  details: () => [...tripKeys.all, "detail"] as const,
  detail: (id: string) => [...tripKeys.details(), id] as const,
  collaborators: (id: string) => [...tripKeys.detail(id), "collaborators"] as const,
  share: (id: string) => [...tripKeys.detail(id), "share"] as const,
  publicShare: (shareToken: string) => ["public-trip-share", shareToken] as const,
  publicShareStatus: (shareToken: string) =>
    ["public-trip-share", shareToken, "status"] as const,
  itineraryVersions: (tripId: string) => [...tripKeys.detail(tripId), "itinerary-versions"] as const,
  itineraryVersion: (tripId: string, versionId: string) =>
    [...tripKeys.itineraryVersions(tripId), versionId] as const
};

export function listTrips(params: ListTripsParams = {}) {
  const searchParams = new URLSearchParams();

  if (params.limit != null) {
    searchParams.set("limit", String(params.limit));
  }

  if (params.offset != null) {
    searchParams.set("offset", String(params.offset));
  }

  const query = searchParams.toString();
  return apiFetch<TripsListResponse>(`/trips${query ? `?${query}` : ""}`);
}

export function getTrip(id: string) {
  return apiFetch<Trip>(`/trips/${id}`);
}

export function listSharedTrips() {
  return apiFetch<SharedTripSummary[]>("/trips/shared-with-me");
}

export function createTrip(input: CreateTripInput) {
  return apiFetch<Trip>("/trips", {
    method: "POST",
    body: JSON.stringify(cleanCreateTripPayload(input))
  });
}

export function generateItinerary(id: string, expectedItineraryRevision: number) {
  return apiFetch<Trip | Itinerary>(`/trips/${id}/generate`, {
    method: "POST",
    body: JSON.stringify({ expectedItineraryRevision })
  });
}

export function updateTripItinerary(
  tripId: string,
  itinerary: Itinerary,
  expectedItineraryRevision: number
) {
  return apiFetch<Trip>(`/trips/${tripId}/itinerary`, {
    method: "PUT",
    body: JSON.stringify({ itinerary, expectedItineraryRevision })
  });
}

export function regenerateItineraryDay(
  tripId: string,
  dayNumber: number,
  instruction: string | undefined,
  expectedItineraryRevision: number
) {
  return apiFetch<Trip>(`/trips/${tripId}/itinerary/days/${dayNumber}/regenerate`, {
    method: "POST",
    body: JSON.stringify(cleanRegenerationPayload(instruction, expectedItineraryRevision))
  });
}

export function regenerateItineraryItem(
  tripId: string,
  dayNumber: number,
  itemIndex: number,
  instruction: string | undefined,
  expectedItineraryRevision: number
) {
  return apiFetch<Trip>(
    `/trips/${tripId}/itinerary/days/${dayNumber}/items/${itemIndex}/regenerate`,
    {
      method: "POST",
      body: JSON.stringify(cleanRegenerationPayload(instruction, expectedItineraryRevision))
    }
  );
}

export function listItineraryVersions(tripId: string) {
  return apiFetch<ListItineraryVersionsResponse>(
    `/trips/${tripId}/itinerary/versions`
  );
}

export function getItineraryVersion(tripId: string, versionId: string) {
  return apiFetch<ItineraryVersionDetail>(
    `/trips/${tripId}/itinerary/versions/${versionId}`
  );
}

export function restoreItineraryVersion(
  tripId: string,
  versionId: string,
  expectedItineraryRevision: number
) {
  return apiFetch<Trip>(
    `/trips/${tripId}/itinerary/versions/${versionId}/restore`,
    {
      method: "POST",
      body: JSON.stringify({ expectedItineraryRevision })
    }
  );
}

export function getTripShare(tripId: string) {
  return apiFetch<TripShareInfo>(`/trips/${tripId}/share`);
}

export function createTripShare(tripId: string, body?: UpdateTripShareRequest) {
  return apiFetch<TripShareInfo>(`/trips/${tripId}/share`, {
    method: "POST",
    ...(body ? { body: JSON.stringify(cleanShareSettingsPayload(body)) } : {})
  });
}

export function updateTripShare(tripId: string, body: UpdateTripShareRequest) {
  return apiFetch<TripShareInfo>(`/trips/${tripId}/share`, {
    method: "PATCH",
    body: JSON.stringify(cleanShareSettingsPayload(body))
  });
}

export function disableTripShare(tripId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/share`, {
    method: "DELETE"
  });
}

export function listTripCollaborators(tripId: string) {
  return apiFetch<TripCollaborator[]>(`/trips/${tripId}/collaborators`);
}

export function inviteTripCollaborator(
  tripId: string,
  input: { email: string; role: CollaboratorRole }
) {
  return apiFetch<TripCollaborator>(`/trips/${tripId}/collaborators`, {
    method: "POST",
    body: JSON.stringify({
      email: input.email.trim(),
      role: input.role
    })
  });
}

export function updateTripCollaboratorRole(
  tripId: string,
  collaboratorId: string,
  input: { role: CollaboratorRole }
) {
  return apiFetch<TripCollaborator>(`/trips/${tripId}/collaborators/${collaboratorId}`, {
    method: "PATCH",
    body: JSON.stringify({ role: input.role })
  });
}

export function removeTripCollaborator(tripId: string, collaboratorId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/collaborators/${collaboratorId}`, {
    method: "DELETE"
  });
}

export function listCollaborationInvitations() {
  return apiFetch<CollaborationInvitation[]>("/collaboration/invitations");
}

export function acceptCollaborationInvitation(tripId: string, collaboratorId: string) {
  return apiFetch<TripCollaborator>(`/trips/${tripId}/collaborators/${collaboratorId}/accept`, {
    method: "POST"
  });
}

export function declineCollaborationInvitation(tripId: string, collaboratorId: string) {
  return apiFetch<{ success: boolean }>(
    `/trips/${tripId}/collaborators/${collaboratorId}/decline`,
    {
      method: "POST"
    }
  );
}

export function getPublicShareStatus(shareToken: string) {
  return apiFetchPublic<PublicShareStatus>(
    `/public/trips/${encodeURIComponent(shareToken)}/status`
  );
}

export function unlockPublicShare(shareToken: string, password: string) {
  return apiFetchPublic<PublicShareUnlockResponse>(
    `/public/trips/${encodeURIComponent(shareToken)}/unlock`,
    {
      method: "POST",
      body: JSON.stringify({ password })
    }
  );
}

export function getPublicTrip(shareToken: string, accessToken?: string | null) {
  return apiFetchPublic<PublicTrip>(
    `/public/trips/${encodeURIComponent(shareToken)}`,
    accessToken
      ? {
          headers: {
            Authorization: `Bearer ${accessToken}`
          }
        }
      : {}
  );
}

function cleanCreateTripPayload(input: CreateTripInput) {
  return {
    destination: input.destination.trim(),
    ...(input.startDate ? { startDate: input.startDate } : {}),
    days: input.days,
    ...(input.budgetAmount != null ? { budgetAmount: input.budgetAmount } : {}),
    budgetCurrency: input.budgetCurrency.trim().toUpperCase(),
    travelers: input.travelers,
    interests: input.interests,
    pace: input.pace
  };
}

function cleanRegenerationPayload(instruction: string | undefined, expectedItineraryRevision: number) {
  const trimmed = instruction?.trim() ?? "";
  return trimmed ? { instruction: trimmed, expectedItineraryRevision } : { expectedItineraryRevision };
}

function cleanShareSettingsPayload(input: UpdateTripShareRequest) {
  return {
    ...(input.expiresAt ? { expiresAt: input.expiresAt } : {}),
    ...(input.clearExpiration ? { clearExpiration: true } : {}),
    ...(input.password ? { password: input.password } : {}),
    ...(input.clearPassword ? { clearPassword: true } : {})
  };
}
