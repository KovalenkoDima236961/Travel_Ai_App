"use client";

import { useTranslations } from "next-intl";
import type { PlanningConstraintIssue } from "@/types/planning-constraints";

type Props = {
  issues: PlanningConstraintIssue[];
};

const severityClass: Record<string, string> = {
  blocking: "border-red-200 bg-red-50 text-red-900",
  warning: "border-amber-200 bg-amber-50 text-amber-900",
  info: "border-sky-200 bg-sky-50 text-sky-900"
};

export function PlanningConstraintIssuesList({ issues }: Props) {
  const t = useTranslations("planningConstraints");

  if (issues.length === 0) {
    return <p className="text-sm text-slate-500">{t("noIssues")}</p>;
  }

  return (
    <div className="space-y-3">
      {issues.map((issue) => (
        <article
          key={`${issue.type}-${issue.message}`}
          className={`rounded-md border p-3 ${severityClass[issue.severity] ?? severityClass.info}`}
        >
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-sm bg-white/70 px-2 py-0.5 text-xs font-semibold">
              {t(`severity.${issue.severity}`)}
            </span>
            <span className="text-xs uppercase tracking-wide opacity-70">{issue.source}</span>
          </div>
          <p className="mt-2 text-sm font-medium">{issue.message}</p>
          {issue.suggestedActions.length > 0 ? (
            <ul className="mt-2 space-y-1 text-sm">
              {issue.suggestedActions.map((action) => (
                <li key={`${issue.type}-${action.type}`}>{action.label}</li>
              ))}
            </ul>
          ) : null}
        </article>
      ))}
    </div>
  );
}
