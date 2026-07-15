import type { BudgetOptimizationProposal } from "@/entities/budget-optimization/model";
import { getCostAmount, getCostCurrency, type EstimatedCost } from "@/entities/budget/model";
import type { CostSplitRule } from "@/entities/cost-splitting/model";
import type { GenerationJob } from "@/entities/generation-job/model";
import type {
  ConflictResolutionMap,
  ItineraryMergeResult
} from "@/entities/itinerary/model/diff-merge/types";
import type { Itinerary, Trip } from "@/entities/trip/model";
import type { PendingItineraryMutation } from "@/lib/offline/types";

export type MergeRecoveryState = {
  latestTrip: Trip;
  mergeResult: ItineraryMergeResult;
  resolutions: ConflictResolutionMap;
  offlineMutation?: PendingItineraryMutation;
};

export type CostSplitEditorTarget =
  | { type: "item"; dayNumber: number; itemIndex: number }
  | { type: "accommodation" };

export type RegeneratingTarget =
  | { type: "day"; dayNumber: number }
  | { type: "item"; dayNumber: number; itemIndex: number };

export function defaultConflictResolutions(
  mergeResult: ItineraryMergeResult
): ConflictResolutionMap {
  return Object.fromEntries(
    mergeResult.conflicts.map((conflict) => [
      conflict.conflictKey,
      conflict.resolution ?? "keep_latest"
    ])
  );
}

export function withPendingOfflineItinerary(
  trip: Trip,
  mutation: PendingItineraryMutation | null
): Trip {
  if (!mutation || mutation.tripId !== trip.id) {
    return trip;
  }

  return {
    ...trip,
    itinerary: mutation.draftItinerary,
    itineraryRevision: mutation.baseRevision
  };
}

export function getCostSplitTargetDetails(
  trip: Trip,
  target: CostSplitEditorTarget
): {
  title: string;
  amount: number;
  currency: string;
  currentSplit?: CostSplitRule | null;
} | null {
  if (target.type === "accommodation") {
    const cost = trip.accommodation?.estimatedCost;
    const amount = getCostAmount(cost);
    if (amount == null) {
      return null;
    }
    return {
      title: "Split accommodation cost",
      amount,
      currency: getCostCurrency(cost) ?? trip.budgetCurrency ?? "EUR",
      currentSplit: cost?.split ?? null
    };
  }

  const day = (trip.itinerary?.days ?? []).find(
    (candidate, index) => (candidate.day || index + 1) === target.dayNumber
  );
  const item = day?.items?.[target.itemIndex] ?? null;
  const cost = item?.estimatedCost;
  const amount = getCostAmount(cost);
  if (!item || amount == null) {
    return null;
  }
  return {
    title: `Split ${item.name}`,
    amount,
    currency: getCostCurrency(cost) ?? trip.budgetCurrency ?? "EUR",
    currentSplit: cost?.split ?? null
  };
}

export function findActiveGenerationJob(jobs: GenerationJob[]) {
  return jobs.find(isActiveGenerationJob) ?? null;
}

export function findProposalCurrentDay(
  itinerary: Itinerary | null,
  proposal: BudgetOptimizationProposal
) {
  const dayNumber = proposal.dayNumber ?? proposal.proposal.dayNumber;
  return (
    (itinerary?.days ?? []).find((day, index) => (day.day || index + 1) === dayNumber) ??
    null
  );
}

export function isActiveGenerationJob(job: GenerationJob) {
  return job.status === "queued" || job.status === "running";
}

export function targetFromGenerationJob(job: GenerationJob): RegeneratingTarget | null {
  if (
    (job.jobType === "item_regeneration" || job.jobType === "quality_improvement_item") &&
    job.dayNumber != null &&
    job.itemIndex != null
  ) {
    return { type: "item", dayNumber: job.dayNumber, itemIndex: job.itemIndex };
  }
  if (
    (job.jobType === "day_regeneration" || job.jobType === "quality_improvement_day") &&
    job.dayNumber != null
  ) {
    return { type: "day", dayNumber: job.dayNumber };
  }
  return null;
}

