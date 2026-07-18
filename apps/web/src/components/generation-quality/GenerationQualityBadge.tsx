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

type GenerationQualityBadgesProps = {
  quality?: GenerationQuality | null;
  source?: string | null;
  className?: string;
};

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

export function GenerationQualityBadges({ quality, source, className }: GenerationQualityBadgesProps) {
  const status = quality?.status;
  const badges: Array<{ label: string; title: string; className: string }> = [];
  if (source !== "manual") {
    badges.push({ label: "AI-generated", title: "AI-generated: created by the planning model.", className: "border-violet-200 bg-violet-50 text-violet-800" });
  }
  if (status === "validated" || status === "repaired_and_validated") {
    badges.push({ label: "Validated", title: "Validated: passed app consistency checks.", className: "border-emerald-200 bg-emerald-50 text-emerald-800" });
  }
  if (status === "repaired_and_validated" || status === "repaired_with_warnings") {
    badges.push({ label: "Repaired", title: "Repaired: app checks found issues that were fixed automatically.", className: "border-blue-200 bg-blue-50 text-blue-800" });
  }
  if (status === "validated_with_warnings" || status === "repaired_with_warnings" || status === "repair_failed" || status === "blocked_by_critical_issues") {
    badges.push({ label: "Needs review", title: "Needs review: some real-world data is missing or stale.", className: "border-amber-200 bg-amber-50 text-amber-800" });
  }
  if (source === "mock" || source === "fallback") {
    badges.push({ label: "Fallback data", title: "Fallback data: a provider was unavailable, so an estimate was used.", className: "border-slate-200 bg-slate-50 text-slate-700" });
  }
  return badges.length > 0 ? (
    <span className={cn("inline-flex flex-wrap items-center gap-1.5", className)}>
      {badges.map((badge) => <span key={badge.label} className={cn("inline-flex rounded-full border px-2 py-0.5 text-[11px] font-semibold", badge.className)} title={badge.title}>{badge.label}</span>)}
    </span>
  ) : null;
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
