import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { AvailabilityCard } from "@/features/availability-search";
import { availabilityKeys } from "@/lib/api/availability";
import { getTripItemDate } from "@/entities/itinerary/model/opening-hours-utils";
import type { AvailabilitySearchResponse } from "@/entities/availability/model";
import type { ItineraryItem, Trip } from "@/entities/trip/model";

// The card is enabled:false and only fetches on click, so tests seed the React
// Query cache under the exact key the card computes, then render statically.
function renderWithResult(result: AvailabilitySearchResponse, item: ItineraryItem) {
  const client = new QueryClient();
  const t = trip();
  const itemDate = formatDate(getTripItemDate(t.startDate!, 1));
  client.setQueryData(
    availabilityKeys.search({
      tripId: t.id,
      dayNumber: 1,
      itemIndex: 0,
      date: itemDate ?? "",
      itemName: item.name
    }),
    result
  );
  return renderToStaticMarkup(
    createElement(
      QueryClientProvider,
      { client },
      createElement(AvailabilityCard, {
        currency: "EUR",
        dayNumber: 1,
        item,
        itemIndex: 0,
        onApplyPrice: async () => {},
        trip: t
      })
    )
  );
}

describe("AvailabilityCard result rendering", () => {
  it("renders an available Ticketmaster result with a safe external booking link", () => {
    const html = renderWithResult(
      baseResult({ status: "available", provider: "ticketmaster", providerDisplayName: "Ticketmaster" }),
      concertItem()
    );
    expect(html).toContain("Ticketmaster");
    expect(html).toContain("Available");
    expect(html).toContain("High confidence");
    expect(html).toContain("Apply price estimate");
    expect(html).toContain("View on provider");
    expect(html).toContain('rel="noopener noreferrer"');
    expect(html).toContain('target="_blank"');
    expect(html).toContain("From");
  });

  it("renders a low-confidence result with a verify warning and no apply button", () => {
    const result = baseResult({ status: "unknown" });
    result.match = { matched: false, confidence: 0.2 };
    const html = renderWithResult(result, concertItem());
    expect(html).toContain("Possible match");
    expect(html).toContain("Verify to apply");
    expect(html).not.toContain("Apply price estimate");
  });

  it("renders a fallback warning when fallbackUsed", () => {
    const result = baseResult({ status: "available", provider: "mock", providerDisplayName: "Mock Tickets" });
    result.fallbackUsed = true;
    const html = renderWithResult(result, concertItem());
    expect(html).toContain("Fallback estimate");
    expect(html).toContain("fallback estimate");
  });

  it("renders a price difference against the current estimate", () => {
    const item = concertItem();
    item.estimatedCost = { amount: 20, currency: "EUR" };
    const html = renderWithResult(baseResult({ status: "available" }), item);
    expect(html).toContain("Current estimate");
    expect(html).toContain("Provider price is higher than current estimate");
  });

  it("renders an unsupported-item provider warning", () => {
    const result = baseResult({ status: "unknown" });
    result.match = { matched: false, confidence: 0 };
    result.options = [];
    result.warnings = ["Availability provider does not support this item type."];
    const html = renderWithResult(result, concertItem());
    expect(html).toContain("does not support this item type");
  });
});

function baseResult(overrides: Partial<AvailabilitySearchResponse>): AvailabilitySearchResponse {
  return {
    status: "available",
    result: "success",
    provider: "ticketmaster",
    providerDisplayName: "Ticketmaster",
    fallbackUsed: false,
    cached: false,
    checkedAt: new Date().toISOString(),
    match: { matched: true, confidence: 0.9, matchedName: "Coldplay", providerEntityId: "abc" },
    options: [
      {
        id: "ticketmaster-abc",
        title: "Coldplay: Music of the Spheres",
        availability: "available",
        priceType: "per_person",
        price: { amount: 55, currency: "EUR", qualifier: "from" },
        startTimes: ["20:00"],
        date: "2026-08-10",
        bookingUrl: "https://www.ticketmaster.com/event/abc",
        providerName: "Ticketmaster",
        providerEntityId: "abc",
        matchConfidence: 0.9,
        location: { name: "Ernst Happel Stadion", address: "Meiereistrasse 7" },
        warnings: ["Verify availability and final price on Ticketmaster."]
      }
    ],
    warnings: ["Availability and prices can change on the provider website."],
    ...overrides
  };
}

function concertItem(): ItineraryItem {
  return { time: "20:00", type: "concert", name: "Coldplay" };
}

function formatDate(value: Date | null) {
  if (!value) {
    return null;
  }
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function trip(): Trip {
  return {
    id: "trip-1",
    destination: "Vienna",
    startDate: "2026-08-10",
    days: 1,
    budgetCurrency: "EUR",
    travelers: 2,
    interests: [],
    pace: "balanced",
    status: "COMPLETED",
    itineraryRevision: 1,
    createdAt: "2026-07-03T10:00:00Z",
    updatedAt: "2026-07-03T10:00:00Z"
  };
}