export function successMessageForGenerationJob(job: GenerationJob) {
  const warningSuffix =
    job.generationQuality &&
    (job.generationQuality.warningIssueCount > 0 || job.generationQuality.highIssueCount > 0)
      ? " Review validation warnings."
      : "";
  if (job.jobType === "full_generation") {
    return `Itinerary generated.${warningSuffix}`;
  }
  if (job.jobType === "budget_optimization_day") {
    return `Budget optimization proposal ready.${warningSuffix}`;
  }
  if (
    (job.jobType === "item_regeneration" || job.jobType === "quality_improvement_item") &&
    job.dayNumber != null &&
    job.itemIndex != null
  ) {
    return `Day ${job.dayNumber} item ${job.itemIndex + 1} regenerated.${warningSuffix}`;
  }
  if (job.dayNumber != null) {
    return `Day ${job.dayNumber} regenerated.${warningSuffix}`;
  }
  return `Itinerary updated.${warningSuffix}`;
}

export function failureMessageForGenerationJob(job: GenerationJob) {
  if (job.errorCode === "itinerary_conflict") {
    return "Generation stopped because the itinerary changed while the job was running. Reload latest version and try again.";
  }
  if (job.errorCode === "no_optimization_found") {
    return "Budget optimization could not find a useful cheaper alternative for that day.";
  }
  if (job.errorCode === "ai_generation_schema_invalid") {
    return "The AI returned an invalid itinerary shape and it could not be saved.";
  }
  if (job.errorCode === "ai_generation_repair_failed") {
    return "The itinerary had validation issues that could not be repaired automatically.";
  }
  if (job.errorCode === "ai_generation_blocked_by_policy") {
    return "Generation was blocked by workspace policy rules.";
  }
  if (job.errorCode === "ai_generation_route_conflict") {
    return "Generation was blocked because route stops or transfers did not line up.";
  }
  if (job.errorCode === "ai_generation_transport_conflict") {
    return "Generation was blocked because activities conflicted with selected transport.";
  }
  if (job.errorCode === "ai_generation_budget_conflict") {
    return "Generation was blocked because the itinerary could not satisfy the budget constraints.";
  }
  if (
    job.errorCode === "ai_generation_validation_failed" ||
    job.errorCode === "ai_output_invalid"
  ) {
    return "The generated itinerary failed reliability validation.";
  }
  if (job.jobType === "budget_optimization_day") {
    return job.errorMessage ?? "Budget optimization failed. The itinerary was not changed.";
  }
  return job.errorMessage ?? "Generation failed. The itinerary was not changed.";
}

export function availabilityResultKey(dayNumber: number, itemIndex: number) {
  return `${dayNumber}:${itemIndex}`;
}

export function availabilityCostCategory(item: {
  type?: string | null;
  place?: { category?: string | null } | null;
}): EstimatedCost["category"] {
  const text = `${item.type ?? ""} ${item.place?.category ?? ""}`.toLowerCase();
  if (text.includes("tour") || text.includes("activity") || text.includes("experience")) {
    return "activity";
  }
  return "ticket";
}

// Mirrors the AvailabilityCard higher-than-estimate threshold.
export function isSignificantPriceChange(
  currentCost: EstimatedCost | null | undefined,
  price: { amount: number; currency: string } | null | undefined
): boolean {
  if (!price) {
    return false;
  }
  const currentAmount = getCostAmount(currentCost);
  const currentCurrency = getCostCurrency(currentCost);
  if (currentAmount == null || !currentCurrency) {
    return false;
  }
  if (currentCurrency.toUpperCase() !== price.currency.toUpperCase()) {
    return false;
  }
  const diff = Math.abs(price.amount - currentAmount);
  return diff >= 10 || (currentAmount > 0 && diff / currentAmount >= 0.2);
}

export function formatAccessSource(source: string) {
  if (source === "owner") {
    return "Owner";
  }
  if (source === "workspace") {
    return "Workspace";
  }
  if (source === "collaborator") {
    return "Trip collaborator";
  }
  if (source === "public") {
    return "Public share";
  }
  return source;
}
