"use client";

import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { formatMoney } from "@/lib/budget/format";
import type {
  CostSplitRule,
  CostSplitType,
  TripTraveler
} from "@/types/cost-splitting";

type CostSplitRuleEditorProps = {
  open: boolean;
  title: string;
  travelers: TripTraveler[];
  currentSplit?: CostSplitRule | null;
  costAmount: number;
  costCurrency: string;
  isSaving?: boolean;
  error?: string | null;
  onClose: () => void;
  onSave: (split: CostSplitRule) => void;
};

export function CostSplitRuleEditor({
  open,
  title,
  travelers,
  currentSplit,
  costAmount,
  costCurrency,
  isSaving = false,
  error,
  onClose,
  onSave
}: CostSplitRuleEditorProps) {
  const [splitType, setSplitType] = useState<CostSplitType>("all_equal");
  const [selectedTravelerIds, setSelectedTravelerIds] = useState<string[]>([]);
  const [percentages, setPercentages] = useState<Record<string, number>>({});

  useEffect(() => {
    if (!open) {
      return;
    }
    const nextType = currentSplit?.type ?? "all_equal";
    setSplitType(nextType);
    setSelectedTravelerIds(
      currentSplit?.travelerIds?.length
        ? currentSplit.travelerIds
        : travelers.map((traveler) => traveler.id)
    );
    setPercentages(
      currentSplit?.percentages ?? equalPercentages(travelers.map((traveler) => traveler.id))
    );
  }, [currentSplit, open, travelers]);

  const activeTravelers = travelers.filter((traveler) => traveler.status === "active");
  const preview = useMemo(
    () => buildPreview(splitType, activeTravelers, selectedTravelerIds, percentages, costAmount),
    [activeTravelers, costAmount, percentages, selectedTravelerIds, splitType]
  );
  const percentTotal = Object.values(percentages).reduce((sum, value) => sum + (Number(value) || 0), 0);
  const canSave =
    activeTravelers.length > 0 &&
    (splitType === "all_equal" ||
      (splitType === "selected_equal" && selectedTravelerIds.length > 0) ||
      (splitType === "custom_percentages" &&
        Object.values(percentages).some((value) => value > 0) &&
        Math.abs(percentTotal - 100) <= 0.01));

  if (!open) {
    return null;
  }

  function toggleTraveler(travelerId: string) {
    setSelectedTravelerIds((current) =>
      current.includes(travelerId)
        ? current.filter((id) => id !== travelerId)
        : [...current, travelerId]
    );
  }

  function save() {
    if (!canSave) {
      return;
    }
    if (splitType === "all_equal") {
      onSave({ type: "all_equal" });
      return;
    }
    if (splitType === "selected_equal") {
      onSave({ type: "selected_equal", travelerIds: selectedTravelerIds });
      return;
    }
    const cleaned = Object.fromEntries(
      Object.entries(percentages)
        .filter(([, value]) => value > 0)
        .map(([id, value]) => [id, Number(value)])
    );
    onSave({
      type: "custom_percentages",
      travelerIds: Object.keys(cleaned),
      percentages: cleaned
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/35 px-4 py-8">
      <div className="w-full max-w-2xl rounded-lg border border-slate-200 bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">{title}</h2>
            <p className="mt-1 text-sm text-slate-600">
              {formatMoney(costAmount, costCurrency)} estimated planning cost
            </p>
          </div>
          <Button onClick={onClose} size="sm" type="button" variant="ghost">
            Close
          </Button>
        </div>

        {activeTravelers.length === 0 ? (
          <div className="mt-4 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
            Add at least one active traveler before saving a split rule.
          </div>
        ) : null}

        <div className="mt-5 space-y-4">
          <label className="block text-sm font-medium text-slate-700" htmlFor="cost-split-type">
            Split type
          </label>
          <Select
            id="cost-split-type"
            onChange={(event) => setSplitType(event.target.value as CostSplitType)}
            value={splitType}
          >
            <option value="all_equal">Equal among all travelers</option>
            <option value="selected_equal">Equal among selected travelers</option>
            <option value="custom_percentages">Custom percentages</option>
          </Select>

          {splitType === "selected_equal" ? (
            <TravelerChecklist
              selectedTravelerIds={selectedTravelerIds}
              travelers={activeTravelers}
              onToggle={toggleTraveler}
            />
          ) : null}

          {splitType === "custom_percentages" ? (
            <div className="space-y-3">
              {activeTravelers.map((traveler) => (
                <label className="grid gap-2 text-sm sm:grid-cols-[minmax(0,1fr)_8rem]" key={traveler.id}>
                  <span className="font-medium text-slate-700">{traveler.name}</span>
                  <Input
                    min={0}
                    onChange={(event) =>
                      setPercentages((current) => ({
                        ...current,
                        [traveler.id]: Number(event.target.value)
                      }))
                    }
                    step="0.01"
                    type="number"
                    value={percentages[traveler.id] ?? 0}
                  />
                </label>
              ))}
              <p className={Math.abs(percentTotal - 100) <= 0.01 ? "text-sm text-emerald-700" : "text-sm text-red-700"}>
                Total: {percentTotal.toFixed(2)}%
              </p>
            </div>
          ) : null}

          <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
            <p className="text-sm font-medium text-slate-700">Preview</p>
            <ul className="mt-2 space-y-1 text-sm">
              {preview.length > 0 ? (
                preview.map((row) => (
                  <li className="flex items-center justify-between gap-3" key={row.travelerId}>
                    <span className="text-slate-600">{row.name}</span>
                    <span className="font-medium text-slate-900">
                      {formatMoney(row.amount, costCurrency)}
                    </span>
                  </li>
                ))
              ) : (
                <li className="text-slate-500">No allocation with the current rule.</li>
              )}
            </ul>
          </div>
        </div>

        {error ? (
          <p className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
            {error}
          </p>
        ) : null}

        <div className="mt-6 flex justify-end gap-2">
          <Button disabled={isSaving} onClick={onClose} type="button" variant="secondary">
            Cancel
          </Button>
          <Button disabled={!canSave || isSaving} onClick={save} type="button">
            {isSaving ? "Saving..." : "Save split"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function TravelerChecklist({
  travelers,
  selectedTravelerIds,
  onToggle
}: {
  travelers: TripTraveler[];
  selectedTravelerIds: string[];
  onToggle: (travelerId: string) => void;
}) {
  return (
    <div className="space-y-2">
      {travelers.map((traveler) => (
        <label className="flex items-center gap-2 text-sm text-slate-700" key={traveler.id}>
          <input
            checked={selectedTravelerIds.includes(traveler.id)}
            className="h-4 w-4 rounded border-slate-300 text-primary-600"
            onChange={() => onToggle(traveler.id)}
            type="checkbox"
          />
          <span>{traveler.name}</span>
        </label>
      ))}
    </div>
  );
}

function buildPreview(
  splitType: CostSplitType,
  travelers: TripTraveler[],
  selectedTravelerIds: string[],
  percentages: Record<string, number>,
  amount: number
) {
  if (splitType === "all_equal") {
    return equalPreview(travelers, amount);
  }
  if (splitType === "selected_equal") {
    return equalPreview(
      travelers.filter((traveler) => selectedTravelerIds.includes(traveler.id)),
      amount
    );
  }
  return travelers
    .map((traveler) => ({
      travelerId: traveler.id,
      name: traveler.name,
      amount: amount * ((percentages[traveler.id] ?? 0) / 100)
    }))
    .filter((row) => row.amount > 0);
}

function equalPreview(travelers: TripTraveler[], amount: number) {
  if (travelers.length === 0) {
    return [];
  }
  const share = amount / travelers.length;
  return travelers.map((traveler) => ({
    travelerId: traveler.id,
    name: traveler.name,
    amount: share
  }));
}

function equalPercentages(travelerIds: string[]) {
  if (travelerIds.length === 0) {
    return {};
  }
  const share = 100 / travelerIds.length;
  return Object.fromEntries(travelerIds.map((id) => [id, Number(share.toFixed(2))]));
}
