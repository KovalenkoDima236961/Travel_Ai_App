import type { QualityIssue } from "@/types/quality";

const MAX_INSTRUCTION_LENGTH = 1000;

export function buildImproveDayInstruction(dayNumber: number, issues: QualityIssue[]): string {
  const relevantIssues = issues.filter(
    (issue) => issue.dayNumber === dayNumber || issue.scope === "trip"
  );
  const bullets = relevantIssues.map(formatDayIssueBullet).filter(Boolean);

  return capInstruction(
    [
      `Improve Day ${dayNumber} of this itinerary using these quality issues:`,
      ...formatBullets(bullets),
      "Keep the rest of the trip consistent. Preserve the user's preferences. Do not add duplicate activities."
    ].join("\n")
  );
}

export function buildImproveItemInstruction(
  dayNumber: number,
  itemIndex: number,
  issues: QualityIssue[]
): string {
  const relevantIssues = issues.filter(
    (issue) => issue.dayNumber === dayNumber && issue.itemIndex === itemIndex
  );
  const bullets = relevantIssues.map(formatItemIssueBullet).filter(Boolean);

  return capInstruction(
    [
      "Improve this itinerary item using these quality issues:",
      ...formatBullets(bullets),
      "Replace it with a better realistic alternative or adjust the timing if appropriate. Keep the day and trip consistent."
    ].join("\n")
  );
}

function formatBullets(bullets: string[]): string[] {
  if (bullets.length === 0) {
    return ["- Improve quality while preserving the itinerary's intent and user preferences."];
  }

  return bullets.map((bullet) => `- ${sanitizeText(bullet)}`);
}

function formatDayIssueBullet(issue: QualityIssue): string {
  if (issue.type === "walking_distance_high") {
    const distanceKm = formatNumber(issue.metadata?.distanceKm);
    const maxWalkingKmPerDay = formatNumber(issue.metadata?.maxWalkingKmPerDay);
    if (distanceKm && maxWalkingKmPerDay) {
      return `Reduce walking distance: estimated ${distanceKm} km, above user preference of ${maxWalkingKmPerDay} km/day.`;
    }
    return "Reduce walking distance and group activities closer together.";
  }

  if (issue.type === "weather_rain_outdoor") {
    return "Rain is likely: prefer indoor activities or add rainy-day alternatives.";
  }

  if (issue.type === "weather_heat_outdoor") {
    return "High heat is likely: avoid long midday outdoor walks and move outdoor activities to cooler times.";
  }

  if (issue.type === "place_may_be_closed") {
    return "Avoid places that may be closed at the scheduled time.";
  }

  if (
    issue.type === "place_match_low_confidence" ||
    issue.type === "place_no_confident_match" ||
    issue.type === "missing_place_coordinates" ||
    issue.type === "conversion_unavailable" ||
    issue.type === "missing_ticket_price" ||
    issue.type === "high_ticket_cost" ||
    issue.type === "provider_price_low_confidence" ||
    issue.type === "availability_unchecked" ||
    issue.type === "availability_unavailable" ||
    issue.type === "availability_limited" ||
    issue.type === "booking_price_higher_than_estimate"
  ) {
    return issue.instructionHint;
  }

  if (issue.type === "place_match_pending_review") {
    return "Use clearer real places where the current auto-match needs review.";
  }

  return issue.instructionHint;
}

function formatItemIssueBullet(issue: QualityIssue): string {
  if (issue.type === "place_may_be_closed") {
    return "The attached place may be closed at the scheduled time.";
  }

  if (issue.type === "place_match_low_confidence") {
    return "The auto-matched place has low confidence.";
  }

  if (issue.type === "place_no_confident_match") {
    return "No confident place match was found for this item.";
  }

  if (issue.type === "missing_place_coordinates") {
    return "The item needs a specific real place with a known map location.";
  }

  if (issue.type === "place_match_pending_review") {
    return "The attached auto-matched place still needs review.";
  }

  if (issue.type === "conversion_unavailable") {
    return "Suggest manual currency or cost correction; do not invent exchange rates.";
  }

  if (issue.type === "missing_ticket_price") {
    return "Add or estimate ticket costs for paid attractions.";
  }

  if (issue.type === "high_ticket_cost") {
    return "Suggest a cheaper or free alternative to this paid attraction.";
  }

  if (issue.type === "provider_price_low_confidence") {
    return "Verify this attraction cost and suggest alternatives if uncertain.";
  }

  if (
    issue.type === "availability_unchecked" ||
    issue.type === "availability_unavailable" ||
    issue.type === "availability_limited" ||
    issue.type === "booking_price_higher_than_estimate"
  ) {
    return issue.instructionHint;
  }

  return issue.instructionHint;
}

function capInstruction(instruction: string): string {
  const normalized = sanitizeText(instruction);
  if (normalized.length <= MAX_INSTRUCTION_LENGTH) {
    return normalized;
  }

  return `${normalized.slice(0, MAX_INSTRUCTION_LENGTH - 3).trimEnd()}...`;
}

function sanitizeText(value: string): string {
  return value
    .replace(/[{}[\]]/g, "")
    .replace(/\s+\n/g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .replace(/[ \t]{2,}/g, " ")
    .trim();
}

function formatNumber(value: unknown): string | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }

  const rounded = value.toFixed(1);
  return rounded.endsWith(".0") ? rounded.slice(0, -2) : rounded;
}
