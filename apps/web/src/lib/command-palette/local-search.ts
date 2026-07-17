import type { Trip } from "@/entities/trip/model";
import type { SearchResult } from "@/types/search";

export function buildCurrentTripLocalResults(trip: Trip | undefined, query: string): SearchResult[] {
  if (!trip || query.trim().length < 2) {
    return [];
  }
  const tokens = tokenize(query);
  const results: SearchResult[] = [];

  trip.route?.stops?.forEach((stop) => {
    if (!matches(tokens, stop.destination, stop.city, stop.country)) {
      return;
    }
    results.push({
      id: `local:route_stop:${trip.id}:${stop.id}`,
      type: "route_stop",
      title: stop.destination || stop.city || stop.country || "Route stop",
      description: [stop.city, stop.country].filter(Boolean).join(" · "),
      context: trip.destination,
      href: `/trips/${trip.id}?tab=route&stopId=${encodeURIComponent(stop.id)}`,
      icon: "map-pin",
      category: "Route & transport",
      score: 1.15,
      metadata: { tripId: trip.id, stopId: stop.id }
    });
  });

  trip.route?.legs?.forEach((leg) => {
    const selected = leg.selectedTransportOption;
    if (
      !matches(
        tokens,
        leg.fromName,
        leg.toName,
        leg.mode,
        selected?.provider,
        selected?.operatorName,
        selected?.serviceName
      )
    ) {
      return;
    }
    results.push({
      id: `local:route_leg:${trip.id}:${leg.id}`,
      type: "route_leg",
      title: [leg.fromName, leg.toName].filter(Boolean).join(" → ") || "Route leg",
      description: [leg.mode, selected?.operatorName || selected?.provider].filter(Boolean).join(" · "),
      context: trip.destination,
      href: `/trips/${trip.id}?tab=route&legId=${encodeURIComponent(leg.id)}`,
      icon: "route",
      category: "Route & transport",
      score: 1.12,
      metadata: { tripId: trip.id, legId: leg.id }
    });
  });

  trip.itinerary?.days?.forEach((day) => {
    day.items.forEach((item, itemIndex) => {
      if (
        !matches(
          tokens,
          item.name,
          item.note,
          item.description,
          item.category,
          item.type,
          item.place?.name,
          item.place?.address,
          day.title,
          day.locationName
        )
      ) {
        return;
      }
      results.push({
        id: `local:itinerary_item:${trip.id}:${day.day}:${itemIndex}`,
        type: "itinerary_item",
        title: item.name,
        description: [`Day ${day.day}`, item.time, item.place?.name, item.note]
          .filter(Boolean)
          .join(" · "),
        context: trip.destination,
        href: `/trips/${trip.id}?tab=itinerary&day=${day.day}&itemIndex=${itemIndex}`,
        icon: "calendar",
        category: "Itinerary",
        score: 1.1,
        metadata: { tripId: trip.id, dayNumber: day.day, itemIndex }
      });
    });
  });

  return results.slice(0, 10);
}

function tokenize(query: string) {
  return query
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter((token) => token.length >= 2);
}

function matches(tokens: string[], ...values: Array<string | null | undefined>) {
  const haystack = values.filter(Boolean).join(" ").toLowerCase();
  return tokens.some((token) => haystack.includes(token));
}
