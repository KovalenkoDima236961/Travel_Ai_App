"use client";

import { useState } from "react";
import { Button } from "@/shared/ui/button";
import type { GenerationJob } from "@/entities/generation-job/model";

type GenerationFailureRecoveryCardProps = {
  job: GenerationJob;
  isRetrying?: boolean;
  onRetry?: () => void;
  onSimplerRequest?: () => void;
  onReloadLatest?: () => void;
  onKeepCurrent?: () => void;
  onEditDetails?: () => void;
};

export function GenerationFailureRecoveryCard({
  job,
  isRetrying = false,
  onRetry,
  onSimplerRequest,
  onReloadLatest,
  onKeepCurrent,
  onEditDetails
}: GenerationFailureRecoveryCardProps) {
  const [copied, setCopied] = useState(false);
  const failure = failureForJob(job);
  const isConflict = failure.category === "itinerary_conflict";
  const safeMessage = job.errorMessageSafe || failure.message;

  async function copyCode() {
    if (!job.errorCode || !navigator.clipboard) {
      return;
    }
    await navigator.clipboard.writeText(job.errorCode);
    setCopied(true);
  }

  return (
    <section className="rounded-[18px] border border-[#E5C3B6] bg-[#FBF0EB] p-5 text-[#843827]" role="alert">
      <h2 className="font-newsreader text-[21px] font-semibold text-cocoa-900">
        {isConflict ? "This trip changed while generation was running" : "We couldn’t finish your itinerary"}
      </h2>
      <p className="mt-2 text-[14px] leading-6 text-cocoa-700">{safeMessage}</p>
      {job.errorCode ? (
        <div className="mt-3 flex flex-wrap items-center gap-2 text-[12.5px] text-cocoa-500">
          <span className="rounded bg-white/80 px-2 py-1 font-mono">{job.errorCode}</span>
          <button className="font-semibold text-clay-deep hover:underline" onClick={copyCode} type="button">
            {copied ? "Copied" : "Copy error code"}
          </button>
        </div>
      ) : null}

      <p className="mt-4 text-[13px] leading-5 text-cocoa-600">{failure.nextStep}</p>
      <div className="mt-4 flex flex-wrap gap-2.5">
        {isConflict ? (
          <>
            {onReloadLatest ? <Button onClick={onReloadLatest} type="button" variant="secondary">Reload latest</Button> : null}
            {onRetry ? <Button disabled={isRetrying} onClick={onRetry} type="button">Start a new generation</Button> : null}
            {onKeepCurrent ? <Button onClick={onKeepCurrent} type="button" variant="secondary">Keep current itinerary</Button> : null}
          </>
        ) : (
          <>
            {onRetry ? <Button disabled={isRetrying} onClick={onRetry} type="button">{isRetrying ? "Trying again…" : "Try again"}</Button> : null}
            {onSimplerRequest ? <Button disabled={isRetrying} onClick={onSimplerRequest} type="button" variant="secondary">Use simpler request</Button> : null}
            {onEditDetails ? <Button onClick={onEditDetails} type="button" variant="secondary">Edit trip details</Button> : null}
          </>
        )}
      </div>
      {!isConflict ? (
        <p className="mt-4 text-[12.5px] leading-5 text-cocoa-500">
          You can also keep this trip without an itinerary and come back when you&apos;re ready.
        </p>
      ) : null}
    </section>
  );
}

type FailureCategory =
  | "ai_timeout"
  | "ai_invalid_json"
  | "ai_validation_failed"
  | "ai_repair_failed"
  | "provider_unavailable"
  | "provider_rate_limited"
  | "itinerary_conflict"
  | "missing_required_trip_data"
  | "permission_denied"
  | "unknown";

function failureForJob(job: GenerationJob): { category: FailureCategory; message: string; nextStep: string } {
  const code = job.errorCode ?? "unknown";
  if (code === "itinerary_conflict") {
    return { category: "itinerary_conflict", message: "The latest trip may have a newer itinerary than this job used.", nextStep: "Reload it before starting a fresh generation so nothing is overwritten." };
  }
  if (["ai_timeout", "ai_generation_timeout", "generation_timeout"].includes(code)) {
    return { category: "ai_timeout", message: "The planning model took too long to respond.", nextStep: "Try again, or use a simpler request with fewer optional constraints." };
  }
  if (["ai_invalid_json", "ai_generation_schema_invalid", "ai_output_invalid"].includes(code)) {
    return { category: "ai_invalid_json", message: "The AI response could not be converted into a valid itinerary.", nextStep: "Try again or ask for a simpler, realistic itinerary." };
  }
  if (["ai_validation_failed", "ai_generation_validation_failed"].includes(code)) {
    return { category: "ai_validation_failed", message: "The generated itinerary did not pass the app’s consistency checks.", nextStep: "Review the trip details, then try a simpler request." };
  }
  if (["ai_repair_failed", "ai_generation_repair_failed"].includes(code)) {
    return { category: "ai_repair_failed", message: "The itinerary needed fixes that could not be applied automatically.", nextStep: "Try again with fewer constraints or adjust the trip details." };
  }
  if (["provider_unavailable", "provider_limits_unavailable", "ai_generation"].includes(code)) {
    return { category: "provider_unavailable", message: "A planning or enrichment provider is temporarily unavailable.", nextStep: "Try again shortly. Your saved trip details are safe." };
  }
  if (["provider_rate_limited", "provider_quota_exceeded"].includes(code)) {
    return { category: "provider_rate_limited", message: "A planning provider has reached a temporary limit.", nextStep: "Try again later, or keep the trip and continue planning manually." };
  }
  if (["missing_required_trip_data", "invalid_input"].includes(code)) {
    return { category: "missing_required_trip_data", message: "Some trip details needed for generation are missing.", nextStep: "Edit the destination, dates, travelers, or route and try again." };
  }
  if (["permission_denied", "forbidden"].includes(code)) {
    return { category: "permission_denied", message: "You don’t have permission to generate this itinerary.", nextStep: "Ask a trip owner or editor to check your access." };
  }
  return { category: "unknown", message: "Something interrupted itinerary generation.", nextStep: "Try again. If it keeps happening, share the error code with support." };
}
