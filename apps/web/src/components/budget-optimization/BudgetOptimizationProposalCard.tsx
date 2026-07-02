"use client";

import { useState } from "react";
import { BudgetOptimizationPreview } from "@/components/budget-optimization/BudgetOptimizationPreview";
import { Button } from "@/components/ui/Button";
import { formatApproxMoney, formatMoney } from "@/lib/budget/format";
import { cn } from "@/lib/utils";
import type { BudgetOptimizationProposal } from "@/types/budget-optimization";
import type { ItineraryDay } from "@/types/trip";

type BudgetOptimizationProposalCardProps = {
  proposal: BudgetOptimizationProposal;
  currentDay?: ItineraryDay | null;
  canMutate: boolean;
  isApplying?: boolean;
  isDiscarding?: boolean;
  onApply: (proposal: BudgetOptimizationProposal) => Promise<void>;
  onDiscard: (proposal: BudgetOptimizationProposal) => Promise<void>;
};

export function BudgetOptimizationProposalCard({
  proposal,
  currentDay,
  canMutate,
  isApplying = false,
  isDiscarding = false,
  onApply,
  onDiscard
}: BudgetOptimizationProposalCardProps) {
  const [showPreview, setShowPreview] = useState(false);
  const content = proposal.proposal;
  const currency = proposal.currency;
  const canApply = canMutate && proposal.status === "pending";

  return (
    <article className="rounded-lg border border-slate-200 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-base font-semibold text-slate-950">
              Day {proposal.dayNumber ?? content.dayNumber} Budget Proposal
            </h3>
            <StatusPill status={proposal.status} />
            <span className="rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium capitalize text-slate-600">
              {content.confidence} confidence
            </span>
          </div>
          <p className="mt-2 text-sm leading-6 text-slate-600">{content.summary}</p>
        </div>
        <div className="text-left sm:text-right">
          <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
            Approx. savings
          </p>
          <p className="mt-1 text-lg font-semibold text-emerald-700">
            {formatApproxMoney(
              proposal.estimatedSavingsAmount ?? content.estimatedSavingsAmount,
              currency
            )}
          </p>
        </div>
      </div>

      <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-3">
        <Metric
          label="Current day"
          value={formatMoney(content.baseDayEstimatedTotal, currency)}
        />
        <Metric
          label="Proposed day"
          value={formatMoney(content.proposedDayEstimatedTotal, currency)}
        />
        <Metric
          label="Target"
          value={
            proposal.targetReductionAmount != null
              ? formatMoney(proposal.targetReductionAmount, currency)
              : "Flexible"
          }
        />
      </dl>

      {content.changes.length > 0 ? (
        <div className="mt-4">
          <p className="text-sm font-semibold text-slate-950">Changes</p>
          <ul className="mt-2 space-y-2">
            {content.changes.map((change, index) => (
              <li className="rounded-md border border-slate-200 bg-slate-50 p-3 text-sm" key={`${change.type}-${index}`}>
                <p className="font-medium text-slate-900">{formatChangeTitle(change)}</p>
                {change.reason ? (
                  <p className="mt-1 leading-5 text-slate-600">{change.reason}</p>
                ) : null}
                {change.estimatedSavingsAmount != null ? (
                  <p className="mt-1 text-xs font-medium text-emerald-700">
                    Saves about{" "}
                    {formatApproxMoney(change.estimatedSavingsAmount, change.currency ?? currency)}
                  </p>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {content.tradeoffs?.length ? (
        <NoteList title="Tradeoffs" items={content.tradeoffs} />
      ) : null}
      {content.warnings?.length ? <NoteList title="Warnings" items={content.warnings} /> : null}

      {showPreview ? (
        <div className="mt-4">
          <BudgetOptimizationPreview currentDay={currentDay} proposal={proposal} />
        </div>
      ) : null}

      <div className="mt-4 flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button
          onClick={() => setShowPreview((value) => !value)}
          type="button"
          variant="secondary"
        >
          {showPreview ? "Hide preview" : "Preview day"}
        </Button>
        {canApply ? (
          <>
            <Button
              disabled={isDiscarding || isApplying}
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
              {isApplying ? "Applying..." : "Apply"}
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

function StatusPill({ status }: { status: BudgetOptimizationProposal["status"] }) {
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

function formatChangeTitle(change: BudgetOptimizationProposal["proposal"]["changes"][number]) {
  const oldName = change.oldItemName?.trim();
  const newName = change.newItemName?.trim();

  if (change.type === "replace_item") {
    return oldName && newName ? `Replace ${oldName} with ${newName}` : "Replace item";
  }
  if (change.type === "remove_item") {
    return oldName ? `Remove ${oldName}` : "Remove item";
  }
  if (change.type === "add_item") {
    return newName ? `Add ${newName}` : "Add item";
  }
  if (change.type === "modify_item_cost") {
    return oldName ?? newName ?? "Update item cost";
  }
  if (change.type === "reorder_item") {
    return oldName ?? newName ?? "Reorder item";
  }
  return oldName ?? newName ?? "Keep item";
}
