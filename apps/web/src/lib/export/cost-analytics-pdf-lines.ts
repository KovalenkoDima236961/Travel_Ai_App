import { formatApproxMoney, formatMoney } from "@/lib/budget/format";
import type { TripPdfLine } from "@/lib/export/trip-pdf-lines";
import type {
  CostAmountBreakdown,
  ExpensiveCostItem,
  TripCostAnalytics,
  TripCostSummary,
  WorkspaceCostAnalytics
} from "@/types/cost-analytics";

const DISCLAIMER =
  "Costs are estimates for planning purposes only. Provider prices, availability, exchange rates, and booking costs may change.";

export function buildTripCostAnalyticsPdfLines(
  analytics: TripCostAnalytics,
  title = "Trip cost analytics"
): TripPdfLine[] {
  const currency = analytics.currency;
  const lines: TripPdfLine[] = [
    { text: title, variant: "title" },
    { text: `Generated ${formatDateTime(analytics.generatedAt)} · Currency ${currency}`, variant: "subtitle" },
    { text: "Summary", variant: "heading" },
    { text: `Estimated total: ${formatApproxMoney(analytics.summary.estimatedTotal, currency)}` },
    {
      text:
        analytics.summary.budgetAmount != null
          ? `Budget: ${formatMoney(analytics.summary.budgetAmount, currency)}`
          : "Budget: not set"
    },
    {
      text:
        analytics.summary.overBudgetAmount != null && analytics.summary.overBudgetAmount > 0
          ? `Over budget: ${formatMoney(analytics.summary.overBudgetAmount, currency)}`
          : `Remaining: ${formatMoney(analytics.summary.remainingAmount, currency)}`
    },
    {
      text: `Missing estimates: ${analytics.summary.missingEstimateCount} · Uncertain estimates: ${analytics.summary.uncertainEstimateCount}`
    }
  ];

  appendBreakdown(lines, "Cost by day", analytics.byDay.map((day) => ({
    label: `Day ${day.dayNumber}${day.date ? ` (${day.date})` : ""}`,
    amount: day.estimatedTotal,
    detail: day.missingEstimateCount > 0 ? `${day.missingEstimateCount} missing` : undefined
  })), currency);
  appendBreakdown(lines, "Cost by category", analytics.byCategory, currency);
  appendBreakdown(lines, "Cost by source", analytics.bySource, currency);
  appendItems(lines, "Most expensive items", analytics.expensiveItems, currency);
  appendWarnings(lines, analytics.warnings);
  return lines;
}

export function buildWorkspaceCostAnalyticsPdfLines(
  analytics: WorkspaceCostAnalytics,
  title = "Workspace cost analytics"
): TripPdfLine[] {
  const currency = analytics.currency;
  const lines: TripPdfLine[] = [
    { text: title, variant: "title" },
    { text: `Generated ${formatDateTime(analytics.generatedAt)} · Currency ${currency}`, variant: "subtitle" },
    { text: "Summary", variant: "heading" },
    { text: `Estimated total: ${formatApproxMoney(analytics.summary.estimatedTotal, currency)}` },
    { text: `Trips included: ${analytics.summary.tripCount}` },
    { text: `Over-budget trips: ${analytics.summary.overBudgetTripCount}` },
    {
      text: `Missing estimates: ${analytics.summary.missingEstimateCount} · Incomplete budgets: ${analytics.summary.incompleteBudgetTripCount}`
    }
  ];

  appendTrips(lines, analytics.expensiveTrips, currency);
  appendBreakdown(lines, "Cost by category", analytics.byCategory, currency);
  appendBreakdown(lines, "Cost by source", analytics.bySource, currency);
  appendBreakdown(lines, "Cost by month", analytics.byMonth.map((month) => ({
    label: month.month,
    amount: month.estimatedTotal,
    detail: `${month.tripCount} ${month.tripCount === 1 ? "trip" : "trips"}`
  })), currency);
  appendItems(lines, "Most expensive items", analytics.expensiveItems, currency);
  appendWarnings(lines, analytics.warnings);
  return lines;
}

function appendTrips(lines: TripPdfLine[], trips: TripCostSummary[], currency: string) {
  if (trips.length === 0) {
    return;
  }
  lines.push({ text: "Top expensive trips", variant: "heading" });
  trips.slice(0, 10).forEach((trip) => {
    lines.push({
      text: `${trip.title}: ${formatApproxMoney(trip.estimatedTotal, currency)}${trip.overBudgetAmount && trip.overBudgetAmount > 0 ? ` · over by ${formatMoney(trip.overBudgetAmount, currency)}` : ""}`,
      variant: "body"
    });
  });
}

function appendBreakdown(
  lines: TripPdfLine[],
  title: string,
  entries: Array<CostAmountBreakdown | { label: string; amount: number; detail?: string }>,
  currency: string
) {
  if (entries.length === 0) {
    return;
  }
  lines.push({ text: title, variant: "heading" });
  entries.slice(0, 12).forEach((entry) => {
    const label =
      "label" in entry
        ? entry.label
        : entry.category ?? entry.source ?? entry.confidence ?? entry.name ?? "Unknown";
    const detail =
      "label" in entry
        ? entry.detail
        : `${entry.percentage}% · ${entry.itemCount} ${entry.itemCount === 1 ? "item" : "items"}`;
    lines.push({
      text: `${formatLabel(label)}: ${formatApproxMoney(entry.amount, currency)}${detail ? ` · ${detail}` : ""}`,
      variant: "body"
    });
  });
}

function appendItems(
  lines: TripPdfLine[],
  title: string,
  items: ExpensiveCostItem[],
  currency: string
) {
  if (items.length === 0) {
    return;
  }
  lines.push({ text: title, variant: "heading" });
  items.slice(0, 12).forEach((item) => {
    const location = item.tripTitle
      ? `${item.tripTitle}${item.dayNumber ? ` Day ${item.dayNumber}` : ""}`
      : item.dayNumber
        ? `Day ${item.dayNumber}`
        : "";
    lines.push({
      text: `${item.name}${location ? ` (${location})` : ""}: ${formatApproxMoney(item.convertedAmount ?? item.amount, currency)} · ${formatLabel(item.category)} · ${formatLabel(item.source)}`,
      variant: "body"
    });
  });
}

function appendWarnings(lines: TripPdfLine[], warnings: string[]) {
  const unique = Array.from(new Set([...warnings, DISCLAIMER]));
  lines.push({ text: "Warnings and limitations", variant: "heading" });
  unique.forEach((warning) => lines.push({ text: warning, variant: "small" }));
}

function formatLabel(value: string) {
  return value
    .split(/[_-]/g)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}
