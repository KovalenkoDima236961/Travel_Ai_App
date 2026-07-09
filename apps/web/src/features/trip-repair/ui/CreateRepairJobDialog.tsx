"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Textarea } from "@/shared/ui/textarea";
import type { ApprovalRiskResponse } from "@/entities/approval-risk/model";
import type {
  CreateRepairJobInput,
  RepairConstraints,
  RepairMode
} from "@/entities/trip-repair/model";
import type { Trip } from "@/entities/trip/model";

const MODE_OPTIONS: { value: RepairMode; label: string }[] = [
  { value: "policy_compliance", label: "Policy compliance" },
  { value: "reduce_budget_risk", label: "Budget risk" },
  { value: "fix_schedule_risk", label: "Schedule risk" },
  { value: "reduce_walking", label: "Walking distance" },
  { value: "add_rest_time", label: "Rest time" },
  { value: "replace_disallowed_items", label: "Disallowed items" },
  { value: "selected_issues", label: "Selected issues" }
];

type CreateRepairJobDialogProps = {
  open: boolean;
  trip: Trip;
  approvalRisk?: ApprovalRiskResponse | null;
  defaultRepairMode?: RepairMode | null;
  disabled?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: CreateRepairJobInput) => Promise<void>;
};

export function CreateRepairJobDialog({
  open,
  trip,
  approvalRisk,
  defaultRepairMode,
  disabled = false,
  error,
  onClose,
  onSubmit
}: CreateRepairJobDialogProps) {
  const riskFactors = useMemo(
    () =>
      (approvalRisk?.factors ?? [])
        .filter((factor) => factor.severity === "critical" || factor.severity === "high")
        .slice(0, 8),
    [approvalRisk?.factors]
  );
  const [repairMode, setRepairMode] = useState<RepairMode>(
    defaultRepairMode ?? "policy_compliance"
  );
  const [selectedRiskFactorTypes, setSelectedRiskFactorTypes] = useState<string[]>([]);
  const [maxChangedItems, setMaxChangedItems] = useState("10");
  const [specialInstructions, setSpecialInstructions] = useState("");
  const [preserveConfirmedItems, setPreserveConfirmedItems] = useState(true);
  const [minimizeChanges, setMinimizeChanges] = useState(true);
  const [preserveUserEditedItems, setPreserveUserEditedItems] = useState(true);
  const [doNotChangeAccommodation, setDoNotChangeAccommodation] = useState(false);
  const [doNotChangeDates, setDoNotChangeDates] = useState(true);
  const [validationError, setValidationError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    setRepairMode(defaultRepairMode ?? "policy_compliance");
    setSelectedRiskFactorTypes(riskFactors.map((factor) => factor.type));
    setMaxChangedItems("10");
    setSpecialInstructions("");
    setPreserveConfirmedItems(true);
    setMinimizeChanges(true);
    setPreserveUserEditedItems(true);
    setDoNotChangeAccommodation(false);
    setDoNotChangeDates(true);
    setValidationError(null);
  }, [defaultRepairMode, open, riskFactors]);

  if (!open) {
    return null;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const parsedMaxChangedItems = parseOptionalInteger(maxChangedItems);
    if (parsedMaxChangedItems != null && (parsedMaxChangedItems < 1 || parsedMaxChangedItems > 50)) {
      setValidationError("Max changed items must be between 1 and 50.");
      return;
    }
    if (repairMode === "selected_issues" && selectedRiskFactorTypes.length === 0) {
      setValidationError("Select at least one risk factor.");
      return;
    }

    const constraints: RepairConstraints = {
      preserveConfirmedItems,
      minimizeChanges,
      preserveUserEditedItems,
      doNotChangeAccommodation,
      doNotChangeDates,
      ...(parsedMaxChangedItems != null ? { maxChangedItems: parsedMaxChangedItems } : {})
    };

    setValidationError(null);
    await onSubmit({
      expectedItineraryRevision: trip.itineraryRevision,
      repairMode,
      selectedIssueTypes: [],
      selectedRiskFactorTypes:
        repairMode === "selected_issues" ? selectedRiskFactorTypes : [],
      constraints,
      specialInstructions
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Repair Trip With AI</h2>
            <p className="mt-1 text-sm leading-6 text-slate-600">
              Create a reviewable proposal for policy, budget, schedule, and risk issues.
            </p>
          </div>
          <Button disabled={disabled} onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>

        <form className="mt-5 space-y-5" onSubmit={handleSubmit}>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="text-sm font-medium text-slate-700">Repair mode</span>
              <select
                className="mt-1 h-11 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none transition focus:border-primary-600 focus:ring-2 focus:ring-primary-100"
                disabled={disabled}
                onChange={(event) => setRepairMode(event.target.value as RepairMode)}
                value={repairMode}
              >
                {MODE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="block">
              <span className="text-sm font-medium text-slate-700">Max changed items</span>
              <Input
                disabled={disabled}
                max={50}
                min={1}
                onChange={(event) => setMaxChangedItems(event.target.value)}
                step="1"
                type="number"
                value={maxChangedItems}
              />
            </label>
          </div>

          {repairMode === "selected_issues" ? (
            <div className="rounded-md border border-slate-200 bg-slate-50 p-4">
              <p className="text-sm font-medium text-slate-800">Risk factors</p>
              {riskFactors.length > 0 ? (
                <div className="mt-3 space-y-3">
                  {riskFactors.map((factor) => (
                    <Checkbox
                      checked={selectedRiskFactorTypes.includes(factor.type)}
                      disabled={disabled}
                      key={factor.type}
                      label={`${factor.title} (${factor.severity})`}
                      onChange={(checked) =>
                        setSelectedRiskFactorTypes((current) =>
                          checked
                            ? [...current, factor.type]
                            : current.filter((value) => value !== factor.type)
                        )
                      }
                    />
                  ))}
                </div>
              ) : (
                <p className="mt-2 text-sm text-slate-500">
                  No high-severity risk factors are loaded for this trip.
                </p>
              )}
            </div>
          ) : null}

          <div className="rounded-md border border-slate-200 bg-slate-50 p-4">
            <p className="text-sm font-medium text-slate-800">Constraints</p>
            <div className="mt-3 grid gap-3 sm:grid-cols-2">
              <Checkbox
                checked={minimizeChanges}
                disabled={disabled}
                label="Minimize changes"
                onChange={setMinimizeChanges}
              />
              <Checkbox
                checked={preserveConfirmedItems}
                disabled={disabled}
                label="Preserve confirmed items"
                onChange={setPreserveConfirmedItems}
              />
              <Checkbox
                checked={preserveUserEditedItems}
                disabled={disabled}
                label="Preserve user edits"
                onChange={setPreserveUserEditedItems}
              />
              <Checkbox
                checked={doNotChangeAccommodation}
                disabled={disabled}
                label="Keep accommodation"
                onChange={setDoNotChangeAccommodation}
              />
              <Checkbox
                checked={doNotChangeDates}
                disabled={disabled}
                label="Keep trip dates"
                onChange={setDoNotChangeDates}
              />
            </div>
          </div>

          <label className="block">
            <span className="text-sm font-medium text-slate-700">Special instructions</span>
            <Textarea
              disabled={disabled}
              maxLength={1000}
              onChange={(event) => setSpecialInstructions(event.target.value)}
              placeholder="Optional constraints for the repair."
              value={specialInstructions}
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
              {disabled ? "Starting..." : "Start repair"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

function Checkbox({
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

function parseOptionalInteger(value: string): number | null {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  const parsed = Number(trimmed);
  return Number.isInteger(parsed) ? parsed : null;
}
