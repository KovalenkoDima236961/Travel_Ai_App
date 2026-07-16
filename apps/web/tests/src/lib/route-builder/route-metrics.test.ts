import { describe, expect, it } from "vitest";
import { getRouteMetrics } from "@/lib/route-builder/route-metrics";

describe("route metrics", () => {
  it("uses selected transport for duration, price, coverage, and confidence", () => {
    const metrics = getRouteMetrics(
      {
        stops: [
          { id: "a", destination: "A" },
          { id: "b", destination: "B" }
        ],
        legs: [
          {
            id: "one",
            fromStopId: "origin",
            toStopId: "a",
            mode: "train",
            selectedTransportOption: {
              id: "selected",
              mode: "train",
              provider: "mock",
              durationMinutes: 90,
              estimatedPrice: { amount: 14, currency: "EUR" },
              confidence: "low"
            }
          },
          {
            id: "two",
            fromStopId: "a",
            toStopId: "b",
            mode: "bus",
            estimatedDurationMinutes: 150,
            estimatedCost: { amount: 22, currency: "EUR" }
          }
        ]
      },
      4
    );

    expect(metrics.totalTransferMinutes).toBe(240);
    expect(metrics.estimatedTransportCost).toBe(36);
    expect(metrics.selectedTransportCoverage).toBe(0.5);
    expect(metrics.lowConfidenceLegCount).toBe(1);
    expect(metrics.intensity).toBe("balanced");
  });
});
