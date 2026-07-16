import { cn } from "@/shared/lib/cn";

type CardSkeletonProps = {
  lines?: number;
  className?: string;
  compact?: boolean;
};

export function CardSkeleton({ lines = 3, className, compact = false }: CardSkeletonProps) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "animate-pulse rounded-lg border border-slate-200 bg-white shadow-soft",
        compact ? "p-4" : "p-5",
        className
      )}
    >
      <div className="h-3 w-24 rounded-full bg-slate-200" />
      <div className="mt-3 h-6 w-2/3 rounded-full bg-slate-200" />
      <div className={compact ? "mt-3 space-y-2" : "mt-5 space-y-3"}>
        {Array.from({ length: lines }, (_, index) => (
          <div
            className="h-3 rounded-full bg-slate-100"
            key={index}
            style={{ width: `${Math.max(48, 100 - index * 13)}%` }}
          />
        ))}
      </div>
    </div>
  );
}
