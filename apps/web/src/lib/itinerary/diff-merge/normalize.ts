import type { Itinerary, ItineraryDay, ItineraryItem } from "@/types/trip";

const volatileKeys = new Set([
  "createdAt",
  "updatedAt",
  "generatedAt",
  "matchedAt",
  "reviewedAt",
  "reviewedBy",
  "enrichedAt",
  "syncedAt",
  "fetchedAt",
  "lastCheckedAt",
  "lastSyncedAt",
  "metadata",
  "placeEnrichment",
  "priceEnrichment"
]);

export function cloneItinerary(itinerary: Itinerary): Itinerary {
  return JSON.parse(JSON.stringify(itinerary)) as Itinerary;
}

export function normalizeItineraryForDiff(itinerary: Itinerary): unknown {
  return normalizeValue({
    ...itinerary,
    days: (itinerary.days ?? []).map((day, index) => normalizeDayForDiff(day, index))
  });
}

export function normalizeDayForDiff(day: ItineraryDay, dayIndex = 0): unknown {
  return normalizeValue({
    ...day,
    day: day.day || dayIndex + 1,
    items: (day.items ?? []).map((item) => normalizeItemForDiff(item))
  });
}

export function normalizeDayMetadataForDiff(day: ItineraryDay, dayIndex = 0): unknown {
  return normalizeValue({
    ...day,
    day: day.day || dayIndex + 1,
    items: undefined
  });
}

export function normalizeItemForDiff(item: ItineraryItem): unknown {
  return normalizeValue(item);
}

export function normalizedText(value: unknown): string {
  return typeof value === "string" ? value.trim().replace(/\s+/g, " ").toLowerCase() : "";
}

export function getItemId(item: ItineraryItem): string | null {
  const record = item as ItineraryItem & { id?: unknown };
  if (typeof record.id === "string" && record.id.trim()) {
    return record.id.trim();
  }
  if (typeof record.id === "number") {
    return String(record.id);
  }
  return null;
}

export function itemIdentitySignature(item: ItineraryItem): string {
  return [
    normalizedText(item.name),
    normalizedText(item.time),
    normalizedText(item.type)
  ].join("|");
}

export function generatedItemKey(
  dayNumber: number,
  itemIndex: number,
  item: ItineraryItem
): string {
  const signature = itemIdentitySignature(item);
  return `generated:${dayNumber}:${itemIndex}:${signature || "item"}`;
}

export function stableStringify(value: unknown): string {
  return JSON.stringify(normalizeValue(value));
}

export function valuesEqual(left: unknown, right: unknown): boolean {
  return stableStringify(left) === stableStringify(right);
}

function normalizeValue(value: unknown): unknown {
  if (value == null) {
    return null;
  }

  if (Array.isArray(value)) {
    return value.map((entry) => normalizeValue(entry));
  }

  if (typeof value !== "object") {
    return value;
  }

  const record = value as Record<string, unknown>;
  const normalized: Record<string, unknown> = {};
  Object.keys(record)
    .filter((key) => !volatileKeys.has(key) && record[key] !== undefined)
    .sort()
    .forEach((key) => {
      normalized[key] = normalizeValue(record[key]);
    });
  return normalized;
}
