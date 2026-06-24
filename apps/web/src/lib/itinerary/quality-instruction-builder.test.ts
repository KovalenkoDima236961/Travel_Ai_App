import { describe, expect, it } from "vitest";
import {
  buildImproveDayInstruction,
  buildImproveItemInstruction
} from "@/lib/itinerary/quality-instruction-builder";
import type { QualityIssue } from "@/types/quality";

function issue(overrides: Partial<QualityIssue>): QualityIssue {
  return {
    id: "issue-1",
    type: "walking_distance_high",
    severity: "warning",
    scope: "day",
    dayNumber: 1,
    title: "High walking distance",
    message: "Day 1 is too far.",
    suggestion: "Reduce walking.",
    instructionHint: "Reduce walking distance and group activities closer together.",
    ...overrides
  };
}

describe("quality instruction builder", () => {
  it("buildImproveDayInstruction includes issue hints", () => {
    const instruction = buildImproveDayInstruction(1, [
      issue({
        metadata: {
          distanceKm: 9.4,
          maxWalkingKmPerDay: 8
        }
      }),
      issue({
        id: "rain",
        type: "weather_rain_outdoor",
        title: "Rain",
        message: "Rain",
        suggestion: "Add indoor alternatives.",
        instructionHint:
          "Make this day more rain-friendly with indoor alternatives and fewer outdoor activities."
      })
    ]);

    expect(instruction).toContain("Improve Day 1");
    expect(instruction).toContain("estimated 9.4 km");
    expect(instruction).toContain("Rain is likely");
    expect(instruction).toContain("Do not add duplicate activities");
  });

  it("caps long day instructions", () => {
    const longIssues = Array.from({ length: 80 }, (_, index) =>
      issue({
        id: `issue-${index}`,
        type: "weather_heat_outdoor",
        title: "Heat",
        message: "Heat",
        suggestion: "Avoid midday heat.",
        instructionHint:
          "Avoid long outdoor walks during midday heat and move outdoor activities to cooler times."
      })
    );

    const instruction = buildImproveDayInstruction(1, longIssues);

    expect(instruction.length).toBeLessThanOrEqual(1000);
    expect(instruction.endsWith("...")).toBe(true);
  });

  it("buildImproveItemInstruction includes item-specific issues", () => {
    const instruction = buildImproveItemInstruction(2, 3, [
      issue({
        id: "closed",
        type: "place_may_be_closed",
        scope: "item",
        dayNumber: 2,
        itemIndex: 3,
        title: "Place may be closed",
        message: "Museum may be closed.",
        suggestion: "Change time.",
        instructionHint:
          "Avoid scheduling this place outside opening hours or replace it with an open alternative."
      }),
      issue({
        id: "other",
        type: "place_no_confident_match",
        scope: "item",
        dayNumber: 1,
        itemIndex: 0,
        title: "No confident place match",
        message: "Other item.",
        suggestion: "Use a clearer place.",
        instructionHint: "Use a more specific real place for this itinerary item."
      })
    ]);

    expect(instruction).toContain("The attached place may be closed");
    expect(instruction).not.toContain("No confident place match");
    expect(instruction).toContain("Replace it with a better realistic alternative");
  });

  it("does not emit raw JSON braces or brackets", () => {
    const instruction = buildImproveItemInstruction(1, 0, [
      issue({
        scope: "item",
        itemIndex: 0,
        instructionHint: "Use this hint with raw looking text: {secret:[value]}"
      })
    ]);

    expect(instruction).not.toContain("{");
    expect(instruction).not.toContain("}");
    expect(instruction).not.toContain("[");
    expect(instruction).not.toContain("]");
  });
});
