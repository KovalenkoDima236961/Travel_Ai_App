import { formatMoney } from "@/entities/budget/model";

export function formatTransportDuration(minutes?: number | null) {
  if (!minutes || minutes <= 0) {
    return "Duration unknown";
  }
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min`;
}

export function formatTransportPrice(price?: { amount: number; currency: string } | null) {
  if (!price) {
    return "Price unknown";
  }
  return formatMoney(price.amount, price.currency);
}

export function formatTransportTime(date?: string | null, time?: string | null) {
  const value = [date, time].filter(Boolean).join(" ");
  return value || "Time unknown";
}

export function providerLabel(provider?: string | null) {
  if (!provider) {
    return "Provider unknown";
  }
  return provider
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
