"use client";

import { useMemo } from "react";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { analyzeItineraryQuality } from "@/entities/itinerary/model/quality-analyzer";
import {
  buildImproveDayInstruction,
  buildImproveItemInstruction
} from "@/entities/itinerary/model/quality-instruction-builder";
import { cn } from "@/lib/utils";
import type { AvailabilityResultByItem } from "@/entities/availability/model";
import type { BudgetSummary } from "@/entities/budget/model";
import type { DayDistanceSummary } from "@/entities/itinerary/model/distance-utils";
import type { QualityIssue, QualityIssueSeverity, QualityIssueType } from "@/entities/quality/model";
import type { RouteEstimate } from "@/entities/route/model";
import type { Trip } from "@/entities/trip/model";
import type { WeatherForecast } from "@/entities/weather/model";

type TripQualityChecksProps = {
  trip: Trip;
  weatherForecast?: WeatherForecast | null;
  routeEstimatesByDay?: Record<number, RouteEstimate | null>;
  fallbackDistanceSummaries?: DayDistanceSummary[];
  maxWalkingKmPerDay?: number | null;
  budgetSummary?: BudgetSummary | null;
  availabilityResultsByItem?: AvailabilityResultByItem;
  onImproveDay?: (dayNumber: number, instruction: string) => Promise<void>;
  onImproveItem?: (
    dayNumber: number,
    itemIndex: number,
    instruction: string
  ) => Promise<void>;
  onOptimizeDayForBudget?: (dayNumber: number) => void;
  isImproving?: boolean;
  isOptimizingBudget?: boolean;
  isEditing?: boolean;
};

const DEFAULT_VISIBLE_ISSUE_COUNT = 5;

const actionableItemIssueTypes: QualityIssueType[] = [
  "place_may_be_closed",
  "place_match_low_confidence",
  "place_no_confident_match",
  "expensive_item",
  "availability_unavailable"
];

