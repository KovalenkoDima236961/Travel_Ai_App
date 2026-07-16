"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { FieldHint, InlineError, StickyMobileActionBar } from "@/components/ui";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import type { Budget } from "@/entities/budget/model";

type BudgetEditFormProps = {
  initial?: Budget | null;
  defaultCurrency: string;
  isSaving?: boolean;
  onSave: (budget: Budget) => void;
  onClear: () => void;
  onCancel: () => void;
};

export function BudgetEditForm({
  initial,
  defaultCurrency,
  isSaving = false,
  onSave,
  onClear,
  onCancel
}: BudgetEditFormProps) {
  const t = useTranslations("budgets");
  const formsT = useTranslations("forms");
  const commonT = useTranslations("common");
  const [amount, setAmount] = useState<string>(
    initial?.amount != null ? String(initial.amount) : ""
  );
  const [currency, setCurrency] = useState<string>(
    (initial?.currency ?? defaultCurrency ?? "EUR").toUpperCase()
  );
  const [error, setError] = useState<string | null>(null);
  const [errorField, setErrorField] = useState<"budget-amount" | "budget-currency" | null>(null);

  function handleSave() {
    const normalizedCurrency = currency.trim().toUpperCase();
    const validation = validateBudgetInput(amount, normalizedCurrency);
    if (validation) {
      setError(formsT(validation.code));
      setErrorField(validation.fieldId);
      return;
    }
    setError(null);
    setErrorField(null);
    onSave({ amount: Number(amount), currency: normalizedCurrency });
  }

  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_6rem]">
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="budget-amount">
            {t("amount")}
          </label>
          <Input
            aria-describedby={errorField === "budget-amount" ? "budget-form-error" : undefined}
            aria-invalid={errorField === "budget-amount"}
            disabled={isSaving}
            id="budget-amount"
            inputMode="decimal"
            min={0}
            onChange={(event) => {
              setAmount(event.target.value);
              setError(null);
              setErrorField(null);
            }}
            placeholder="700"
            step="0.01"
            type="number"
            value={amount}
          />
        </div>
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="budget-currency">
            {t("currency")}
          </label>
          <Input
            aria-describedby={`budget-currency-hint${errorField === "budget-currency" ? " budget-form-error" : ""}`}
            aria-invalid={errorField === "budget-currency"}
            disabled={isSaving}
            id="budget-currency"
            maxLength={3}
            onChange={(event) => {
              setCurrency(event.target.value.toUpperCase());
              setError(null);
              setErrorField(null);
            }}
            placeholder="EUR"
            value={currency}
          />
          <FieldHint id="budget-currency-hint">{t("currencyHint")}</FieldHint>
        </div>
      </div>

      {error ? <InlineError id="budget-form-error" message={error} /> : null}

      <div className="hidden flex-wrap gap-2 md:flex">
        <Button disabled={isSaving} onClick={handleSave} size="sm" type="button">
          {isSaving ? commonT("saving") : t("save")}
        </Button>
        <Button
          disabled={isSaving}
          onClick={onCancel}
          size="sm"
          type="button"
          variant="ghost"
        >
          {commonT("cancel")}
        </Button>
        {initial ? (
          <Button
            disabled={isSaving}
            onClick={onClear}
            size="sm"
            type="button"
            variant="ghost"
          >
            {t("clear")}
          </Button>
        ) : null}
      </div>
      <StickyMobileActionBar
        onCancel={onCancel}
        onPrimary={handleSave}
        pending={isSaving}
        pendingLabel={commonT("saving")}
        primaryLabel={t("save")}
      />
    </div>
  );
}

export type BudgetInputErrorCode = "amountRequired" | "amountNonNegative" | "currencyCode";

export function validateBudgetInput(
  amount: string,
  currency: string
): { fieldId: "budget-amount" | "budget-currency"; code: BudgetInputErrorCode } | null {
  const parsedAmount = Number(amount);
  if (amount.trim() === "" || !Number.isFinite(parsedAmount)) {
    return { fieldId: "budget-amount", code: "amountRequired" };
  }
  if (parsedAmount < 0) {
    return { fieldId: "budget-amount", code: "amountNonNegative" };
  }
  if (!/^[A-Z]{3}$/.test(currency)) {
    return { fieldId: "budget-currency", code: "currencyCode" };
  }
  return null;
}
