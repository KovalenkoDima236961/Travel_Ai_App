"use client";

import { useState } from "react";
import { RepairJobStatusCard } from "./RepairJobStatusCard";
import { RepairProposalPreview } from "./RepairProposalPreview";
import { Button } from "@/shared/ui/button";
import { formatApproxMoney } from "@/entities/budget/model";
import { cn } from "@/lib/utils";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { RepairProposal } from "@/entities/trip-repair/model";
import type { Itinerary } from "@/entities/trip/model";

type RepairProposalsPanelProps = {
  tripId: string;
  currentItinerary: Itinerary;
  proposals: RepairProposal[];
  activeJob?: GenerationJob | null;
  canMutate: boolean;
  error?: string | null;
  isLoading?: boolean;
  isApplying?: boolean;
  isDiscarding?: boolean;
  isCancellingJob?: boolean;
  onApply: (proposal: RepairProposal) => Promise<void>;
  onCancelJob?: (job: GenerationJob) => Promise<void> | void;
  onCreateRepair?: () => void;
  onDiscard: (proposal: RepairProposal) => Promise<void>;
};

export function RepairProposalsPanel({
  tripId,
  currentItinerary,
  proposals,
  activeJob,
  canMutate,
  error,
  isLoading = false,
  isApplying = false,
  isDiscarding = false,
  isCancellingJob = false,
  onApply,
  onCancelJob,
  onCreateRepair,
  onDiscard
}: RepairProposalsPanelProps) {
  if (!isLoading && proposals.length === 0 && !activeJob && !error && !onCreateRepair) {
    return null;
  }

  return (
    <section id="repair" className="scroll-mt-24 rounded-lg border border-slate-200 bg-white p-5 shadow-soft">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">
            Policy-aware repair
          </p>
          <h2 className="mt-1 text-lg font-semibold text-slate-950">AI Repair Proposals</h2>
          <p className="mt-1 text-sm leading-6 text-slate-600">
            Review policy, risk, and schedule repairs before applying them.
          </p>
        </div>
        {onCreateRepair ? (
          <Button disabled={!canMutate || Boolean(activeJob)} onClick={onCreateRepair} type="button">
            Repair with AI
          </Button>
        ) : null}
      </div>

      <div className="mt-4 space-y-4">
        {activeJob ? (
          <RepairJobStatusCard
            cancelling={isCancellingJob}
            job={activeJob}
            onCancel={onCancelJob}
          />
        ) : null}
        {error ? (
          <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {error}
          </div>
        ) : null}
        {isLoading ? (
          <p className="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm text-slate-500">
            Loading repair proposals...
          </p>
        ) : null}
        {!isLoading && proposals.length === 0 && !activeJob ? (
          <p className="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm text-slate-500">
            No pending repair proposals.
          </p>
        ) : null}
        {proposals.map((proposal) => (
          <RepairProposalCard
            canMutate={canMutate}
            currentItinerary={currentItinerary}
            isApplying={isApplying}
            isDiscarding={isDiscarding}
            key={proposal.id}
            onApply={onApply}
            onDiscard={onDiscard}
            proposal={proposal}
            tripId={tripId}
          />
        ))}
      </div>
    </section>
  );
}

