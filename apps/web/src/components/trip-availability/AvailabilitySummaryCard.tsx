"use client";

import type { TripAvailabilitySummary } from "@/types/trip-availability";

type AvailabilitySummaryCardProps = {
  summary?: TripAvailabilitySummary;
};

export function AvailabilitySummaryCard({ summary }: AvailabilitySummaryCardProps) {
  return (
    <div className="grid gap-2 sm:grid-cols-3">
      <Metric label="Submitted" value={summary?.submittedCount ?? 0} />
      <Metric label="Missing" value={summary?.missingCount ?? 0} />
      <Metric label="Travelers" value={summary?.totalCollaborators ?? 0} />
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-[14px] bg-sand-50 p-3">
      <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
        {label}
      </p>
      <p className="mt-1 font-newsreader text-[25px] font-semibold text-cocoa-900">{value}</p>
    </div>
  );
}
