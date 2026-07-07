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
import { PageContainer } from "@/components/layout/PageContainer";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import {
  canCreateTripsInWorkspace,
  canManageWorkspace,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import { useWorkspaceCostAnalytics } from "@/hooks/useWorkspaceCostAnalytics";
import type { CostInsight, WorkspaceCostAnalyticsParams } from "@/entities/cost-analytics/model";
import {
  COMMON_CURRENCIES,
  monthBreakdown,
  rangeForPreset,
  type DatePreset
} from "../model/workspaceCostAnalyticsModel";
import { WorkspaceBudgetOverview } from "./WorkspaceBudgetOverview";

export function WorkspaceCostAnalyticsPageContent() {
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

      <WorkspaceBudgetOverview
        activeBudget={activeBudget}
        canManage={canManage}
        onUseBudgetPeriod={useBudgetPeriod}
        workspaceId={workspaceId}
        workspaceLoaded={workspaceQuery.isSuccess}
      />

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
