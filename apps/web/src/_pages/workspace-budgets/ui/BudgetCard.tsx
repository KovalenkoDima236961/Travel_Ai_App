import Link from "next/link";
import { Button, buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import {
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import { useWorkspaceBudgetSummary } from "@/features/workspace-budget";
import type { WorkspaceBudget } from "@/entities/workspace-budget/model";
import { formatBudgetPeriod } from "../model/workspaceBudgetsPageModel";

type BudgetCardProps = {
  budget: WorkspaceBudget;
  canManage: boolean;
  onEdit: (budget: WorkspaceBudget) => void;
  onArchive: (budget: WorkspaceBudget) => void;
  onMakePrimary: (budget: WorkspaceBudget) => void;
};

export function BudgetCard({
  budget,
  canManage,
  onEdit,
  onArchive,
  onMakePrimary
}: BudgetCardProps) {
  const summaryQuery = useWorkspaceBudgetSummary({
    workspaceId: budget.workspaceId,
    budgetId: budget.id,
    enabled: budget.status === "active"
  });
  const summary = summaryQuery.data?.summary;
  const utilization = summary?.utilizationPercent ?? 0;
  const progress = Math.min(Math.max(utilization, 0), 100);
  const over = (summary?.overBudgetAmount ?? 0) > 0;

  return (
    <Card className="flex h-full flex-col gap-5">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="break-words text-lg font-semibold text-slate-950">{budget.name}</h3>
            {budget.isPrimary ? (
              <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-semibold text-primary-700">
                Primary
              </span>
            ) : null}
            <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700">
              {budget.status}
            </span>
          </div>
          {budget.description ? (
            <p className="mt-2 text-sm leading-6 text-slate-600">{budget.description}</p>
          ) : null}
        </div>
        <div className="text-right text-sm font-semibold text-slate-950">
          {formatPlainMoney(budget.amount, budget.currency)}
        </div>
      </div>

      <div className="grid gap-3 text-sm sm:grid-cols-3">
        <Metric label="Period" value={formatBudgetPeriod(budget)} />
        <Metric
          label="Estimated"
          value={summary ? formatAnalyticsMoney(summary.estimatedTotal, budget.currency) : "Loading..."}
        />
        <Metric
          label={over ? "Over" : "Remaining"}
          tone={over ? "danger" : "ok"}
          value={
            summary
              ? formatAnalyticsMoney(
                  over ? summary.overBudgetAmount : summary.remainingAmount,
                  budget.currency
                )
              : "Loading..."
          }
        />
      </div>

      {budget.status === "active" ? (
        <div>
          <div className="flex items-center justify-between text-xs font-semibold text-slate-500">
            <span>Utilization</span>
            <span>{formatPercent(summary?.utilizationPercent)}</span>
          </div>
          <div className="mt-2 h-3 rounded-full bg-slate-100">
            <div
              className={over ? "h-3 rounded-full bg-red-600" : "h-3 rounded-full bg-primary-600"}
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      ) : null}

      <div className="mt-auto flex flex-wrap gap-2">
        <Link
          className={buttonStyles({ variant: "secondary", size: "sm" })}
          href={`/workspaces/${budget.workspaceId}/budgets/${budget.id}`}
        >
          View summary
        </Link>
        {canManage && budget.status === "active" ? (
          <>
            <Button onClick={() => onEdit(budget)} size="sm" type="button" variant="secondary">
              Edit
            </Button>
            {!budget.isPrimary ? (
              <Button onClick={() => onMakePrimary(budget)} size="sm" type="button" variant="secondary">
                Make primary
              </Button>
            ) : null}
            <Button onClick={() => onArchive(budget)} size="sm" type="button" variant="danger">
              Archive
            </Button>
          </>
        ) : null}
      </div>
    </Card>
  );
}

function Metric({
  label,
  value,
  tone = "default"
}: {
  label: string;
  value: string;
  tone?: "default" | "ok" | "danger";
}) {
  return (
    <div className="rounded-md bg-slate-50 p-3">
      <p className="text-xs font-semibold uppercase text-slate-500">{label}</p>
      <p
        className={
          tone === "danger"
            ? "mt-1 break-words font-semibold text-red-700"
            : tone === "ok"
              ? "mt-1 break-words font-semibold text-emerald-700"
              : "mt-1 break-words font-semibold text-slate-900"
        }
      >
        {value}
      </p>
    </div>
  );
}
