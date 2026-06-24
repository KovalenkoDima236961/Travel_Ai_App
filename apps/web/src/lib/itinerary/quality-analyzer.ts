import { getOpeningStatus, getTripItemDate } from "@/lib/itinerary/opening-hours-utils";
import { getEffectiveReviewStatus } from "@/lib/itinerary/place-enrichment-review-utils";
import { isValidCoordinate } from "@/lib/itinerary/map-utils";
import type { DayDistanceSummary } from "@/lib/itinerary/distance-utils";
import { getDayDistanceSummaries } from "@/lib/itinerary/distance-utils";
import type { QualityIssue, QualityIssueSeverity, QualitySummary } from "@/types/quality";
import type { RouteEstimate } from "@/types/route";
import type { Itinerary, ItineraryDay, ItineraryItem } from "@/types/trip";
import type { WeatherDay, WeatherForecast } from "@/types/weather";

type AnalyzeItineraryQualityParams = {
  itinerary: Itinerary;
  tripStartDate?: string | null;
  weatherForecast?: WeatherForecast | null;
  routeEstimatesByDay?: Record<number, RouteEstimate | null>;
  fallbackDistanceSummaries?: DayDistanceSummary[];
  maxWalkingKmPerDay?: number | null;
  placeMatchConfidenceThreshold?: number;
};

type DistanceForDay = {
  distanceKm: number;
  source: "route" | "straight_line";
};

const DEFAULT_PLACE_MATCH_CONFIDENCE_THRESHOLD = 0.75;
const RAIN_PRECIPITATION_THRESHOLD = 60;
const HEAT_TEMPERATURE_THRESHOLD_C = 32;

const outdoorTypeTerms = [
  "place",
  "park",
  "viewpoint",
  "walking",
  "outdoor",
  "attraction",
  "landmark"
];

const outdoorTextTerms = ["walk", "park", "viewpoint", "outdoor", "garden", "square"];

const mapReadyTypes = [
  "place",
  "food",
  "activity",
  "restaurant",
  "cafe",
  "museum",
  "landmark",
  "park"
];

const severityRank: Record<QualityIssueSeverity, number> = {
  critical: 0,
  warning: 1,
  info: 2
};

export function analyzeItineraryQuality({
  itinerary,
  tripStartDate,
  weatherForecast,
  routeEstimatesByDay,
  fallbackDistanceSummaries,
  maxWalkingKmPerDay,
  placeMatchConfidenceThreshold = DEFAULT_PLACE_MATCH_CONFIDENCE_THRESHOLD
}: AnalyzeItineraryQualityParams): QualitySummary {
  const issues: QualityIssue[] = [];
  const summaries =
    fallbackDistanceSummaries ?? getDayDistanceSummaries(itinerary, maxWalkingKmPerDay);
  const summariesByDay = new Map(summaries.map((summary) => [summary.dayNumber, summary]));

  for (const [dayIndex, day] of (itinerary.days ?? []).entries()) {
    const dayNumber = day.day || dayIndex + 1;

    const distance = getDistanceForDay(
      dayNumber,
      routeEstimatesByDay?.[dayNumber],
      summariesByDay.get(dayNumber)
    );
    const walkingIssue = getWalkingDistanceIssue(dayNumber, distance, maxWalkingKmPerDay);
    if (walkingIssue) {
      issues.push(walkingIssue);
    }

    const weatherDay = getWeatherForTripDay(weatherForecast, tripStartDate, dayNumber);
    if (weatherDay && isOutdoorHeavyDay(day)) {
      issues.push(...getWeatherIssues(dayNumber, weatherDay));
    }

    for (const [itemIndex, item] of (day.items ?? []).entries()) {
      const openingIssue = getOpeningHoursIssue(item, dayNumber, itemIndex, tripStartDate);
      if (openingIssue) {
        issues.push(openingIssue);
      }

      issues.push(
        ...getPlaceEnrichmentIssues(
          item,
          dayNumber,
          itemIndex,
          placeMatchConfidenceThreshold
        )
      );
    }
  }

  return summarizeIssues(issues);
}

