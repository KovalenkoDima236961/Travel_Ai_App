// @vitest-environment jsdom

import { axe } from "jest-axe";
import { describe, expect, it, vi } from "vitest";
import { BudgetEditForm } from "@/features/trip-budget/ui/BudgetEditForm";
import { renderWithProviders, screen, userEvent } from "../../../../test/test-utils";

describe("BudgetEditForm behavior", () => {
  it("announces invalid amounts and submits normalized valid values", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    const { container } = renderWithProviders(
      <BudgetEditForm defaultCurrency="eur" onCancel={vi.fn()} onClear={vi.fn()} onSave={onSave} />
    );

    await user.click(screen.getAllByRole("button", { name: "Save budget" })[0]);
    expect(await screen.findByText("Enter an amount.")).toBeVisible();
    expect(screen.getByLabelText("Amount")).toHaveAttribute("aria-invalid", "true");

    await user.type(screen.getByLabelText("Amount"), "720.50");
    await user.clear(screen.getByLabelText("Currency"));
    await user.type(screen.getByLabelText("Currency"), "usd");
    await user.click(screen.getAllByRole("button", { name: "Save budget" })[0]);

    expect(onSave).toHaveBeenCalledWith({ amount: 720.5, currency: "USD" });
    expect(await axe(container)).toHaveNoViolations();
  });

  it("disables all mutation controls while saving", () => {
    renderWithProviders(
      <BudgetEditForm
        defaultCurrency="EUR"
        initial={{ amount: 600, currency: "EUR" }}
        isSaving
        onCancel={vi.fn()}
        onClear={vi.fn()}
        onSave={vi.fn()}
      />
    );

    expect(screen.getAllByRole("button", { name: "Saving…" })[0]).toBeDisabled();
    expect(screen.getAllByRole("button", { name: "Cancel" })[0]).toBeDisabled();
    expect(screen.getByRole("button", { name: "Clear budget" })).toBeDisabled();
  });
});
