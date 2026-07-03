import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { AvailabilityCard } from "@/components/availability/AvailabilityCard";
import type { Trip } from "@/types/trip";

describe("AvailabilityCard", () => {
  it("renders the initial check button", () => {
    const client = new QueryClient();
    const html = renderToStaticMarkup(
      createElement(
        QueryClientProvider,
        { client },
        createElement(AvailabilityCard, {
          currency: "EUR",
          dayNumber: 1,
          item: {
            time: "10:00",
            type: "attraction",
            name: "Colosseum",
            place: {
              provider: "mock",
              providerPlaceId: "mock-colosseum",
              name: "Colosseum",
              address: "Piazza del Colosseo",
              latitude: 41.8902,
              longitude: 12.4922
            }
          },
          itemIndex: 0,
          trip: trip()
        })
      )
    );

    expect(html).toContain("Check availability");
    expect(html).toContain("Availability and prices may change");
  });
});

function trip(): Trip {
  return {
    id: "trip-1",
    destination: "Rome",
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
