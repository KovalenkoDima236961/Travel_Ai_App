import type { ApprovalRiskSuggestedAction } from "@/entities/approval-risk/model";

type RouterLike = {
  push: (href: string) => void;
};

export type ApprovalRiskActionContext = {
  tripId: string;
  workspaceId?: string | null;
  router?: RouterLike;
  openBudgetOptimization?: (dayNumber?: number | null) => void;
  openRegenerateDay?: (dayNumber?: number | null) => void;
  setActiveTab?: (tab: string) => void;
};

export function handleRiskAction(
  action: ApprovalRiskSuggestedAction,
  context: ApprovalRiskActionContext
): boolean {
  const target = action.target ?? {};
  const tripId = target.tripId ?? context.tripId;
  const workspaceId = target.workspaceId ?? context.workspaceId ?? undefined;

  switch (action.type) {
    case "open_budget_optimization":
      if (context.openBudgetOptimization) {
        context.openBudgetOptimization(target.dayNumber);
        return true;
      }
      return push(context.router, `/trips/${tripId}?budgetOptimizeDay=${target.dayNumber ?? ""}`);
    case "open_trip_analytics":
      return push(context.router, `/trips/${tripId}#budget`);
    case "open_workspace_budget":
      return workspaceId ? push(context.router, `/workspaces/${workspaceId}/budgets`) : false;
    case "open_cost_splitting":
      return push(context.router, `/trips/${tripId}#cost-splitting`);
    case "check_availability":
    case "open_item":
    case "add_missing_costs":
      return push(context.router, itemHref(tripId, target.dayNumber, target.itemIndex));
    case "open_accommodation":
      return push(context.router, `/trips/${tripId}#accommodation`);
    case "fix_policy_violation":
      return push(context.router, `/trips/${tripId}#workspace-policy`);
    case "regenerate_day":
      if (context.openRegenerateDay) {
        context.openRegenerateDay(target.dayNumber);
        return true;
      }
      return push(context.router, itemHref(tripId, target.dayNumber, null));
    case "optimize_route":
      return push(context.router, `/trips/${tripId}#route`);
    case "review_ai_adaptation":
      return push(context.router, `/trips/${tripId}#itinerary`);
    case "open_approval_checklist":
      return push(context.router, `/trips/${tripId}#approval`);
    default:
      return false;
  }
}

function itemHref(tripId: string, dayNumber?: number | null, itemIndex?: number | null) {
  const hash =
    dayNumber != null && itemIndex != null
      ? `#day-${dayNumber}-item-${itemIndex}`
      : dayNumber != null
        ? `#day-${dayNumber}`
        : "#itinerary";
  return `/trips/${tripId}${hash}`;
}

function push(router: RouterLike | undefined, href: string) {
  if (router) {
    router.push(href);
    return true;
  }
  if (typeof window !== "undefined") {
    window.location.href = href;
    return true;
  }
  return false;
}

