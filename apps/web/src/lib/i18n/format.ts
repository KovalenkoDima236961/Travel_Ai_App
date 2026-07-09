import { LANGUAGE_LOCALES, type SupportedLanguage } from "./languages";

type DateInput = Date | string | number;

function toDate(value: DateInput): Date {
  return value instanceof Date ? value : new Date(value);
}

export function formatDate(value: DateInput, language: SupportedLanguage): string {
  return new Intl.DateTimeFormat(LANGUAGE_LOCALES[language], {
    dateStyle: "medium"
  }).format(toDate(value));
}

export function formatDateTime(value: DateInput, language: SupportedLanguage): string {
  return new Intl.DateTimeFormat(LANGUAGE_LOCALES[language], {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(toDate(value));
}

export function formatMoney(
  amount: number,
  currency: string,
  language: SupportedLanguage
): string {
  return new Intl.NumberFormat(LANGUAGE_LOCALES[language], {
    style: "currency",
    currency: currency.toUpperCase()
  }).format(amount);
}

export function formatNumber(value: number, language: SupportedLanguage): string {
  return new Intl.NumberFormat(LANGUAGE_LOCALES[language]).format(value);
}

export function formatPercent(value: number, language: SupportedLanguage): string {
  return new Intl.NumberFormat(LANGUAGE_LOCALES[language], {
    style: "percent"
  }).format(value);
}
