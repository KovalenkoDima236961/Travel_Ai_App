"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { CostBreakdownBars } from "@/components/analytics/CostBreakdownBars";
import { CostInsightsPanel } from "@/components/analytics/CostInsightsPanel";
import { CostReportExportMenu } from "@/components/analytics/CostReportExportMenu";
import { CostSummaryCards, type CostSummaryCard } from "@/components/analytics/CostSummaryCards";
import { CostWarningsPanel } from "@/components/analytics/CostWarningsPanel";
import { ExpensiveItemsTable } from "@/components/analytics/ExpensiveItemsTable";
import { WorkspaceTripsCostTable } from "@/components/analytics/WorkspaceTripsCostTable";
import {
  formatAnalyticsLabel,
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import {
  canCreateTripsInWorkspace,
  canManageWorkspace,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import { useWorkspaceCostAnalytics } from "@/hooks/useWorkspaceCostAnalytics";
import type {
  CostAmountBreakdown,
  CostInsight,
  WorkspaceCostAnalyticsParams
} from "@/types/cost-analytics";

const COMMON_CURRENCIES = ["EUR", "USD", "GBP", "JPY", "CAD", "AUD"];
type DatePreset = "all" | "this-year" | "next-12" | "custom";

export default function WorkspaceCostAnalyticsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceCostAnalyticsPageContent />
    </ProtectedRoute>
  );
}

