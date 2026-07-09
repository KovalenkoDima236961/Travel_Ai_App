import { cn } from "@/lib/utils";
import type { CostInsight } from "@/entities/cost-analytics/model";
import { SparklesIcon } from "./icons";

type InsightsPanelProps = {
  insights: CostInsight[];
  canEdit?: boolean;
  onAction?: (insight: CostInsight) => void;
};

/**
 * Slice-local restyle of the shared CostInsightsPanel as the mock's icon-tile
 * callout, generalised over the real severity levels and every insight the
 * trip analytics API returns (the mock shows a single one).
 */
export function InsightsPanel({ insights, canEdit = false, onAction }: InsightsPanelProps) {
  if (insights.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-4">
      {insights.map((insight, index) => {
        const editAction =
          insight.action?.type === "optimize_budget" ||
          insight.action?.type === "optimize_trip_day" ||
          insight.action?.type === "update_price" ||
          insight.action?.type === "check_availability" ||
          insight.action?.type === "check_missing_prices";
        const showAction = Boolean(insight.action) && (!editAction || canEdit);
        const tone = toneStyles(insight.severity);
        return (
          <div
            className={cn("rounded-[20px] border px-7 py-6", tone.container)}
            key={`${insight.type}-${index}`}
          >
            <div className="flex items-start gap-3.5">
              <span
                className={cn(
                  "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
                  tone.tile
                )}
              >
                <SparklesIcon className="h-5 w-5" />
              </span>
              <div className="min-w-0">
                <h3 className="text-[15.5px] font-semibold text-cocoa-900">{insight.title}</h3>
                <p className="mt-1.5 text-sm leading-[1.55] text-cocoa-500">{insight.message}</p>
                {showAction && onAction ? (
                  <button
                    className={cn(
                      "mt-3.5 inline-flex h-[38px] items-center gap-2 rounded-full border px-4 text-[13.5px] font-semibold transition",
                      tone.button
                    )}
                    onClick={() => onAction(insight)}
                    type="button"
                  >
                    {actionLabel(insight.action?.type)}
                  </button>
                ) : null}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function toneStyles(severity: CostInsight["severity"]) {
  if (severity === "critical") {
    return {
      container: "border-[#E5C3B6] bg-[#FBF0EB]",
      tile: "bg-clay-tint text-clay-deep",
      button: "border-[#E5C3B6] bg-white text-clay-deep hover:bg-clay-tint"
    };
  }
  if (severity === "warning") {
    return {
      container: "border-[#EFD9B8] bg-[#FFFDF7]",
      tile: "bg-[#FAEFDA] text-[#96682A]",
      button: "border-[#EAD6B2] bg-white text-[#96682A] hover:bg-[#FAEFDA]"
    };
  }
  return {
    container: "border-sand-300 bg-white",
    tile: "bg-clay-tint text-clay-deep",
    button: "border-[#E5C3B6] bg-[#FBF0EB] text-clay-deep hover:bg-clay-tint"
  };
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