export function TripQualityChecks({
  trip,
  weatherForecast,
  routeEstimatesByDay,
  fallbackDistanceSummaries,
  maxWalkingKmPerDay,
  budgetSummary,
  availabilityResultsByItem,
  onImproveDay,
  onImproveItem,
  onOptimizeDayForBudget,
  isImproving = false,
  isOptimizingBudget = false,
  isEditing = false
}: TripQualityChecksProps) {
  const itinerary = trip.itinerary;

  const summary = useMemo(() => {
    if (!itinerary) {
      return null;
    }

    return analyzeItineraryQuality({
      itinerary,
      tripStartDate: trip.startDate,
      weatherForecast,
      routeEstimatesByDay,
      fallbackDistanceSummaries,
      maxWalkingKmPerDay,
      accommodation: trip.accommodation ?? null,
      tripBudget: trip.budget,
      budgetSummary,
      availabilityResultsByItem
    });
  }, [
    availabilityResultsByItem,
    budgetSummary,
    fallbackDistanceSummaries,
    itinerary,
    maxWalkingKmPerDay,
    routeEstimatesByDay,
    trip.accommodation,
    trip.budget,
    trip.startDate,
    weatherForecast
  ]);

  const allIssues = useMemo(() => (summary ? flattenIssues(summary.byDay) : []), [summary]);

  if (!itinerary || !summary) {
    return null;
  }

  const visibleIssues = allIssues.slice(0, DEFAULT_VISIBLE_ISSUE_COUNT);
  const hasHiddenIssues = allIssues.length > DEFAULT_VISIBLE_ISSUE_COUNT;
  const canImprove = Boolean(onImproveDay && onImproveItem);
  const canOptimizeBudget = Boolean(onOptimizeDayForBudget);
  const tripIssues = summary.tripIssues;
  const highestCostDayNumber =
    budgetSummary && budgetSummary.byDay.length > 0
      ? [...budgetSummary.byDay].sort((left, right) => right.estimatedTotal - left.estimatedTotal)[0]
          .dayNumber
      : null;

  async function improveHighestCostDay() {
    if (highestCostDayNumber == null) {
      return;
    }
    await improveDay(highestCostDayNumber, summary?.byDay[highestCostDayNumber] ?? []);
  }

  async function improveDay(dayNumber: number, issues: QualityIssue[]) {
    if (!onImproveDay) {
      return;
    }
    const instruction = buildImproveDayInstruction(dayNumber, issues);
    await onImproveDay(dayNumber, instruction);
  }

  async function improveItem(dayNumber: number, itemIndex: number, issues: QualityIssue[]) {
    if (!onImproveItem) {
      return;
    }
    const instruction = buildImproveItemInstruction(dayNumber, itemIndex, issues);
    await onImproveItem(dayNumber, itemIndex, instruction);
  }

  return (
    <Card>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-950">Trip Quality Checks</h2>
          <p className="mt-1 text-sm text-slate-600">
            {summary.total === 0
              ? "No major issues found."
              : `${summary.total} ${summary.total === 1 ? "issue" : "issues"} found.`}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <SeverityPill label="Critical" severity="critical" value={summary.critical} />
          <SeverityPill label="Warning" severity="warning" value={summary.warning} />
          <SeverityPill label="Info" severity="info" value={summary.info} />
        </div>
      </div>

      {isEditing ? (
        <div className="mt-4 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          Save or cancel edits before improving with AI.
        </div>
      ) : null}

      {tripIssues.length > 0 ? (
        <div className="mt-5 space-y-3 border-b border-slate-100 pb-4">
          {tripIssues.map((issue) => (
            <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between" key={issue.id}>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <SeverityBadge severity={issue.severity} />
                  <p className="font-medium text-slate-950">{issue.title}</p>
                </div>
                <p className="mt-1 text-sm leading-6 text-slate-600">{issue.message}</p>
                <p className="mt-1 text-xs leading-5 text-slate-500">{issue.suggestion}</p>
              </div>
              {canImprove &&
              issue.type === "trip_budget_exceeded" &&
              highestCostDayNumber != null ? (
                <div className="flex flex-wrap gap-2">
                  {canOptimizeBudget ? (
                    <Button
                      disabled={isEditing || isOptimizingBudget}
                      onClick={() => onOptimizeDayForBudget?.(highestCostDayNumber)}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      {isOptimizingBudget
                        ? "Optimizing..."
                        : `Optimize Day ${highestCostDayNumber}`}
                    </Button>
                  ) : null}
                  <Button
                    disabled={isEditing || isImproving}
                    onClick={improveHighestCostDay}
                    size="sm"
                    type="button"
                    variant="secondary"
                  >
                    {isImproving ? "Improving..." : `Improve day ${highestCostDayNumber}`}
                  </Button>
                </div>
              ) : null}
            </div>
          ))}
        </div>
      ) : null}

      {summary.total === 0 ? (
        <div className="mt-4 rounded-md border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
          Checks update as route estimates, weather, opening hours, and place match data become
          available.
        </div>
      ) : (
        <div className="mt-5 space-y-4">
          <IssueGroups
            canImprove={canImprove}
            canOptimizeBudget={canOptimizeBudget}
            disabled={isEditing || isImproving}
            isOptimizingBudget={isOptimizingBudget}
            isImproving={isImproving}
            issues={visibleIssues}
            onImproveDay={improveDay}
            onImproveItem={improveItem}
            onOptimizeDayForBudget={onOptimizeDayForBudget}
          />

          {hasHiddenIssues ? (
            <details className="rounded-md border border-slate-200 bg-slate-50 p-3">
              <summary className="cursor-pointer select-none text-sm font-medium text-primary-700 hover:text-primary-600">
                Show all checks
              </summary>
              <div className="mt-4">
                <IssueGroups
                  canImprove={canImprove}
                  canOptimizeBudget={canOptimizeBudget}
                  disabled={isEditing || isImproving}
                  isOptimizingBudget={isOptimizingBudget}
                  isImproving={isImproving}
                  issues={allIssues}
                  onImproveDay={improveDay}
                  onImproveItem={improveItem}
                  onOptimizeDayForBudget={onOptimizeDayForBudget}
                />
              </div>
            </details>
          ) : null}
        </div>
      )}
    </Card>
  );
}

type IssueGroupsProps = {
  issues: QualityIssue[];
  canImprove: boolean;
  canOptimizeBudget: boolean;
  disabled: boolean;
  isImproving: boolean;
  isOptimizingBudget: boolean;
  onImproveDay: (dayNumber: number, issues: QualityIssue[]) => Promise<void>;
  onImproveItem: (
    dayNumber: number,
    itemIndex: number,
    issues: QualityIssue[]
  ) => Promise<void>;
  onOptimizeDayForBudget?: (dayNumber: number) => void;
};

