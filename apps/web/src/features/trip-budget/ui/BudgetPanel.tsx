"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BudgetEditForm } from "./BudgetEditForm";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { approvalRiskKeys } from "@/lib/api/approval-risk";
import { budgetKeys, getTripBudgetSummary, updateTripBudget } from "@/lib/api/budget";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { tripKeys } from "@/lib/api/trips";
import { formatApproxMoney, formatMoney } from "@/entities/budget/model";
import { getErrorMessage } from "@/lib/utils";
import type { Budget, BudgetSummary } from "@/entities/budget/model";
import type { Trip } from "@/entities/trip/model";

type BudgetPanelProps = {
  trip: Trip;
  canEdit: boolean;
  offline?: boolean;
  offlineSummary?: BudgetSummary | null;
  perPersonAverage?: { amount: number; currency: string } | null;
  optimizationDisabled?: boolean;
  onOpenBudgetOptimization?: (dayNumber: number) => void;
};

export function BudgetPanel({
  trip,
  canEdit,
  offline = false,
  offlineSummary,
  perPersonAverage = null,
  optimizationDisabled = false,
  onOpenBudgetOptimization
}: BudgetPanelProps) {
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const summaryQuery = useQuery({
    queryKey: budgetKeys.summary(trip.id),
    queryFn: () => getTripBudgetSummary(trip.id),
    enabled: !offline
  });

  const updateMutation = useMutation({
    mutationFn: (budget: Budget | null) => updateTripBudget(trip.id, budget),
    onSuccess: async (_budget, variables) => {
      setError(null);
      setMessage(variables == null ? "Budget cleared." : "Budget updated.");
      setIsEditing(false);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(trip.id) }),
        queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(trip.id) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(trip.id) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(trip.id) })
      ]);
    },
    onError: (err) => {
      setMessage(null);
      setError(getErrorMessage(err, "Could not update the budget."));
    }
  });

  const summary = offline ? offlineSummary : summaryQuery.data;
  const currency = summary?.currency ?? trip.budget?.currency ?? trip.budgetCurrency ?? "EUR";
  const currentBudget: Budget | null = trip.budget ?? null;

  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <h2 className="text-lg font-semibold text-slate-950">Budget</h2>
        {canEdit && !isEditing ? (
          <Button onClick={() => setIsEditing(true)} size="sm" type="button" variant="secondary">
            {currentBudget ? "Edit" : "Set budget"}
          </Button>
        ) : null}
      </div>

      {message ? <p className="mt-2 text-sm text-emerald-700">{message}</p> : null}
      {error ? <p className="mt-2 text-sm text-red-700">{error}</p> : null}

      {isEditing ? (
        <div className="mt-4">
          <BudgetEditForm
            defaultCurrency={currency}
            initial={currentBudget}
            isSaving={updateMutation.isPending}
            onCancel={() => {
              setIsEditing(false);
              setError(null);
            }}
            onClear={() => updateMutation.mutate(null)}
            onSave={(budget) => updateMutation.mutate(budget)}
          />
        </div>
      ) : (
        <BudgetSummaryView
          currency={currency}
          isLoading={!offline && summaryQuery.isLoading}
          onOpenBudgetOptimization={canEdit ? onOpenBudgetOptimization : undefined}
          optimizationDisabled={optimizationDisabled}
          perPersonAverage={perPersonAverage}
          summary={summary ?? null}
        />
      )}
    </Card>
  );
}

