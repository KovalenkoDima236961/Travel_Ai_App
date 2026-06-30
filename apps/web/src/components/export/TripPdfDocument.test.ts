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

  it("uses converted private budget summary when provided", () => {
    const text = buildTripPdfLines({
      destination: "Tokyo",
      days: 2,
      source: "private",
      budgetAmount: 900,
      budgetCurrency: "EUR",
      budgetSummary: {
        currency: "EUR",
        tripBudget: 900,
        estimatedTotal: 842.3,
        remaining: 57.7,
        overBudgetBy: 0,
        missingEstimateCount: 0,
        estimatedItemCount: 3,
        convertedItemCount: 2,
        unconvertedItemCount: 0,
        originalCurrencyTotals: [
          { currency: "JPY", amount: 45000 },
          { currency: "EUR", amount: 120 }
        ],
        conversionWarnings: [],
        exchangeRateInfo: {
          provider: "mock",
          asOf: "2026-06-30T12:00:00Z",
          fallbackUsed: false
        },
        byDay: [],
        byCategory: []
      },
      itinerary: { days: [] }
    })
      .map((line) => line.text)
      .join("\n");

    expect(text).toContain("Estimated total: ≈€842.30");
    expect(text).toContain("JPY: ¥45,000");
    expect(text).toContain("Approximate conversions");
  });
});
