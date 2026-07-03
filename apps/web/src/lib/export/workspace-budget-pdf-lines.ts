import { formatApproxMoney, formatMoney } from "@/lib/budget/format";
import type { TripPdfLine } from "@/lib/export/trip-pdf-lines";
import type { WorkspaceBudgetSummary } from "@/types/workspace-budget";

const DISCLAIMER =
  "Costs are estimates for planning purposes only. Provider prices, availability, exchange rates, and booking costs may change.";

export function buildWorkspaceBudgetPdfLines(
  summary: WorkspaceBudgetSummary,
  title = "Workspace budget"
): TripPdfLine[] {
  const currency = summary.budget.currency;
  const lines: TripPdfLine[] = [
    { text: title, variant: "title" },
    { text: `Generated ${formatDateTime(summary.generatedAt)} · Currency ${currency}`, variant: "subtitle" },
    { text: "Budget summary", variant: "heading" },
    { text: `Budget: ${summary.budget.name} · ${formatMoney(summary.budget.amount, currency)}` },
    { text: `Period: ${summary.budget.periodStart ?? "open"} to ${summary.budget.periodEnd ?? "open"}` },
    { text: `Estimated total: ${formatApproxMoney(summary.summary.estimatedTotal, currency)}` },
    {
      text:
        summary.summary.overBudgetAmount > 0
          ? `Over budget: ${formatMoney(summary.summary.overBudgetAmount, currency)}`
          : `Remaining: ${formatMoney(summary.summary.remainingAmount, currency)}`
    },
    { text: `Utilization: ${summary.summary.utilizationPercent}% · Trips included: ${summary.summary.tripCount}` },
    {
      text: `Missing estimates: ${summary.summary.missingEstimateCount} · Uncertain estimates: ${summary.summary.uncertainEstimateCount}`
    }
  ];

  appendTrips(lines, summary, currency);
  appendBreakdown(lines, "Cost by category", summary.byCategory, currency, "category");
  appendBreakdown(lines, "Cost by source", summary.bySource, currency, "source");
  appendItems(lines, summary, currency);
  appendInsights(lines, summary);
  appendWarnings(lines, summary.warnings);
  return lines;
}

function appendTrips(lines: TripPdfLine[], summary: WorkspaceBudgetSummary, currency: string) {
  if (summary.byTrip.length === 0) {
    return;
  }
  lines.push({ text: "Cost by trip", variant: "heading" });
  summary.byTrip.slice(0, 12).forEach((trip) => {
    lines.push({
      text: `${trip.title}: ${formatApproxMoney(trip.estimatedTotal, currency)} · ${trip.percentageOfBudget}% of budget · ${trip.missingEstimateCount} missing`,
      variant: "body"
    });
  });
}

function appendBreakdown(
  lines: TripPdfLine[],
  title: string,
  entries: WorkspaceBudgetSummary["byCategory"] | WorkspaceBudgetSummary["bySource"],
  currency: string,
  key: "category" | "source"
) {
  if (entries.length === 0) {
    return;
  }
  lines.push({ text: title, variant: "heading" });
  entries.slice(0, 12).forEach((entry) => {
    const label = key === "category" ? "category" in entry ? entry.category : "" : "source" in entry ? entry.source : "";
    lines.push({
      text: `${formatLabel(label ?? "Unknown")}: ${formatApproxMoney(entry.amount, currency)} · ${entry.percentageOfBudget ?? 0}% of budget · ${entry.percentageOfEstimatedTotal}% of total`,
      variant: "body"
    });
  });
}

function appendItems(lines: TripPdfLine[], summary: WorkspaceBudgetSummary, currency: string) {
  if (summary.expensiveItems.length === 0) {
    return;
  }
  lines.push({ text: "Expensive items", variant: "heading" });
  summary.expensiveItems.slice(0, 12).forEach((item) => {
    lines.push({
      text: `${item.name}${item.tripTitle ? ` (${item.tripTitle})` : ""}: ${formatApproxMoney(item.convertedAmount ?? item.amount, currency)} · ${formatLabel(item.category)} · ${formatLabel(item.source)}`,
      variant: "body"
    });
  });
}

function appendInsights(lines: TripPdfLine[], summary: WorkspaceBudgetSummary) {
  if (summary.insights.length === 0) {
    return;
  }
  lines.push({ text: "Insights", variant: "heading" });
  summary.insights.forEach((insight) => {
    lines.push({
      text: `${formatLabel(insight.severity)} · ${insight.title}: ${insight.message}`,
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
