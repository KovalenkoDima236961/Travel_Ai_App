"use client";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { costSourceLabel, getCostAmount } from "@/entities/budget/model";
import { COST_CATEGORIES, COST_CONFIDENCES } from "@/entities/budget/model";
import type { CostCategory, CostConfidence, EstimatedCost } from "@/entities/budget/model";

type ItemCostEditorProps = {
  cost: EstimatedCost | null | undefined;
  tripCurrency: string;
  idPrefix: string;
  disabled?: boolean;
  onChange: (cost: EstimatedCost | null) => void;
};

/**
 * ItemCostEditor edits a single itinerary item's structured cost estimate. Any
 * edit marks the cost as manually sourced; the parent itinerary editor persists
 * the change through the existing PUT /itinerary flow (with conflict detection).
 */
export function ItemCostEditor({
  cost,
  tripCurrency,
  idPrefix,
  disabled,
  onChange
}: ItemCostEditorProps) {
  const amount = getCostAmount(cost);
  const currency = (cost?.currency ?? tripCurrency ?? "").toUpperCase();
  const category = (cost?.category ?? "other") as CostCategory;
  const confidence = (cost?.confidence ?? "medium") as CostConfidence;
  const note = cost?.note ?? "";
  const sourceLabel = costSourceLabel(cost);

  function emit(next: Partial<EstimatedCost> & { amount?: number | null }) {
    const merged: EstimatedCost = {
      amount: next.amount !== undefined ? next.amount : amount,
      currency: next.currency ?? (currency || tripCurrency),
      category: next.category ?? category,
      confidence: next.confidence ?? confidence,
      note: next.note ?? (note || null),
      // Any manual edit re-stamps the source.
      source: "manual"
    };
    onChange(merged);
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {sourceLabel ? (
        <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-600 sm:col-span-2">
          Source: <span className="font-medium text-slate-800">{sourceLabel}</span>
        </div>
      ) : null}

      <div className="grid gap-1">
        <label className="text-sm font-medium text-slate-700" htmlFor={`${idPrefix}-amount`}>
          Cost amount
        </label>
        <Input
          disabled={disabled}
          id={`${idPrefix}-amount`}
          inputMode="decimal"
          min={0}
          onChange={(event) =>
            emit({ amount: event.target.value === "" ? null : Number(event.target.value) })
          }
          placeholder="0"
          step="0.01"
          type="number"
          value={amount ?? ""}
        />
      </div>

      <div className="grid gap-1">
        <label className="text-sm font-medium text-slate-700" htmlFor={`${idPrefix}-currency`}>
          Currency
        </label>
        <Input
          disabled={disabled}
          id={`${idPrefix}-currency`}
          maxLength={3}
          onChange={(event) => emit({ currency: event.target.value.toUpperCase() })}
          placeholder="EUR"
          value={currency}
        />
      </div>

      <div className="grid gap-1">
        <label className="text-sm font-medium text-slate-700" htmlFor={`${idPrefix}-category`}>
          Category
        </label>
        <Select
          disabled={disabled}
          id={`${idPrefix}-category`}
          onChange={(event) => emit({ category: event.target.value as CostCategory })}
          value={category}
        >
          {COST_CATEGORIES.map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </Select>
      </div>

      <div className="grid gap-1">
        <label className="text-sm font-medium text-slate-700" htmlFor={`${idPrefix}-confidence`}>
          Confidence
        </label>
        <Select
          disabled={disabled}
          id={`${idPrefix}-confidence`}
          onChange={(event) => emit({ confidence: event.target.value as CostConfidence })}
          value={confidence}
        >
          {COST_CONFIDENCES.map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </Select>
      </div>

      <div className="grid gap-1 sm:col-span-2">
        <label className="text-sm font-medium text-slate-700" htmlFor={`${idPrefix}-note`}>
          Cost note
        </label>
        <Input
          disabled={disabled}
          id={`${idPrefix}-note`}
          maxLength={300}
          onChange={(event) => emit({ note: event.target.value })}
          placeholder="Approximate price"
          value={note}
        />
      </div>

      {cost ? (
        <div className="sm:col-span-2">
          <Button
            disabled={disabled}
            onClick={() => onChange(null)}
            size="sm"
            type="button"
            variant="ghost"
          >
            Remove cost
          </Button>
        </div>
      ) : null}
    </div>
  );
}
