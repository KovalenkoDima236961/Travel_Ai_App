import { cn } from "@/shared/lib/cn";
import type { TripLifecycle } from "@/types/library";

const styles: Record<TripLifecycle, string> = {
  draft: "bg-sand-200 text-cocoa-600",
  planning: "bg-[#FFF0D9] text-[#9A651D]",
  ready: "bg-[#E4F3E8] text-[#2F7A57]",
  active: "bg-[#E0EEF9] text-[#27638E]",
  completed: "bg-[#EEEAF7] text-[#65528E]",
  archived: "bg-sand-200 text-cocoa-500"
};

export function TripLifecycleBadge({ lifecycle }: { lifecycle: TripLifecycle }) {
  return <span className={cn("inline-flex rounded-full px-2.5 py-1 text-xs font-semibold capitalize", styles[lifecycle])}>{lifecycle}</span>;
}
