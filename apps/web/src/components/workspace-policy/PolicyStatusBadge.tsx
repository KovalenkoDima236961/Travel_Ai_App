import { cn } from "@/lib/utils";
import type { PolicyEvaluationStatus } from "@/types/workspace-policy";

const META: Record<PolicyEvaluationStatus, { label: string; className: string }> = {
  ok: { label: "Policy OK", className: "bg-emerald-100 text-emerald-800" },
  info: { label: "Policy info", className: "bg-sky-100 text-sky-800" },
  warning: { label: "Policy warning", className: "bg-amber-100 text-amber-900" },
  blocking: { label: "Policy blocking", className: "bg-red-100 text-red-800" },
  not_applicable: { label: "No policy", className: "bg-slate-100 text-slate-700" }
};

export function PolicyStatusBadge({
  status,
  className
}: {
  status: PolicyEvaluationStatus;
  className?: string;
}) {
  const meta = META[status];
  return (
    <span
      className={cn(
        "inline-flex rounded-full px-2.5 py-1 text-xs font-semibold",
        meta.className,
        className
      )}
    >
      {meta.label}
    </span>
  );
}
