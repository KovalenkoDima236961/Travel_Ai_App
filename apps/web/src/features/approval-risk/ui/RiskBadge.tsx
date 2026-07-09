import { cn } from "@/shared/lib/cn";
import type { ApprovalRiskLevel } from "@/entities/approval-risk/model";

const RISK_META: Record<ApprovalRiskLevel, { label: string; mark: string; className: string }> = {
  low: {
    label: "Low risk",
    mark: "Low",
    className: "border-emerald-200 bg-emerald-50 text-emerald-800"
  },
  medium: {
    label: "Medium risk",
    mark: "Med",
    className: "border-amber-200 bg-amber-50 text-amber-800"
  },
  high: {
    label: "High risk",
    mark: "High",
    className: "border-orange-200 bg-orange-50 text-orange-800"
  },
  critical: {
    label: "Critical risk",
    mark: "Crit",
    className: "border-red-200 bg-red-50 text-red-800"
  },
  unknown: {
    label: "Unknown risk",
    mark: "?",
    className: "border-slate-200 bg-slate-50 text-slate-700"
  },
  not_applicable: {
    label: "Risk not applicable",
    mark: "N/A",
    className: "border-slate-200 bg-white text-slate-500"
  }
};

export function RiskBadge({
  status,
  score,
  className
}: {
  status?: ApprovalRiskLevel | null;
  score?: number | null;
  className?: string;
}) {
  const meta = RISK_META[status ?? "unknown"] ?? RISK_META.unknown;
  return (
    <span
      className={cn(
        "inline-flex h-8 items-center gap-1.5 rounded-md border px-2.5 text-xs font-semibold",
        meta.className,
        className
      )}
      title={score != null ? `${meta.label} · ${score}/100` : meta.label}
    >
      <span aria-hidden>{meta.mark}</span>
      <span>{score != null ? `${meta.label} · ${score}/100` : meta.label}</span>
    </span>
  );
}

