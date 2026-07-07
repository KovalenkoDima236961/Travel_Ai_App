"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { useCreateTemplateAdaptationJob } from "../model/useCreateTemplateAdaptationJob";
import { useTemplateAdaptationJob } from "../model/useTemplateAdaptationJob";
import { getErrorMessage } from "@/lib/utils";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { TripTemplate } from "@/entities/trip-template/model";

type AdaptTemplateWithAiDialogProps = {
  open: boolean;
  template: TripTemplate | null;
  onClose: () => void;
  onUseDirectly?: (template: TripTemplate) => void;
};

const STATUS_LABEL: Record<GenerationJob["status"], string> = {
  queued: "Queued",
  running: "Adapting template with AI…",
  completed: "Completed",
  failed: "Failed",
  cancelled: "Cancelled"
};

export function AdaptTemplateWithAiDialog({
  open,
  template,
  onClose,
  onUseDirectly
}: AdaptTemplateWithAiDialogProps) {
  const router = useRouter();
  const { editableWorkspaces, currentWorkspace } = useWorkspaces();
  const createJob = useCreateTemplateAdaptationJob();

  const [title, setTitle] = useState("");
  const [destination, setDestination] = useState("");
  const [startDate, setStartDate] = useState("");
  const [durationDays, setDurationDays] = useState("3");
  const [workspaceId, setWorkspaceId] = useState("");
  const [budgetAmount, setBudgetAmount] = useState("");
  const [budgetCurrency, setBudgetCurrency] = useState("EUR");
  const [travelers, setTravelers] = useState("2");
  const [pace, setPace] = useState("balanced");
  const [interests, setInterests] = useState("");
  const [avoid, setAvoid] = useState("");
  const [specialInstructions, setSpecialInstructions] = useState("");
  const [fallback, setFallback] = useState(true);
  const [localError, setLocalError] = useState<string | null>(null);
  const [job, setJob] = useState<{ id: string; tripId: string } | null>(null);

  useEffect(() => {
    if (!open || !template) {
      return;
    }
    setTitle(defaultAdaptTitle(template.title));
    setDestination("");
    setStartDate("");
    setDurationDays(String(template.durationDays || 3));
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
    setInterests("");
    setAvoid("");
    setSpecialInstructions("");
    setFallback(true);
    setLocalError(null);
    setJob(null);
  }, [currentWorkspace, editableWorkspaces, open, template]);

  const tracked = useTemplateAdaptationJob({
    tripId: job?.tripId,
    jobId: job?.id,
    enabled: Boolean(job)
  });

  const summary = tracked.summary;
  const status = tracked.job?.status ?? (job ? "queued" : null);

  const durationHint = useMemo(() => {
    if (!template) {
      return "";
    }
    return `Template is ${template.durationDays} ${
      template.durationDays === 1 ? "day" : "days"
    }.`;
  }, [template]);

  if (!open || !template) {
    return null;
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!template) {
      return;
    }
    const parsedDuration = Number(durationDays);
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
    if (!Number.isInteger(parsedDuration) || parsedDuration < 1 || parsedDuration > 30) {
      setLocalError("Duration must be between 1 and 30 days.");
      return;
    }
    if (parsedBudget != null && (!Number.isFinite(parsedBudget) || parsedBudget < 0)) {
      setLocalError("Budget must be zero or greater.");
      return;
    }
    if (specialInstructions.length > 1000) {
      setLocalError("Special instructions must be at most 1000 characters.");
      return;
    }
    try {
      setLocalError(null);
      const created = await createJob.mutateAsync({
        templateId: template.id,
        input: {
          title,
          destination,
          startDate,
          durationDays: parsedDuration,
          workspaceId: workspaceId || null,
          budget:
            parsedBudget != null
              ? { amount: parsedBudget, currency: budgetCurrency }
              : null,
          travelers: parsedTravelers,
          pace,
          interests: interests ? interests.split(",") : [],
          avoid: avoid ? avoid.split(",") : [],
          specialInstructions,
          fallbackToDeterministic: fallback
        }
      });
      setJob({ id: created.id, tripId: created.tripId });
    } catch (error) {
      setLocalError(getErrorMessage(error, "Could not start AI adaptation."));
    }
  }

  function openTrip() {
    if (tracked.createdTripId) {
      onClose();
      router.push(`/trips/${tracked.createdTripId}?adaptedFromTemplate=${template!.id}`);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <Card className="max-h-[90vh] w-full max-w-2xl overflow-y-auto">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Adapt with AI</h2>
            <p className="mt-1 text-sm text-slate-600">
              Re-target “{template.title}” to a new destination. {durationHint} The result is a
              draft you can review and edit.
            </p>
          </div>
          <Button onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>

        {job ? (
          <AdaptationJobStatus
            status={status}
            errorMessage={tracked.job?.errorCode ? tracked.job?.errorMessage ?? tracked.job?.errorCode : null}
            summary={summary}
            fallbackWasEnabled={fallback}
            onOpenTrip={openTrip}
            onUseDirectly={onUseDirectly ? () => onUseDirectly(template) : undefined}
            onClose={onClose}
          />
        ) : (
          <form className="mt-6 space-y-5" onSubmit={submit}>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block text-sm font-medium text-slate-700">
                Trip title
                <Input className="mt-2" onChange={(e) => setTitle(e.target.value)} value={title} />
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Destination
                <Input
                  className="mt-2"
                  onChange={(e) => setDestination(e.target.value)}
                  placeholder="e.g. Vienna"
                  value={destination}
                />
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Start date
                <Input className="mt-2" onChange={(e) => setStartDate(e.target.value)} type="date" value={startDate} />
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Duration (days)
                <Input
                  className="mt-2"
                  max="30"
                  min="1"
                  onChange={(e) => setDurationDays(e.target.value)}
                  type="number"
                  value={durationDays}
                />
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Scope
                <Select className="mt-2" onChange={(e) => setWorkspaceId(e.target.value)} value={workspaceId}>
                  <option value="">Personal trip</option>
                  {editableWorkspaces.map((workspace) => (
                    <option key={workspace.id} value={workspace.id}>
                      {workspace.name}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Pace
                <Select className="mt-2" onChange={(e) => setPace(e.target.value)} value={pace}>
                  <option value="relaxed">Relaxed</option>
                  <option value="balanced">Balanced</option>
                  <option value="intensive">Intensive</option>
                </Select>
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Budget amount
                <Input
                  className="mt-2"
                  min="0"
                  onChange={(e) => setBudgetAmount(e.target.value)}
                  placeholder="Optional"
                  step="0.01"
                  type="number"
                  value={budgetAmount}
                />
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Budget currency
                <Select className="mt-2" onChange={(e) => setBudgetCurrency(e.target.value)} value={budgetCurrency}>
                  {["EUR", "USD", "GBP", "CZK"].map((code) => (
                    <option key={code} value={code}>
                      {code}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Travelers
                <Input className="mt-2" min="1" onChange={(e) => setTravelers(e.target.value)} type="number" value={travelers} />
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Interests
                <Input
                  className="mt-2"
                  onChange={(e) => setInterests(e.target.value)}
                  placeholder="museums, food, architecture"
                  value={interests}
                />
              </label>
              <label className="block text-sm font-medium text-slate-700">
                Avoid
                <Input
                  className="mt-2"
                  onChange={(e) => setAvoid(e.target.value)}
                  placeholder="nightclubs"
                  value={avoid}
                />
              </label>
            </div>
            <label className="block text-sm font-medium text-slate-700">
              Special instructions
              <Textarea
                className="mt-2"
                maxLength={1000}
                onChange={(e) => setSpecialInstructions(e.target.value)}
                placeholder="e.g. Make it suitable for first-time visitors."
                rows={3}
                value={specialInstructions}
              />
            </label>
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input
                checked={fallback}
                onChange={(e) => setFallback(e.target.checked)}
                type="checkbox"
              />
              If AI adaptation fails, create a deterministic template copy instead
            </label>
            {localError ? (
              <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
                {localError}
              </div>
            ) : null}
            <div className="flex flex-wrap justify-end gap-2">
              <Button onClick={onClose} type="button" variant="secondary">
                Cancel
              </Button>
              <Button disabled={createJob.isPending} type="submit">
                {createJob.isPending ? "Starting…" : "Adapt with AI"}
              </Button>
            </div>
          </form>
        )}
      </Card>
    </div>
  );
}

function AdaptationJobStatus({
  status,
  errorMessage,
  summary,
  fallbackWasEnabled,
  onOpenTrip,
  onUseDirectly,
  onClose
}: {
  status: GenerationJob["status"] | null;
  errorMessage: string | null;
  summary: import("@/entities/template-adaptation/model").TemplateAdaptationSummary | null;
  fallbackWasEnabled: boolean;
  onOpenTrip: () => void;
  onUseDirectly?: () => void;
  onClose: () => void;
}) {
  const isTerminal = status === "completed" || status === "failed" || status === "cancelled";
  return (
    <div className="mt-6 space-y-4" data-testid="adaptation-job-status">
      <div className="rounded-md border border-slate-200 bg-slate-50 p-4">
        <p className="text-sm font-medium text-slate-800">
          {status ? STATUS_LABEL[status] : "Starting…"}
        </p>
        {!isTerminal ? (
          <p className="mt-1 text-sm text-slate-600">
            Adapting the template, validating the itinerary, and creating your trip…
          </p>
        ) : null}
      </div>

      {status === "completed" ? (
        <div className="space-y-3">
          <p className="text-sm font-medium text-emerald-700">Trip created.</p>
          {summary?.fallbackUsed ? (
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
              AI adaptation failed, so this trip was created as a deterministic template copy.
            </div>
          ) : null}
          {summary?.majorChanges?.length ? (
            <div>
              <p className="text-sm font-medium text-slate-700">Major changes</p>
              <ul className="mt-1 list-disc pl-5 text-sm text-slate-600">
                {summary.majorChanges.map((change, index) => (
                  <li key={index}>{change}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {summary?.warnings?.length ? (
            <div>
              <p className="text-sm font-medium text-amber-700">Please review</p>
              <ul className="mt-1 list-disc pl-5 text-sm text-amber-700">
                {summary.warnings.map((warning, index) => (
                  <li key={index}>{warning}</li>
                ))}
              </ul>
            </div>
          ) : null}
          <div className="flex flex-wrap justify-end gap-2">
            <Button onClick={onClose} type="button" variant="secondary">
              Close
            </Button>
            <Button onClick={onOpenTrip} type="button">
              Open trip
            </Button>
          </div>
        </div>
      ) : null}

      {status === "failed" ? (
        <div className="space-y-3">
          <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {errorMessage ?? "AI adaptation failed."}
          </div>
          {!fallbackWasEnabled ? (
            <p className="text-sm text-slate-600">
              Retry with the deterministic fallback enabled, or use the template directly.
            </p>
          ) : null}
          <div className="flex flex-wrap justify-end gap-2">
            {onUseDirectly ? (
              <Button onClick={onUseDirectly} type="button" variant="secondary">
                Use template directly
              </Button>
            ) : null}
            <Button onClick={onClose} type="button">
              Close
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function defaultAdaptTitle(templateTitle: string) {
  const stripped = templateTitle.replace(/\btemplate\b/gi, "").trim();
  return stripped ? `Trip from ${stripped}` : `Trip from ${templateTitle}`;
}
