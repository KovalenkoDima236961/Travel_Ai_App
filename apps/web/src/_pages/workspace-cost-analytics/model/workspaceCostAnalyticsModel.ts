import type { CostAmountBreakdown } from "@/entities/cost-analytics/model";

export const COMMON_CURRENCIES = ["EUR", "USD", "GBP", "JPY", "CAD", "AUD"];

export type DatePreset = "all" | "this-year" | "next-12" | "custom";

export function rangeForPreset(preset: DatePreset, customFrom: string, customTo: string) {
  if (preset === "custom") {
    return { from: customFrom || null, to: customTo || null };
  }
  if (preset === "this-year") {
    const year = new Date().getFullYear();
    return { from: `${year}-01-01`, to: `${year}-12-31` };
  }
  if (preset === "next-12") {
    const now = new Date();
    const end = new Date(now);
    end.setFullYear(end.getFullYear() + 1);
    return { from: formatDateInput(now), to: formatDateInput(end) };
  }
  return { from: null, to: null };
}

export function monthBreakdown(
  months: Array<{ month: string; estimatedTotal: number; tripCount: number }>
): CostAmountBreakdown[] {
  const total = months.reduce((sum, month) => sum + month.estimatedTotal, 0);
  return months.map((month) => ({
    name: month.month,
    amount: month.estimatedTotal,
    percentage: total > 0 ? Math.round((month.estimatedTotal / total) * 10000) / 100 : 0,
    itemCount: month.tripCount
  }));
}

function formatDateInput(date: Date) {
  return date.toISOString().slice(0, 10);
}
