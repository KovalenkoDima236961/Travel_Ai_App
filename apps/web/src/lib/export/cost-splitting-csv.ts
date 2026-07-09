import type { CostSplittingSummary } from "@/entities/cost-splitting/model";
import type { SupportedLanguage } from "@/lib/i18n/languages";
import { localizeCsvText } from "./csv-localization";

const DISCLAIMER =
  "Estimated planning costs only. This is not a payment request, invoice, accounting record, or settlement calculation.";

export function generateCostSplittingCsv(
  summary: CostSplittingSummary,
  language: SupportedLanguage = "en"
): string {
  return localizeCsvText(joinSections([
    section("Summary", [
      ["Metric", "Value", "Currency"],
      ["Estimated total", summary.summary.estimatedTotal, summary.currency],
      ["Allocated total", summary.summary.allocatedTotal, summary.currency],
      ["Unassigned total", summary.summary.unassignedTotal, summary.currency],
      ["Traveler count", summary.summary.travelerCount, ""],
      ["Missing estimate count", summary.summary.missingEstimateCount, ""],
      ["Default split count", summary.summary.defaultSplitCount, ""],
      ["Invalid split count", summary.summary.invalidSplitCount, ""],
      ["Converted item count", summary.summary.convertedItemCount, ""],
      ["Unconverted item count", summary.summary.unconvertedItemCount, ""]
    ]),
    section("Per traveler totals", [
      ["Traveler", "Email", "Role", "Allocated total", "Percentage of total"],
      ...summary.travelers.map((traveler) => [
        traveler.name,
        traveler.email ?? "",
        traveler.role,
        traveler.allocatedTotal,
        traveler.percentageOfTotal
      ])
    ]),
    section("Per traveler category breakdown", [
      ["Traveler", "Category", "Amount", "Currency"],
      ...summary.travelers.flatMap((traveler) =>
        traveler.byCategory.map((category) => [
          traveler.name,
          category.category,
          category.amount,
          summary.currency
        ])
      )
    ]),
    section("Allocated items", [
      ["Traveler", "Type", "Day", "Item index", "Name", "Category", "Allocated amount", "Original amount", "Original currency", "Split type", "Rule source"],
      ...summary.travelers.flatMap((traveler) =>
        traveler.items.map((item) => [
          traveler.name,
          item.type,
          item.dayNumber ?? "",
          item.itemIndex != null ? item.itemIndex + 1 : "",
          item.name,
          item.category,
          item.allocatedAmount,
          item.originalCostAmount,
          item.originalCostCurrency,
          item.splitType,
          item.ruleSource
        ])
      )
    ]),
    section("Unassigned costs", [
      ["Type", "Day", "Item index", "Name", "Amount", "Currency", "Reason"],
      ...summary.unassignedCosts.map((cost) => [
        cost.type,
        cost.dayNumber ?? "",
        cost.itemIndex != null ? cost.itemIndex + 1 : "",
        cost.name,
        cost.amount,
        cost.currency,
        cost.reason
      ])
    ]),
    section("Warnings", [["Warning"], ...[...summary.warnings, DISCLAIMER].map((warning) => [warning])])
  ]), language);
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
