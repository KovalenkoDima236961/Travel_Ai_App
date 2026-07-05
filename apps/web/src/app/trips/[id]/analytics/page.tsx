"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { CostBreakdownBars } from "@/components/analytics/CostBreakdownBars";
import { CostByDayChart } from "@/components/analytics/CostByDayChart";
import { CostInsightsPanel } from "@/components/analytics/CostInsightsPanel";
import { CostReportExportMenu } from "@/components/analytics/CostReportExportMenu";
import { CostSummaryCards, type CostSummaryCard } from "@/components/analytics/CostSummaryCards";
import { CostWarningsPanel } from "@/components/analytics/CostWarningsPanel";
import { ExpensiveItemsTable } from "@/components/analytics/ExpensiveItemsTable";
import {
  formatAnalyticsLabel,
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { getTrip, tripKeys } from "@/lib/api/trips";
import { useTripCostAnalytics } from "@/hooks/useTripCostAnalytics";
import type { CostInsight } from "@/types/cost-analytics";

const COMMON_CURRENCIES = ["EUR", "USD", "GBP", "JPY", "CAD", "AUD"];

export default function TripCostAnalyticsPage() {
  return (
    <ProtectedRoute>
      <TripCostAnalyticsPageContent />
    </ProtectedRoute>
  );
}

function TripCostAnalyticsPageContent() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const tripId = params.id;
  const [currency, setCurrency] = useState("EUR");

  const tripQuery = useQuery({
    queryKey: tripKeys.detail(tripId),
    queryFn: () => getTrip(tripId),
    enabled: Boolean(tripId)
  });

  useEffect(() => {
    const next = tripQuery.data?.budgetCurrency?.trim().toUpperCase();
    if (next) {
      setCurrency(next);
    }
  }, [tripQuery.data?.budgetCurrency]);

  const analyticsQuery = useTripCostAnalytics({
    tripId,
    currency,
    enabled: Boolean(tripId)
  });

  const trip = tripQuery.data ?? null;
  const analytics = analyticsQuery.data ?? null;
  const canEdit = trip?.access?.canEdit ?? false;
  const topCategory = analytics?.byCategory[0];
  const expensiveItems = useMemo(
    () => analytics?.expensiveItems.map((item) => ({ ...item, tripId })) ?? [],
    [analytics, tripId]
  );
  const summaryCards = useMemo<CostSummaryCard[]>(() => {
    if (!analytics) {
      return [];
    }
    const overBudget = (analytics.summary.overBudgetAmount ?? 0) > 0;
    return [
      {
        label: "Estimated total",
        value: formatAnalyticsMoney(analytics.summary.estimatedTotal, analytics.currency),
        detail: `${analytics.summary.convertedItemCount} converted · ${analytics.summary.unconvertedItemCount} unconverted`
      },
      {
        label: overBudget ? "Over budget" : "Remaining",
        value: overBudget
          ? formatPlainMoney(analytics.summary.overBudgetAmount, analytics.currency)
          : formatPlainMoney(analytics.summary.remainingAmount, analytics.currency),
        detail:
          analytics.summary.budgetAmount != null
            ? `${formatPercent(analytics.summary.budgetUtilizationPercent)} utilized`
            : "No budget set",
        tone: overBudget ? "danger" : "ok"
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
    const action = insight.action;
    if (!action) {
      return;
    }
    if (action.type === "export_report") {
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    const day = action.dayNumber;
    const item = action.itemIndex ?? 0;
    if (action.type === "optimize_budget") {
      router.push(day ? `/trips/${tripId}?budgetOptimizeDay=${day}` : `/trips/${tripId}`);
      return;
    }
    router.push(day ? `/trips/${tripId}#day-${day}-item-${item}` : `/trips/${tripId}`);
  }

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href={`/trips/${tripId}`}>
            Back to trip
          </Link>
          <h1 className="mt-3 text-3xl font-semibold text-slate-950">
            Cost analytics
          </h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
            Approximate planning estimates by day, category, source, and confidence.
          </p>
        </div>
        <div className="flex flex-col items-start gap-2 sm:items-end">
          <label className="text-sm font-medium text-slate-700" htmlFor="trip-analytics-currency">
            Currency
          </label>
          <select
            className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900"
            id="trip-analytics-currency"
            onChange={(event) => setCurrency(event.target.value)}
            value={currency}
          >
            {COMMON_CURRENCIES.map((code) => (
              <option key={code} value={code}>
                {code}
              </option>
            ))}
          </select>
        </div>
      </div>

      {tripQuery.isLoading || analyticsQuery.isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading cost analytics...
        </div>
      ) : null}

      {tripQuery.isError || analyticsQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {tripQuery.error instanceof Error
            ? tripQuery.error.message
            : analyticsQuery.error instanceof Error
              ? analyticsQuery.error.message
              : "Could not load cost analytics."}
        </div>
      ) : null}

      {analytics ? (
        <div className="space-y-6">
          <Card className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-semibold text-slate-950">
                {trip?.destination ?? "Trip"} cost report
              </p>
              <p className="mt-1 text-sm text-slate-600">
                Generated {new Date(analytics.generatedAt).toLocaleString()}.
              </p>
            </div>
            <CostReportExportMenu
              analytics={analytics}
              scope="trip"
              title={`${trip?.destination ?? "trip"} cost analytics`}
            />
          </Card>

          <CostSummaryCards cards={summaryCards} />

          <div className="grid gap-6 xl:grid-cols-2">
            <CostByDayChart days={analytics.byDay} currency={analytics.currency} />
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
              entries={analytics.byConfidence}
              title="Cost by confidence"
              valueKey="confidence"
            />
          </div>

          <ExpensiveItemsTable
            canEdit={canEdit}
            currency={analytics.currency}
            items={expensiveItems}
            onOptimizeDay={(dayNumber) => router.push(`/trips/${tripId}?budgetOptimizeDay=${dayNumber}`)}
          />

          <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
            <CostInsightsPanel
              canEdit={canEdit}
              insights={analytics.insights}
              onAction={handleInsightAction}
            />
            <CostWarningsPanel warnings={analytics.warnings} />
          </div>

          <div className="flex flex-wrap gap-2">
            <Link className={buttonStyles({ variant: "secondary" })} href={`/trips/${tripId}`}>
              Open itinerary
            </Link>
            <Link className={buttonStyles({ variant: "secondary" })} href={`/trips/${tripId}#cost-splitting`}>
              View per-traveler split
            </Link>
          </div>
        </div>
      ) : null}
    </PageContainer>
  );
}
