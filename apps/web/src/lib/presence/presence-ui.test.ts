import { describe, expect, it } from "vitest";

import {
  getOtherEditingUsers,
  getPresenceDisplayName,
  getPresenceEditingWarning
} from "@/lib/presence/presence-ui";
import type { TripPresenceSnapshot, TripPresenceUser } from "@/types/presence";

const currentUser: TripPresenceUser = {
  userId: "u1",
  displayName: null,
  role: "owner",
  state: "viewing",
  connectedAt: "2026-06-25T12:00:00Z",
  lastSeenAt: "2026-06-25T12:00:00Z"
};

const collaborator: TripPresenceUser = {
  userId: "u2",
  displayName: "Anna",
  role: "editor",
  state: "editing",
  connectedAt: "2026-06-25T12:01:00Z",
  lastSeenAt: "2026-06-25T12:01:00Z"
};

function snapshot(users: TripPresenceUser[]): TripPresenceSnapshot {
  return { tripId: "trip-1", users };
}

describe("presence UI helpers", () => {
  it("renders the current user as You", () => {
    expect(getPresenceDisplayName(currentUser, "u1")).toBe("You");
  });

  it("renders collaborator display names when present", () => {
    expect(getPresenceDisplayName(collaborator, "u1")).toBe("Anna");
  });

  it("falls back to Collaborator when display name is missing", () => {
    expect(getPresenceDisplayName({ ...collaborator, displayName: null }, "u1")).toBe(
      "Collaborator"
    );
  });

  it("finds other users currently editing", () => {
    expect(getOtherEditingUsers(snapshot([currentUser, collaborator]), "u1")).toEqual([
      collaborator
    ]);
  });

  it("does not warn when only the current user is editing", () => {
    expect(
      getPresenceEditingWarning(snapshot([{ ...currentUser, state: "editing" }]), "u1")
    ).toBeNull();
  });

  it("warns when another user is editing", () => {
    expect(getPresenceEditingWarning(snapshot([currentUser, collaborator]), "u1")).toBe(
      "Anna is currently editing this itinerary. Be careful before saving changes."
    );
  });

  it("summarizes multiple editors", () => {
    expect(
      getPresenceEditingWarning(
        snapshot([
          currentUser,
          collaborator,
          { ...collaborator, userId: "u3", displayName: "Ben" }
        ]),
        "u1"
      )
    ).toBe("2 collaborators are currently editing this itinerary.");
  });
});
