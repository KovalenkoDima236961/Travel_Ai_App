"use client";

import { useTranslations } from "next-intl";
import { ErrorState, SectionLoadingState } from "@/components/ui";
import { CollaboratorReadinessRow } from "./CollaboratorReadinessRow";
import { ReadinessCategoryBadge } from "./ReadinessCategoryBadge";
import { ReadinessScoreBadge } from "./ReadinessScoreBadge";
import { ReadinessTopActions } from "./ReadinessTopActions";
import { categoryLabel, scoreBarClass } from "./readiness-ui";
import type { GroupReadiness } from "@/types/group-readiness";

type GroupReadinessPanelProps = {
  tripId: string;
  readiness?: GroupReadiness | null;
  loading?: boolean;
  error?: Error | null;
  canNudge?: boolean;
  onRetry?: () => void;
  retrying?: boolean;
};

export function GroupReadinessPanel({
  tripId,
  readiness,
  loading = false,
  error = null,
  canNudge = false,
  onRetry,
  retrying = false
}: GroupReadinessPanelProps) {
  const loadingT = useTranslations("loading");
  const errorsT = useTranslations("errors");

  if (loading && !readiness) {
    return (
      <section id="group-readiness" className="scroll-mt-24">
        <SectionLoadingState cards={2} label={loadingT("groupReadiness")} />
      </section>
    );
  }
  if (error && !readiness) {
    return (
      <section id="group-readiness" className="scroll-mt-24">
        <ErrorState
          className="rounded-[18px]"
          description={errorsT("groupReadinessDescription")}
          developmentDetails={error.message}
          retryAction={onRetry ? { onRetry, pending: retrying } : undefined}
          title={errorsT("groupReadinessTitle")}
        />
      </section>
    );
  }
  if (!readiness) {
    return null;
  }

  const readyCount = readiness.members.filter((member) => member.level === "ready").length;
  const currentUser = readiness.members.find((member) => member.isCurrentUser);
  const currentUserItems = currentUser?.items ?? [];

  return (
    <section id="group-readiness" className="scroll-mt-24 space-y-4">
      <div className="rounded-[18px] border border-sand-300 bg-white p-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Group readiness
            </p>
            <h2 className="mt-1 font-newsreader text-[28px] font-semibold text-cocoa-900">
              Who is ready
            </h2>
            <p className="mt-2 max-w-[720px] text-[14px] leading-[1.65] text-cocoa-600">
              {readiness.summary}
            </p>
          </div>
          <ReadinessScoreBadge level={readiness.level} score={readiness.score} />
        </div>

        <div className="mt-5 h-2 overflow-hidden rounded-full bg-sand-200">
          <div
            className={`h-full rounded-full ${scoreBarClass(readiness.score)}`}
            style={{ width: `${Math.max(0, Math.min(readiness.score, 100))}%` }}
          />
        </div>

        <dl className="mt-5 grid gap-3 sm:grid-cols-3">
          <div className="rounded-[14px] bg-sand-50 p-4">
            <dt className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Ready
            </dt>
            <dd className="mt-1 text-[20px] font-semibold text-cocoa-900">
              {readyCount}/{readiness.members.length}
            </dd>
          </div>
          <div className="rounded-[14px] bg-sand-50 p-4">
            <dt className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Open items
            </dt>
            <dd className="mt-1 text-[20px] font-semibold text-cocoa-900">
              {readiness.members.reduce((count, member) => count + member.items.length, 0)}
            </dd>
          </div>
          <div className="rounded-[14px] bg-sand-50 p-4">
            <dt className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Categories
            </dt>
            <dd className="mt-1 text-[20px] font-semibold text-cocoa-900">
              {readiness.categorySummary.length}
            </dd>
          </div>
        </dl>
      </div>

      {currentUserItems.length > 0 ? (
        <div className="rounded-[18px] border border-[#EAD9B8] bg-[#FDF7E8] p-5">
          <h3 className="text-[16px] font-semibold text-cocoa-900">Your action items</h3>
          <div className="mt-3 flex flex-wrap gap-2">
            {currentUserItems.map((item) => (
              <a
                key={`${item.category}:${item.id}`}
                href={item.action?.href ?? "#group-readiness"}
                className="inline-flex rounded-full border border-[#EAD9B8] bg-white px-3 py-1.5 text-[13px] font-semibold text-[#96682A] transition hover:border-[#D4B77E]"
              >
                {item.action?.label ?? item.title}
              </a>
            ))}
          </div>
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_340px]">
        <div className="space-y-3">
          {readiness.members.map((member) => (
            <CollaboratorReadinessRow
              canNudge={canNudge}
              key={member.userId}
              member={member}
              tripId={tripId}
            />
          ))}
        </div>
        <div className="space-y-4">
          <ReadinessTopActions actions={readiness.topActions} />
          <div className="rounded-[16px] border border-sand-300 bg-white p-4">
            <h3 className="text-[14px] font-semibold text-cocoa-900">Category summary</h3>
            <div className="mt-3 space-y-3">
              {readiness.categorySummary.map((summary) => (
                <div key={summary.category} className="rounded-[12px] bg-sand-50 p-3">
                  <div className="flex items-center justify-between gap-2">
                    <ReadinessCategoryBadge category={summary.category} />
                    <span className="text-[12px] font-semibold text-cocoa-700">
                      {summary.readyCount}/{summary.totalCount}
                    </span>
                  </div>
                  <p className="mt-2 text-[12.5px] text-cocoa-500">
                    {summary.openIssueCount > 0
                      ? `${summary.openIssueCount} open ${categoryLabel(summary.category).toLowerCase()} item(s).`
                      : "No open items."}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
