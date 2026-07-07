import Link from "next/link";
import { Button, buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import {
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import type { ActiveWorkspaceBudget } from "@/entities/cost-analytics/model";

type WorkspaceBudgetOverviewProps = {
  activeBudget: ActiveWorkspaceBudget | null;
  canManage: boolean;
  workspaceId: string;
  workspaceLoaded: boolean;
  onUseBudgetPeriod: () => void;
};

export function WorkspaceBudgetOverview({
  activeBudget,
  canManage,
  workspaceId,
  workspaceLoaded,
  onUseBudgetPeriod
}: WorkspaceBudgetOverviewProps) {
  if (!activeBudget && !workspaceLoaded) {
    return null;
  }

  if (!activeBudget) {
    return (
      <Card className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-slate-600">
          {canManage ? "No primary workspace budget is set." : "No workspace budget set."}
        </p>
        {canManage ? (
          <Link className={buttonStyles({ size: "sm" })} href={`/workspaces/${workspaceId}/budgets`}>
            Create workspace budget
          </Link>
        ) : null}
      </Card>
    );
  }

  return (
    <Card className="mb-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-lg font-semibold text-slate-950">{activeBudget.name}</h2>
            <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-semibold text-primary-700">
              Primary budget
            </span>
          </div>
          <p className="mt-2 text-sm text-slate-600">
            {formatPlainMoney(activeBudget.amount, activeBudget.currency)} ·{" "}
            {activeBudget.periodStart || activeBudget.periodEnd
              ? `${activeBudget.periodStart ?? "open"} - ${activeBudget.periodEnd ?? "open"}`
              : "All workspace trips"}
          </p>
        </div>
        <div className="grid gap-3 text-sm sm:grid-cols-3 lg:min-w-[32rem]">
          <div className="rounded-md bg-slate-50 p-3">
            <p className="text-xs font-semibold uppercase text-slate-500">Estimated</p>
            <p className="mt-1 font-semibold text-slate-950">
              {formatAnalyticsMoney(activeBudget.estimatedTotal, activeBudget.currency)}
            </p>
          </div>
          <div className="rounded-md bg-slate-50 p-3">
            <p className="text-xs font-semibold uppercase text-slate-500">
              {activeBudget.overBudgetAmount > 0 ? "Over" : "Remaining"}
            </p>
            <p
              className={
                activeBudget.overBudgetAmount > 0
                  ? "mt-1 font-semibold text-red-700"
                  : "mt-1 font-semibold text-emerald-700"
              }
            >
              {formatAnalyticsMoney(
                activeBudget.overBudgetAmount > 0
                  ? activeBudget.overBudgetAmount
                  : activeBudget.remainingAmount,
                activeBudget.currency
              )}
            </p>
          </div>
          <div className="rounded-md bg-slate-50 p-3">
            <p className="text-xs font-semibold uppercase text-slate-500">Utilization</p>
            <p className="mt-1 font-semibold text-slate-950">
              {formatPercent(activeBudget.utilizationPercent)}
            </p>
          </div>
        </div>
      </div>
      <div className="mt-5 flex flex-wrap gap-2">
        <Link
          className={buttonStyles({ variant: "secondary", size: "sm" })}
          href={`/workspaces/${workspaceId}/budgets/${activeBudget.id}`}
        >
          View budget
        </Link>
        <Button onClick={onUseBudgetPeriod} size="sm" type="button" variant="secondary">
          Use budget period
        </Button>
      </div>
    </Card>
  );
}
