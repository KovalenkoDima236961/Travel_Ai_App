"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import type {
  CreateWorkspaceBudgetInput,
  WorkspaceBudget
} from "@/types/workspace-budget";

type WorkspaceBudgetFormDialogProps = {
  open: boolean;
  title: string;
  submitLabel: string;
  initialBudget?: WorkspaceBudget | null;
  isSubmitting?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: CreateWorkspaceBudgetInput) => void;
};

const COMMON_CURRENCIES = ["EUR", "USD", "GBP", "JPY", "CAD", "AUD"];

export function WorkspaceBudgetFormDialog({
  open,
  title,
  submitLabel,
  initialBudget,
  isSubmitting = false,
  error,
  onClose,
  onSubmit
}: WorkspaceBudgetFormDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");
  const [currency, setCurrency] = useState("EUR");
  const [periodStart, setPeriodStart] = useState("");
  const [periodEnd, setPeriodEnd] = useState("");
  const [isPrimary, setIsPrimary] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    setName(initialBudget?.name ?? "");
    setDescription(initialBudget?.description ?? "");
    setAmount(initialBudget ? String(initialBudget.amount) : "");
    setCurrency(initialBudget?.currency ?? "EUR");
    setPeriodStart(initialBudget?.periodStart ?? "");
    setPeriodEnd(initialBudget?.periodEnd ?? "");
    setIsPrimary(initialBudget?.isPrimary ?? false);
    setLocalError(null);
  }, [initialBudget, open]);

  const normalizedCurrency = useMemo(() => currency.trim().toUpperCase(), [currency]);

  function submit(event: FormEvent) {
    event.preventDefault();
    const parsedAmount = Number(amount);
    const trimmedName = name.trim();
    if (!trimmedName) {
      setLocalError("Name is required.");
      return;
    }
    if (trimmedName.length < 2 || trimmedName.length > 100) {
      setLocalError("Name must be between 2 and 100 characters.");
      return;
    }
    if (!Number.isFinite(parsedAmount) || parsedAmount < 0) {
      setLocalError("Amount must be greater than or equal to 0.");
      return;
    }
    if (!/^[A-Z]{3}$/.test(normalizedCurrency)) {
      setLocalError("Currency must be a 3-letter code.");
      return;
    }
    if (periodStart && periodEnd && periodStart > periodEnd) {
      setLocalError("Start date must be before or equal to end date.");
      return;
    }

    onSubmit({
      name: trimmedName,
      description: description.trim() || null,
      amount: parsedAmount,
      currency: normalizedCurrency,
      periodStart: periodStart || null,
      periodEnd: periodEnd || null,
      isPrimary
    });
  }

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <Card className="max-h-[90vh] w-full max-w-2xl overflow-y-auto">
        <div className="flex items-start justify-between gap-4">
          <h2 className="text-xl font-semibold text-slate-950">{title}</h2>
          <Button onClick={onClose} size="sm" type="button" variant="ghost">
            Close
          </Button>
        </div>
        <form className="mt-6 space-y-5" onSubmit={submit}>
          <label className="block text-sm font-medium text-slate-700">
            Name
            <Input
              className="mt-2"
              onChange={(event) => setName(event.target.value)}
              required
              value={name}
            />
          </label>
          <label className="block text-sm font-medium text-slate-700">
            Description
            <Textarea
              className="mt-2"
              maxLength={500}
              onChange={(event) => setDescription(event.target.value)}
              value={description}
            />
          </label>
          <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_10rem]">
            <label className="block text-sm font-medium text-slate-700">
              Amount
              <Input
                className="mt-2"
                min="0"
                onChange={(event) => setAmount(event.target.value)}
                required
                step="0.01"
                type="number"
                value={amount}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Currency
              <select
                className="mt-2 block h-11 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900"
                onChange={(event) => setCurrency(event.target.value)}
                value={currency}
              >
                {COMMON_CURRENCIES.map((code) => (
                  <option key={code} value={code}>
                    {code}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-medium text-slate-700">
              Period start
              <Input
                className="mt-2"
                onChange={(event) => setPeriodStart(event.target.value)}
                type="date"
                value={periodStart}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Period end
              <Input
                className="mt-2"
                onChange={(event) => setPeriodEnd(event.target.value)}
                type="date"
                value={periodEnd}
              />
            </label>
          </div>
          <label className="flex items-center gap-3 text-sm font-medium text-slate-700">
            <input
              checked={isPrimary}
              className="h-4 w-4 rounded border-slate-300 text-primary-600"
              onChange={(event) => setIsPrimary(event.target.checked)}
              type="checkbox"
            />
            Primary budget
          </label>
          {localError || error ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
              {localError ?? error}
            </p>
          ) : null}
          <div className="flex flex-wrap justify-end gap-2">
            <Button onClick={onClose} type="button" variant="secondary">
              Cancel
            </Button>
            <Button disabled={isSubmitting} type="submit">
              {isSubmitting ? "Saving..." : submitLabel}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
