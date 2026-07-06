import { cn } from "@/lib/utils";
import type { ApprovalStatus } from "@/types/approval";

const STATUS_META: Record<
  ApprovalStatus,
  { label: string; className: string; icon: string }
> = {
  not_required: {
    label: "Not required",
    className: "bg-slate-100 text-slate-600 border-slate-200",
    icon: "—"
  },
  draft: {
    label: "Draft",
    className: "bg-slate-100 text-slate-700 border-slate-300",
    icon: "✎"
  },
  pending_approval: {
    label: "Pending approval",
    className: "bg-amber-100 text-amber-800 border-amber-300",
    icon: "⏳"
  },
  changes_requested: {
    label: "Changes requested",
    className: "bg-orange-100 text-orange-800 border-orange-300",
    icon: "↺"
  },
  approved: {
    label: "Approved",
    className: "bg-emerald-100 text-emerald-800 border-emerald-300",
    icon: "✓"
  },
  cancelled: {
    label: "Cancelled",
    className: "bg-slate-100 text-slate-500 border-slate-300",
    icon: "⊘"
  }
};

export function ApprovalStatusBadge({
  status,
  className
}: {
  status: ApprovalStatus;
  className?: string;
}) {
  const meta = STATUS_META[status] ?? STATUS_META.draft;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium",
        meta.className,
        className
      )}
    >
      <span aria-hidden>{meta.icon}</span>
      {meta.label}
    </span>
  );
}