function RepairProposalCard({
  tripId,
  proposal,
  currentItinerary,
  canMutate,
  isApplying,
  isDiscarding,
  onApply,
  onDiscard
}: {
  tripId: string;
  proposal: RepairProposal;
  currentItinerary: Itinerary;
  canMutate: boolean;
  isApplying: boolean;
  isDiscarding: boolean;
  onApply: (proposal: RepairProposal) => Promise<void>;
  onDiscard: (proposal: RepairProposal) => Promise<void>;
}) {
  const [showPreview, setShowPreview] = useState(false);
  const summary = proposal.summary;
  const canApply = canMutate && proposal.status === "pending";

  return (
    <article className="rounded-lg border border-slate-200 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-base font-semibold text-slate-950">
              {repairModeLabel(proposal.repairMode)}
            </h3>
            <StatusPill status={proposal.status} />
          </div>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            {summary.majorChanges[0] ?? "AI generated a repair proposal for this itinerary."}
          </p>
        </div>
        <RiskDelta proposal={proposal} />
      </div>

      <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-4">
        <Metric label="Changed" value={String(summary.changedItemCount)} />
        <Metric label="Added" value={String(summary.addedItemCount)} />
        <Metric label="Removed" value={String(summary.removedItemCount)} />
        <Metric label="Moved" value={String(summary.movedItemCount)} />
      </dl>

      {summary.estimatedCostBefore && summary.estimatedCostAfter ? (
        <p className="mt-3 text-sm text-slate-600">
          Cost moves from{" "}
          <span className="font-medium text-slate-900">
            {formatApproxMoney(
              summary.estimatedCostBefore.amount,
              summary.estimatedCostBefore.currency
            )}
          </span>{" "}
          to{" "}
          <span className="font-medium text-slate-900">
            {formatApproxMoney(
              summary.estimatedCostAfter.amount,
              summary.estimatedCostAfter.currency
            )}
          </span>
          .
        </p>
      ) : null}

      {summary.issuesAddressed.length > 0 ? (
        <NoteList title="Addressed" items={summary.issuesAddressed.slice(0, 4)} />
      ) : null}
      {summary.issuesRemaining.length > 0 ? (
        <NoteList title="Remaining" items={summary.issuesRemaining.slice(0, 4)} />
      ) : null}
      {summary.warnings.length > 0 ? (
        <NoteList title="Warnings" items={summary.warnings.slice(0, 3)} />
      ) : null}

      {showPreview ? (
        <div className="mt-4">
          <RepairProposalPreview
            currentItinerary={currentItinerary}
            proposal={proposal}
            tripId={tripId}
          />
        </div>
      ) : null}

      <div className="mt-4 flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button
          onClick={() => setShowPreview((value) => !value)}
          type="button"
          variant="secondary"
        >
          {showPreview ? "Hide preview" : "Preview repair"}
        </Button>
        {canApply ? (
          <>
            <Button
              disabled={isApplying || isDiscarding}
              onClick={() => onDiscard(proposal)}
              type="button"
              variant="secondary"
            >
              {isDiscarding ? "Discarding..." : "Discard"}
            </Button>
            <Button
              disabled={isApplying || isDiscarding}
              onClick={() => onApply(proposal)}
              type="button"
            >
              {isApplying ? "Applying..." : "Apply repair"}
            </Button>
          </>
        ) : null}
      </div>
    </article>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-1 font-semibold text-slate-950">{value}</dd>
    </div>
  );
}

function NoteList({ title, items }: { title: string; items: string[] }) {
  return (
    <div className="mt-4">
      <p className="text-sm font-semibold text-slate-950">{title}</p>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-sm leading-6 text-slate-600">
        {items.map((item, index) => (
          <li key={`${title}-${index}`}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

function RiskDelta({ proposal }: { proposal: RepairProposal }) {
  if (proposal.baseRiskScore == null || proposal.proposedRiskScore == null) {
    return null;
  }
  const delta = proposal.proposedRiskScore - proposal.baseRiskScore;
  return (
    <div className="text-left sm:text-right">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">Risk score</p>
      <p className={cn("mt-1 text-lg font-semibold", delta <= 0 ? "text-emerald-700" : "text-amber-700")}>
        {proposal.baseRiskScore}
        {" -> "}
        {proposal.proposedRiskScore}
      </p>
    </div>
  );
}

function StatusPill({ status }: { status: RepairProposal["status"] }) {
  return (
    <span
      className={cn(
        "rounded-full border px-2.5 py-1 text-xs font-medium capitalize",
        status === "pending" && "border-amber-200 bg-amber-50 text-amber-800",
        status === "applied" && "border-emerald-200 bg-emerald-50 text-emerald-700",
        (status === "discarded" || status === "expired") &&
          "border-slate-200 bg-slate-50 text-slate-600",
        status === "failed" && "border-red-200 bg-red-50 text-red-700"
      )}
    >
      {status}
    </span>
  );
}

function repairModeLabel(mode: string) {
  return mode
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
