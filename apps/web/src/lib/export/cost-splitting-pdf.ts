import { formatApproxMoney } from "@/lib/budget/format";
import { downloadPdfLines } from "@/lib/export/pdf";
import { slugifyForFilename } from "@/lib/export/export-filenames";
import type { TripPdfLine } from "@/lib/export/trip-pdf-lines";
import type { CostSplittingSummary } from "@/types/cost-splitting";

const DISCLAIMER =
  "Estimated planning costs only. This is not a payment request, invoice, accounting record, or settlement calculation.";

export function downloadCostSplittingPdf(summary: CostSplittingSummary, title: string): void {
  downloadPdfLines(
    buildCostSplittingPdfLines(summary, title),
    `${slugifyForFilename(title || "trip")}-cost-split-report.pdf`
  );
}

export function buildCostSplittingPdfLines(
  summary: CostSplittingSummary,
  title = "Cost split report"
): TripPdfLine[] {
  const lines: TripPdfLine[] = [
    { text: title, variant: "title" },
    {
      text: `Generated ${formatDateTime(summary.generatedAt)} · Currency ${summary.currency}`,
      variant: "subtitle"
    },
    { text: "Summary", variant: "heading" },
    {
      text: `Estimated total: ${formatApproxMoney(summary.summary.estimatedTotal, summary.currency)}`
    },
    {
      text: `Allocated total: ${formatApproxMoney(summary.summary.allocatedTotal, summary.currency)}`
    },
    {
      text: `Unassigned total: ${formatApproxMoney(summary.summary.unassignedTotal, summary.currency)}`
    },
    {
      text: `Travelers: ${summary.summary.travelerCount} · Missing estimates: ${summary.summary.missingEstimateCount} · Invalid splits: ${summary.summary.invalidSplitCount}`
    }
  ];

  if (summary.travelers.length > 0) {
    lines.push({ text: "Traveler totals", variant: "heading" });
    summary.travelers.forEach((traveler) => {
      lines.push({
        text: `${traveler.name}: ${formatApproxMoney(traveler.allocatedTotal, summary.currency)} · ${traveler.percentageOfTotal}%`,
        variant: "body"
      });
    });
  }

  if (summary.byCategory.length > 0) {
    lines.push({ text: "Category breakdown", variant: "heading" });
    summary.byCategory.forEach((category) => {
      lines.push({
        text: `${formatLabel(category.category)}: ${formatApproxMoney(category.amount, summary.currency)}`,
        variant: "body"
      });
    });
  }

  if (summary.unassignedCosts.length > 0) {
    lines.push({ text: "Unassigned costs", variant: "heading" });
    summary.unassignedCosts.slice(0, 12).forEach((cost) => {
      const location = cost.dayNumber ? `Day ${cost.dayNumber}` : "Accommodation";
      lines.push({
        text: `${cost.name} (${location}): ${formatApproxMoney(cost.amount, cost.currency)} · ${formatLabel(cost.reason)}`,
        variant: "body"
      });
    });
  }

  lines.push({ text: "Warnings and limitations", variant: "heading" });
  [...summary.warnings, DISCLAIMER].forEach((warning) => {
    lines.push({ text: warning, variant: "small" });
  });
  return lines;
}

function formatLabel(value: string): string {
  return value
    .split(/[_-]/g)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}
