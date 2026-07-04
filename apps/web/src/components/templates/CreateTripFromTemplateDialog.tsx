"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { useTripTemplateMutations } from "@/hooks/useTripTemplates";
import { getErrorMessage } from "@/lib/utils";
import type { Trip } from "@/types/trip";
import type { TripTemplate } from "@/types/trip-template";

type CreateTripFromTemplateDialogProps = {
  open: boolean;
  template: TripTemplate | null;
  onClose: () => void;
  onCreated?: (trip: Trip) => void;
};

export function CreateTripFromTemplateDialog({
  open,
  template,
  onClose,
  onCreated
}: CreateTripFromTemplateDialogProps) {
  const router = useRouter();
  const { editableWorkspaces, currentWorkspace } = useWorkspaces();
  const mutations = useTripTemplateMutations();
  const [title, setTitle] = useState("");
  const [destination, setDestination] = useState("");
  const [startDate, setStartDate] = useState("");
  const [workspaceId, setWorkspaceId] = useState("");
  const [budgetAmount, setBudgetAmount] = useState("");
  const [budgetCurrency, setBudgetCurrency] = useState("EUR");
  const [travelers, setTravelers] = useState("2");
  const [pace, setPace] = useState("balanced");
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    if (!open || !template) {
      return;
    }
    setTitle(defaultTripTitle(template.title));
    setDestination(template.destinationHint ?? "");
    setStartDate("");
    setWorkspaceId(
      currentWorkspace &&
        editableWorkspaces.some((workspace) => workspace.id === currentWorkspace.id)
        ? currentWorkspace.id
        : ""
    );
    setBudgetAmount("");
    setBudgetCurrency(template.defaultCurrency || template.estimatedTotalCurrency || "EUR");
    setTravelers("2");
    setPace("balanced");
    setLocalError(null);
  }, [currentWorkspace, editableWorkspaces, open, template]);

  const estimateLabel = useMemo(() => {
    if (!template?.estimatedTotalAmount) {
      return null;
    }
    return `${template.estimatedTotalAmount} ${
      template.estimatedTotalCurrency || template.defaultCurrency || "EUR"
    } estimated`;
  }, [template]);

  if (!open || !template) {
    return null;
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!template) {
      return;
    }
    const parsedBudget = budgetAmount.trim() ? Number(budgetAmount) : null;
    const parsedTravelers = Number(travelers);
    if (!title.trim()) {
      setLocalError("Trip title is required.");
      return;
    }
    if (!destination.trim()) {
      setLocalError("Destination is required.");
      return;
    }
    if (!startDate) {
      setLocalError("Start date is required.");
      return;
    }
    if (parsedBudget != null && (!Number.isFinite(parsedBudget) || parsedBudget < 0)) {
      setLocalError("Budget must be zero or greater.");
      return;
    }
    if (!Number.isInteger(parsedTravelers) || parsedTravelers < 1) {
      setLocalError("Travelers must be at least 1.");
      return;
    }
    try {
      setLocalError(null);
      const trip = await mutations.createTripFromTemplate.mutateAsync({
        templateId: template.id,
        input: {
          title,
          destination,
          startDate,
          workspaceId: workspaceId || null,
          budget:
            parsedBudget != null
              ? { amount: parsedBudget, currency: budgetCurrency }
              : null,
          travelers: parsedTravelers,
          pace
        }
      });
      onCreated?.(trip);
      onClose();
      router.push(`/trips/${trip.id}`);
    } catch (error) {
      setLocalError(getErrorMessage(error, "Could not create trip from template."));
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <Card className="max-h-[90vh] w-full max-w-2xl overflow-y-auto">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Use template</h2>
            <p className="mt-1 text-sm text-slate-600">
              {template.durationDays} {template.durationDays === 1 ? "day" : "days"}
              {estimateLabel ? ` · ${estimateLabel}` : ""}
            </p>
          </div>
          <Button onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>
        <form className="mt-6 space-y-5" onSubmit={submit}>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-medium text-slate-700">
              Trip title
              <Input className="mt-2" onChange={(event) => setTitle(event.target.value)} value={title} />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Destination
              <Input
                className="mt-2"
                onChange={(event) => setDestination(event.target.value)}
                value={destination}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Start date
              <Input
                className="mt-2"
                onChange={(event) => setStartDate(event.target.value)}
                type="date"
                value={startDate}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Scope
              <Select
                className="mt-2"
                onChange={(event) => setWorkspaceId(event.target.value)}
                value={workspaceId}
              >
                <option value="">Personal trip</option>
                {editableWorkspaces.map((workspace) => (
                  <option key={workspace.id} value={workspace.id}>
                    {workspace.name}
                  </option>
                ))}
              </Select>
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Budget amount
              <Input
                className="mt-2"
                min="0"
                onChange={(event) => setBudgetAmount(event.target.value)}
                placeholder="Optional"
                step="0.01"
                type="number"
                value={budgetAmount}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Budget currency
              <Select
                className="mt-2"
                onChange={(event) => setBudgetCurrency(event.target.value)}
                value={budgetCurrency}
              >
                {["EUR", "USD", "GBP", "CZK"].map((code) => (
                  <option key={code} value={code}>
                    {code}
                  </option>
                ))}
              </Select>
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Travelers
              <Input
                className="mt-2"
                min="1"
                onChange={(event) => setTravelers(event.target.value)}
                type="number"
                value={travelers}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Pace
              <Select className="mt-2" onChange={(event) => setPace(event.target.value)} value={pace}>
                <option value="relaxed">Relaxed</option>
                <option value="balanced">Balanced</option>
                <option value="packed">Intensive</option>
              </Select>
            </label>
          </div>
          {localError ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {localError}
            </div>
          ) : null}
          <div className="flex flex-wrap justify-end gap-2">
            <Button onClick={onClose} type="button" variant="secondary">
              Cancel
            </Button>
            <Button disabled={mutations.createTripFromTemplate.isPending} type="submit">
              {mutations.createTripFromTemplate.isPending ? "Creating..." : "Create trip"}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}

function defaultTripTitle(templateTitle: string) {
  const stripped = templateTitle.replace(/\btemplate\b/gi, "").trim();
  return stripped ? `Trip from ${stripped}` : `Trip from ${templateTitle}`;
}
