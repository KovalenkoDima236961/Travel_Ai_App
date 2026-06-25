import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { SoftEditLockWarningDialog } from "@/components/edit-locks/SoftEditLockWarningDialog";
import type { EditLockView } from "@/types/edit-locks";

const lock: EditLockView = {
  locked: true,
  scope: "itinerary",
  tripId: "trip-1",
  lockedByUserId: "user-1",
  lockedByDisplayName: "Anna",
  lockedByRole: "editor",
  lockedByCurrentUser: false
};

describe("SoftEditLockWarningDialog", () => {
  it("renders collaborator warning and actions", () => {
    const html = renderToStaticMarkup(
      <SoftEditLockWarningDialog lock={lock} onCancel={() => {}} onContinue={() => {}} />
    );

    expect(html).toContain("Someone is already editing");
    expect(html).toContain("Anna is currently editing this itinerary");
    expect(html).toContain("Cancel");
    expect(html).toContain("Continue anyway");
  });

  it("falls back when display name is unavailable", () => {
    const html = renderToStaticMarkup(
      <SoftEditLockWarningDialog
        lock={{ ...lock, lockedByDisplayName: null }}
        onCancel={() => {}}
        onContinue={() => {}}
      />
    );

    expect(html).toContain("A collaborator is currently editing this itinerary");
  });
});
