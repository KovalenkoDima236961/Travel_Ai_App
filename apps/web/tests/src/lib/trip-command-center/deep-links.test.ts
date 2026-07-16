import { describe, expect, it } from "vitest";
import { getDeepLinkTarget } from "@/lib/trip-command-center/navigation";

describe("trip detail deep links", () => {
  it("maps route legs and stops to stable targets", () => {
    expect(getDeepLinkTarget("route", new URLSearchParams("legId=leg-7"))).toEqual({
      sectionId: "route",
      targetId: "route-leg-leg-7"
    });
    expect(getDeepLinkTarget("route", new URLSearchParams("stopId=stop-2"))).toEqual({
      sectionId: "route",
      targetId: "route-stop-stop-2"
    });
  });

  it("maps budget, health, expense, and activity targets", () => {
    expect(getDeepLinkTarget("budget", new URLSearchParams("category=food"))?.targetId).toBe(
      "budget-category-food"
    );
    expect(getDeepLinkTarget("health", new URLSearchParams("issueId=health-1"))?.targetId).toBe(
      "trip-health-issue-health-1"
    );
    expect(getDeepLinkTarget("expenses", new URLSearchParams("expenseId=expense-1"))?.targetId).toBe(
      "expense-expense-1"
    );
    expect(getDeepLinkTarget("activity", new URLSearchParams("eventId=event-1"))?.targetId).toBe(
      "activity-event-event-1"
    );
  });

  it("falls back to a section and rejects unknown tabs", () => {
    expect(getDeepLinkTarget("checklist", new URLSearchParams())).toEqual({ sectionId: "checklist" });
    expect(getDeepLinkTarget("missing", new URLSearchParams())).toBeNull();
  });
});
