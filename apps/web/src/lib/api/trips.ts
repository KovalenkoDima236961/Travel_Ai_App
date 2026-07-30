import { apiFetch, apiFetchPublic } from "@/shared/api/client";
import type {
  ItineraryVersionDetail,
  ListItineraryVersionsResponse
} from "@/entities/itinerary/model";
import type {
  PublicShareStatus,
  PublicShareUnlockResponse,
  PublicTrip,
  TripShareInfo,
  UpdateTripShareRequest
} from "@/entities/share/model";
import type {
  CollaborationInvitation,
  CollaboratorRole,
  SharedTripSummary,
  TripCollaborator,
  TripInvitation,
  TripMember,
  TripSuggestion,
  TripSuggestionStatus,
  TripSuggestionTargetType,
  TripSuggestionType,
  TripVoteSummary,
  TripVoteTargetType,
  TripVoteType
} from "@/entities/collaboration/model";
import type { TripRoute } from "@/entities/route/model";
import type { CreateTripInput, Itinerary, Trip, TripScope, TripsListResponse } from "@/entities/trip/model";
import type {
  CreateTripContract,
  ExpectedRevisionContract,
  TripContract,
  TripsListContract
} from "@/lib/api/contracts";

type ContractTrip = Trip & TripContract;
type ContractTripsList = TripsListResponse & TripsListContract;

type ListTripsParams = {
  limit?: number;
  offset?: number;
  scope?: TripScope | "all";
  workspaceId?: string | null;
};

export const tripKeys = {
  all: ["trips"] as const,
  lists: () => [...tripKeys.all, "list"] as const,
  list: (params: ListTripsParams) => [...tripKeys.lists(), params] as const,
  shared: () => [...tripKeys.all, "shared-with-me"] as const,
  invitations: () => ["collaboration", "invitations"] as const,
  tripInvitations: (id: string) => [...tripKeys.detail(id), "invitations"] as const,
  details: () => [...tripKeys.all, "detail"] as const,
  detail: (id: string) => [...tripKeys.details(), id] as const,
  route: (id: string) => [...tripKeys.detail(id), "route"] as const,
  collaborators: (id: string) => [...tripKeys.detail(id), "collaborators"] as const,
  members: (id: string) => [...tripKeys.detail(id), "members"] as const,
  suggestions: (id: string, status?: TripSuggestionStatus) =>
    [...tripKeys.detail(id), "suggestions", status ?? "all"] as const,
  votes: (id: string, targetType?: TripVoteTargetType, targetId?: string) =>
    [...tripKeys.detail(id), "votes", targetType ?? "all", targetId ?? "all"] as const,
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

  if (params.scope) {
    searchParams.set("scope", params.scope);
  }

  if (params.workspaceId) {
    searchParams.set("workspaceId", params.workspaceId);
  }

  const query = searchParams.toString();
  return apiFetch<ContractTripsList>(`/trips${query ? `?${query}` : ""}`);
}

export function getTrip(id: string) {
  return apiFetch<ContractTrip>(`/trips/${id}`);
}

export function getTripRoute(id: string) {
  return apiFetch<{ route: TripRoute | null }>(`/trips/${id}/route`);
}

