import { describe, expect, it } from "vitest";
import { validateBudgetInput } from "@/features/trip-budget/ui/BudgetEditForm";

describe("budget form validation", () => {
  it("rejects missing and negative amounts", () => {
    expect(validateBudgetInput("", "EUR")).toEqual({
      fieldId: "budget-amount",
      code: "amountRequired"
    });
    expect(validateBudgetInput("-1", "EUR")).toEqual({
      fieldId: "budget-amount",
      code: "amountNonNegative"
    });
  });

  it("requires a three-letter currency and accepts safe values", () => {
    expect(validateBudgetInput("120.50", "EU")).toEqual({
      fieldId: "budget-currency",
      code: "currencyCode"
    });
    expect(validateBudgetInput("120.50", "EUR")).toBeNull();
  });
});
