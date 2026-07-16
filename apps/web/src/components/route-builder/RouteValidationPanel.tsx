import { useTranslations } from "next-intl";
import type { RouteBuilderIssue } from "@/lib/route-builder/route-validation";
import { RouteValidationIssueCard } from "./RouteValidationIssueCard";

type RouteValidationPanelProps = {
  issues: RouteBuilderIssue[];
  onAction?: (issue: RouteBuilderIssue) => void;
};

export function RouteValidationPanel({ issues, onAction }: RouteValidationPanelProps) {
  const t = useTranslations("route");
  const sorted = [...issues].sort((left, right) => severityRank(right.severity) - severityRank(left.severity));
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">{t("validation")}</p>
          <h3 className="mt-1 font-newsreader text-[21px] font-semibold text-cocoa-900">
            {issues.length === 0 ? t("routeLooksGood") : t("issueCount", { count: issues.length })}
          </h3>
        </div>
        {issues.length === 0 ? (
          <span className="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-[12px] font-semibold text-emerald-700">
            ✓ {t("checksPassed")}
          </span>
        ) : null}
      </div>
      {issues.length > 0 ? (
        <div className="mt-4 grid gap-2 lg:grid-cols-2">
          {sorted.map((issue) => <RouteValidationIssueCard issue={issue} key={issue.id} onAction={onAction} />)}
        </div>
      ) : (
        <p className="mt-2 text-[13px] text-cocoa-500">{t("validationAdvisory")}</p>
      )}
    </section>
  );
}

function severityRank(severity: RouteBuilderIssue["severity"]) {
  return severity === "error" ? 3 : severity === "warning" ? 2 : 1;
}
