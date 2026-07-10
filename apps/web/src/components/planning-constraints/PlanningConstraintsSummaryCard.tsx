"use client";

import { useTranslations } from "next-intl";
import { Card } from "@/shared/ui/card";
import type { PlanningConstraintSummary } from "@/types/planning-constraints";

type Props = {
  summary: PlanningConstraintSummary;
};

export function PlanningConstraintsSummaryCard({ summary }: Props) {
  const t = useTranslations("planningConstraints");
  const rows = [
    [t("language"), summary.language],
    [t("budget"), summary.budget],
    [t("pace"), summary.pace],
    [t("transport"), summary.transport],
    [t("styles"), summary.tripStyles.length ? summary.tripStyles.join(", ") : t("notSet")],
    [t("workspaceRules"), String(summary.workspacePolicyRules)]
  ];

  return (
    <Card className="p-4">
      <div className="grid gap-3 sm:grid-cols-2">
        {rows.map(([label, value]) => (
          <div key={label}>
            <dt className="text-xs font-medium uppercase text-slate-500">{label}</dt>
            <dd className="mt-1 text-sm font-medium text-slate-900">{value}</dd>
          </div>
        ))}
      </div>
      <div className="mt-4 flex gap-2 text-sm">
        <span className="rounded-md bg-amber-50 px-2 py-1 text-amber-800">
          {t("warningCount", { count: summary.warningCount })}
        </span>
        <span className="rounded-md bg-red-50 px-2 py-1 text-red-800">
          {t("blockerCount", { count: summary.blockerCount })}
        </span>
      </div>
    </Card>
  );
}
