import type { Trip } from "@/entities/trip/model";
import {
  getCachedChecklist,
  getCachedExpenses,
  getCachedReminders,
  getCachedTrip,
  listCachedTrips
} from "@/lib/offline/trip-cache";
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

export async function buildOfflineCachedResults(
  userId: string | null | undefined,
  query: string,
  limit = 20
): Promise<SearchResult[]> {
  const normalizedUserId = userId?.trim();
  if (!normalizedUserId || query.trim().length < 2) {
    return [];
  }
  const tokens = tokenize(query);
  const cachedTrips = await listCachedTrips(normalizedUserId);
  const results: SearchResult[] = [];

  for (const cached of cachedTrips.slice(0, 20)) {
    if (results.length >= limit) {
      break;
    }
    const detail = await getCachedTrip(cached.tripId, normalizedUserId);
    const trip = detail?.trip ?? cached.trip;
    if (matches(tokens, trip.destination, trip.status)) {
      results.push(offlineResult({
        id: `offline:trip:${trip.id}`,
        type: "trip",
        title: trip.destination,
        description: "Offline result",
        href: `/trips/${trip.id}`,
        icon: "map",
        category: "Trips",
        metadata: { tripId: trip.id }
      }));
    }

    results.push(
      ...buildCurrentTripLocalResults(trip, query)
        .slice(0, Math.max(0, limit - results.length))
        .map((item) =>
          offlineResult({
            ...item,
            id: item.id.startsWith("offline:") ? item.id : `offline:${item.id}`,
            description: [item.description, "Offline result"].filter(Boolean).join(" · ")
          })
        )
    );

    if (results.length >= limit) {
      break;
    }
    const [checklist, reminders, expenses] = await Promise.all([
      getCachedChecklist(trip.id, normalizedUserId),
      getCachedReminders(trip.id, normalizedUserId),
      getCachedExpenses(trip.id, normalizedUserId)
    ]);

    for (const item of checklist?.checklist.checklist?.items ?? []) {
      if (results.length >= limit) break;
      if (!matches(tokens, item.title, item.description, item.category, item.priority)) continue;
      results.push(offlineResult({
        id: `offline:checklist_item:${trip.id}:${item.id}`,
        type: "checklist_item",
        title: item.title,
        description: [item.priority, item.category, "Offline result"].filter(Boolean).join(" · "),
        context: trip.destination,
        href: `/trips/${trip.id}?tab=checklist&itemId=${encodeURIComponent(item.id)}`,
        icon: "check-square",
        category: "Prepare",
        metadata: { tripId: trip.id, itemId: item.id }
      }));
    }

    for (const reminder of reminders?.reminders ?? []) {
      if (results.length >= limit) break;
      if (!matches(tokens, reminder.title, reminder.description, reminder.category, reminder.priority, reminder.status)) continue;
      results.push(offlineResult({
        id: `offline:reminder:${trip.id}:${reminder.id}`,
        type: "reminder",
        title: reminder.title,
        description: [reminder.priority, reminder.category, reminder.status, "Offline result"].filter(Boolean).join(" · "),
        context: trip.destination,
        href: `/trips/${trip.id}?tab=reminders&reminderId=${encodeURIComponent(reminder.id)}`,
        icon: "bell",
        category: "Prepare",
        metadata: { tripId: trip.id, reminderId: reminder.id }
      }));
    }

    for (const expense of expenses?.expenses ?? []) {
      if (results.length >= limit) break;
      if (!matches(tokens, expense.title, expense.description, expense.category, expense.amount.currency)) continue;
      results.push(offlineResult({
        id: `offline:expense:${trip.id}:${expense.id}`,
        type: "expense",
        title: expense.title,
        description: [
          `${expense.amount.amount.toFixed(2)} ${expense.amount.currency}`,
          expense.category,
          "Offline result"
        ].filter(Boolean).join(" · "),
        context: trip.destination,
        href: `/trips/${trip.id}?tab=expenses&expenseId=${encodeURIComponent(expense.id)}`,
        icon: "receipt-text",
        category: "Money",
        metadata: { tripId: trip.id, expenseId: expense.id }
      }));
    }
  }

  return dedupeByHref(results).slice(0, limit);
}

function offlineResult(item: Omit<SearchResult, "score"> & { score?: number }): SearchResult {
  return {
    ...item,
    score: item.score ?? 1,
    sourceService: "offline-cache",
    metadata: {
      ...item.metadata,
      offline: true
    }
  };
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

function dedupeByHref(items: SearchResult[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    if (seen.has(item.href)) {
      return false;
    }
    seen.add(item.href);
    return true;
  });
}
