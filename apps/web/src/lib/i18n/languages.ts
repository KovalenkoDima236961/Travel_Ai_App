export const SUPPORTED_LANGUAGES = ["en", "es", "uk", "fr"] as const;

export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

export const DEFAULT_LANGUAGE: SupportedLanguage = "en";
export const LANGUAGE_STORAGE_KEY = "app_language";

export const LANGUAGE_LABELS: Record<SupportedLanguage, string> = {
  en: "English",
  es: "Español",
  uk: "Українська",
  fr: "Français"
};

export const LANGUAGE_LOCALES: Record<SupportedLanguage, string> = {
  en: "en-US",
  es: "es-ES",
  uk: "uk-UA",
  fr: "fr-FR"
};

export function isSupportedLanguage(input: unknown): input is SupportedLanguage {
  return (
    typeof input === "string" &&
    SUPPORTED_LANGUAGES.includes(input.toLowerCase() as SupportedLanguage)
  );
}

export function normalizeLanguage(input: unknown): SupportedLanguage {
  if (typeof input !== "string") {
    return DEFAULT_LANGUAGE;
  }
  const baseLanguage = input.trim().toLowerCase().split(/[-_]/, 1)[0];
  return isSupportedLanguage(baseLanguage) ? baseLanguage : DEFAULT_LANGUAGE;
}

export function getStoredLanguage(): SupportedLanguage | null {
  if (typeof window === "undefined") {
    return null;
  }
  const stored = window.localStorage.getItem(LANGUAGE_STORAGE_KEY);
  return isSupportedLanguage(stored) ? stored : null;
}

export function setStoredLanguage(language: SupportedLanguage): void {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
  }
}

export function getBrowserLanguage(): SupportedLanguage {
  if (typeof navigator === "undefined") {
    return DEFAULT_LANGUAGE;
  }
  const candidates = [...(navigator.languages ?? []), navigator.language];
  for (const candidate of candidates) {
    const baseLanguage = candidate?.trim().toLowerCase().split(/[-_]/, 1)[0];
    if (isSupportedLanguage(baseLanguage)) {
      return baseLanguage;
    }
  }
  return DEFAULT_LANGUAGE;
}

export function getInitialLanguage(): SupportedLanguage {
  return getStoredLanguage() ?? getBrowserLanguage();
}
