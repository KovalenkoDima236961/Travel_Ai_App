"use client";

import { useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import type { Budget } from "@/types/budget";

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
  const [amount, setAmount] = useState<string>(
    initial?.amount != null ? String(initial.amount) : ""
  );
  const [currency, setCurrency] = useState<string>(
    (initial?.currency ?? defaultCurrency ?? "EUR").toUpperCase()
  );
  const [error, setError] = useState<string | null>(null);

  function handleSave() {
    const parsedAmount = Number(amount);
    if (amount.trim() === "" || Number.isNaN(parsedAmount)) {
      setError("Enter a budget amount.");
      return;
    }
    if (parsedAmount < 0) {
      setError("Budget amount must be 0 or more.");
      return;
    }
    const normalizedCurrency = currency.trim().toUpperCase();
    if (!/^[A-Z]{3}$/.test(normalizedCurrency)) {
      setError("Currency must be a 3-letter code (e.g. EUR).");
      return;
    }
    setError(null);
    onSave({ amount: parsedAmount, currency: normalizedCurrency });
  }

  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_6rem]">
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="budget-amount">
            Amount
          </label>
          <Input
            disabled={isSaving}
            id="budget-amount"
            inputMode="decimal"
            min={0}
            onChange={(event) => setAmount(event.target.value)}
            placeholder="700"
            step="0.01"
            type="number"
            value={amount}
          />
        </div>
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="budget-currency">
            Currency
          </label>
          <Input
            disabled={isSaving}
            id="budget-currency"
            maxLength={3}
            onChange={(event) => setCurrency(event.target.value.toUpperCase())}
            placeholder="EUR"
            value={currency}
          />
        </div>
      </div>

      {error ? <p className="text-sm text-red-700">{error}</p> : null}

      <div className="flex flex-wrap gap-2">
        <Button disabled={isSaving} onClick={handleSave} size="sm" type="button">
          {isSaving ? "Saving..." : "Save budget"}
        </Button>
        <Button
          disabled={isSaving}
          onClick={onCancel}
          size="sm"
          type="button"
          variant="ghost"
        >
          Cancel
        </Button>
        {initial ? (
          <Button
            disabled={isSaving}
            onClick={onClear}
            size="sm"
            type="button"
            variant="ghost"
          >
            Clear budget
          </Button>
        ) : null}
      </div>
    </div>
  );
}