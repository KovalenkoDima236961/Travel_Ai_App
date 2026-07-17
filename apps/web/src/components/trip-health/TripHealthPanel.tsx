"use client";

import { useTranslations } from "next-intl";
import { ErrorState, SectionLoadingState } from "@/components/ui";
import { ContextualTip } from "@/components/onboarding/ContextualTip";
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
  onRetry?: () => void;
  retrying?: boolean;
};

export function TripHealthPanel({
  health,
  loading = false,
  error = null,
  onRetry,
  retrying = false
}: TripHealthPanelProps) {
  const loadingT = useTranslations("loading");
  const errorsT = useTranslations("errors");

  if (loading && !health) {
    return (
      <section id="health" className="scroll-mt-24">
        <SectionLoadingState cards={2} label={loadingT("tripHealth")} />
      </section>
    );
  }
  if (error && !health) {
    return (
      <section id="health" className="scroll-mt-24">
        <ErrorState
          className="rounded-[18px]"
          description={errorsT("tripHealthDescription")}
          developmentDetails={error.message}
          retryAction={onRetry ? { onRetry, pending: retrying } : undefined}
          title={errorsT("tripHealthTitle")}
        />
      </section>
    );
  }
  if (!health) {
    return null;
  }

  return (
    <section id="health" className="scroll-mt-24">
      <div className="flex flex-col gap-4">
        <ContextualTip tipId="trip_health" />
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
