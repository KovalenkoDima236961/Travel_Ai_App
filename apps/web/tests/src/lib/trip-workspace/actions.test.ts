import { describe, expect, it } from "vitest";
import { getTripWorkspaceActions, type TripWorkspaceActionContext } from "@/lib/trip-workspace/actions";

function context(overrides: Partial<TripWorkspaceActionContext> = {}): TripWorkspaceActionContext {
  return {
    tripId: "trip-1",
    role: "owner",
    canEdit: true,
    canManageCollaborators: true,
    canManageShare: true,
    canRestoreVersion: true,
    canArchive: true,
    archived: false,
    completed: true,
    online: true,
    hasItinerary: true,
    flags: {
      public_sharing_enabled: true,
      data_exports_enabled: true,
      offline_mode_enabled: true,
      trip_workspace_shared_actions_enabled: true
    },
    ...overrides
  };
}

describe("Trip Workspace action registry", () => {
  it("includes owner actions and canonical destinations", () => {
    const actions = getTripWorkspaceActions(context());
    expect(actions.map((action) => action.id)).toEqual(expect.arrayContaining([
      "edit_itinerary",
      "invite_people",
      "share_trip",
      "export_trip",
      "save_template",
      "archive_trip"
    ]));
    expect(actions.find((action) => action.id === "invite_people")?.href).toBe(
      "/trips/trip-1/group?view=people&action=invite"
    );
  });

  it("does not expose mutation actions to viewers", () => {
    const actions = getTripWorkspaceActions(context({
      role: "viewer",
      canEdit: false,
      canManageCollaborators: false,
      canManageShare: false,
      canArchive: false
    }));
    expect(actions.map((action) => action.id)).not.toEqual(expect.arrayContaining([
      "edit_itinerary",
      "invite_people",
      "share_trip",
      "archive_trip",
      "restore_trip"
    ]));
  });

  it("switches archived owners to a restore action", () => {
    const actions = getTripWorkspaceActions(context({ archived: true }));
    expect(actions.some((action) => action.id === "archive_trip")).toBe(false);
    expect(actions.some((action) => action.id === "restore_trip")).toBe(true);
    expect(actions.some((action) => action.id === "edit_itinerary")).toBe(false);
  });
});
