"use client";

import { useTranslations } from "next-intl";
import { ErrorState, SectionLoadingState } from "@/components/ui";
import { BudgetConfidenceBadge } from "./BudgetConfidenceBadge";
import { BudgetCoverageBreakdown } from "./BudgetCoverageBreakdown";
import { BudgetImprovementActions } from "./BudgetImprovementActions";
import { BudgetRiskIssuesList } from "./BudgetRiskIssuesList";
import { CostSourceQualityTable } from "./CostSourceQualityTable";
import { PlannedVsActualAccuracyCard } from "./PlannedVsActualAccuracyCard";
import { formatMoney } from "@/entities/budget/model";
import type { BudgetConfidence } from "@/types/budget-confidence";

type BudgetConfidenceCardProps = {
  confidence?: BudgetConfidence | null;
  isLoading?: boolean;
  error?: string | null;
  onRetry?: () => void;
  retrying?: boolean;
};

export function BudgetConfidenceCard({
  confidence,
  isLoading = false,
  error = null,
  onRetry,
  retrying = false
}: BudgetConfidenceCardProps) {
  const errorsT = useTranslations("errors");
  const loadingT = useTranslations("loading");

  if (isLoading) {
    return <SectionLoadingState compact label={loadingT("budget")} />;
  }

  if (error) {
    return (
      <ErrorState
        compact
        description={errorsT("budgetConfidenceDescription")}
        developmentDetails={error}
        retryAction={onRetry ? { onRetry, pending: retrying } : undefined}
        title={errorsT("budgetConfidenceTitle")}
      />
    );
  }

  if (!confidence) {
    return (
      <ErrorState
        compact
        description={errorsT("budgetConfidenceDescription")}
        retryAction={onRetry ? { onRetry, pending: retrying } : undefined}
        title={errorsT("budgetConfidenceTitle")}
      />
    );
  }

  return (
    <section className="rounded-md border border-slate-200 bg-white p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Budget confidence
          </p>
          <div className="mt-1 flex flex-wrap items-center gap-2">
            <span className="text-2xl font-semibold text-slate-950">{confidence.score}</span>
            <span className="text-sm text-slate-500">/ 100</span>
            <BudgetConfidenceBadge level={confidence.level} />
            <BudgetConfidenceBadge riskLevel={confidence.riskLevel} />
          </div>
        </div>
        <div className="text-right text-sm">
          <p className="font-medium text-slate-950">
            {formatMoney(confidence.estimatedTotal.amount, confidence.currency)}
          </p>
          <p className="text-xs text-slate-500">estimated total</p>
        </div>
      </div>

      <p className="mt-3 text-sm leading-6 text-slate-600">{confidence.summary}</p>

      <div className="mt-4 grid gap-3 sm:grid-cols-3">
        <Metric
          label="Coverage"
          value={`${confidence.coverage.overall}%`}
        />
        <Metric
          label="Actual spend"
          value={formatMoney(confidence.actualTotal.amount, confidence.actualTotal.currency)}
        />
        <Metric
          label="Issues"
          value={String(confidence.issues.length)}
        />
      </div>

      {confidence.warnings.length > 0 ? (
        <div className="mt-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          {confidence.warnings.slice(0, 2).join(" ")}
        </div>
      ) : null}

      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <BudgetCoverageBreakdown coverage={confidence.coverage} />
        <PlannedVsActualAccuracyCard plannedVsActual={confidence.plannedVsActual} />
      </div>

      <div className="mt-4 space-y-4">
        <BudgetRiskIssuesList issues={confidence.issues} />
        <BudgetImprovementActions recommendations={confidence.recommendations} />
        <CostSourceQualityTable sources={confidence.sourceQuality} />
      </div>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-sm font-semibold text-slate-950">{value}</p>
    </div>
  );
}
