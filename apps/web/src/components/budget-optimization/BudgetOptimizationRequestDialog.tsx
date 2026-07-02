"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { formatMoney } from "@/lib/budget/format";
import type { BudgetSummary } from "@/types/budget";
import type {
  BudgetOptimizationConstraints,
  BudgetOptimizationJobRequest
} from "@/types/budget-optimization";
import type { Trip } from "@/types/trip";

type BudgetOptimizationRequestDialogProps = {
  open: boolean;
  trip: Trip;
  budgetSummary: BudgetSummary | null;
  defaultDayNumber?: number | null;
  disabled?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: BudgetOptimizationJobRequest) => Promise<void>;
};

type ConstraintKey = keyof Pick<
  BudgetOptimizationConstraints,
  "preserveMustSeeItems" | "keepMealCount" | "avoidReplacingManualCosts"
>;

export function BudgetOptimizationRequestDialog({
  open,
  trip,
  budgetSummary,
  defaultDayNumber,
  disabled = false,
  error,
  onClose,
  onSubmit
}: BudgetOptimizationRequestDialogProps) {
  const currency = budgetSummary?.currency ?? trip.budgetCurrency ?? "EUR";
  const dayOptions = trip.itinerary?.days ?? [];
  const initialDayNumber =
    defaultDayNumber ??
    getSuggestedDayNumber(budgetSummary) ??
    dayOptions[0]?.day ??
    1;
  const initialTarget = useMemo(
    () => getSuggestedTargetReduction(budgetSummary, initialDayNumber),
    [budgetSummary, initialDayNumber]
  );
  const [dayNumber, setDayNumber] = useState(initialDayNumber);
  const [targetReduction, setTargetReduction] = useState(
    initialTarget != null ? String(Math.round(initialTarget)) : ""
  );
  const [maxWalkingIncreaseKm, setMaxWalkingIncreaseKm] = useState("2");
  const [instruction, setInstruction] = useState("");
  const [preserveMustSeeItems, setPreserveMustSeeItems] = useState(true);
  const [keepMealCount, setKeepMealCount] = useState(true);
  const [avoidReplacingManualCosts, setAvoidReplacingManualCosts] = useState(true);
  const [validationError, setValidationError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const nextDayNumber =
      defaultDayNumber ??
      getSuggestedDayNumber(budgetSummary) ??
      dayOptions[0]?.day ??
      1;
    const nextTarget = getSuggestedTargetReduction(budgetSummary, nextDayNumber);
    setDayNumber(nextDayNumber);
    setTargetReduction(nextTarget != null ? String(Math.round(nextTarget)) : "");
    setMaxWalkingIncreaseKm("2");
    setInstruction("");
    setPreserveMustSeeItems(true);
    setKeepMealCount(true);
    setAvoidReplacingManualCosts(true);
    setValidationError(null);
  }, [budgetSummary, dayOptions, defaultDayNumber, open]);

  if (!open) {
    return null;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const parsedTarget = parseOptionalNumber(targetReduction);
    const parsedWalking = parseOptionalNumber(maxWalkingIncreaseKm);
    if (parsedTarget != null && parsedTarget < 0) {
      setValidationError("Target reduction must be 0 or more.");
      return;
    }
    if (parsedWalking != null && parsedWalking < 0) {
      setValidationError("Maximum walking increase must be 0 or more.");
      return;
    }
    if (dayNumber < 1) {
      setValidationError("Choose a day to optimize.");
      return;
    }

    setValidationError(null);
    await onSubmit({
      scope: "day",
      dayNumber,
      targetReductionAmount: parsedTarget,
      currency,
      expectedItineraryRevision: trip.itineraryRevision,
      constraints: {
        preserveMustSeeItems,
        keepMealCount,
        avoidReplacingManualCosts,
        ...(parsedWalking != null ? { maxWalkingIncreaseKm: parsedWalking } : {})
      },
      instruction
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Optimize Day for Budget</h2>
            <p className="mt-1 text-sm leading-6 text-slate-600">
              Create a reviewable AI proposal. The itinerary will not change until you apply it.
            </p>
          </div>
          <Button disabled={disabled} onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>

        <form className="mt-5 space-y-5" onSubmit={handleSubmit}>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="text-sm font-medium text-slate-700">Day</span>
              <select
                className="mt-1 h-11 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none transition focus:border-primary-600 focus:ring-2 focus:ring-primary-100"
                disabled={disabled}
                onChange={(event) => {
                  const nextDay = Number(event.target.value);
                  setDayNumber(nextDay);
                  const nextTarget = getSuggestedTargetReduction(budgetSummary, nextDay);
                  setTargetReduction(nextTarget != null ? String(Math.round(nextTarget)) : "");
                }}
                value={dayNumber}
              >
                {dayOptions.map((day, index) => {
                  const optionDayNumber = day.day || index + 1;
                  const daySummary = budgetSummary?.byDay.find(
                    (candidate) => candidate.dayNumber === optionDayNumber
                  );
                  return (
                    <option key={optionDayNumber} value={optionDayNumber}>
                      Day {optionDayNumber}
                      {daySummary
                        ? ` - ${formatMoney(daySummary.estimatedTotal, currency)}`
                        : ""}
                    </option>
                  );
                })}
              </select>
            </label>

            <label className="block">
              <span className="text-sm font-medium text-slate-700">Target reduction</span>
              <Input
                disabled={disabled}
                min={0}
                onChange={(event) => setTargetReduction(event.target.value)}
                placeholder="Optional"
                step="1"
                type="number"
                value={targetReduction}
              />
              <span className="mt-1 block text-xs text-slate-500">{currency}</span>
            </label>
          </div>

          <div className="rounded-md border border-slate-200 bg-slate-50 p-4">
            <p className="text-sm font-medium text-slate-800">Constraints</p>
            <div className="mt-3 space-y-3">
              <ConstraintCheckbox
                checked={preserveMustSeeItems}
                disabled={disabled}
                label="Preserve must-see and high-value items"
                onChange={setConstraint("preserveMustSeeItems")}
              />
              <ConstraintCheckbox
                checked={keepMealCount}
                disabled={disabled}
                label="Keep meal and rest balance"
                onChange={setConstraint("keepMealCount")}
              />
              <ConstraintCheckbox
                checked={avoidReplacingManualCosts}
                disabled={disabled}
                label="Avoid replacing manually priced items"
                onChange={setConstraint("avoidReplacingManualCosts")}
              />
              <label className="block">
                <span className="text-sm font-medium text-slate-700">
                  Max walking increase
                </span>
                <Input
                  disabled={disabled}
                  min={0}
                  onChange={(event) => setMaxWalkingIncreaseKm(event.target.value)}
                  step="0.5"
                  type="number"
                  value={maxWalkingIncreaseKm}
                />
                <span className="mt-1 block text-xs text-slate-500">Kilometers</span>
              </label>
            </div>
          </div>

          <label className="block">
            <span className="text-sm font-medium text-slate-700">Instruction</span>
            <Textarea
              disabled={disabled}
              maxLength={2000}
              onChange={(event) => setInstruction(event.target.value)}
              placeholder="Optional, for example: keep the historical theme of the day."
              value={instruction}
            />
          </label>

          {validationError || error ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {validationError ?? error}
            </div>
          ) : null}

          <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button disabled={disabled} onClick={onClose} type="button" variant="secondary">
              Cancel
            </Button>
            <Button disabled={disabled} type="submit">
              {disabled ? "Starting..." : "Start optimization"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );

  function setConstraint(key: ConstraintKey) {
    return (checked: boolean) => {
      if (key === "preserveMustSeeItems") {
        setPreserveMustSeeItems(checked);
      } else if (key === "keepMealCount") {
        setKeepMealCount(checked);
      } else {
        setAvoidReplacingManualCosts(checked);
      }
    };
  }
}

function ConstraintCheckbox({
  checked,
  disabled,
  label,
  onChange
}: {
  checked: boolean;
  disabled: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex items-start gap-3 text-sm text-slate-700">
      <input
        checked={checked}
        className="mt-1 h-4 w-4 rounded border-slate-300 text-primary-600 focus:ring-primary-500"
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      <span>{label}</span>
    </label>
  );
}

function getSuggestedDayNumber(summary: BudgetSummary | null): number | null {
  if (!summary?.byDay.length) {
    return null;
  }
  const overBudgetDay = [...summary.byDay]
    .filter((day) => (day.overDailyBudgetBy ?? 0) > 0)
    .sort((left, right) => (right.overDailyBudgetBy ?? 0) - (left.overDailyBudgetBy ?? 0))[0];
  if (overBudgetDay) {
    return overBudgetDay.dayNumber;
  }
  return [...summary.byDay].sort(
    (left, right) => right.estimatedTotal - left.estimatedTotal
  )[0].dayNumber;
}

function getSuggestedTargetReduction(
  summary: BudgetSummary | null,
  dayNumber: number
): number | null {
  const day = summary?.byDay.find((candidate) => candidate.dayNumber === dayNumber);
  if (!day) {
    return null;
  }
  if ((day.overDailyBudgetBy ?? 0) > 0) {
    return day.overDailyBudgetBy ?? null;
  }
  return day.estimatedTotal > 0 ? day.estimatedTotal * 0.15 : null;
}

function parseOptionalNumber(value: string): number | null {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : null;
}
