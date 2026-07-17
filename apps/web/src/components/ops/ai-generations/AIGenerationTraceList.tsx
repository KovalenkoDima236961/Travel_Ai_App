import Link from "next/link";

import type { AIGenerationTrace } from "@/entities/ops/model";
import { formatOpsDate } from "@/_pages/ops/model/opsPageModel";
import { MONO, SMALL_OUTLINE_BUTTON } from "@/_pages/ops/ui/opsStyles";
import { cn } from "@/shared/lib/cn";
import { AIGenerationQualityBadge } from "./AIGenerationQualityBadge";
import { AIGenerationStatusBadge } from "./AIGenerationStatusBadge";

function duration(value?: number | null) { return value == null ? "—" : `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}s`; }

export function AIGenerationTraceList({ items }: { items: AIGenerationTrace[] }) {
  return <div className="overflow-x-auto rounded-[14px] border border-sand-200"><table className="min-w-full text-left"><thead><tr className="bg-sand-50 text-[11.5px] uppercase tracking-[0.04em] text-[#A08D78]"><th className="px-4 py-3 font-semibold">Created</th><th className="px-4 py-3 font-semibold">Generation</th><th className="px-4 py-3 font-semibold">Status</th><th className="px-4 py-3 font-semibold">Quality</th><th className="px-4 py-3 font-semibold">Provider</th><th className="px-4 py-3 font-semibold">Prompt</th><th className="px-4 py-3 font-semibold">Duration</th><th className="px-4 py-3 font-semibold">Error</th><th className="px-4 py-3" /></tr></thead><tbody>{items.map((trace) => <tr key={trace.id} className="border-t border-sand-200"><td className="px-4 py-3 text-[12.5px] text-cocoa-400">{formatOpsDate(trace.createdAt)}</td><td className="px-4 py-3 text-[13px] text-cocoa-900">{trace.generationType}</td><td className="px-4 py-3"><AIGenerationStatusBadge status={trace.status} /></td><td className="px-4 py-3"><AIGenerationQualityBadge status={trace.qualityStatus} /></td><td className="px-4 py-3 text-[12.5px] text-cocoa-500">{trace.provider}{trace.model ? ` · ${trace.model}` : ""}</td><td className={cn("px-4 py-3 text-[12px] text-cocoa-500", MONO)}>{trace.promptVersion ?? "—"}</td><td className="px-4 py-3 text-[12.5px] text-cocoa-500">{duration(trace.durationMs)}</td><td className={cn("px-4 py-3 text-[12px] text-cocoa-500", MONO)}>{trace.errorCode ?? "—"}</td><td className="px-4 py-3 text-right"><Link className={SMALL_OUTLINE_BUTTON} href={`/ops/ai-generations/${trace.id}`}>Open</Link></td></tr>)}{items.length === 0 ? <tr><td colSpan={9} className="px-4 py-8 text-center text-[13px] text-cocoa-400">No AI generation traces found.</td></tr> : null}</tbody></table></div>;
}
