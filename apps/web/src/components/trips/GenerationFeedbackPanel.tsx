"use client";

import { FeedbackChips } from "@/components/personalization/FeedbackChips";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { Trip } from "@/entities/trip/model";
import type { FeedbackType } from "@/types/personalization";

const alphaFeedbackChips: Array<{ type: FeedbackType; label: string }> = [
  { type: "like", label: "Useful" },
  { type: "dislike", label: "Not useful" },
  { type: "too_packed", label: "Too busy" },
  { type: "too_expensive", label: "Too expensive" },
  { type: "too_much_walking", label: "Too much walking" },
  { type: "not_my_vibe", label: "Wrong style" },
  { type: "other", label: "Other" }
];

export function GenerationFeedbackPanel({
  job,
  trip
}: {
  job: GenerationJob;
  trip: Trip;
}) {
  const quality = job.generationQuality ?? job.resultPayload?.generationQuality ?? null;
  const qualityStatus = quality?.status ?? "unknown";
  const fallbackUsed =
    Boolean(job.resultPayload && "fallback" in job.resultPayload) ||
    qualityStatus.includes("fallback");

  return (
    <section className="rounded-lg border border-sand-300 bg-white p-4" aria-labelledby="alpha-generation-feedback-title">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 id="alpha-generation-feedback-title" className="text-[14px] font-semibold text-cocoa-900">
            Was this itinerary useful?
          </h2>
        </div>
        <FeedbackChips
          chips={alphaFeedbackChips}
          input={{
            tripId: trip.id,
            workspaceId: trip.workspaceId ?? null,
            entityType: "general",
            entityId: job.id,
            metadata: {
              source: "alpha_generation_feedback",
              category: qualityStatus,
              style: [job.jobType, fallbackUsed ? "fallback_used" : "primary_used"]
            }
          }}
        />
      </div>
    </section>
  );
}