function WorkspaceCostAnalyticsPageContent() {
  const params = useParams<{ workspaceId: string }>();
  const router = useRouter();
  const workspaceId = params.workspaceId;
  const { setCurrentWorkspace } = useWorkspaces();
  const [currency, setCurrency] = useState("EUR");
  const [datePreset, setDatePreset] = useState<DatePreset>("all");
  const [customFrom, setCustomFrom] = useState("");
  const [customTo, setCustomTo] = useState("");

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

  const analyticsParams = useMemo<WorkspaceCostAnalyticsParams>(() => {
    const range = rangeForPreset(datePreset, customFrom, customTo);
    return {
      currency,
      from: range.from,
      to: range.to
    };
  }, [currency, customFrom, customTo, datePreset]);

  const analyticsQuery = useWorkspaceCostAnalytics({
    workspaceId,
    params: analyticsParams,
    enabled: Boolean(workspaceId)
  });

  const workspace = workspaceQuery.data ?? null;
  const analytics = analyticsQuery.data ?? null;
  const canEdit = workspace ? canCreateTripsInWorkspace(workspace.currentUserRole) : false;
  const canManage = workspace ? canManageWorkspace(workspace.currentUserRole) : false;
  const activeBudget = analytics?.activeBudget ?? null;
  const topCategory = analytics?.byCategory[0];
  const monthEntries = useMemo(
    () => (analytics ? monthBreakdown(analytics.byMonth) : []),
    [analytics]
  );
  const summaryCards = useMemo<CostSummaryCard[]>(() => {
    if (!analytics) {
      return [];
    }
    return [
      {
        label: "Workspace total",
        value: formatAnalyticsMoney(analytics.summary.estimatedTotal, analytics.currency),
        detail:
          analytics.summary.budgetTotal != null
            ? `Budget ${formatPlainMoney(analytics.summary.budgetTotal, analytics.currency)}`
            : "Known trip budgets only"
      },
      {
        label: "Trips included",
        value: String(analytics.summary.tripCount),
        detail: `${analytics.summary.overBudgetTripCount} over budget`,
        tone: analytics.summary.overBudgetTripCount > 0 ? "warning" : "ok"
      },
      {
        label: "Missing estimates",
        value: String(analytics.summary.missingEstimateCount),
        detail: `${analytics.summary.uncertainEstimateCount} uncertain`,
        tone: analytics.summary.missingEstimateCount > 0 ? "warning" : "ok"
      },
      {
        label: "Top category",
        value: topCategory
          ? formatAnalyticsLabel(topCategory.category ?? topCategory.name)
          : "No costs",
        detail: topCategory
          ? `${formatAnalyticsMoney(topCategory.amount, analytics.currency)} · ${formatPercent(topCategory.percentage)}`
          : undefined
      }
    ];
  }, [analytics, topCategory]);

  function handleInsightAction(insight: CostInsight) {
    if (
      (insight.type === "workspace_budget_exceeded" ||
        insight.type === "workspace_budget_nearing_limit" ||
        insight.action?.type === "open_workspace_analytics") &&
      activeBudget
    ) {
      router.push(`/workspaces/${workspaceId}/budgets/${activeBudget.id}`);
      return;
    }
    if (insight.action?.type === "export_report") {
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    if (insight.action?.tripId) {
      router.push(`/trips/${insight.action.tripId}/analytics`);
    }
  }

  function useBudgetPeriod() {
    if (!activeBudget) {
      return;
    }
    if (!activeBudget.periodStart && !activeBudget.periodEnd) {
      setDatePreset("all");
      setCustomFrom("");
      setCustomTo("");
      return;
    }
    setDatePreset("custom");
    setCustomFrom(activeBudget.periodStart ?? "");
    setCustomTo(activeBudget.periodEnd ?? "");
  }

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href={`/workspaces/${workspaceId}`}>
            Back to workspace
          </Link>
          <h1 className="mt-3 text-3xl font-semibold text-slate-950">
            Workspace cost analytics
          </h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
            Aggregate approximate planning costs across workspace trips.
          </p>
        </div>
        <Link className={buttonStyles({ variant: "secondary" })} href="/trips/new">
          Create trip
        </Link>
      </div>

      <Card className="mb-6">
        <div className="grid gap-4 md:grid-cols-4">
          <label className="text-sm font-medium text-slate-700">
            Currency
            <select
              className="mt-2 block h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900"
              onChange={(event) => setCurrency(event.target.value)}
              value={currency}
            >
              {COMMON_CURRENCIES.map((code) => (
                <option key={code} value={code}>
                  {code}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium text-slate-700">
            Date range
            <select
              className="mt-2 block h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900"
              onChange={(event) => setDatePreset(event.target.value as DatePreset)}
              value={datePreset}
            >
              <option value="all">All trips</option>
              <option value="this-year">This year</option>
              <option value="next-12">Next 12 months</option>
              <option value="custom">Custom</option>
            </select>
          </label>
          <label className="text-sm font-medium text-slate-700">
            From
            <input
              className="mt-2 block h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900 disabled:bg-slate-50"
              disabled={datePreset !== "custom"}
              onChange={(event) => setCustomFrom(event.target.value)}
              type="date"
              value={customFrom}
            />
          </label>
          <label className="text-sm font-medium text-slate-700">
            To
            <input
              className="mt-2 block h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900 disabled:bg-slate-50"
              disabled={datePreset !== "custom"}
              onChange={(event) => setCustomTo(event.target.value)}
              type="date"
              value={customTo}
            />
          </label>
        </div>
      </Card>

      {activeBudget ? (
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
            <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/workspaces/${workspaceId}/budgets/${activeBudget.id}`}>
              View budget
            </Link>
            <Button onClick={useBudgetPeriod} size="sm" type="button" variant="secondary">
              Use budget period
            </Button>
          </div>
        </Card>
      ) : workspaceQuery.isSuccess ? (
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
      ) : null}

      {workspaceQuery.isLoading || analyticsQuery.isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading workspace cost analytics...
        </div>
      ) : null}

      {workspaceQuery.isError || analyticsQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {workspaceQuery.error instanceof Error
            ? workspaceQuery.error.message
            : analyticsQuery.error instanceof Error
              ? analyticsQuery.error.message
              : "Could not load workspace analytics."}
        </div>
      ) : null}

      {analytics ? (
        <div className="space-y-6">
          <Card className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-semibold text-slate-950">
                {workspace?.name ?? "Workspace"} cost report
              </p>
              <p className="mt-1 text-sm text-slate-600">
                Generated {new Date(analytics.generatedAt).toLocaleString()}.
              </p>
            </div>
            <CostReportExportMenu
              analytics={analytics}
              scope="workspace"
              title={`${workspace?.name ?? "workspace"} cost analytics`}
            />
          </Card>

          <CostSummaryCards cards={summaryCards} />

          <div className="grid gap-6 xl:grid-cols-2">
            <WorkspaceTripsCostTable
              currency={analytics.currency}
              trips={analytics.byTrip}
            />
            <CostBreakdownBars
              currency={analytics.currency}
              entries={analytics.byCategory}
              title="Cost by category"
              valueKey="category"
            />
            <CostBreakdownBars
              currency={analytics.currency}
              entries={analytics.bySource}
              title="Cost by source"
              valueKey="source"
            />
            <CostBreakdownBars
              currency={analytics.currency}
              entries={monthEntries}
              title="Cost by month"
              valueKey="name"
            />
          </div>

          <ExpensiveItemsTable
            canEdit={canEdit}
            currency={analytics.currency}
            items={analytics.expensiveItems}
            showTrip
          />

          <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
            <CostInsightsPanel
              canEdit={canEdit}
              insights={analytics.insights}
              onAction={handleInsightAction}
            />
            <CostWarningsPanel warnings={analytics.warnings} />
          </div>
        </div>
      ) : null}
    </PageContainer>
  );
}

function rangeForPreset(preset: DatePreset, customFrom: string, customTo: string) {
  if (preset === "custom") {
    return { from: customFrom || null, to: customTo || null };
  }
  if (preset === "this-year") {
    const year = new Date().getFullYear();
    return { from: `${year}-01-01`, to: `${year}-12-31` };
  }
  if (preset === "next-12") {
    const now = new Date();
    const end = new Date(now);
    end.setFullYear(end.getFullYear() + 1);
    return { from: formatDateInput(now), to: formatDateInput(end) };
  }
  return { from: null, to: null };
}

function monthBreakdown(months: Array<{ month: string; estimatedTotal: number; tripCount: number }>): CostAmountBreakdown[] {
  const total = months.reduce((sum, month) => sum + month.estimatedTotal, 0);
  return months.map((month) => ({
    name: month.month,
    amount: month.estimatedTotal,
    percentage: total > 0 ? Math.round((month.estimatedTotal / total) * 10000) / 100 : 0,
    itemCount: month.tripCount
  }));
}

function formatDateInput(date: Date) {
  return date.toISOString().slice(0, 10);
}
