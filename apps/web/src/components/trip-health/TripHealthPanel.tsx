"use client";

import { ReadinessChecklist } from "./ReadinessChecklist";
import { TopFixesCard } from "./TopFixesCard";
import { TripHealthCategoryGrid } from "./TripHealthCategoryGrid";
import { TripHealthIssueList } from "./TripHealthIssueList";
import { TripHealthScoreCard } from "./TripHealthScoreCard";
import type { TripHealth } from "@/types/trip-health";

type TripHealthPanelProps = {
  health?: TripHealth | null;
  loading?: boolean;
  error?: Error | null;
};

export function TripHealthPanel({ health, loading = false, error = null }: TripHealthPanelProps) {
  if (loading && !health) {
    return (
      <section id="health" className="scroll-mt-24 rounded-[18px] border border-sand-300 bg-white p-6">
        <p className="text-[14px] text-cocoa-500">Evaluating trip health...</p>
      </section>
    );
  }
  if (error && !health) {
    return (
      <section id="health" className="scroll-mt-24 rounded-[18px] border border-[#E5C3B6] bg-[#FBF0EB] p-6">
        <h2 className="font-newsreader text-[22px] font-semibold text-[#B3402E]">
          Trip health unavailable
        </h2>
        <p className="mt-2 text-[14px] leading-[1.6] text-[#9A4A3A]">
          {error.message || "Could not evaluate trip health."}
        </p>
      </section>
    );
  }
  if (!health) {
    return null;
  }

  return (
    <section id="health" className="scroll-mt-24">
      <div className="flex flex-col gap-4">
        <TripHealthScoreCard health={health} />
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_340px]">
          <TopFixesCard fixes={health.topFixes} />
          <ReadinessChecklist issues={health.issues} />
        </div>
        <TripHealthCategoryGrid categories={health.categories} />
        <TripHealthIssueList issues={health.issues} />
      </div>
    </section>
  );
}