function getDistanceForDay(
  dayNumber: number,
  routeEstimate?: RouteEstimate | null,
  fallbackSummary?: DayDistanceSummary
): DistanceForDay | null {
  if (routeEstimate && Number.isFinite(routeEstimate.distanceKm)) {
    return {
      distanceKm: routeEstimate.distanceKm,
      source: "route"
    };
  }

  if (fallbackSummary && Number.isFinite(fallbackSummary.straightLineDistanceKm)) {
    return {
      distanceKm: fallbackSummary.straightLineDistanceKm,
      source: "straight_line"
    };
  }

  return null;
}

function getWalkingDistanceIssue(
  dayNumber: number,
  distance: DistanceForDay | null,
  maxWalkingKmPerDay?: number | null
): QualityIssue | null {
  if (
    !distance ||
    !Number.isFinite(distance.distanceKm) ||
    typeof maxWalkingKmPerDay !== "number" ||
    maxWalkingKmPerDay <= 0 ||
    distance.distanceKm <= maxWalkingKmPerDay
  ) {
    return null;
  }

  const severity: QualityIssueSeverity =
    distance.distanceKm > maxWalkingKmPerDay * 1.5 ? "critical" : "warning";
  const estimateLabel = distance.source === "route" ? "route estimate" : "straight-line estimate";

  return {
    id: `day-${dayNumber}-walking-distance-high`,
    type: "walking_distance_high",
    severity,
    scope: "day",
    dayNumber,
    title: "High walking distance",
    message: `Day ${dayNumber} is estimated at ${formatKm(
      distance.distanceKm
    )}, above your preference of ${formatKm(maxWalkingKmPerDay)}/day.`,
    suggestion: "Reduce walking by reordering places or replacing distant activities.",
    instructionHint: "Reduce walking distance and group activities closer together.",
    metadata: {
      distanceKm: distance.distanceKm,
      maxWalkingKmPerDay,
      estimateSource: distance.source,
      estimateLabel
    }
  };
}

function getOpeningHoursIssue(
  item: ItineraryItem,
  dayNumber: number,
  itemIndex: number,
  tripStartDate?: string | null
): QualityIssue | null {
  const openingHours = item.place?.openingHours;
  if (!openingHours || openingHours.length === 0) {
    return null;
  }

  const status = getOpeningStatus({
    startDate: tripStartDate,
    dayNumber,
    itemTime: item.time,
    openingHours
  });

  if (status.status !== "closed") {
    return null;
  }

  const placeName = item.place?.name || item.name || "This place";
  const scheduledTime = item.time || "the scheduled time";

  return {
    id: `day-${dayNumber}-item-${itemIndex}-place-closed`,
    type: "place_may_be_closed",
    severity: "warning",
    scope: "item",
    dayNumber,
    itemIndex,
    title: "Place may be closed",
    message: `${placeName} may be closed at ${scheduledTime}.`,
    suggestion: "Change the time or replace this place.",
    instructionHint:
      "Avoid scheduling this place outside opening hours or replace it with an open alternative.",
    metadata: {
      placeName,
      itemTime: item.time,
      openingStatusLabel: status.label
    }
  };
}

