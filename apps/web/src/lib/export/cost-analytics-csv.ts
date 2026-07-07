import type {
  CostAmountBreakdown,
  TripCostAnalytics,
  WorkspaceCostAnalytics
} from "@/entities/cost-analytics/model";

const DISCLAIMER =
  "Costs are estimates for planning purposes only. Provider prices, availability, exchange rates, and booking costs may change.";

export function generateTripCostAnalyticsCsv(analytics: TripCostAnalytics): string {
  return joinSections([
    section("Summary", [
      ["Metric", "Value", "Currency"],
      ["Estimated total", analytics.summary.estimatedTotal, analytics.currency],
      ["Budget amount", analytics.summary.budgetAmount ?? "", analytics.currency],
      ["Remaining amount", analytics.summary.remainingAmount ?? "", analytics.currency],
      ["Over budget amount", analytics.summary.overBudgetAmount ?? "", analytics.currency],
      ["Budget utilization percent", analytics.summary.budgetUtilizationPercent ?? "", "%"],
      ["Item estimated total", analytics.summary.itemEstimatedTotal, analytics.currency],
      ["Accommodation total", analytics.summary.accommodationTotal ?? "", analytics.currency],
      ["Missing estimate count", analytics.summary.missingEstimateCount, ""],
      ["Uncertain estimate count", analytics.summary.uncertainEstimateCount, ""],
      ["Converted item count", analytics.summary.convertedItemCount, ""],
      ["Unconverted item count", analytics.summary.unconvertedItemCount, ""]
    ]),
    section("Cost by day", [
      ["Day", "Date", "Estimated total", "Budget share", "Over budget", "Missing estimates"],
      ...analytics.byDay.map((day) => [
        day.dayNumber,
        day.date ?? "",
        day.estimatedTotal,
        day.budgetShare ?? "",
        day.overBudgetAmount ?? "",
        day.missingEstimateCount
      ])
    ]),
    breakdownSection("Cost by category", analytics.byCategory, "category"),
    breakdownSection("Cost by source", analytics.bySource, "source"),
    section("Expensive items", [
      ["Day", "Item index", "Name", "Type", "Category", "Amount", "Currency", "Converted amount", "Source", "Confidence", "Percent of trip"],
      ...analytics.expensiveItems.map((item) => [
        item.dayNumber ?? "",
        item.itemIndex != null ? item.itemIndex + 1 : "",
        item.name,
        item.type,
        item.category,
        item.amount,
        item.currency,
        item.convertedAmount ?? "",
        item.source,
        item.confidence,
        item.percentageOfTrip
      ])
    ]),
    section("Warnings", [["Warning"], ...[...analytics.warnings, DISCLAIMER].map((warning) => [warning])])
  ]);
}

export function generateWorkspaceCostAnalyticsCsv(analytics: WorkspaceCostAnalytics): string {
  return joinSections([
    section("Summary", [
      ["Metric", "Value", "Currency"],
      ["Trips included", analytics.summary.tripCount, ""],
      ["Estimated total", analytics.summary.estimatedTotal, analytics.currency],
      ["Budget total", analytics.summary.budgetTotal ?? "", analytics.currency],
      ["Over-budget trips", analytics.summary.overBudgetTripCount, ""],
      ["Missing estimate count", analytics.summary.missingEstimateCount, ""],
      ["Uncertain estimate count", analytics.summary.uncertainEstimateCount, ""],
      ["Converted item count", analytics.summary.convertedItemCount, ""],
      ["Unconverted item count", analytics.summary.unconvertedItemCount, ""],
      ["Incomplete budget trips", analytics.summary.incompleteBudgetTripCount, ""]
    ]),
    section("Cost by trip", [
      ["Trip ID", "Title", "Destination", "Start date", "End date", "Budget", "Estimated total", "Over budget", "Missing estimates"],
      ...analytics.byTrip.map((trip) => [
        trip.tripId,
        trip.title,
        trip.destination,
        trip.startDate ?? "",
        trip.endDate ?? "",
        trip.budgetAmount ?? "",
        trip.estimatedTotal,
        trip.overBudgetAmount ?? "",
        trip.missingEstimateCount
      ])
    ]),
    breakdownSection("Cost by category", analytics.byCategory, "category"),
    breakdownSection("Cost by source", analytics.bySource, "source"),
    section("Cost by month", [
      ["Month", "Estimated total", "Trip count"],
      ...analytics.byMonth.map((month) => [month.month, month.estimatedTotal, month.tripCount])
    ]),
    section("Expensive items", [
      ["Trip", "Destination", "Day", "Item index", "Name", "Type", "Category", "Amount", "Currency", "Converted amount", "Source", "Confidence"],
      ...analytics.expensiveItems.map((item) => [
        item.tripTitle ?? "",
        item.destination ?? "",
        item.dayNumber ?? "",
        item.itemIndex != null ? item.itemIndex + 1 : "",
        item.name,
        item.type,
        item.category,
        item.amount,
        item.currency,
        item.convertedAmount ?? "",
        item.source,
        item.confidence
      ])
    ]),
    section("Warnings", [["Warning"], ...[...analytics.warnings, DISCLAIMER].map((warning) => [warning])])
  ]);
}

function breakdownSection(
  title: string,
  entries: CostAmountBreakdown[],
  key: "category" | "source" | "confidence"
): string {
  return section(title, [
    [title.replace("Cost by ", ""), "Amount", "Percentage", "Item count"],
    ...entries.map((entry) => [
      entry[key] ?? entry.name ?? "",
      entry.amount,
      entry.percentage,
      entry.itemCount
    ])
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
