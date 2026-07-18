"use client";

import { GenerationQualityBadges, GenerationWarningsPanel } from "@/components/generation-quality";
import { Button } from "@/shared/ui/button";
import type { GenerationJob } from "@/entities/generation-job/model";

type GenerationProgressCardProps = {
  job: GenerationJob;
  canCancel?: boolean;
  isCancelling?: boolean;
  onCancel?: () => void;
};

const PROGRESS_STEPS = [
  "Understanding your trip details",
  "Applying your preferences",
  "Building the itinerary",
  "Checking schedule quality",
  "Checking budget and route issues",
  "Adding place and cost context",
  "Saving your itinerary"
];

export function GenerationProgressCard({
  job,
  canCancel = false,
  isCancelling = false,
  onCancel
}: GenerationProgressCardProps) {
  const quality = job.generationQuality ?? job.resultPayload?.generationQuality ?? null;
  const stage = generationStageForJob(job);
  const isComplete = stage === "completed";

  return (
    <section
      aria-live="polite"
      className="rounded-[18px] border border-[#BFD8E8] bg-[#F2F8FC] p-5 text-[#244C66]"
    >
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="font-newsreader text-[21px] font-semibold text-cocoa-900">
              {isComplete ? "Your itinerary is ready" : "Creating your itinerary"}
            </h2>
            <GenerationQualityBadges quality={quality} source={itinerarySource(job)} />
          </div>
          <p className="mt-1.5 text-[14px] leading-6 text-cocoa-600">{stageMessage(stage)}</p>
          {!isComplete && stage !== "cancelled" ? (
            <p className="mt-2 text-[13px] leading-5 text-cocoa-500">
              This can take a little while with a local model. You can leave this page and come
              back; the job status will update.
            </p>
          ) : null}
        </div>
        {canCancel && job.status === "queued" && onCancel ? (
          <Button disabled={isCancelling} onClick={onCancel} size="sm" type="button" variant="secondary">
            {isCancelling ? "Cancelling…" : "Cancel"}
          </Button>
        ) : null}
      </div>

      <ol className="mt-5 grid gap-2 sm:grid-cols-2" aria-label="What we’re doing">
        {PROGRESS_STEPS.map((step, index) => {
          const state = progressState(index, stage);
          return (
            <li key={step} className="flex items-center gap-2.5 rounded-lg bg-white/70 px-3 py-2 text-[13px]">
              <span
                aria-hidden="true"
                className={
                  state === "done"
                    ? "flex h-5 w-5 items-center justify-center rounded-full bg-[#3E6B5A] text-[11px] text-white"
                    : state === "current"
                      ? "flex h-5 w-5 items-center justify-center rounded-full border-2 border-[#3E7397] bg-white text-[10px] text-[#3E7397]"
                      : "flex h-5 w-5 items-center justify-center rounded-full border border-sand-400 bg-sand-100 text-[10px] text-cocoa-400"
                }
              >
                {state === "done" ? "✓" : state === "current" ? "…" : ""}
              </span>
              <span className={state === "next" ? "text-cocoa-500" : "font-medium text-cocoa-800"}>{step}</span>
            </li>
          );
        })}
      </ol>

      {!isComplete && stage !== "cancelled" ? (
        <p className="mt-4 text-[12.5px] leading-5 text-cocoa-500">
          We&apos;ll show validation and review notes when it finishes. The list reflects the
          overall work, not a precise backend substep.
        </p>
      ) : null}
      <GenerationWarningsPanel compact quality={quality} />
    </section>
  );
}

export type GenerationProgressStage =
  | "not_started"
  | "queued"
  | "running"
  | "validating"
  | "repairing"
  | "enriching"
  | "saving"
  | "completed"
  | "failed"
  | "cancelled"
  | "conflict";

export function generationStageForJob(job: GenerationJob): GenerationProgressStage {
  if (job.status === "failed" && job.errorCode === "itinerary_conflict") {
    return "conflict";
  }
  if (job.status === "completed" || job.status === "failed" || job.status === "cancelled") {
    return job.status;
  }
  // The API intentionally exposes coarse job states. Do not present fabricated
  // validation/enrichment milestones until the worker reports them explicitly.
  return job.status;
}

function progressState(index: number, stage: GenerationProgressStage) {
  if (stage === "completed") {
    return "done" as const;
  }
  if (stage === "queued") {
    return index === 0 ? ("current" as const) : ("next" as const);
  }
  if (stage === "running") {
    return index === 2 ? ("current" as const) : ("next" as const);
  }
  return "next" as const;
}

function stageMessage(stage: GenerationProgressStage) {
  switch (stage) {
    case "queued":
      return "Preparing your trip details for the planning worker.";
    case "running":
      return "Working on your itinerary. We’ll run final checks before it is saved.";
    case "completed":
      return "Your itinerary has been saved. Review the most important notes before diving in.";
    case "cancelled":
      return "This queued generation was cancelled before it started.";
    default:
      return "Preparing your itinerary.";
  }
}

function itinerarySource(job: GenerationJob) {
  const source = job.resultPayload?.source;
  return typeof source === "string" ? source : undefined;
}
