import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { ApprovalChecklist } from "@/features/trip-approval";
import type { ApprovalChecklist as ApprovalChecklistData } from "@/entities/approval/model";

function checklist(overrides: Partial<ApprovalChecklistData> = {}): ApprovalChecklistData {
  return {
    status: "warning",
    warningCount: 1,
    criticalCount: 0,
    blockerCount: 0,
    items: [
      {
        key: "itinerary_exists",
        status: "ok",
        severity: "blocker",
        title: "Itinerary exists",
        message: "Trip has a generated itinerary."
      },
      {
        key: "budget_exists",
        status: "warning",
        severity: "warning",
        title: "Trip budget set",
        message: "No trip budget is set."
      }
    ],
    ...overrides
  };
}

describe("ApprovalChecklist", () => {
  it("renders ok and warning items with the acknowledgement note", () => {
    const html = renderToStaticMarkup(<ApprovalChecklist checklist={checklist()} />);
    expect(html).toContain("Itinerary exists");
    expect(html).toContain("Trip budget set");
    expect(html).toContain("Warnings do not block submission");
  });

  it("renders a blocked item", () => {
    const html = renderToStaticMarkup(
      <ApprovalChecklist
        checklist={checklist({
          status: "blocked",
          blockerCount: 1,
          criticalCount: 1,
          items: [
            {
              key: "itinerary_exists",
              status: "blocked",
              severity: "blocker",
              title: "Itinerary exists",
              message: "Add at least one itinerary day."
            }
          ]
        })}
      />
    );
    expect(html).toContain("Add at least one itinerary day.");
  });

  it("marks acknowledged warnings", () => {
    const html = renderToStaticMarkup(
      <ApprovalChecklist checklist={checklist()} acknowledgedWarnings={["budget_exists"]} />
    );
    expect(html).toContain("acknowledged");
  });
});
