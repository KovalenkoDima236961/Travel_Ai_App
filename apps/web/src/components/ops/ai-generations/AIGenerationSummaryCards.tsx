import type { AIGenerationTrace } from "@/entities/ops/model";
import { AIGenerationQualityBadge } from "./AIGenerationQualityBadge";
import { AIGenerationStatusBadge } from "./AIGenerationStatusBadge";

const duration = (value?: number | null) => value == null ? "—" : `${(value / 1000).toFixed(1)}s`;

export function AIGenerationSummaryCards({ trace }: { trace: AIGenerationTrace }) {
  const cards = [["Status", <AIGenerationStatusBadge key="status" status={trace.status} />], ["Quality", <AIGenerationQualityBadge key="quality" status={trace.qualityStatus} />], ["Provider", `${trace.provider}${trace.model ? ` · ${trace.model}` : ""}`], ["Duration", duration(trace.durationMs)], ["Prompt", trace.promptVersion ?? "—"], ["Queue wait", duration(trace.queueWaitMs)]] as const;
  return <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">{cards.map(([label, value]) => <div key={label} className="rounded-2xl border border-sand-300 bg-white px-5 py-4"><p className="text-[11.5px] font-semibold uppercase tracking-[0.06em] text-[#A08D78]">{label}</p><div className="mt-2 text-[13px] font-semibold text-cocoa-800">{value}</div></div>)}</section>;
}
