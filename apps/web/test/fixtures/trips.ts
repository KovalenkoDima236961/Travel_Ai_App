import type { Trip, TripsListResponse } from "@/entities/trip/model";
import type { TripContract, TripsListContract } from "@/lib/api/contracts";
import { itineraryFixture } from "./itinerary";
import { routeFixture } from "./route";
import { TEST_USER_IDS } from "./users";

export const TEST_TRIP_ID = "20000000-0000-4000-8000-000000000001";

export const tripFixture: Trip = {
  id: TEST_TRIP_ID,
  userId: TEST_USER_IDS.owner,
  workspaceId: null,
  scope: "personal",
  tripType: "single_destination",
  route: routeFixture,
  destination: "Vienna",
  startDate: "2026-04-10",
  days: 2,
  budgetAmount: 600,
  budgetCurrency: "EUR",
  travelers: 2,
  interests: ["food", "culture"],
  pace: "balanced",
  status: "COMPLETED",
  itinerary: itineraryFixture,
  itineraryRevision: 3,
  access: {
    role: "owner",
    source: "owner",
    canEdit: true,
    canManageCollaborators: true,
    canManageShare: true,
    canRestoreVersion: true,
    canDelete: true
  },
  createdAt: "2026-02-01T10:00:00Z",
  updatedAt: "2026-02-01T10:10:00Z"
};

tripFixture satisfies TripContract;

export const viewerTripFixture: Trip = {
  ...tripFixture,
  access: {
    role: "viewer",
    source: "collaborator",
    canEdit: false,
    canManageCollaborators: false,
    canManageShare: false,
    canRestoreVersion: false,
    canDelete: false
  }
};

export const tripsListFixture: TripsListResponse = {
  items: [tripFixture],
  limit: 20,
  offset: 0
};

tripsListFixture satisfies TripsListContract;
