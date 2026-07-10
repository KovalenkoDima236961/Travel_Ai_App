"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { PlanningConstraintIssuesList } from "@/components/planning-constraints/PlanningConstraintIssuesList";
import { PlanningConstraintsSummaryCard } from "@/components/planning-constraints/PlanningConstraintsSummaryCard";
import type { PlanningConstraintsPreviewResponse } from "@/types/planning-constraints";

type Props = {
  preview?: PlanningConstraintsPreviewResponse | null;
  isLoading?: boolean;
  error?: string | null;
  onPreview?: () => void;
};

export function PlanningConstraintsPreviewPanel({
  preview,
  isLoading = false,
  error,
  onPreview
}: Props) {
  const t = useTranslations("planningConstraints");
  const issues = preview ? [...preview.blockers, ...preview.warnings] : [];

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-slate-900">{t("title")}</h2>
        {onPreview ? (
          <Button disabled={isLoading} size="sm" variant="secondary" onClick={onPreview}>
            {isLoading ? t("previewing") : t("previewButton")}
          </Button>
        ) : null}
      </div>
      {error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          {error}
        </div>
      ) : null}
      {preview ? (
        <>
          <PlanningConstraintsSummaryCard summary={preview.summary} />
          <PlanningConstraintIssuesList issues={issues} />
        </>
      ) : (
        <p className="text-sm text-slate-500">{t("emptyPreview")}</p>
      )}
    </section>
  );
}
