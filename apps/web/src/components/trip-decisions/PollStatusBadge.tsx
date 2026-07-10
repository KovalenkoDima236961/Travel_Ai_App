import type { TripPollStatus } from "@/types/trip-decisions";

const LABELS: Record<TripPollStatus, string> = {
  open: "Open",
  closed: "Closed",
  archived: "Archived"
};

export function PollStatusBadge({ status }: { status: TripPollStatus }) {
  const color =
    status === "open"
      ? "border-[#C9E4D0] bg-[#F2F7F1] text-[#38543F]"
      : status === "closed"
        ? "border-sand-300 bg-sand-100 text-cocoa-500"
        : "border-slate-200 bg-slate-100 text-slate-500";

  return (
    <span className={`rounded-full border px-2.5 py-1 text-[12px] font-semibold ${color}`}>
      {LABELS[status]}
    </span>
  );
}
