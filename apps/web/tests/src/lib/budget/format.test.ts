import { describe, expect, it } from "vitest";
import {
  costSourceLabel,
  costBadgeLabel,
  formatApproxMoney,
  formatMoney,
  getCostAmount,
  getCostCurrency,
  isManualCost,
  isProviderCost
} from "@/entities/budget/model";

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

  it("formats approximate money with a leading marker", () => {
    expect(formatApproxMoney(14.8, "EUR")).toBe("≈€14.80");
    expect(formatApproxMoney(null, "EUR")).toBe("—");
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

  it("detects provider source and formats source labels", () => {
    expect(isProviderCost({ amount: 18, source: "provider" })).toBe(true);
    expect(costSourceLabel({ amount: 18, source: "provider" })).toBe("Provider estimate");
    expect(costSourceLabel({ amount: 18, source: "ai" })).toBe("AI estimate");
  });
});

describe("costBadgeLabel", () => {
  it("renders a compact badge with category", () => {
    expect(costBadgeLabel({ amount: 18, currency: "EUR", category: "ticket" })).toBe("€18 ticket");
  });

  it("labels provider ticket costs as estimates", () => {
    expect(
      costBadgeLabel({ amount: 18, currency: "EUR", category: "ticket", source: "provider" })
    ).toBe("€18 estimated ticket");
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
