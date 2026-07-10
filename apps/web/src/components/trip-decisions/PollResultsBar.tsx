import type { PollOptionResult } from "@/types/trip-decisions";

export function PollResultsBar({ result }: { result: PollOptionResult }) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between gap-3 text-[12px] font-semibold text-cocoa-500">
        <span className="truncate">{result.label}</span>
        <span>
          {result.voteCount} {result.voteCount === 1 ? "vote" : "votes"} · {result.percentage}%
        </span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-sand-200">
        <div
          className="h-full rounded-full bg-clay"
          style={{ width: `${Math.min(100, Math.max(0, result.percentage))}%` }}
        />
      </div>
      {result.averageRating != null ? (
        <p className="text-[12px] text-cocoa-400">
          Average rating {result.averageRating.toFixed(1)}
        </p>
      ) : null}
    </div>
  );
}
