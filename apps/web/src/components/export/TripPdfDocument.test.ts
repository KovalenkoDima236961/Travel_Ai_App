import { describe, expect, it } from "vitest";
import { buildTripPdfLines } from "@/lib/export/trip-pdf-lines";
import type { ExportTrip } from "@/lib/export/trip-export-adapter";

describe("buildTripPdfLines accommodation", () => {
  it("includes private accommodation details and cost", () => {
    const lines = buildTripPdfLines({
      destination: "Rome",
      startDate: "2026-08-10",
      days: 2,
      budgetAmount: 500,
      budgetCurrency: "EUR",
      source: "private",
      accommodation: {
        name: "Hotel Roma",
        type: "hotel",
        address: "Via Roma 10",
        checkInDate: "2026-08-10",
        checkOutDate: "2026-08-12",
        estimatedCost: { amount: 120, currency: "EUR", category: "accommodation" },
        notes: "Near Termini"
      },
      itinerary: { days: [] }
    });

    const text = lines.map((line) => line.text).join("\n");
    expect(text).toContain("Accommodation");
    expect(text).toContain("Hotel Roma - Hotel");
    expect(text).toContain("Via Roma 10");
    expect(text).toContain("Estimated stay cost: €120 stay");
    expect(text).toContain("Accommodation: €120");
  });

  it("omits structured accommodation for public exports", () => {
    const trip: ExportTrip = {
      destination: "Rome",
      days: 2,
      source: "public",
      accommodation: {
        name: "Hotel Roma",
        type: "hotel",
        address: "Via Roma 10"
      },
      itinerary: { days: [] }
    };

    const text = buildTripPdfLines(trip)
      .map((line) => line.text)
      .join("\n");
    expect(text).not.toContain("Hotel Roma");
    expect(text).not.toContain("Via Roma 10");
  });
});
