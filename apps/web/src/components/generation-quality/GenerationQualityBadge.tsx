import { cn } from "@/shared/lib/cn";
import type { GenerationQuality, GenerationQualityStatus } from "@/types/generation-quality";

type GenerationQualityBadgeProps = {
  quality?: GenerationQuality | null;
  status?: GenerationQualityStatus;
  className?: string;
};

export function GenerationQualityBadge({
  quality,
  status,
  className
}: GenerationQualityBadgeProps) {
  const value = status ?? quality?.status;
  if (!value || value === "not_validated") {
    return null;
  }

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium",
        badgeClassName(value),
        className
      )}
    >
      {badgeLabel(value, quality)}
    </span>
  );
}

function badgeLabel(status: GenerationQualityStatus, quality?: GenerationQuality | null) {
  const warningCount = quality?.warningIssueCount ?? 0;
  const highCount = quality?.highIssueCount ?? 0;
  const criticalCount = quality?.criticalIssueCount ?? 0;

  switch (status) {
    case "validated":
      return "Validated";
    case "validated_with_warnings":
      return warningCount > 0 ? `Validated, ${warningCount} warning(s)` : "Validated with warnings";
    case "repaired_and_validated":
      return "Repaired and validated";
    case "repaired_with_warnings":
      return warningCount > 0 ? `Repaired, ${warningCount} warning(s)` : "Repaired with warnings";
    case "repair_failed":
      return "Repair failed";
    case "schema_invalid":
      return "Schema invalid";
    case "blocked_by_policy":
      return "Blocked by policy";
    case "blocked_by_critical_issues":
      return criticalCount + highCount > 0
        ? `Blocked, ${criticalCount + highCount} issue(s)`
        : "Blocked by validation";
    case "ai_output_invalid":
      return "AI output invalid";
    case "not_validated":
      return "Not validated";
  }
}

function badgeClassName(status: GenerationQualityStatus) {
  switch (status) {
    case "validated":
    case "repaired_and_validated":
      return "border-emerald-200 bg-emerald-50 text-emerald-800";
    case "validated_with_warnings":
    case "repaired_with_warnings":
      return "border-amber-200 bg-amber-50 text-amber-800";
    case "repair_failed":
    case "schema_invalid":
    case "blocked_by_policy":
    case "blocked_by_critical_issues":
    case "ai_output_invalid":
      return "border-red-200 bg-red-50 text-red-800";
    case "not_validated":
      return "border-slate-200 bg-slate-50 text-slate-700";
  }
}
