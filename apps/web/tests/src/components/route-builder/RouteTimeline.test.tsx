import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { NextIntlClientProvider } from "next-intl";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { RouteTimeline } from "@/components/route-builder/RouteTimeline";
import type { TripRoute } from "@/entities/route/model";
import messages from "../../../../messages/en.json";

describe("RouteTimeline", () => {
  it("renders stops, legs, selected transport, and the not-booked disclaimer", () => {
    const html = render(
      <RouteTimeline
        currency="EUR"
        route={sampleRoute()}
        itinerary={{ days: [{ day: 1, title: "Vienna", primaryStopId: "vienna", items: [] }] }}
      />
    );
    expect(html).toContain("Bratislava");
    expect(html).toContain("Vienna");
    expect(html).toContain("Railjet");
    expect(html).toContain("€14");
    expect(html).toContain("Not booked");
  });

  it("renders missing and stale transport states", () => {
    const route = sampleRoute();
    route.legs![0] = {
      ...route.legs![0],
      providerMetadata: { stale: true },
      warnings: ["Transport option may no longer match this leg."]
    };
    route.legs!.push({ id: "leg_2", fromStopId: "vienna", toStopId: "salzburg", mode: "bus" });
    route.stops.push({ id: "salzburg", destination: "Salzburg", country: "Austria" });
    const html = render(<RouteTimeline route={route} />);
    expect(html).toContain("Stale");
    expect(html).toContain("Missing transport option");
  });

  it("uses a vertical accessible timeline layout", () => {
    const html = render(<RouteTimeline route={sampleRoute()} />);
    expect(html).toContain('aria-label="Route timeline"');
    expect(html).toContain("border-l-2");
  });
});

function render(node: ReactNode) {
  const queryClient = new QueryClient();
  return renderToStaticMarkup(
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={queryClient}>{node}</QueryClientProvider>
    </NextIntlClientProvider>
  );
}

function sampleRoute(): TripRoute {
  return {
    origin: { name: "Bratislava", country: "Slovakia" },
    stops: [{ id: "vienna", destination: "Vienna", country: "Austria", nights: 2 }],
    legs: [
      {
        id: "leg_1",
        fromStopId: "origin",
        toStopId: "vienna",
        fromName: "Bratislava",
        toName: "Vienna",
        mode: "train",
        selectedTransportOption: {
          id: "railjet-1",
          mode: "train",
          provider: "national_rail",
          operatorName: "Railjet",
          departureTime: "09:00",
          arrivalTime: "10:05",
          durationMinutes: 65,
          estimatedPrice: { amount: 14, currency: "EUR" },
          confidence: "high"
        }
      }
    ]
  };
}
