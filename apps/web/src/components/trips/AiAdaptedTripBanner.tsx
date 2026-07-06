"use client";

import { useSearchParams } from "next/navigation";

/** Shown immediately after a trip is created by AI template adaptation. It is
 * intentionally query-param driven (the dialog navigates with
 * `?adaptedFromTemplate=<id>`) so it appears right after creation without
 * requiring the client to re-derive provenance from version/activity metadata.
 */
export function AiAdaptedTripBanner({ className }: { className?: string }) {
  const searchParams = useSearchParams();
  if (!searchParams?.get("adaptedFromTemplate")) {
    return null;
  }
  const fallbackUsed = searchParams.get("fallbackUsed") === "true";
  return (
    <div
      className={`rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-900 ${className ?? ""}`}
      data-testid="ai-adapted-trip-banner"
    >
      <p className="font-medium">Created by AI adapting a template.</p>
      <p className="mt-1 text-primary-800">
        Please review costs, availability, and timing before relying on this plan. Costs are
        estimates and availability is unchecked.
      </p>
      {fallbackUsed ? (
        <p className="mt-2 rounded-md border border-amber-200 bg-amber-50 p-2 text-amber-800">
          AI adaptation failed, so this trip was created as a deterministic template copy.
        </p>
      ) : null}
    </div>
  );
}