function IssueGroups({
  issues,
  canImprove,
  canOptimizeBudget,
  disabled,
  isImproving,
  isOptimizingBudget,
  onImproveDay,
  onImproveItem,
  onOptimizeDayForBudget
}: IssueGroupsProps) {
  const groups = groupIssuesByDay(issues);

  return (
    <div className="divide-y divide-slate-100">
      {groups.map(({ dayNumber, dayIssues }) => {
        const dayActionIssues = dayIssues.filter(
          (issue) =>
            issue.scope === "day" &&
            (issue.severity === "critical" || issue.severity === "warning")
        );
        const itemActionGroups = getActionableItemGroups(dayIssues);
        const hasBudgetIssue = dayIssues.some(isBudgetOptimizationIssue);

        return (
          <section className="py-4 first:pt-0 last:pb-0" key={dayNumber}>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <h3 className="text-sm font-semibold text-slate-950">Day {dayNumber}</h3>
              <div className="flex flex-wrap gap-2">
                {canOptimizeBudget && hasBudgetIssue ? (
                  <Button
                    disabled={disabled || isOptimizingBudget}
                    onClick={() => onOptimizeDayForBudget?.(dayNumber)}
                    size="sm"
                    type="button"
                    variant="secondary"
                  >
                    {isOptimizingBudget ? "Optimizing..." : "Optimize for budget"}
                  </Button>
                ) : null}
                {canImprove && dayActionIssues.length > 0 ? (
                  <Button
                    disabled={disabled}
                    onClick={() => onImproveDay(dayNumber, dayIssues)}
                    size="sm"
                    type="button"
                    variant="secondary"
                  >
                    {isImproving ? "Improving..." : "Improve day"}
                  </Button>
                ) : null}
              </div>
            </div>

            <ul className="mt-3 space-y-3">
              {dayIssues.map((issue) => (
                <IssueRow issue={issue} key={issue.id} />
              ))}
            </ul>

            {canImprove && itemActionGroups.length > 0 ? (
              <div className="mt-3 flex flex-wrap gap-2">
                {itemActionGroups.map(({ itemIndex, itemIssues }) => (
                  <Button
                    disabled={disabled}
                    key={itemIndex}
                    onClick={() => onImproveItem(dayNumber, itemIndex, itemIssues)}
                    size="sm"
                    type="button"
                    variant="secondary"
                  >
                    {isImproving ? "Improving..." : `Improve item ${itemIndex + 1}`}
                  </Button>
                ))}
              </div>
            ) : null}
          </section>
        );
      })}
    </div>
  );
}

function isBudgetOptimizationIssue(issue: QualityIssue) {
  return (
    issue.type === "day_budget_high" ||
    issue.type === "high_ticket_cost" ||
    issue.type === "expensive_item" ||
    issue.type === "booking_price_higher_than_estimate"
  );
}

function IssueRow({ issue }: { issue: QualityIssue }) {
  return (
    <li className="grid gap-2 sm:grid-cols-[6rem_minmax(0,1fr)]">
      <div>
        <SeverityBadge severity={issue.severity} />
      </div>
      <div className="min-w-0">
        <p className="font-medium text-slate-950">{issue.title}</p>
        <p className="mt-1 text-sm leading-6 text-slate-600">{issue.message}</p>
        <p className="mt-1 text-xs leading-5 text-slate-500">{issue.suggestion}</p>
        {issue.type === "place_match_pending_review" ? (
          <a
            className="mt-1 inline-flex text-xs font-medium text-primary-700 hover:text-primary-600"
            href="#place-matches"
          >
            Review this match in Place Matches.
          </a>
        ) : null}
        {issue.type === "availability_unchecked" ? (
          <a
            className="mt-1 inline-flex text-xs font-medium text-primary-700 hover:text-primary-600"
            href={`#day-${issue.dayNumber}-item-${issue.itemIndex}-availability`}
          >
            Check availability.
          </a>
        ) : null}
      </div>
    </li>
  );
}

function SeverityPill({
  label,
  severity,
  value
}: {
  label: string;
  severity: QualityIssueSeverity;
  value: number;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium",
        severityClasses(severity)
      )}
    >
      <span>{value}</span>
      <span>{label}</span>
    </span>
  );
}

function SeverityBadge({ severity }: { severity: QualityIssueSeverity }) {
  return (
    <span
      className={cn(
        "inline-flex rounded-full border px-2.5 py-1 text-xs font-medium capitalize",
        severityClasses(severity)
      )}
    >
      {severity}
    </span>
  );
}

function severityClasses(severity: QualityIssueSeverity) {
  if (severity === "critical") {
    return "border-red-200 bg-red-50 text-red-700";
  }
  if (severity === "warning") {
    return "border-amber-200 bg-amber-50 text-amber-800";
  }
  return "border-slate-200 bg-slate-50 text-slate-600";
}

function flattenIssues(byDay: Record<number, QualityIssue[]>): QualityIssue[] {
  return Object.entries(byDay)
    .sort(([leftDay], [rightDay]) => Number(leftDay) - Number(rightDay))
    .flatMap(([, issues]) => issues);
}

function groupIssuesByDay(issues: QualityIssue[]) {
  const groups = new Map<number, QualityIssue[]>();

  for (const issue of issues) {
    if (typeof issue.dayNumber !== "number") {
      continue;
    }

    groups.set(issue.dayNumber, [...(groups.get(issue.dayNumber) ?? []), issue]);
  }

  return Array.from(groups.entries()).map(([dayNumber, dayIssues]) => ({
    dayNumber,
    dayIssues
  }));
}

function getActionableItemGroups(dayIssues: QualityIssue[]) {
  const groups = new Map<number, QualityIssue[]>();

  for (const issue of dayIssues) {
    if (
      issue.scope !== "item" ||
      typeof issue.itemIndex !== "number" ||
      !actionableItemIssueTypes.includes(issue.type)
    ) {
      continue;
    }

    groups.set(issue.itemIndex, [...(groups.get(issue.itemIndex) ?? []), issue]);
  }

  return Array.from(groups.entries())
    .sort(([leftIndex], [rightIndex]) => leftIndex - rightIndex)
    .map(([itemIndex, itemIssues]) => ({ itemIndex, itemIssues }));
}
