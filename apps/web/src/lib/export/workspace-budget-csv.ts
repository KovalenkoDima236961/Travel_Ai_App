import type { WorkspaceBudgetSummary } from "@/types/workspace-budget";

const DISCLAIMER =
  "Costs are estimates for planning purposes only. Provider prices, availability, exchange rates, and booking costs may change.";

export function generateWorkspaceBudgetCsv(summary: WorkspaceBudgetSummary): string {
  const currency = summary.budget.currency;
  return joinSections([
    section("Budget summary", [
      ["Metric", "Value", "Currency"],
      ["Budget name", summary.budget.name, ""],
      ["Budget amount", summary.budget.amount, currency],
      ["Period start", summary.budget.periodStart ?? "", ""],
      ["Period end", summary.budget.periodEnd ?? "", ""],
      ["Estimated total", summary.summary.estimatedTotal, currency],
      ["Remaining amount", summary.summary.remainingAmount, currency],
      ["Over budget amount", summary.summary.overBudgetAmount, currency],
      ["Utilization percent", summary.summary.utilizationPercent, "%"],
      ["Trips included", summary.summary.tripCount, ""],
      ["Missing estimates", summary.summary.missingEstimateCount, ""],
      ["Uncertain estimates", summary.summary.uncertainEstimateCount, ""],
      ["Converted item count", summary.summary.convertedItemCount, ""],
      ["Unconverted item count", summary.summary.unconvertedItemCount, ""]
    ]),
    section("Cost by trip", [
      ["Trip ID", "Title", "Destination", "Start date", "Estimated total", "Percent of budget", "Missing estimates", "Over trip budget"],
      ...summary.byTrip.map((trip) => [
        trip.tripId,
        trip.title,
        trip.destination,
        trip.startDate ?? "",
        trip.estimatedTotal,
        trip.percentageOfBudget,
        trip.missingEstimateCount,
        trip.overTripBudgetAmount ?? ""
      ])
    ]),
    section("Cost by category", [
      ["Category", "Amount", "Percent of budget", "Percent of estimated total", "Item count"],
      ...summary.byCategory.map((entry) => [
        entry.category ?? "",
        entry.amount,
        entry.percentageOfBudget ?? "",
        entry.percentageOfEstimatedTotal,
        entry.itemCount
      ])
    ]),
    section("Cost by source", [
      ["Source", "Amount", "Percent of budget", "Percent of estimated total", "Item count"],
      ...summary.bySource.map((entry) => [
        entry.source ?? "",
        entry.amount,
        entry.percentageOfBudget ?? "",
        entry.percentageOfEstimatedTotal,
        entry.itemCount
      ])
    ]),
    section("Expensive items", [
      ["Trip", "Day", "Item index", "Name", "Category", "Amount", "Currency", "Converted amount", "Source", "Confidence"],
      ...summary.expensiveItems.map((item) => [
        item.tripTitle ?? "",
        item.dayNumber ?? "",
        item.itemIndex != null ? item.itemIndex + 1 : "",
        item.name,
        item.category,
        item.amount,
        item.currency,
        item.convertedAmount ?? "",
        item.source,
        item.confidence
      ])
    ]),
    section("Insights", [
      ["Severity", "Type", "Title", "Message"],
      ...summary.insights.map((insight) => [
        insight.severity,
        insight.type,
        insight.title,
        insight.message
      ])
    ]),
    section("Warnings", [["Warning"], ...[...summary.warnings, DISCLAIMER].map((warning) => [warning])])
  ]);
}

function section(title: string, rows: Array<Array<string | number>>): string {
  return [title, ...rows.map((row) => row.map(csvCell).join(","))].join("\n");
}

function joinSections(sections: string[]): string {
  return `${sections.join("\n\n")}\n`;
}

function csvCell(value: string | number): string {
  const text = String(value);
  if (/[",\n\r]/.test(text)) {
    return `"${text.replace(/"/g, '""')}"`;
  }
  return text;
}
