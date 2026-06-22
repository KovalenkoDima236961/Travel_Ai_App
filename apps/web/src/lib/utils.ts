export function cn(...classes: Array<string | false | null | undefined>) {
  return classes.filter(Boolean).join(" ");
}

export function formatDate(
  value: string,
  options: Intl.DateTimeFormatOptions = { dateStyle: "medium" }
) {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("en", options).format(date);
}

export function formatMoney(amount: number, currency = "EUR") {
  return new Intl.NumberFormat("en", {
    style: "currency",
    currency,
    maximumFractionDigits: Number.isInteger(amount) ? 0 : 2
  }).format(amount);
}

export function formatBudget(amount: number | null | undefined, currency = "EUR") {
  if (amount == null) {
    return "Not set";
  }

  return formatMoney(amount, currency);
}

export function formatInterestLabel(value: string) {
  return value
    .split(/[_-]/g)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function formatPaceLabel(value: string) {
  if (value === "packed" || value === "intensive") {
    return "Intensive";
  }

  return formatInterestLabel(value);
}

export function getErrorMessage(error: unknown, fallback = "Something went wrong.") {
  return error instanceof Error ? error.message : fallback;
}
