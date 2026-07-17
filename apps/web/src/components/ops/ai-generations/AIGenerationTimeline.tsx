import type { AIGenerationTraceEvent } from "@/entities/ops/model";
import { formatOpsDate } from "@/_pages/ops/model/opsPageModel";

export function AIGenerationTimeline({ events }: { events: AIGenerationTraceEvent[] }) {
  return <ol className="space-y-3 border-l border-sand-300 pl-5">{events.map((event) => <li key={event.id} className="relative"><span className="absolute -left-[25px] top-1.5 h-2.5 w-2.5 rounded-full bg-clay" /><div className="flex flex-wrap items-baseline justify-between gap-2"><p className="text-[13.5px] font-semibold text-cocoa-800">{event.title}</p><time className="text-[12px] text-cocoa-400">{formatOpsDate(event.createdAt)}</time></div>{event.message ? <p className="mt-1 text-[13px] text-cocoa-500">{event.message}</p> : null}</li>)}{events.length === 0 ? <li className="text-[13px] text-cocoa-400">No timeline events were retained.</li> : null}</ol>;
}
