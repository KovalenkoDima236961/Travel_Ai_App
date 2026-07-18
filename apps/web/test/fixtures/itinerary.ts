import type { Itinerary } from "@/entities/trip/model";

export const itineraryFixture: Itinerary = {
  destination: "Vienna",
  summary: "A deterministic two-day city break.",
  travelers: 2,
  pace: "balanced",
  currency: "EUR",
  totalBudget: 600,
  generatedAt: "2026-02-01T10:00:00Z",
  source: "mock",
  days: [
    {
      day: 1,
      date: "2026-04-10",
      title: "Historic centre",
      items: [
        {
          time: "09:00",
          endTime: "11:00",
          type: "activity",
          name: "Walk the Ringstrasse",
          estimatedCost: { amount: 0, currency: "EUR", category: "activity", confidence: "high", source: "ai" }
        },
        {
          time: "12:00",
          endTime: "13:00",
          type: "food",
          name: "Naschmarkt lunch",
          estimatedCost: { amount: 24, currency: "EUR", category: "food", confidence: "high", source: "manual" }
        }
      ]
    },
    {
      day: 2,
      date: "2026-04-11",
      title: "Museums and parks",
      items: [
        {
          time: "10:00",
          endTime: "12:00",
          type: "place",
          name: "Belvedere Palace",
          estimatedCost: { amount: 18, currency: "EUR", category: "ticket", confidence: "high", source: "provider" }
        }
      ]
    }
  ]
};
