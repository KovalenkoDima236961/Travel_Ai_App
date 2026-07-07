import type { TripStatus } from "@/entities/trip/model";

export type CollaboratorRole = "viewer" | "editor";
export type CollaboratorStatus = "pending" | "accepted" | "removed";

export type TripCollaborator = {
  id: string;
  tripId: string;
  userId: string;
  email?: string | null;
  displayName?: string | null;
  role: CollaboratorRole;
  status: CollaboratorStatus;
  invitedByUserId: string;
  invitedAt: string;
  acceptedAt?: string | null;
  removedAt?: string | null;
};

export type CollaborationInvitation = {
  collaboratorId: string;
  tripId: string;
  destination: string;
  role: CollaboratorRole;
  invitedByUserId: string;
  invitedAt: string;
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