function getWeatherIssues(dayNumber: number, weatherDay: WeatherDay): QualityIssue[] {
  const issues: QualityIssue[] = [];

  if (weatherDay.precipitationChance >= RAIN_PRECIPITATION_THRESHOLD) {
    issues.push({
      id: `day-${dayNumber}-weather-rain-outdoor`,
      type: "weather_rain_outdoor",
      severity: "warning",
      scope: "day",
      dayNumber,
      title: "Rain may affect outdoor plans",
      message: `Day ${dayNumber} has a ${weatherDay.precipitationChance}% chance of rain and several outdoor activities.`,
      suggestion: "Add indoor alternatives or reduce weather-sensitive outdoor time.",
      instructionHint:
        "Make this day more rain-friendly with indoor alternatives and fewer outdoor activities.",
      metadata: {
        date: weatherDay.date,
        precipitationChance: weatherDay.precipitationChance
      }
    });
  }

  if (weatherDay.temperatureMaxC >= HEAT_TEMPERATURE_THRESHOLD_C) {
    issues.push({
      id: `day-${dayNumber}-weather-heat-outdoor`,
      type: "weather_heat_outdoor",
      severity: "warning",
      scope: "day",
      dayNumber,
      title: "Heat may affect outdoor plans",
      message: `Day ${dayNumber} is forecast at up to ${formatTemperature(
        weatherDay.temperatureMaxC
      )}C with several outdoor activities.`,
      suggestion: "Move outdoor time to cooler hours or add shaded indoor alternatives.",
      instructionHint:
        "Avoid long outdoor walks during midday heat and move outdoor activities to cooler times.",
      metadata: {
        date: weatherDay.date,
        temperatureMaxC: weatherDay.temperatureMaxC
      }
    });
  }

  return issues;
}

function getPlaceEnrichmentIssues(
  item: ItineraryItem,
  dayNumber: number,
  itemIndex: number,
  placeMatchConfidenceThreshold: number
): QualityIssue[] {
  const meta = item.placeEnrichment;
  if (!meta) {
    return [];
  }

  const issues: QualityIssue[] = [];
  const itemName = item.name || "This itinerary item";

  if (meta.status === "matched" && getEffectiveReviewStatus(meta) === "pending") {
    issues.push({
      id: `day-${dayNumber}-item-${itemIndex}-place-review-pending`,
      type: "place_match_pending_review",
      severity: "info",
      scope: "item",
      dayNumber,
      itemIndex,
      title: "Auto-matched place needs review",
      message: `${itemName} has an auto-matched place that still needs review.`,
      suggestion: "Accept, change, or remove this place match.",
      instructionHint: "Verify or replace the auto-matched place.",
      metadata: {
        status: meta.status,
        reviewStatus: getEffectiveReviewStatus(meta),
        confidence: meta.confidence ?? null
      }
    });
  }

  if (
    meta.status === "matched" &&
    typeof meta.confidence === "number" &&
    Number.isFinite(meta.confidence) &&
    meta.confidence < placeMatchConfidenceThreshold
  ) {
    issues.push({
      id: `day-${dayNumber}-item-${itemIndex}-place-low-confidence`,
      type: "place_match_low_confidence",
      severity: "warning",
      scope: "item",
      dayNumber,
      itemIndex,
      title: "Low-confidence place match",
      message: `${itemName} has a ${Math.round(meta.confidence * 100)}% place match confidence.`,
      suggestion: "Replace this item with a clearer, better-matched place.",
      instructionHint: "Replace this item with a clearer, better-matched place.",
      metadata: {
        confidence: meta.confidence,
        threshold: placeMatchConfidenceThreshold
      }
    });
  }

  if (meta.status === "no_match") {
    issues.push({
      id: `day-${dayNumber}-item-${itemIndex}-place-no-confident-match`,
      type: "place_no_confident_match",
      severity: "info",
      scope: "item",
      dayNumber,
      itemIndex,
      title: "No confident place match",
      message: `${itemName} does not have a confident place match.`,
      suggestion: "Use a more specific real place for this itinerary item.",
      instructionHint: "Use a more specific real place for this itinerary item.",
      metadata: {
        status: meta.status,
        query: meta.query ?? null
      }
    });
  }

  if (shouldCheckMissingCoordinates(item) && !hasValidPlaceCoordinates(item)) {
    issues.push({
      id: `day-${dayNumber}-item-${itemIndex}-missing-place-coordinates`,
      type: "missing_place_coordinates",
      severity: "info",
      scope: "item",
      dayNumber,
      itemIndex,
      title: "Missing map location",
      message: `${itemName} is missing a map-ready place location.`,
      suggestion: "Attach a real place to enable map and route checks.",
      instructionHint: "Use a specific real place with a known location.",
      metadata: {
        itemType: item.type,
        hasPlace: Boolean(item.place)
      }
    });
  }

  return issues;
}

