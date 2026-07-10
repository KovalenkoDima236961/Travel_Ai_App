"use client";

import { FormEvent, useEffect, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import type { Workspace } from "@/entities/workspace/model";
import type {
  CreateTripFromRouteAlternativeInput,
  RouteAlternative
} from "@/types/route-alternatives";

type CreateTripFromRouteAlternativeDialogProps = {
  alternative: RouteAlternative | null;
  sessionWorkspaceId?: string | null;
  workspaces?: Workspace[];
  isPending?: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: (input: CreateTripFromRouteAlternativeInput) => void;
};

export function CreateTripFromRouteAlternativeDialog({
  alternative,
  sessionWorkspaceId = null,
  workspaces = [],
  isPending = false,
  error = null,
  onClose,
  onConfirm
}: CreateTripFromRouteAlternativeDialogProps) {
  const [title, setTitle] = useState("");
  const [startDate, setStartDate] = useState("");
  const [budgetAmount, setBudgetAmount] = useState("");
  const [budgetCurrency, setBudgetCurrency] = useState("EUR");
  const [travelers, setTravelers] = useState("2");
  const [workspaceId, setWorkspaceId] = useState("");
  const [autoGenerateItinerary, setAutoGenerateItinerary] = useState(false);

  useEffect(() => {
    if (!alternative) {
      return;
    }
    setTitle(alternative.title);
    setBudgetAmount(
      alternative.estimatedBudget?.amount != null ? String(alternative.estimatedBudget.amount) : ""
    );
    setBudgetCurrency(alternative.estimatedBudget?.currency ?? "EUR");
    setWorkspaceId(sessionWorkspaceId ?? "");
  }, [alternative, sessionWorkspaceId]);

  if (!alternative) {
    return null;
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const amount = budgetAmount === "" ? undefined : Number(budgetAmount);
    const travelerCount = travelers === "" ? undefined : Number(travelers);
    onConfirm({
      title,
      startDate: startDate || undefined,
      budget:
        amount != null && Number.isFinite(amount)
          ? { amount, currency: budgetCurrency.toUpperCase(), confidence: "medium" }
          : undefined,
      travelers: travelerCount != null && Number.isFinite(travelerCount) ? travelerCount : undefined,
      workspaceId: workspaceId || undefined,
      autoGenerateItinerary
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-cocoa-900/55 p-4 backdrop-blur-sm">
      <div className="w-full max-w-xl rounded-[20px] bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-[11px] font-bold uppercase tracking-[0.12em] text-clay">
              Create trip from route
            </p>
            <h2 className="mt-1 font-newsreader text-[27px] font-semibold text-cocoa-900">
              {alternative.title}
            </h2>
          </div>
          <Button type="button" variant="ghost" onClick={onClose}>
            Close
          </Button>
        </div>

        <form className="mt-5 space-y-4" onSubmit={submit}>
          <label className="block text-sm font-semibold text-cocoa-700">
            Title
            <Input value={title} onChange={(event) => setTitle(event.target.value)} required />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-semibold text-cocoa-700">
              Start date
              <Input type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} />
            </label>
            <label className="block text-sm font-semibold text-cocoa-700">
              Travelers
              <Input
                type="number"
                min={1}
                value={travelers}
                onChange={(event) => setTravelers(event.target.value)}
              />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-semibold text-cocoa-700">
              Budget
              <Input
                type="number"
                min={0}
                step="0.01"
                value={budgetAmount}
                onChange={(event) => setBudgetAmount(event.target.value)}
              />
            </label>
            <label className="block text-sm font-semibold text-cocoa-700">
              Currency
              <Input
                value={budgetCurrency}
                maxLength={3}
                onChange={(event) => setBudgetCurrency(event.target.value.toUpperCase())}
              />
            </label>
          </div>
          {workspaces.length > 0 ? (
            <label className="block text-sm font-semibold text-cocoa-700">
              Workspace
              <select
                className="mt-2 h-11 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none transition focus:border-primary-600 focus:ring-2 focus:ring-primary-100"
                value={workspaceId}
                onChange={(event) => setWorkspaceId(event.target.value)}
              >
                <option value="">Personal trip</option>
                {workspaces.map((workspace) => (
                  <option key={workspace.id} value={workspace.id}>
                    {workspace.name}
                  </option>
                ))}
              </select>
            </label>
          ) : null}
          <label className="flex items-center gap-2 text-sm font-medium text-cocoa-700">
            <input
              checked={autoGenerateItinerary}
              onChange={(event) => setAutoGenerateItinerary(event.target.checked)}
              type="checkbox"
            />
            Generate itinerary after creating the trip
          </label>
          {error ? (
            <div className="rounded-[12px] border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">
              {error}
            </div>
          ) : null}
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={onClose} disabled={isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Creating..." : "Create trip"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
