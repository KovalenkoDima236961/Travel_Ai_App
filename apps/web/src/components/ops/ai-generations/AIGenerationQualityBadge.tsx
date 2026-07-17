import { cn } from "@/shared/lib/cn";

export function AIGenerationQualityBadge({ status }: { status?: string | null }) {
  const value = status?.replace(/_/g, " ") ?? "not validated";
  const tone = status?.includes("blocked") || status === "repair_failed" || status === "schema_invalid"
    ? "bg-[#FBF0EB] text-[#B3402E]"
    : status?.includes("warnings")
      ? "bg-[#FAEFDA] text-[#96682A]"
      : status
        ? "bg-[#EDF3EA] text-[#2F7A57]"
        : "bg-[#F4EDE4] text-[#8A7A6A]";
  return <span className={cn("inline-flex rounded-full px-2.5 py-[3px] text-[11.5px] font-semibold", tone)}>{value}</span>;
}
