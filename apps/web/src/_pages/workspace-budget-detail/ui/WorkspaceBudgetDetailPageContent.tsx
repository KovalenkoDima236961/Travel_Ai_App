"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { CostBreakdownBars } from "@/components/analytics/CostBreakdownBars";
import { CostInsightsPanel } from "@/components/analytics/CostInsightsPanel";
import { CostSummaryCards, type CostSummaryCard } from "@/components/analytics/CostSummaryCards";
import { CostWarningsPanel } from "@/components/analytics/CostWarningsPanel";
import { ExpensiveItemsTable } from "@/components/analytics/ExpensiveItemsTable";
import {
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button, buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { WorkspaceBudgetExportMenu } from "@/features/workspace-budget";
import { WorkspaceBudgetFormDialog } from "@/features/workspace-budget";
import {
  canCreateTripsInWorkspace,
  canManageWorkspace,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import { useWorkspaceBudgetSummary } from "@/features/workspace-budget";
import { useWorkspaceBudgetMutations } from "@/features/workspace-budget";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import type { CostAmountBreakdown, CostInsight } from "@/entities/cost-analytics/model";
import type { CreateWorkspaceBudgetInput } from "@/entities/workspace-budget/model";
import {
  formatBudgetPeriod,
  mutationMessage
} from "../model/workspaceBudgetDetailModel";
import { BudgetTripsTable, BudgetUtilization } from "./BudgetDetailSections";

export function WorkspaceBudgetDetailPageContent() {
  const params = useParams<{ workspaceId: string; budgetId: string }>();
  const router = useRouter();
  const workspaceId = params.workspaceId;
  const budgetId = params.budgetId;
  const { setCurrentWorkspace } = useWorkspaces();
  const [editOpen, setEditOpen] = useState(false);
  const summaryQuery = useWorkspaceBudgetSummary({ workspaceId, budgetId });
  const mutations = useWorkspaceBudgetMutations(workspaceId);

  const workspaceQuery = useQuery({
    queryKey: workspaceKeys.detail(workspaceId),
    queryFn: () => getWorkspace(workspaceId),
    enabled: Boolean(workspaceId)
  });

  useEffect(() => {
    if (workspaceQuery.isSuccess) {
      setCurrentWorkspace(workspaceId);
    }
  }, [setCurrentWorkspace, workspaceId, workspaceQuery.isSuccess]);

  const workspace = workspaceQuery.data ?? null;
  const summary = summaryQuery.data ?? null;
  const budget = summary?.budget ?? null;
  const canManage = workspace ? canManageWorkspace(workspace.currentUserRole) : false;
  const canEditTrips = workspace ? canCreateTripsInWorkspace(workspace.currentUserRole) : false;
  const currency = budget?.currency ?? "EUR";
  const categoryEntries = useMemo<CostAmountBreakdown[]>(
    () =>
      summary
        ? summary.byCategory.map((entry) => ({
            category: entry.category as CostAmountBreakdown["category"],
            name: entry.category,
            amount: entry.amount,
            percentage: entry.percentageOfEstimatedTotal,
            itemCount: entry.itemCount
          }))
        : [],
    [summary]
  );
  const sourceEntries = useMemo<CostAmountBreakdown[]>(
    () =>
      summary
        ? summary.bySource.map((entry) => ({
            source: entry.source as CostAmountBreakdown["source"],
            name: entry.source,
            amount: entry.amount,
            percentage: entry.percentageOfEstimatedTotal,
            itemCount: entry.itemCount
          }))
        : [],
    [summary]
  );
  const summaryCards = useMemo<CostSummaryCard[]>(() => {
    if (!summary || !budget) {
      return [];
    }
    const over = summary.summary.overBudgetAmount > 0;
    return [
      {
        label: "Estimated total",
        value: formatAnalyticsMoney(summary.summary.estimatedTotal, currency),
        detail: `Budget ${formatPlainMoney(budget.amount, currency)}`
      },
      {
        label: over ? "Over budget" : "Remaining",
        value: formatAnalyticsMoney(
          over ? summary.summary.overBudgetAmount : summary.summary.remainingAmount,
          currency
        ),
        detail: formatPercent(summary.summary.utilizationPercent),
        tone: over ? "danger" : "ok"
      },
      {
        label: "Trips included",
        value: String(summary.summary.tripCount),
        detail: formatBudgetPeriod(budget)
      },
      {
        label: "Missing estimates",
        value: String(summary.summary.missingEstimateCount),
        detail: `${summary.summary.uncertainEstimateCount} uncertain`,
        tone: summary.summary.missingEstimateCount > 0 ? "warning" : "ok"
      }
    ];
  }, [budget, currency, summary]);

  function submitEdit(input: CreateWorkspaceBudgetInput) {
    if (!budget) {
      return;
    }
    mutations.updateBudget.mutate(
      { budgetId: budget.id, input },
      {
        onSuccess: () => {
          setEditOpen(false);
          void summaryQuery.refetch();
        }
      }
    );
  }

  function archiveBudget() {
    if (!budget) {
      return;
    }
    const confirmed = window.confirm("Archive this workspace budget? Existing trips will not be changed.");
    if (!confirmed) {
      return;
    }
    const reason = window.prompt("Archive reason") ?? undefined;
    mutations.archiveBudget.mutate(
      { budgetId: budget.id, reason },
      { onSuccess: () => router.push(`/workspaces/${workspaceId}/budgets`) }
    );
  }

  function handleInsightAction(insight: CostInsight) {
    if (insight.action?.type === "open_workspace_analytics") {
      router.push(`/workspaces/${workspaceId}/analytics`);
      return;
    }
    if (insight.action?.tripId) {
      router.push(`/trips/${insight.action.tripId}/analytics`);
    }
  }

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href={`/workspaces/${workspaceId}/budgets`}>
            Back to budgets
          </Link>
          <h1 className="mt-3 text-3xl font-semibold text-slate-950">
            {budget?.name ?? "Workspace budget"}
          </h1>
          {budget ? (
            <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
              {formatPlainMoney(budget.amount, budget.currency)} · {formatBudgetPeriod(budget)}
            </p>
          ) : null}
        </div>
        {budget ? (
          <div className="flex flex-wrap gap-2">
            <Link className={buttonStyles({ variant: "secondary" })} href={`/workspaces/${workspaceId}/analytics`}>
              Workspace analytics
            </Link>
            {canManage && budget.status === "active" ? (
              <>
                <Button onClick={() => setEditOpen(true)} type="button" variant="secondary">
                  Edit
                </Button>
                {!budget.isPrimary ? (
                  <Button
                    onClick={() => mutations.makePrimary.mutate(budget.id, { onSuccess: () => void summaryQuery.refetch() })}
                    type="button"
                    variant="secondary"
                  >
                    Make primary
                  </Button>
                ) : null}
                <Button onClick={archiveBudget} type="button" variant="danger">
                  Archive
                </Button>
              </>
            ) : null}
          </div>
        ) : null}
      </div>

      {summaryQuery.isLoading || workspaceQuery.isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading budget summary...
        </div>
      ) : null}

      {summaryQuery.isError || workspaceQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {summaryQuery.error instanceof Error
            ? summaryQuery.error.message
            : workspaceQuery.error instanceof Error
              ? workspaceQuery.error.message
              : "Could not load budget summary."}
        </div>
      ) : null}

      {summary && budget ? (
        <div className="space-y-6">
          <Card className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-semibold text-slate-950">Budget report</p>
              <p className="mt-1 text-sm text-slate-600">
                Generated {new Date(summary.generatedAt).toLocaleString()}.
              </p>
            </div>
            <WorkspaceBudgetExportMenu summary={summary} title={`${budget.name} budget`} />
          </Card>

          <CostSummaryCards cards={summaryCards} />
          <BudgetUtilization amount={budget.amount} currency={currency} summary={summary.summary} />

          <div className="grid gap-6 xl:grid-cols-2">
            <BudgetTripsTable currency={currency} trips={summary.byTrip} />
            <CostBreakdownBars
              currency={currency}
              entries={categoryEntries}
              title="Cost by category"
              valueKey="category"
            />
            <CostBreakdownBars
              currency={currency}
              entries={sourceEntries}
              title="Cost by source"
              valueKey="source"
            />
          </div>

          <ExpensiveItemsTable
            canEdit={canEditTrips}
            currency={currency}
            items={summary.expensiveItems}
            showTrip
          />

          <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
            <CostInsightsPanel
              canEdit={canEditTrips}
              insights={summary.insights}
              onAction={handleInsightAction}
            />
            <CostWarningsPanel warnings={summary.warnings} />
          </div>
        </div>
      ) : null}

      {budget ? (
        <WorkspaceBudgetFormDialog
          error={mutationMessage(mutations.updateBudget.error)}
          initialBudget={budget}
          isSubmitting={mutations.updateBudget.isPending}
          onClose={() => setEditOpen(false)}
          onSubmit={submitEdit}
          open={editOpen}
          submitLabel="Save changes"
          title="Edit workspace budget"
        />
      ) : null}
    </PageContainer>
  );
}

