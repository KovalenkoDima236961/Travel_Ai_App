import type { TripRoute } from "@/entities/route/model";

export const routeFixture: TripRoute = {
  origin: { name: "Bratislava", country: "Slovakia", coordinates: { lat: 48.1486, lng: 17.1077 } },
  returnToOrigin: true,
  stops: [
    {
      id: "stop-vienna",
      destination: "Vienna",
      city: "Vienna",
      country: "Austria",
      arrivalDate: "2026-04-10",
      departureDate: "2026-04-12",
      nights: 2,
      coordinates: { lat: 48.2082, lng: 16.3738 }
    }
  ],
  legs: [
    {
      id: "leg-bratislava-vienna",
      fromStopId: "origin",
      toStopId: "stop-vienna",
      fromName: "Bratislava",
      toName: "Vienna",
      mode: "train",
      estimatedDistanceKm: 80,
      estimatedDurationMinutes: 70,
      estimatedCost: { amount: 16, currency: "EUR", category: "transport", confidence: "high", source: "provider" },
      warnings: []
    }
  ],
  preferences: {
    preferredModes: ["train", "public_transport"],
    avoidModes: ["flight"],
    carAvailable: false,
    maxTransferHoursPerDay: 4,
    tripStyles: ["city_break"]
  }
};
