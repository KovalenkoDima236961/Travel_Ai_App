import type { ProviderQuotaStatus } from "@/entities/ops/model";

/**
 * Presentational tokens for the Ops Dashboard restyle (Ops.dc.html). The warm
 * editorial palette lives in tailwind.config.ts (clay/sand/cocoa); the status
 * pill / tile hexes below are one-off dashboard signal colors from the design
 * (green #2F7A57, blue #4E6E86, gold #96682A, red #B3402E) that don't map to a
 * token, so they stay as arbitrary values.
 */

// JetBrains Mono is loaded for this slice only; the variable is applied on the
// Ops root wrapper, so this class resolves to the mono face inside Ops and to
// next/font's fallback everywhere the var is defined.
export const MONO = "font-[family-name:var(--font-jetbrains-mono)]";

export const CARD = "rounded-[20px] border border-sand-300 bg-white px-6 py-6 sm:px-[26px]";
export const CARD_HEADING = "font-newsreader text-[20px] font-semibold text-cocoa-900";
export const MICRO_LABEL =
  "text-[11.5px] font-semibold uppercase tracking-[0.06em] text-[#A08D78]";

export const OUTLINE_BUTTON =
  "inline-flex h-[38px] items-center justify-center gap-2 rounded-full border border-sand-400 bg-white px-4 text-[13.5px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60";

export const SMALL_OUTLINE_BUTTON =
  "inline-flex h-[34px] items-center justify-center rounded-full border border-sand-400 bg-white px-3.5 text-[12.5px] font-semibold text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60";

export const SMALL_DANGER_BUTTON =
  "inline-flex h-[34px] items-center justify-center rounded-full border border-[#E5C3B6] bg-[#FBF0EB] px-3.5 text-[12.5px] font-semibold text-[#B3402E] transition hover:bg-[#F7E4DB] disabled:cursor-not-allowed disabled:opacity-60";

export const OPS_INPUT =
  "h-[38px] w-full rounded-[10px] border border-sand-400 bg-[#FFFDFA] px-3 text-[13px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint";
export const OPS_SELECT = OPS_INPUT;

const PILL_BASE =
  "inline-flex items-center rounded-full px-2.5 py-[3px] text-[11.5px] font-semibold";

export function statusPillClass(status?: string) {
  const tone =
    status === "failed" || status === "down"
      ? "bg-[#FBF0EB] text-[#B3402E]"
      : status === "running"
        ? "bg-[#EAF0F4] text-[#4E6E86]"
        : status === "degraded"
          ? "bg-[#FAEFDA] text-[#96682A]"
          : status === "completed" || status === "healthy" || status === "operational"
            ? "bg-[#EDF3EA] text-[#2F7A57]"
            : "bg-[#F4EDE4] text-[#8A7A6A]";
  return `${PILL_BASE} ${tone}`;
}

export function quotaPillClass(status: ProviderQuotaStatus) {
  const tone =
    status === "quota_exceeded"
      ? "bg-[#FBF0EB] text-[#B3402E]"
      : status === "rate_limited_recently" || status === "nearing_quota"
        ? "bg-[#FAEFDA] text-[#96682A]"
        : status === "healthy"
          ? "bg-[#EDF3EA] text-[#2F7A57]"
          : "bg-[#F4EDE4] text-[#8A7A6A]";
  return `${PILL_BASE} ${tone}`;
}