export function updateTripRoute(
  id: string,
  input: { expectedItineraryRevision?: number; route: TripRoute | null }
) {
  return apiFetch<ContractTrip>(`/trips/${id}/route`, {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export function listSharedTrips() {
  return apiFetch<SharedTripSummary[]>("/trips/shared-with-me");
}

export function createTrip(input: CreateTripInput) {
  return apiFetch<ContractTrip>("/trips", {
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
  const body: ExpectedRevisionContract & { itinerary: Itinerary } = {
    itinerary,
    expectedItineraryRevision
  };
  return apiFetch<ContractTrip>(`/trips/${tripId}/itinerary`, {
    method: "PUT",
    body: JSON.stringify(body)
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
  input: { email: string; role: CollaboratorRole; message?: string; expiresAt?: string | null }
) {
  return apiFetch<TripCollaborator>(`/trips/${tripId}/collaborators`, {
    method: "POST",
    body: JSON.stringify({
      email: input.email.trim(),
      role: input.role,
      ...(input.message?.trim() ? { message: input.message.trim() } : {}),
      ...(input.expiresAt ? { expiresAt: input.expiresAt } : {})
    })
  });
}

export async function listTripInvitations(tripId: string): Promise<TripInvitation[]> {
  const response = await apiFetch<{ items: TripInvitation[] }>(`/trips/${tripId}/invitations`);
  return response?.items ?? [];
}

export function createTripInvitation(
  tripId: string,
  input: { email: string; role: CollaboratorRole; message?: string; expiresAt?: string | null }
) {
  return apiFetch<TripInvitation>(`/trips/${tripId}/invitations`, {
    method: "POST",
    body: JSON.stringify({
      email: input.email.trim(),
      role: input.role,
      ...(input.message?.trim() ? { message: input.message.trim() } : {}),
      ...(input.expiresAt ? { expiresAt: input.expiresAt } : {})
    })
  });
}

export function resendTripInvitation(
  tripId: string,
  invitationId: string,
  input: { message?: string; expiresAt?: string | null } = {}
) {
  return apiFetch<TripInvitation>(`/trips/${tripId}/invitations/${invitationId}/resend`, {
    method: "POST",
    body: JSON.stringify({
      ...(input.message?.trim() ? { message: input.message.trim() } : {}),
      ...(input.expiresAt ? { expiresAt: input.expiresAt } : {})
    })
  });
}

export function revokeTripInvitation(tripId: string, invitationId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/invitations/${invitationId}/revoke`, {
    method: "POST"
  });
}

export async function listTripMembers(tripId: string): Promise<TripMember[]> {
  const response = await apiFetch<{ items: TripMember[] }>(`/trips/${tripId}/members`);
  return response?.items ?? [];
}

export function transferTripOwnership(tripId: string, newOwnerUserId: string) {
  return apiFetch<ContractTrip>(`/trips/${tripId}/members/transfer-ownership`, {
    method: "POST",
    body: JSON.stringify({ newOwnerUserId })
  });
}

export function leaveTrip(tripId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/members/leave`, {
    method: "POST"
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

export function acceptCollaborationInvitation(
  tripId: string,
  collaboratorId: string,
  invitationId?: string | null
) {
  const path = invitationId
    ? `/trips/${tripId}/invitations/${invitationId}/accept`
    : `/trips/${tripId}/collaborators/${collaboratorId}/accept`;
  return apiFetch<TripCollaborator | TripInvitation>(path, {
    method: "POST"
  });
}

export function declineCollaborationInvitation(
  tripId: string,
  collaboratorId: string,
  invitationId?: string | null
) {
  const path = invitationId
    ? `/trips/${tripId}/invitations/${invitationId}/decline`
    : `/trips/${tripId}/collaborators/${collaboratorId}/decline`;
  return apiFetch<{ success: boolean }>(
    path,
    {
      method: "POST"
    }
  );
}

export async function listTripSuggestions(
  tripId: string,
  status?: TripSuggestionStatus
): Promise<TripSuggestion[]> {
  const query = status ? `?${new URLSearchParams({ status }).toString()}` : "";
  const response = await apiFetch<{ items: TripSuggestion[] }>(
    `/trips/${tripId}/suggestions${query}`
  );
  return response?.items ?? [];
}

export function createTripSuggestion(
  tripId: string,
  input: {
    suggestionType: TripSuggestionType;
    targetType: TripSuggestionTargetType;
    targetId?: string;
    before?: unknown;
    after?: unknown;
    comment?: string;
    metadata?: Record<string, unknown>;
  }
) {
  return apiFetch<TripSuggestion>(`/trips/${tripId}/suggestions`, {
    method: "POST",
    body: JSON.stringify({
      suggestionType: input.suggestionType,
      targetType: input.targetType,
      ...(input.targetId?.trim() ? { targetId: input.targetId.trim() } : {}),
      ...(input.before !== undefined ? { before: input.before } : {}),
      ...(input.after !== undefined ? { after: input.after } : {}),
      ...(input.comment?.trim() ? { comment: input.comment.trim() } : {}),
      ...(input.metadata ? { metadata: input.metadata } : {})
    })
  });
}

export function acceptTripSuggestion(
  tripId: string,
  suggestionId: string,
  expectedItineraryRevision?: number
) {
  return apiFetch<TripSuggestion>(`/trips/${tripId}/suggestions/${suggestionId}/accept`, {
    method: "POST",
    body: JSON.stringify(
      expectedItineraryRevision != null ? { expectedItineraryRevision } : {}
    )
  });
}

export function rejectTripSuggestion(tripId: string, suggestionId: string) {
  return apiFetch<TripSuggestion>(`/trips/${tripId}/suggestions/${suggestionId}/reject`, {
    method: "POST"
  });
}

export function resolveTripSuggestion(tripId: string, suggestionId: string) {
  return apiFetch<TripSuggestion>(`/trips/${tripId}/suggestions/${suggestionId}/resolve`, {
    method: "POST"
  });
}

export async function listTripVotes(
  tripId: string,
  input: { targetType?: TripVoteTargetType; targetId?: string } = {}
): Promise<TripVoteSummary[]> {
  const query = new URLSearchParams();
  if (input.targetType) {
    query.set("targetType", input.targetType);
  }
  if (input.targetId) {
    query.set("targetId", input.targetId);
  }
  const suffix = query.toString();
  const response = await apiFetch<{ items: TripVoteSummary[] }>(
    `/trips/${tripId}/votes${suffix ? `?${suffix}` : ""}`
  );
  return response?.items ?? [];
}

export function setTripVote(
  tripId: string,
  input: {
    targetType: TripVoteTargetType;
    targetId: string;
    voteType: TripVoteType;
    metadata?: Record<string, unknown>;
  }
) {
  return apiFetch<TripVoteSummary>(`/trips/${tripId}/votes`, {
    method: "POST",
    body: JSON.stringify({
      targetType: input.targetType,
      targetId: input.targetId.trim(),
      voteType: input.voteType,
      ...(input.metadata ? { metadata: input.metadata } : {})
    })
  });
}

export function deleteTripVote(tripId: string, voteId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/votes/${voteId}`, {
    method: "DELETE"
  });
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

function cleanCreateTripPayload(input: CreateTripInput): CreateTripContract {
  return {
    destination: input.destination.trim(),
    ...(input.tripType ? { tripType: input.tripType } : {}),
    ...(input.route ? { route: input.route } : {}),
    ...(input.workspaceId ? { workspaceId: input.workspaceId } : {}),
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
