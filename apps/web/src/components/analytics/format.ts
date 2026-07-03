import { formatApproxMoney, formatMoney } from "@/lib/budget/format";

export function formatAnalyticsMoney(
  amount: number | null | undefined,
  currency: string
): string {
  return formatApproxMoney(amount, currency);
}

export function formatPlainMoney(
  amount: number | null | undefined,
  currency: string
): string {
  return formatMoney(amount, currency);
}

export function formatPercent(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) {
    return "—";
  }
  return `${new Intl.NumberFormat("en", { maximumFractionDigits: 2 }).format(value)}%`;
}

export function formatAnalyticsLabel(value: string | null | undefined): string {
  const text = (value ?? "").trim();
  if (!text) {
    return "Unknown";
  }
  return text
    .split(/[_-]/g)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function formatAnalyticsDate(value: string | null | undefined): string {
  if (!value) {
    return "Not set";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(date);
}
