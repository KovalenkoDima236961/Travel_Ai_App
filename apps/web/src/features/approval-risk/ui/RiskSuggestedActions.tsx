"use client";

import { Button } from "@/shared/ui/button";
import type { ApprovalRiskSuggestedAction } from "@/entities/approval-risk/model";

export function RiskSuggestedActions({
  actions,
  onAction
}: {
  actions: ApprovalRiskSuggestedAction[];
  onAction?: (action: ApprovalRiskSuggestedAction) => boolean;
}) {
  if (actions.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-wrap gap-2">
      {actions.map((action, index) => {
        const key = `${action.type}-${action.target?.dayNumber ?? "trip"}-${action.target?.itemIndex ?? "item"}-${index}`;
        const clickable = Boolean(onAction);
        return clickable ? (
          <Button
            key={key}
            onClick={() => onAction?.(action)}
            size="sm"
            type="button"
            variant={action.priority === "high" ? "secondary" : "ghost"}
          >
            {action.label}
          </Button>
        ) : (
          <span
            key={key}
            className="inline-flex min-h-9 items-center rounded-md border border-slate-200 bg-slate-50 px-3 text-sm text-slate-600"
          >
            {action.label}
          </span>
        );
      })}
    </div>
  );
}

