import { describe, expect, it } from "vitest";
import {
  costBadgeLabel,
  formatMoney,
  getCostAmount,
  getCostCurrency,
  isManualCost
} from "@/lib/budget/format";

describe("formatMoney", () => {
  it("formats a valid currency", () => {
    expect(formatMoney(18, "EUR")).toBe("€18");
    expect(formatMoney(18.5, "EUR")).toBe("€18.50");
  });

  it("returns a dash for missing amounts", () => {
    expect(formatMoney(null, "EUR")).toBe("—");
    expect(formatMoney(undefined, "EUR")).toBe("—");
  });

  it("falls back gracefully for invalid currency codes", () => {
    expect(formatMoney(20, "ZZZ123")).toBe("20");
    expect(formatMoney(20, "")).toBe("20");
  });
});

describe("cost helpers", () => {
  it("reads the structured object amount and currency", () => {
    const cost = { amount: 25.5, currency: "eur", category: "food" as const };
    expect(getCostAmount(cost)).toBe(25.5);
    expect(getCostCurrency(cost)).toBe("EUR");
  });

  it("reads the legacy bare-number form", () => {
    expect(getCostAmount(12)).toBe(12);
    expect(getCostCurrency(12)).toBeNull();
  });

  it("treats missing amounts as null", () => {
    expect(getCostAmount(null)).toBeNull();
    expect(getCostAmount({ currency: "EUR" })).toBeNull();
  });

  it("detects manual source", () => {
    expect(isManualCost({ amount: 5, source: "manual" })).toBe(true);
    expect(isManualCost({ amount: 5, source: "ai" })).toBe(false);
  });
});

describe("costBadgeLabel", () => {
  it("renders a compact badge with category", () => {
    expect(costBadgeLabel({ amount: 18, currency: "EUR", category: "ticket" })).toBe("€18 ticket");
  });

  it("adds approx for low confidence and uses the fallback currency", () => {
    expect(costBadgeLabel({ amount: 15, category: "food", confidence: "low" }, "EUR")).toBe(
      "€15 food (approx.)"
    );
  });

  it("returns null when there is no amount", () => {
    expect(costBadgeLabel(null, "EUR")).toBeNull();
    expect(costBadgeLabel({ currency: "EUR" }, "EUR")).toBeNull();
  });
});
