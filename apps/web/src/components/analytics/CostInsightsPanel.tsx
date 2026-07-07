import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { cn } from "@/lib/utils";
import type { CostInsight } from "@/entities/cost-analytics/model";

type CostInsightsPanelProps = {
  insights: CostInsight[];
  canEdit?: boolean;
  onAction?: (insight: CostInsight) => void;
};

export function CostInsightsPanel({
  insights,
  canEdit = false,
  onAction
}: CostInsightsPanelProps) {
  return (
    <Card>
      <h2 className="text-lg font-semibold text-slate-950">Insights</h2>
      {insights.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500">No cost insights yet.</p>
      ) : (
        <div className="mt-5 space-y-3">
          {insights.map((insight, index) => {
            const editAction =
              insight.action?.type === "optimize_budget" ||
              insight.action?.type === "optimize_trip_day" ||
              insight.action?.type === "update_price" ||
              insight.action?.type === "check_availability" ||
              insight.action?.type === "check_missing_prices";
            const showAction = Boolean(insight.action) && (!editAction || canEdit);
            return (
              <div
                className={cn(
                  "rounded-md border p-4",
                  insight.severity === "critical" && "border-red-200 bg-red-50",
                  insight.severity === "warning" && "border-amber-200 bg-amber-50",
                  insight.severity === "info" && "border-slate-200 bg-slate-50"
                )}
                key={`${insight.type}-${index}`}
              >
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <p className="font-medium text-slate-950">{insight.title}</p>
                    <p className="mt-1 text-sm leading-6 text-slate-700">{insight.message}</p>
                  </div>
                  {showAction && onAction ? (
                    <Button onClick={() => onAction(insight)} size="sm" type="button" variant="secondary">
                      {actionLabel(insight.action?.type)}
                    </Button>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
}

function actionLabel(type: string | undefined) {
  switch (type) {
    case "optimize_budget":
    case "optimize_trip_day":
      return "Optimize";
    case "check_availability":
    case "check_missing_prices":
      return "Check";
    case "update_price":
      return "Update";
    case "open_item":
      return "Open item";
    case "open_trip":
      return "Open trip";
    case "open_workspace_analytics":
      return "Open analytics";
    case "export_report":
    case "export_budget_report":
      return "Export";
    default:
      return "Open";
  }
}
