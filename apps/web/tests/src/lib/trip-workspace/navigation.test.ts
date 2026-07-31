import { describe, expect, it } from "vitest";
import {
  buildTripWorkspaceHref,
  normalizeTripWorkspaceHref,
  resolveTripWorkspaceLocation,
  TRIP_WORKSPACE_SECTION_IDS
} from "@/lib/trip-workspace/navigation";
import { resolveTripWorkspaceDeepLink } from "@/lib/trip-workspace/deep-link";

describe("Trip Workspace navigation", () => {
  it("exposes the six stable primary sections", () => {
    expect(TRIP_WORKSPACE_SECTION_IDS).toEqual([
      "overview",
      "plan",
      "money",
      "group",
      "prepare",
      "more"
    ]);
  });

  it("resolves canonical route sections and validates subviews", () => {
    expect(
      resolveTripWorkspaceLocation(
        "/trips/trip-1/plan",
        new URLSearchParams("view=timeline&day=3")
      )
    ).toMatchObject({ tripId: "trip-1", section: "plan", view: "timeline", isLegacy: false });
    expect(
      resolveTripWorkspaceLocation(
        "/trips/trip-1/money",
        new URLSearchParams("view=not-a-view")
      )
    ).toMatchObject({ section: "money", view: "overview" });
  });

  it("interprets every important legacy tab without a redirect loop", () => {
    expect(
      resolveTripWorkspaceLocation("/trips/trip-1", new URLSearchParams("tab=route"))
    ).toMatchObject({ section: "plan", view: "route", isLegacy: true });
    expect(
      resolveTripWorkspaceLocation("/trips/trip-1", new URLSearchParams("tab=expenses"))
    ).toMatchObject({ section: "money", view: "expenses", isLegacy: true });
    expect(
      resolveTripWorkspaceLocation("/trips/trip-1", new URLSearchParams("tab=checklist"))
    ).toMatchObject({ section: "prepare", view: "checklist", isLegacy: true });
  });

  it("canonicalizes legacy entity parameters while preserving context", () => {
    expect(
      normalizeTripWorkspaceHref(
        "/trips/trip-1?tab=itinerary&day=3&itemId=item-7&filter=mine"
      )
    ).toBe("/trips/trip-1/plan?day=3&filter=mine&item=item-7");
    expect(
      normalizeTripWorkspaceHref("/trips/trip-1?tab=receipts&receiptId=receipt-2")
    ).toBe("/trips/trip-1/money?view=receipts&receipt=receipt-2");
  });

  it("builds readable deep links", () => {
    expect(buildTripWorkspaceHref("trip-1", "plan", { view: "timeline", day: 2 })).toBe(
      "/trips/trip-1/plan?view=timeline&day=2"
    );
  });

  it("resolves canonical deep-link targets and permission-safe states", () => {
    expect(
      resolveTripWorkspaceDeepLink({
        pathname: "/trips/trip-1/money",
        searchParams: new URLSearchParams("view=expenses&expense=expense-2")
      })
    ).toMatchObject({
      state: "resolving",
      section: "money",
      view: "expenses",
      targetId: "expense-expense-2"
    });
    expect(
      resolveTripWorkspaceDeepLink({
        pathname: "/trips/trip-1/group",
        searchParams: new URLSearchParams("view=discussion&comment=comment-4"),
        hasTripAccess: false
      }).state
    ).toBe("forbidden");
    expect(
      resolveTripWorkspaceDeepLink({
        pathname: "/trips/trip-1/money",
        searchParams: new URLSearchParams("view=expenses&expense=expense-2"),
        offline: true,
        targetAvailableOffline: false
      }).state
    ).toBe("offline_unavailable");
    expect(
      resolveTripWorkspaceDeepLink({
        pathname: "/trips/trip-1/plan",
        searchParams: new URLSearchParams("view=timeline"),
        featureEnabled: false
      }).state
    ).toBe("feature_disabled");
  });
});
