import { cn } from "@/lib/utils";
import type { ApprovalChecklist as ApprovalChecklistData, ApprovalChecklistItem } from "@/entities/approval/model";

const ITEM_META: Record<ApprovalChecklistItem["status"], { icon: string; className: string; label: string }> = {
  ok: { icon: "✓", className: "text-emerald-600", label: "OK" },
  warning: { icon: "!", className: "text-amber-600", label: "Warning" },
  blocked: { icon: "✕", className: "text-red-600", label: "Blocker" },
  info: { icon: "i", className: "text-slate-500", label: "Info" }
};

export function ApprovalChecklist({
  checklist,
  acknowledgedWarnings,
  className
}: {
  checklist: ApprovalChecklistData;
  acknowledgedWarnings?: string[];
  className?: string;
}) {
  const acknowledged = new Set(acknowledgedWarnings ?? []);
  return (
    <div className={cn("space-y-3", className)}>
      <ul className="space-y-2">
        {checklist.items.map((item) => {
          const meta = ITEM_META[item.status] ?? ITEM_META.info;
          const isAcknowledged = item.status === "warning" && acknowledged.has(item.key);
          return (
            <li key={item.key} className="flex items-start gap-2 text-sm">
              <span
                className={cn("mt-0.5 font-bold", meta.className)}
                aria-label={meta.label}
                title={meta.label}
              >
                {meta.icon}
              </span>
              <span className="flex-1 text-slate-700">
                <span className="font-medium text-slate-900">{item.title}</span>
                {item.message ? <span className="text-slate-500"> — {item.message}</span> : null}
                {item.key === "workspace_policy" ? (
                  <a className="ml-1 text-primary-700 hover:underline" href="#workspace-policy">
                    View policy details
                  </a>
                ) : null}
                {isAcknowledged ? (
                  <span className="ml-1 text-xs text-slate-400">(acknowledged)</span>
                ) : null}
              </span>
            </li>
          );
        })}
      </ul>
      <p className="text-xs text-slate-500">
        Warnings do not block submission, but reviewers will see them.
      </p>
    </div>
  );
}
