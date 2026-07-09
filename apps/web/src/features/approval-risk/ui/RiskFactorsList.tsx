"use client";

import { RiskSuggestedActions } from "./RiskSuggestedActions";
import { cn } from "@/shared/lib/cn";
import type {
  ApprovalRiskFactor,
  ApprovalRiskFactorSeverity,
  ApprovalRiskSuggestedAction
} from "@/entities/approval-risk/model";

const SEVERITIES: { value: ApprovalRiskFactorSeverity; label: string }[] = [
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" }
];

const SEVERITY_CLASS: Record<ApprovalRiskFactorSeverity, string> = {
  critical: "border-red-200 bg-red-50 text-red-800",
  high: "border-orange-200 bg-orange-50 text-orange-800",
  medium: "border-amber-200 bg-amber-50 text-amber-800",
  low: "border-slate-200 bg-white text-slate-700"
};

export function RiskFactorsList({
  factors,
  defaultOpen = false,
  onAction
}: {
  factors: ApprovalRiskFactor[];
  defaultOpen?: boolean;
  onAction?: (action: ApprovalRiskSuggestedAction) => boolean;
}) {
  if (factors.length === 0) {
    return null;
  }

  return (
    <details className="rounded-md border border-slate-200 bg-white p-4" open={defaultOpen}>
      <summary className="cursor-pointer text-sm font-semibold text-slate-900">
        Risk factors ({factors.length})
      </summary>
      <div className="mt-4 space-y-4">
        {SEVERITIES.map(({ value, label }) => {
          const group = factors.filter((factor) => factor.severity === value);
          if (group.length === 0) {
            return null;
          }
          return (
            <section key={value} className="space-y-2">
              <h5 className="text-xs font-semibold uppercase tracking-wide text-slate-400">
                {label}
              </h5>
              <ul className="space-y-2">
                {group.map((factor) => (
                  <li
                    key={`${factor.type}-${factor.points}-${factor.message}`}
                    className="rounded-md border border-slate-200 p-3"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="min-w-0">
                        <p className="text-sm font-semibold text-slate-950">{factor.title}</p>
                        <p className="mt-1 text-sm text-slate-600">{factor.message}</p>
                      </div>
                      <span
                        className={cn(
                          "inline-flex h-8 items-center rounded-md border px-2.5 text-xs font-semibold",
                          SEVERITY_CLASS[factor.severity]
                        )}
                      >
                        +{factor.points}
                      </span>
                    </div>
                    <p className="mt-2 text-xs text-slate-500">
                      Source: {factor.source.replaceAll("_", " ")}
                      {factor.affected?.affectedCount
                        ? ` · ${factor.affected.affectedCount} affected`
                        : ""}
                    </p>
                    {factor.affected?.affectedItems?.length ? (
                      <p className="mt-1 text-xs text-slate-500">
                        {factor.affected.affectedItems
                          .slice(0, 4)
                          .map((item) =>
                            item.dayNumber != null
                              ? `Day ${item.dayNumber}${item.itemIndex != null ? `, item ${item.itemIndex + 1}` : ""}${item.name ? `: ${item.name}` : ""}`
                              : item.name || item.category || "Affected item"
                          )
                          .join("; ")}
                      </p>
                    ) : null}
                    {factor.suggestedActions?.length ? (
                      <div className="mt-3">
                        <RiskSuggestedActions
                          actions={factor.suggestedActions}
                          onAction={onAction}
                        />
                      </div>
                    ) : null}
                  </li>
                ))}
              </ul>
            </section>
          );
        })}
      </div>
    </details>
  );
}

