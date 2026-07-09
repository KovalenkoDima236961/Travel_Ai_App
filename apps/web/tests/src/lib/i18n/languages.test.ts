import { describe, expect, it } from "vitest";
import {
  DEFAULT_LANGUAGE,
  isSupportedLanguage,
  normalizeLanguage
} from "@/lib/i18n/languages";
import {
  formatDate,
  formatMoney,
  formatNumber,
  formatPercent
} from "@/lib/i18n/format";
import { localizeCsvText } from "@/lib/export/csv-localization";

describe("language normalization", () => {
  it.each([
    ["en-US", "en"],
    ["en-GB", "en"],
    ["es-MX", "es"],
    ["uk-UA", "uk"],
    ["fr-CA", "fr"]
  ])("maps %s to %s", (input, expected) => {
    expect(normalizeLanguage(input)).toBe(expected);
  });

  it("falls back to English and rejects unsupported values", () => {
    expect(normalizeLanguage("de-DE")).toBe(DEFAULT_LANGUAGE);
    expect(isSupportedLanguage("de")).toBe(false);
  });
});

describe("CSV localization", () => {
  it("localizes stable export headings without changing data cells", () => {
    const csv = "Summary\nDay,Time,Item,Cost\n1,09:00,Prague Castle,20\n";
    expect(localizeCsvText(csv, "uk")).toBe(
      "Підсумок\nДень,Час,Елемент,Вартість\n1,09:00,Prague Castle,20\n"
    );
  });
});

describe("localized formatting", () => {
  it.each(["en", "es", "uk", "fr"] as const)("formats values for %s", (language) => {
    expect(formatDate("2026-09-10T12:00:00Z", language)).not.toHaveLength(0);
    expect(formatMoney(1234.5, "EUR", language)).toContain("1");
    expect(formatNumber(1234.5, language)).toContain("1");
    expect(formatPercent(0.25, language)).toContain("25");
  });
});
