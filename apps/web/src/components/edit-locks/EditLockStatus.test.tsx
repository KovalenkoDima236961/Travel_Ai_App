import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { EditLockStatus } from "@/components/edit-locks/EditLockStatus";
import type { EditLockView } from "@/types/edit-locks";

const baseLock: EditLockView = {
  locked: true,
  scope: "itinerary",
  tripId: "trip-1",
  lockedByUserId: "user-1",
  lockedByRole: "editor",
  lockedByCurrentUser: false
};

describe("EditLockStatus", () => {
  it("hides when no lock is active", () => {
    expect(renderToStaticMarkup(<EditLockStatus lock={null} />)).toBe("");
    expect(
      renderToStaticMarkup(<EditLockStatus lock={{ locked: false, scope: "itinerary", tripId: "trip-1" }} />)
    ).toBe("");
  });

  it("shows current user ownership", () => {
    const html = renderToStaticMarkup(
      <EditLockStatus lock={{ ...baseLock, lockedByCurrentUser: true }} />
    );

    expect(html).toContain("You are editing this itinerary");
  });

  it("shows collaborator ownership", () => {
    const html = renderToStaticMarkup(
      <EditLockStatus lock={{ ...baseLock, lockedByDisplayName: "Anna" }} />
    );

    expect(html).toContain("Anna is editing this itinerary");
  });
});
