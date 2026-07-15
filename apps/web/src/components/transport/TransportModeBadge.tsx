import { transportModeLabel } from "@/components/routes/route-options";
import { cn } from "@/shared/lib/cn";

type Props = {
  mode?: string | null;
  className?: string;
};

export function TransportModeBadge({ mode, className }: Props) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-md bg-sand-200 px-2.5 py-1 text-[12px] font-semibold text-cocoa-600",
        className
      )}
    >
      {transportModeLabel(mode)}
    </span>
  );
}
