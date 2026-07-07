import { Button } from "@/shared/ui/button";
import { formatDate } from "@/lib/utils";
import type { GenerationJob } from "@/entities/generation-job/model";

type GenerationJobStatusCardProps = {
  job: GenerationJob;
  canCancel?: boolean;
  isCancelling?: boolean;
  onCancel?: () => void;
};

export function GenerationJobStatusCard({
  job,
  canCancel = false,
  isCancelling = false,
  onCancel
}: GenerationJobStatusCardProps) {
  const copy = getStatusCopy(job);

  return (
    <div className={copy.className}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-sm font-semibold">{copy.title}</p>
          <p className="mt-1 text-sm leading-6">{copy.message}</p>
          {job.status === "failed" && job.errorMessage ? (
            <p className="mt-2 text-sm leading-6">{job.errorMessage}</p>
          ) : null}
          <p className="mt-2 text-xs opacity-80">
            Started {job.startedAt ? formatDate(job.startedAt, dateTimeFormat) : "not yet"} -
            Queued {formatDate(job.createdAt, dateTimeFormat)}
          </p>
        </div>
        {canCancel && job.status === "queued" && onCancel ? (
          <Button
            disabled={isCancelling}
            onClick={onCancel}
            size="sm"
            type="button"
            variant="secondary"
          >
            {isCancelling ? "Cancelling..." : "Cancel"}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

const dateTimeFormat: Intl.DateTimeFormatOptions = {
  dateStyle: "medium",
  timeStyle: "short"
};

function getStatusCopy(job: GenerationJob) {
  const budgetOptimization = job.jobType === "budget_optimization_day";
  switch (job.status) {
    case "queued":
      return {
        title: budgetOptimization ? "Budget optimization queued..." : "Generation queued...",
        message: describeTarget(job),
        className: "mb-4 rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-900"
      };
    case "running":
      return {
        title: budgetOptimization ? "Optimizing budget..." : "Generating itinerary...",
        message: describeTarget(job),
        className: "mb-4 rounded-lg border border-blue-200 bg-blue-50 p-4 text-blue-900"
      };
    case "completed":
      return {
        title: budgetOptimization ? "Budget proposal ready" : "Generation completed",
        message: budgetOptimization
          ? "Review the proposal before applying it to the itinerary."
          : "The itinerary has been updated.",
        className:
          "mb-4 rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-emerald-800"
      };
    case "failed":
      return {
        title: budgetOptimization ? "Budget optimization failed" : "Generation failed",
        message: conflictMessage(job) ?? "The itinerary was not changed.",
        className: "mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-red-800"
      };
    case "cancelled":
      return {
        title: budgetOptimization ? "Budget optimization cancelled" : "Generation cancelled",
        message: "The queued job was cancelled before it started.",
        className: "mb-4 rounded-lg border border-slate-200 bg-slate-50 p-4 text-slate-700"
      };
  }
}

function describeTarget(job: GenerationJob) {
  if (job.jobType === "full_generation") {
    return "Building a full itinerary in the background.";
  }
  if (job.jobType === "budget_optimization_day" && job.dayNumber != null) {
    return `Creating a budget optimization proposal for Day ${job.dayNumber}.`;
  }
  if (job.dayNumber != null && job.itemIndex != null) {
    return `Updating Day ${job.dayNumber}, item ${job.itemIndex + 1}.`;
  }
  if (job.dayNumber != null) {
    return `Updating Day ${job.dayNumber}.`;
  }
  return "Updating the itinerary in the background.";
}

function conflictMessage(job: GenerationJob) {
  if (job.errorCode === "no_optimization_found") {
    return "No useful cheaper alternative was found for that day. Try a different target or instruction.";
  }
  if (job.errorCode !== "itinerary_conflict") {
    return null;
  }
  return "Generation stopped because the itinerary changed while the job was running. Reload latest version and try again.";
}
