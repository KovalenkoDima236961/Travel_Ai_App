import { HealthCategoryIcon } from "./HealthCategoryIcon";
import { HealthSeverityBadge } from "./HealthSeverityBadge";
import { categoryLabel } from "./health-ui";
import type { TripHealthIssue } from "@/types/trip-health";

export function TripHealthIssueCard({ issue }: { issue: TripHealthIssue }) {
  return (
    <article
      className="scroll-mt-28 rounded-[14px] border border-sand-200 bg-white p-4 outline-none transition-shadow"
      id={`trip-health-issue-${issue.id}`}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 gap-3">
          <HealthCategoryIcon category={issue.category} />
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <HealthSeverityBadge severity={issue.severity} />
              <span className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
                {categoryLabel[issue.category]}
              </span>
            </div>
            <h3 className="mt-2 text-[15px] font-semibold text-cocoa-900">{issue.title}</h3>
            <p className="mt-1 text-[14px] leading-[1.6] text-cocoa-500">
              {issue.description}
            </p>
            {issue.impact ? (
              <p className="mt-2 text-[13px] leading-[1.5] text-cocoa-400">
                Impact: {issue.impact}
              </p>
            ) : null}
            {issue.recommendation ? (
              <p className="mt-1 text-[13px] leading-[1.5] text-cocoa-400">
                Recommendation: {issue.recommendation}
              </p>
            ) : null}
          </div>
        </div>
        {issue.action ? (
          <a
            href={issue.action.href}
            className="inline-flex h-9 shrink-0 items-center justify-center rounded-full border border-sand-400 bg-sand-50 px-4 text-[13px] font-semibold text-cocoa-700 transition hover:border-sand-600 hover:bg-white hover:text-cocoa-900"
          >
            {issue.action.label}
          </a>
        ) : null}
      </div>
    </article>
  );
}
