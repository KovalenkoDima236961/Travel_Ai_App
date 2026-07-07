import type { TripStatus } from "@/entities/trip/model";
import { cn } from "@/lib/utils";

type TripStatusBadgeProps = {
  status: TripStatus;
};

const badgeClasses: Record<TripStatus, string> = {
  DRAFT: "border-slate-200 bg-slate-100 text-slate-700",
  PROCESSING: "border-amber-200 bg-amber-100 text-amber-800",
  COMPLETED: "border-emerald-200 bg-emerald-100 text-emerald-800",
  FAILED: "border-red-200 bg-red-100 text-red-800"
};

export function TripStatusBadge({ status }: TripStatusBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold",
        badgeClasses[status]
      )}
    >
      {status}
    </span>
  );
}