function BudgetSummaryView({
  summary,
  currency,
  isLoading,
  optimizationDisabled,
  perPersonAverage,
  onOpenBudgetOptimization
}: {
  summary: BudgetSummary | null;
  currency: string;
  isLoading: boolean;
  optimizationDisabled: boolean;
  perPersonAverage?: { amount: number; currency: string } | null;
  onOpenBudgetOptimization?: (dayNumber: number) => void;
}) {
  if (isLoading) {
    return <p className="mt-4 text-sm text-slate-500">Loading budget summary…</p>;
  }
  if (!summary) {
    return <p className="mt-4 text-sm text-slate-500">Budget summary is unavailable.</p>;
  }

  const overBudget = (summary.overBudgetBy ?? 0) > 0;
  const hasBudget = summary.tripBudget != null;
  const suggestedOptimizationDayNumber = getSuggestedOptimizationDay(summary);
  const canOptimize =
    suggestedOptimizationDayNumber != null && Boolean(onOpenBudgetOptimization);
  const hasTransportEstimate = summary.byCategory.some(
    (category) => category.category === "transport" && category.estimatedTotal > 0
  );

  return (
    <div className="mt-4 space-y-4 text-sm">
      <dl className="space-y-3">
        <SummaryRow
          label="Trip budget"
          value={hasBudget ? formatMoney(summary.tripBudget, currency) : "No budget set"}
        />
        <SummaryRow label="Estimated total" value={formatApproxMoney(summary.estimatedTotal, currency)} />
        {perPersonAverage ? (
          <SummaryRow
            label="Estimated per-person average"
            value={formatApproxMoney(perPersonAverage.amount, perPersonAverage.currency)}
          />
        ) : null}
        {hasBudget ? (
          <SummaryRow
            emphasis={overBudget ? "danger" : "ok"}
            label={overBudget ? "Over budget by" : "Remaining"}
            value={formatMoney(
              overBudget ? summary.overBudgetBy : summary.remaining,
              currency
            )}
          />
        ) : null}
      </dl>

      {overBudget ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          <p>
            This trip is estimated to go over budget by{" "}
            {formatApproxMoney(summary.overBudgetBy, currency)}.
          </p>
          {canOptimize ? (
            <Button
              className="mt-3"
              disabled={optimizationDisabled}
              onClick={() => onOpenBudgetOptimization?.(suggestedOptimizationDayNumber)}
              size="sm"
              type="button"
              variant="secondary"
            >
              Optimize Day {suggestedOptimizationDayNumber} for budget
            </Button>
          ) : null}
        </div>
      ) : null}

      {summary.originalCurrencyTotals?.length ? (
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Original totals
          </p>
          <ul className="mt-2 space-y-1">
            {summary.originalCurrencyTotals.map((total) => (
              <li className="flex items-center justify-between gap-3" key={total.currency}>
                <span className="text-slate-600">{total.currency}</span>
                <span className="text-slate-900">{formatMoney(total.amount, total.currency)}</span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {summary.conversionWarnings?.length ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          <p className="font-medium">
            Some costs could not be converted and are not included in the total.
          </p>
          <ul className="mt-2 space-y-1 text-xs">
            {summary.conversionWarnings.map((warning, index) => (
              <li key={`${warning.currency}-${warning.reason}-${index}`}>
                {warning.amount != null
                  ? `${formatMoney(warning.amount, warning.currency)} - ${formatWarningReason(warning.reason)}`
                  : `${warning.currency} - ${formatWarningReason(warning.reason)}`}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {summary.missingEstimateCount > 0 ? (
        <p className="text-xs text-slate-500">
          {summary.missingEstimateCount} item
          {summary.missingEstimateCount === 1 ? "" : "s"} likely need a cost estimate.
        </p>
      ) : null}

      {summary.unsupportedCurrencyCount && !summary.conversionWarnings?.length ? (
        <p className="text-xs text-slate-500">
          {summary.unsupportedCurrencyCount} item
          {summary.unsupportedCurrencyCount === 1 ? "" : "s"} use a different currency and are
          excluded from totals.
        </p>
      ) : null}

      {summary.exchangeRateInfo ? (
        <p className="text-xs text-slate-500">
          Approximate exchange rates from {formatProvider(summary.exchangeRateInfo.provider)}
          {summary.exchangeRateInfo.asOf
            ? `, as of ${formatExchangeRateDate(summary.exchangeRateInfo.asOf)}`
            : ""}
          {summary.exchangeRateInfo.fallbackUsed ? " (mock fallback used)" : ""}.
        </p>
      ) : null}

      {summary.byDay.length > 0 ? (
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Daily totals</p>
          <ul className="mt-2 space-y-1">
            {summary.byDay.map((day) => {
              const dayOver = (day.overDailyBudgetBy ?? 0) > 0;
              return (
                <li className="flex items-center justify-between gap-3" key={day.dayNumber}>
                  <span className="text-slate-600">Day {day.dayNumber}</span>
                  <span className={dayOver ? "font-medium text-red-700" : "text-slate-900"}>
                    {formatApproxMoney(day.estimatedTotal, currency)}
                    {day.dailyBudgetShare != null
                      ? ` / ${formatMoney(day.dailyBudgetShare, currency)}`
                      : ""}
                  </span>
                </li>
              );
            })}
          </ul>
          {!overBudget && canOptimize ? (
            <Button
              className="mt-3"
              disabled={optimizationDisabled}
              onClick={() => onOpenBudgetOptimization?.(suggestedOptimizationDayNumber)}
              size="sm"
              type="button"
              variant="secondary"
            >
              Optimize Day {suggestedOptimizationDayNumber} for budget
            </Button>
          ) : null}
        </div>
      ) : null}

      {summary.byCategory.length > 0 ? (
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            By category
          </p>
          <ul className="mt-2 space-y-1">
            {summary.byCategory.map((category) => (
              <li className="flex items-center justify-between gap-3" key={category.category}>
                <span className="capitalize text-slate-600">{category.category}</span>
                <span className="text-slate-900">
                  {formatApproxMoney(category.estimatedTotal, currency)}
                  <span className="ml-1 text-xs text-slate-400">
                    ({category.itemCount})
                  </span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {hasTransportEstimate ? (
        <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          Transport prices are estimates from selected options or planning data. Verify before booking.
        </p>
      ) : null}
    </div>
  );
}

function getSuggestedOptimizationDay(summary: BudgetSummary): number | null {
  if (summary.byDay.length === 0) {
    return null;
  }
  const overBudgetDay = [...summary.byDay]
    .filter((day) => (day.overDailyBudgetBy ?? 0) > 0)
    .sort((left, right) => (right.overDailyBudgetBy ?? 0) - (left.overDailyBudgetBy ?? 0))[0];
  if (overBudgetDay) {
    return overBudgetDay.dayNumber;
  }
  if ((summary.overBudgetBy ?? 0) <= 0) {
    return null;
  }
  return [...summary.byDay].sort(
    (left, right) => right.estimatedTotal - left.estimatedTotal
  )[0].dayNumber;
}

function SummaryRow({
  label,
  value,
  emphasis
}: {
  label: string;
  value: string;
  emphasis?: "ok" | "danger";
}) {
  const valueClass =
    emphasis === "danger"
      ? "font-semibold text-red-700"
      : emphasis === "ok"
        ? "font-semibold text-emerald-700"
        : "font-medium text-slate-900";
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="text-slate-600">{label}</dt>
      <dd className={valueClass}>{value}</dd>
    </div>
  );
}

function formatProvider(provider: string | null | undefined): string {
  const value = (provider ?? "").trim();
  if (!value) {
    return "the configured provider";
  }
  return value
    .split(/[_-]/g)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function formatWarningReason(reason: string): string {
  switch (reason) {
    case "unsupported_currency":
      return "unsupported currency";
    case "provider_unavailable":
      return "provider unavailable";
    case "conversion_disabled":
      return "conversion disabled";
    default:
      return "conversion unavailable";
  }
}

function formatExchangeRateDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(date);
}
