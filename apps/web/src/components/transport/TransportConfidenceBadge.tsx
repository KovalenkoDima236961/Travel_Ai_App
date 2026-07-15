import { cn } from "@/shared/lib/cn";
import type { TransportConfidence } from "@/types/transport";

type Props = {
  confidence?: TransportConfidence | string | null;
  className?: string;
};

const styles: Record<string, string> = {
  high: "border-emerald-200 bg-emerald-50 text-emerald-700",
  medium: "border-amber-200 bg-amber-50 text-amber-700",
  low: "border-slate-200 bg-slate-50 text-slate-600"
};

export function TransportConfidenceBadge({ confidence, className }: Props) {
  const normalized = confidence ?? "low";
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-md border px-2 py-0.5 text-[12px] font-semibold",
        styles[normalized] ?? styles.low,
        className
      )}
    >
      {normalized} confidence
    </span>
  );
}
