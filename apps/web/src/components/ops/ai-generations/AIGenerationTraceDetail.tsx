import Link from "next/link";
import type { AIGenerationTraceDetail as Detail } from "@/entities/ops/model";
import { CARD, CARD_HEADING, OUTLINE_BUTTON } from "@/_pages/ops/ui/opsStyles";
import { AIGenerationContextSummary } from "./AIGenerationContextSummary";
import { AIGenerationSummaryCards } from "./AIGenerationSummaryCards";
import { AIGenerationTimeline } from "./AIGenerationTimeline";
import { PromptSnapshotViewer } from "./PromptSnapshotViewer";
import { RAGSummaryPanel } from "./RAGSummaryPanel";
import { RepairAttemptsPanel } from "./RepairAttemptsPanel";
import { SafeErrorDetails } from "./SafeErrorDetails";
import { ValidationSummaryPanel } from "./ValidationSummaryPanel";

export function AIGenerationTraceDetail({ detail }: { detail: Detail }) { const { trace } = detail; return <div className="space-y-6"><div className="flex flex-wrap items-start justify-between gap-4"><div><Link className={OUTLINE_BUTTON} href="/ops/ai-generations">Back to AI Generations</Link><h1 className="mt-5 font-newsreader text-[34px] font-medium tracking-[-0.02em] text-cocoa-900">AI generation trace</h1><p className="mt-2 font-mono text-xs text-cocoa-400">{trace.id}</p></div></div><AIGenerationSummaryCards trace={trace} /><section className={CARD}><h2 className={CARD_HEADING}>Timeline</h2><div className="mt-5"><AIGenerationTimeline events={detail.events} /></div></section><section className={CARD}><h2 className={CARD_HEADING}>Safe context</h2><div className="mt-5 grid gap-7 xl:grid-cols-2"><AIGenerationContextSummary title="Input summary" value={trace.inputSummary} /><AIGenerationContextSummary title="Constraints summary" value={trace.constraintsSummary} /><RAGSummaryPanel value={trace.ragSummary} /><AIGenerationContextSummary title="Prompt summary" value={trace.promptSummary} /></div></section><section className={CARD}><h2 className={CARD_HEADING}>Validation & repair</h2><div className="mt-5 grid gap-7 xl:grid-cols-2"><ValidationSummaryPanel value={trace.validationSummary} /><RepairAttemptsPanel value={trace.repairSummary} /></div></section><section className={CARD}><h2 className={CARD_HEADING}>Output</h2><div className="mt-5"><AIGenerationContextSummary title="Output summary" value={trace.outputSummary} /></div></section><SafeErrorDetails trace={trace} /><section className={CARD}><PromptSnapshotViewer snapshot={detail.promptSnapshot} /></section></div>; }
