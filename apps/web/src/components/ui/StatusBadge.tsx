import { cn } from "@/shared/lib/cn";

export type StatusBadgeTone = "neutral" | "success" | "warning" | "danger" | "info";

const TONE_CLASSES: Record<StatusBadgeTone, string> = {
  neutral: "border-slate-200 bg-slate-100 text-slate-700",
  success: "border-emerald-200 bg-emerald-50 text-emerald-800",
  warning: "border-amber-200 bg-amber-50 text-amber-900",
  danger: "border-red-200 bg-red-50 text-red-800",
  info: "border-blue-200 bg-blue-50 text-blue-800"
};

export function StatusBadge({
  label,
  tone = "neutral",
  ariaLabel,
  className
}: {
  label: string;
  tone?: StatusBadgeTone;
  ariaLabel?: string;
  className?: string;
}) {
  return (
    <span
      aria-label={ariaLabel}
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold",
        TONE_CLASSES[tone],
        className
      )}
    >
      {label}
    </span>
  );
}
