"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import {
  formatAnalyticsLabel,
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import { getTrip, tripKeys } from "@/lib/api/trips";
import { useTripCostAnalytics } from "@/hooks/useTripCostAnalytics";
import type { CostInsight } from "@/entities/cost-analytics/model";
import { COMMON_CURRENCIES } from "../model/tripCostAnalyticsPageModel";
import { AnalyticsHeader } from "./AnalyticsHeader";
import { BreakdownCard } from "./BreakdownCard";
import { CostByDayCard } from "./CostByDayCard";
import { ExpensiveItemsCard } from "./ExpensiveItemsCard";
import { ExportReportMenu } from "./ExportReportMenu";
import { InsightsPanel } from "./InsightsPanel";
import { SummaryCards, type SummaryCard } from "./SummaryCards";
import { WarningsPanel } from "./WarningsPanel";
import { instrumentSans, newsreader } from "./fonts";
import { ArrowLeftIcon } from "./icons";

const FOOTER_LINK =
  "inline-flex h-[42px] items-center rounded-full border border-sand-400 bg-white px-[18px] text-sm font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900";

export function TripCostAnalyticsPageContent() {
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
  const summaryCards = useMemo<SummaryCard[]>(() => {
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

  const isLoading = tripQuery.isLoading || analyticsQuery.isLoading;
  const isError = tripQuery.isError || analyticsQuery.isError;
  const errorMessage =
    tripQuery.error instanceof Error
      ? tripQuery.error.message
      : analyticsQuery.error instanceof Error
        ? analyticsQuery.error.message
        : "Could not load cost analytics.";

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <AnalyticsHeader />

      <div className="mx-auto max-w-[1280px] px-6 pb-[72px] pt-9 sm:px-10">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div>
            <Link
              className="inline-flex items-center gap-2 text-sm font-medium text-clay-deep hover:text-clay"
              href={`/trips/${tripId}`}
            >
              <ArrowLeftIcon className="h-[15px] w-[15px]" />
              Back to {trip?.destination ?? "trip"}
            </Link>
            <h1 className="mt-3.5 font-newsreader text-[44px] font-medium leading-[1.02] tracking-[-0.02em] text-cocoa-900">
              Cost analytics
            </h1>
            <p className="mt-3 max-w-xl text-[15px] text-cocoa-500">
              Approximate planning estimates by day, category, source, and confidence.
            </p>
            {analytics ? (
              <p className="mt-2 text-[13px] text-cocoa-400">
                Generated {new Date(analytics.generatedAt).toLocaleString()}
              </p>
            ) : null}
          </div>
          <div className="flex items-center gap-2.5">
            <label className="sr-only" htmlFor="trip-analytics-currency">
              Currency
            </label>
            <select
              className="h-[42px] rounded-full border border-sand-400 bg-white px-[18px] text-sm text-cocoa-900"
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
            {analytics ? (
              <ExportReportMenu
                analytics={analytics}
                title={`${trip?.destination ?? "trip"} cost analytics`}
              />
            ) : null}
          </div>
        </div>

        {isLoading ? (
          <div className="mt-8 rounded-[18px] border border-sand-300 bg-white px-6 py-5 text-sm text-cocoa-500">
            Loading cost analytics…
          </div>
        ) : null}

        {isError ? (
          <div className="mt-8 rounded-[18px] border border-[#E5C3B6] bg-[#FBF0EB] px-6 py-5 text-sm text-clay-deep">
            {errorMessage}
          </div>
        ) : null}

        {analytics ? (
          <div className="mt-8 flex flex-col gap-6">
            <SummaryCards cards={summaryCards} />

            <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
              <CostByDayCard currency={analytics.currency} days={analytics.byDay} />
              <BreakdownCard
                currency={analytics.currency}
                entries={analytics.byCategory}
                title="Cost by category"
                valueKey="category"
              />
            </div>

            <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
              <BreakdownCard
                currency={analytics.currency}
                entries={analytics.bySource}
                title="Cost by source"
                valueKey="source"
              />
              <BreakdownCard
                currency={analytics.currency}
                entries={analytics.byConfidence}
                title="Cost by confidence"
                valueKey="confidence"
              />
            </div>

            <ExpensiveItemsCard
              canEdit={canEdit}
              currency={analytics.currency}
              items={expensiveItems}
              onOptimizeDay={(dayNumber) => router.push(`/trips/${tripId}?budgetOptimizeDay=${dayNumber}`)}
            />

            <InsightsPanel
              canEdit={canEdit}
              insights={analytics.insights}
              onAction={handleInsightAction}
            />

            <WarningsPanel warnings={analytics.warnings} />

            <div className="flex flex-wrap gap-2.5">
              <Link className={FOOTER_LINK} href={`/trips/${tripId}`}>
                Open itinerary
              </Link>
              <Link className={FOOTER_LINK} href={`/trips/${tripId}#cost-splitting`}>
                View per-traveler split
              </Link>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
