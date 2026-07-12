"use client";

import type { TripAvailabilitySummary } from "@/types/trip-availability";

export function MissingAvailabilityNotice({
  summary
}: {
  summary?: TripAvailabilitySummary;
}) {
  if (!summary || summary.missingCount === 0) {
    return null;
  }

  const names = summary.missingUsers
    .slice(0, 4)
    .map((user) => user.displayName)
    .join(", ");

  return (
    <div className="rounded-[14px] bg-sand-50 p-4 text-[13px] text-cocoa-600">
      Missing availability from {summary.missingCount} traveler
      {summary.missingCount === 1 ? "" : "s"}
      {names ? `: ${names}` : ""}.
    </div>
  );
}
