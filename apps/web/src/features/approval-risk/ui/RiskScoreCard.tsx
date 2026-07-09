"use client";

import { RiskBadge } from "./RiskBadge";
import { RiskSuggestedActions } from "./RiskSuggestedActions";
import type {
  ApprovalRiskResponse,
  ApprovalRiskSuggestedAction
} from "@/entities/approval-risk/model";

export function RiskScoreCard({
  risk,
  onAction
}: {
  risk: ApprovalRiskResponse;
  onAction?: (action: ApprovalRiskSuggestedAction) => boolean;
}) {
  if (risk.status === "not_applicable") {
    return (
      <div className="rounded-md border border-slate-200 bg-slate-50 p-4">
        <div className="flex items-center justify-between gap-3">
          <h4 className="text-sm font-semibold text-slate-900">Approval risk</h4>
          <RiskBadge status="not_applicable" />
        </div>
        <p className="mt-2 text-sm text-slate-500">Risk scoring applies to workspace trips.</p>
      </div>
    );
  }

  const title =
    risk.score != null ? `${riskLabel(risk.status)} · ${risk.score}/${risk.maxScore}` : "Risk unavailable";

  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-wide text-slate-400">Approval risk</p>
          <h4 className="mt-1 text-base font-semibold capitalize text-slate-950">{title}</h4>
        </div>
        <RiskBadge status={risk.status} score={risk.score} />
      </div>

      {risk.topReasons.length > 0 ? (
        <ul className="mt-3 list-disc space-y-1 pl-5 text-sm text-slate-700">
          {risk.topReasons.slice(0, 3).map((reason) => (
            <li key={reason}>{reason}</li>
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-sm text-slate-600">No material approval risks were found.</p>
      )}

      {risk.suggestedActions.length > 0 ? (
        <div className="mt-3">
          <RiskSuggestedActions actions={risk.suggestedActions.slice(0, 4)} onAction={onAction} />
        </div>
      ) : null}

      {risk.warnings.length > 0 ? (
        <p className="mt-3 text-xs text-slate-500">{risk.warnings[0]}</p>
      ) : null}
    </div>
  );
}

function riskLabel(status: ApprovalRiskResponse["status"]) {
  switch (status) {
    case "low":
      return "Low risk";
    case "medium":
      return "Medium risk";
    case "high":
      return "High risk";
    case "critical":
      return "Critical risk";
    default:
      return "Risk unavailable";
  }
}
