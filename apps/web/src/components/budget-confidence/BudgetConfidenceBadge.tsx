import { cn } from "@/shared/lib/cn";
import type {
  BudgetConfidenceLevel,
  BudgetRiskLevel
} from "@/types/budget-confidence";

type BudgetConfidenceBadgeProps = {
  level?: BudgetConfidenceLevel | null;
  riskLevel?: BudgetRiskLevel | null;
  className?: string;
};

export function BudgetConfidenceBadge({
  level,
  riskLevel,
  className
}: BudgetConfidenceBadgeProps) {
  const value = level ?? riskLevel ?? "low";
  return (
    <span
      className={cn(
        "inline-flex h-7 items-center rounded-full border px-2.5 text-xs font-semibold capitalize",
        level ? levelClasses(level) : riskClasses(riskLevel ?? "low"),
        className
      )}
    >
      {formatLabel(value)}
    </span>
  );
}

function levelClasses(value: BudgetConfidenceLevel) {
  switch (value) {
    case "very_high":
    case "high":
      return "border-emerald-200 bg-emerald-50 text-emerald-700";
    case "medium":
      return "border-amber-200 bg-amber-50 text-amber-800";
    case "very_low":
      return "border-red-200 bg-red-50 text-red-700";
    case "low":
    default:
      return "border-orange-200 bg-orange-50 text-orange-700";
  }
}

function riskClasses(value: BudgetRiskLevel) {
  switch (value) {
    case "low":
      return "border-emerald-200 bg-emerald-50 text-emerald-700";
    case "medium":
      return "border-amber-200 bg-amber-50 text-amber-800";
    case "critical":
      return "border-red-200 bg-red-50 text-red-700";
    case "high":
    default:
      return "border-orange-200 bg-orange-50 text-orange-700";
  }
}

function formatLabel(value: string) {
  return value.replaceAll("_", " ");
}
