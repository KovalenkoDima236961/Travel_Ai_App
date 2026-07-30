import type { TripStatus } from "@/entities/trip/model";

export type CollaboratorRole = "viewer" | "editor" | "commenter" | "guest" | "admin";
export type CollaboratorStatus =
  | "pending"
  | "accepted"
  | "declined"
  | "expired"
  | "revoked"
  | "removed";

export type TripCollaborator = {
  id: string;
  tripId: string;
  userId: string;
  email?: string | null;
  displayName?: string | null;
  role: CollaboratorRole;
  status: CollaboratorStatus;
  invitedByUserId: string;
  message?: string | null;
  invitedAt: string;
  expiresAt?: string | null;
  acceptedAt?: string | null;
  declinedAt?: string | null;
  revokedAt?: string | null;
  removedAt?: string | null;
  lastSeenAt?: string | null;
  permissions?: Record<string, unknown> | null;
};

export type CollaborationInvitation = {
  collaboratorId: string;
  invitationId?: string | null;
  tripId: string;
  destination: string;
  role: CollaboratorRole;
  invitedByUserId: string;
  email?: string | null;
  message?: string | null;
  invitedAt: string;
  expiresAt?: string | null;
};

export type TripInvitationStatus = "pending" | "accepted" | "declined" | "expired" | "revoked";

export type TripInvitation = {
  id: string;
  tripId: string;
  inviterUserId: string;
  invitedUserId?: string | null;
  email: string;
  role: CollaboratorRole;
  status: TripInvitationStatus;
  message?: string | null;
  expiresAt: string;
  acceptedAt?: string | null;
  declinedAt?: string | null;
  revokedAt?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type TripMemberStatus = "active" | "invited" | "removed";

export type TripMember = {
  userId: string;
  email?: string | null;
  displayName?: string | null;
  role: CollaboratorRole | "owner" | (string & {});
  status: TripMemberStatus;
  joinedAt?: string | null;
  invitedBy?: string | null;
  lastSeenAt?: string | null;
  permissions: Record<string, boolean>;
  isSelf: boolean;
};

export type TripSuggestionType =
  | "activity_replacement"
  | "time_change"
  | "budget_adjustment"
  | "route_change"
  | "note";

export type TripSuggestionTargetType =
  | "trip"
  | "day"
  | "itinerary_item"
  | "budget_item"
  | "route"
  | "attachment";

export type TripSuggestionStatus = "open" | "accepted" | "rejected" | "resolved";

export type TripSuggestion = {
  id: string;
  tripId: string;
  authorUserId: string;
  suggestionType: TripSuggestionType;
  targetType: TripSuggestionTargetType;
  targetId?: string | null;
  status: TripSuggestionStatus;
  before?: unknown;
  after?: unknown;
  comment?: string | null;
  metadata?: Record<string, unknown> | null;
  appliedItineraryRevision?: number | null;
  resolvedAt?: string | null;
  resolvedByUserId?: string | null;
  createdAt: string;
  updatedAt: string;
  isAuthor?: boolean;
  canResolve?: boolean;
};

export type TripVoteTargetType =
  | "activity"
  | "restaurant"
  | "hotel"
  | "destination"
  | "suggestion";

export type TripVoteType = "thumbs_up" | "thumbs_down" | "heart" | "star";

export type TripVoteSummary = {
  targetType: TripVoteTargetType;
  targetId: string;
  counts: Partial<Record<TripVoteType, number>>;
  currentVote?: TripVoteType | null;
};

export type SharedTripSummary = {
  id: string;
  destination: string;
  startDate?: string | null;
  days: number;
  role: CollaboratorRole;
  ownerUserId?: string | null;
  status: TripStatus;
  itineraryRevision: number;
  updatedAt?: string | null;
};