function shouldCheckMissingCoordinates(item: ItineraryItem): boolean {
  if (!item.placeEnrichment) {
    return false;
  }

  const itemType = item.type?.toLowerCase() ?? "";
  return mapReadyTypes.some((type) => itemType.includes(type));
}

function hasValidPlaceCoordinates(item: ItineraryItem): boolean {
  return isValidCoordinate(item.place?.latitude, item.place?.longitude);
}

function isOutdoorHeavyDay(day: ItineraryDay): boolean {
  const items = day.items ?? [];
  if (items.length === 0) {
    return false;
  }

  const outdoorCount = items.filter(isOutdoorItem).length;
  if (outdoorCount < 2) {
    return false;
  }

  return outdoorCount / items.length >= 0.4;
}

function isOutdoorItem(item: ItineraryItem): boolean {
  const itemType = item.type?.toLowerCase() ?? "";
  if (outdoorTypeTerms.some((term) => itemType.includes(term))) {
    return true;
  }

  const itemText = `${item.name ?? ""} ${item.note ?? ""}`.toLowerCase();
  return outdoorTextTerms.some((term) => itemText.includes(term));
}

function getWeatherForTripDay(
  weatherForecast: WeatherForecast | null | undefined,
  tripStartDate: string | null | undefined,
  dayNumber: number
): WeatherDay | null {
  const days = weatherForecast?.days ?? [];
  if (days.length === 0) {
    return null;
  }

  if (tripStartDate) {
    const itemDate = getTripItemDate(tripStartDate, dayNumber);
    const dateKey = itemDate ? formatLocalDate(itemDate) : null;
    const matchingDay = dateKey ? days.find((day) => day.date === dateKey) : null;
    if (matchingDay) {
      return matchingDay;
    }
  }

  return days[dayNumber - 1] ?? null;
}

function summarizeIssues(issues: QualityIssue[]): QualitySummary {
  const sortedIssues = [...issues].sort(compareIssues);
  const byDay: Record<number, QualityIssue[]> = {};
  const itemIssues: QualityIssue[] = [];
  const tripIssues: QualityIssue[] = [];

  for (const issue of sortedIssues) {
    if (typeof issue.dayNumber === "number") {
      byDay[issue.dayNumber] = [...(byDay[issue.dayNumber] ?? []), issue];
    }
    if (issue.scope === "item") {
      itemIssues.push(issue);
    }
    if (issue.scope === "trip") {
      tripIssues.push(issue);
    }
  }

  return {
    total: sortedIssues.length,
    critical: sortedIssues.filter((issue) => issue.severity === "critical").length,
    warning: sortedIssues.filter((issue) => issue.severity === "warning").length,
    info: sortedIssues.filter((issue) => issue.severity === "info").length,
    byDay,
    itemIssues,
    tripIssues
  };
}

function compareIssues(left: QualityIssue, right: QualityIssue) {
  const severityDelta = severityRank[left.severity] - severityRank[right.severity];
  if (severityDelta !== 0) {
    return severityDelta;
  }

  const leftDay = left.dayNumber ?? Number.MAX_SAFE_INTEGER;
  const rightDay = right.dayNumber ?? Number.MAX_SAFE_INTEGER;
  if (leftDay !== rightDay) {
    return leftDay - rightDay;
  }

  const leftItem = left.itemIndex ?? Number.MAX_SAFE_INTEGER;
  const rightItem = right.itemIndex ?? Number.MAX_SAFE_INTEGER;
  if (leftItem !== rightItem) {
    return leftItem - rightItem;
  }

  return left.id.localeCompare(right.id);
}

function formatKm(value: number): string {
  const rounded = value.toFixed(1);
  return `${rounded.endsWith(".0") ? rounded.slice(0, -2) : rounded} km`;
}

function formatTemperature(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}

function formatLocalDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}
