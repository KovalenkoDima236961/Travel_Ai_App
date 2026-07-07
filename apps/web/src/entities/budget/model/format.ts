import type { CostCategory, EstimatedCost } from "@/entities/budget/model";

/**
 * formatMoney renders an amount in the given currency using Intl.NumberFormat,
 * falling back gracefully when the currency code is missing or invalid.
 */
export function formatMoney(
  amount: number | null | undefined,
  currency: string | null | undefined
): string {
  if (amount == null || Number.isNaN(amount)) {
    return "—";
  }

  const code = (currency ?? "").trim().toUpperCase();
  if (/^[A-Z]{3}$/.test(code)) {
    try {
      return new Intl.NumberFormat("en", {
        style: "currency",
        currency: code,
        maximumFractionDigits: Number.isInteger(amount) ? 0 : 2
      }).format(amount);
    } catch {
      // Fall through to the plain-number fallback below.
    }
  }

  const formatted = new Intl.NumberFormat("en", {
    maximumFractionDigits: Number.isInteger(amount) ? 0 : 2
  }).format(amount);
  // Only append a plausible alphabetic code; drop noise like "ZZZ123".
  return /^[A-Z]{2,4}$/.test(code) ? `${formatted} ${code}` : formatted;
}

export function formatApproxMoney(
  amount: number | null | undefined,
  currency: string | null | undefined
): string {
  const formatted = formatMoney(amount, currency);
  return formatted === "—" ? formatted : `≈${formatted}`;
}

/**
 * A raw itinerary item cost may arrive as the structured object or, from older
 * data, as a bare number. These helpers normalise reading either shape.
 */
type RawCost = EstimatedCost | number | null | undefined;

export function getCostAmount(cost: RawCost): number | null {
  if (cost == null) {
    return null;
  }
  if (typeof cost === "number") {
    return Number.isFinite(cost) ? cost : null;
  }
  if (typeof cost.amount === "number" && Number.isFinite(cost.amount)) {
    return cost.amount;
  }
  return null;
}

export function getCostCurrency(cost: RawCost): string | null {
  if (cost == null || typeof cost === "number") {
    return null;
  }
  const code = (cost.currency ?? "").trim().toUpperCase();
  return code || null;
}

export function getCostCategory(cost: RawCost): CostCategory | null {
  if (cost == null || typeof cost === "number") {
    return null;
  }
  return cost.category ?? null;
}

export function isManualCost(cost: RawCost): boolean {
  return typeof cost === "object" && cost != null && cost.source === "manual";
}

export function isProviderCost(cost: RawCost): boolean {
  return typeof cost === "object" && cost != null && cost.source === "provider";
}

export function costSourceLabel(cost: RawCost): string | null {
  if (typeof cost !== "object" || cost == null) {
    return null;
  }
  if (cost.source === "manual") {
    return "Manual";
  }
  if (cost.source === "provider") {
    return "Provider estimate";
  }
  if (cost.source === "ai") {
    return "AI estimate";
  }
  return null;
}

const CATEGORY_LABELS: Record<CostCategory, string> = {
  food: "food",
  transport: "transport",
  ticket: "ticket",
  activity: "activity",
  accommodation: "stay",
  shopping: "shopping",
  other: "other"
};

/**
 * costBadgeLabel renders a compact badge like "€18 ticket" / "€15 food (approx.)".
 * Returns null when the cost has no usable amount.
 */
export function costBadgeLabel(cost: RawCost, fallbackCurrency?: string | null): string | null {
  const amount = getCostAmount(cost);
  if (amount == null) {
    return null;
  }
  const currency = getCostCurrency(cost) ?? fallbackCurrency ?? null;
  const money = formatMoney(amount, currency);
  const category = getCostCategory(cost);
  const providerPrefix = isProviderCost(cost) && (category === "ticket" || category === "activity")
    ? " estimated"
    : "";
  const label = category ? `${providerPrefix} ${CATEGORY_LABELS[category]}` : "";
  const approx =
    typeof cost === "object" && cost != null && cost.confidence === "low" ? " (approx.)" : "";
  return `${money}${label}${approx}`;
}
