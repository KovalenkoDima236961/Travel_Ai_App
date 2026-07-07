import { BudgetOptimizationProposalCard } from "@/features/budget-optimization";
import type { BudgetOptimizationProposal } from "@/entities/budget-optimization/model";
import type { Itinerary } from "@/entities/trip/model";
import { findProposalCurrentDay } from "../model/tripDetailPageModel";

type BudgetOptimizationProposalsPanelProps = {
  proposals: BudgetOptimizationProposal[];
  currentItinerary: Itinerary | null;
  canMutate: boolean;
  error: string | null;
  isLoading: boolean;
  isApplying: boolean;
  isDiscarding: boolean;
  onApply: (proposal: BudgetOptimizationProposal) => Promise<void>;
  onDiscard: (proposal: BudgetOptimizationProposal) => Promise<void>;
};

export function BudgetOptimizationProposalsPanel({
  proposals,
  currentItinerary,
  canMutate,
  error,
  isLoading,
  isApplying,
  isDiscarding,
  onApply,
  onDiscard
}: BudgetOptimizationProposalsPanelProps) {
  if (!isLoading && proposals.length === 0 && !error) {
    return null;
  }

  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xl font-semibold text-slate-950">
          Budget Optimization Proposals
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          Review cheaper day plans before applying them to the itinerary.
        </p>
      </div>

      {error ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          {error}
        </div>
      ) : null}

      {isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-600">
          Loading budget optimization proposals...
        </div>
      ) : null}

      {proposals.map((proposal) => (
        <BudgetOptimizationProposalCard
          canMutate={canMutate}
          currentDay={findProposalCurrentDay(currentItinerary, proposal)}
          isApplying={isApplying}
          isDiscarding={isDiscarding}
          key={proposal.id}
          onApply={onApply}
          onDiscard={onDiscard}
          proposal={proposal}
        />
      ))}
    </section>
  );
}
