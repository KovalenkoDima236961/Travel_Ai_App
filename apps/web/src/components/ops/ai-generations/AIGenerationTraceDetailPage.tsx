"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "@/shared/api/client";
import { getAIGenerationTrace, opsKeys } from "@/lib/api/ops";
import { invalidateOps } from "@/_pages/ops/model/opsPageModel";
import { instrumentSans, jetBrainsMono, newsreader } from "@/_pages/ops/ui/fonts";
import { OpsHeader } from "@/_pages/ops/ui/OpsHeader";
import { CARD, CARD_HEADING } from "@/_pages/ops/ui/opsStyles";
import { cn } from "@/shared/lib/cn";
import { AIGenerationTraceDetail } from "./AIGenerationTraceDetail";

export function AIGenerationTraceDetailPage({ traceId }: { traceId: string }) {
  const queryClient = useQueryClient();
  const trace = useQuery({ queryKey: opsKeys.aiGenerationDetail(traceId), queryFn: () => getAIGenerationTrace(traceId), staleTime: 10_000 });
  const denied = trace.error instanceof ApiError && trace.error.status === 403;
  const missing = trace.error instanceof ApiError && [404, 503].includes(trace.error.status);
  return <div className={cn(newsreader.variable, instrumentSans.variable, jetBrainsMono.variable, "min-h-screen bg-sand-50 font-instrument text-cocoa-700")}><OpsHeader onRefresh={() => invalidateOps(queryClient)} /><div className="mx-auto max-w-[1360px] px-6 pb-[72px] pt-8 sm:px-10">{denied ? <section className={CARD}><h1 className={CARD_HEADING}>Ops access required</h1><p className="mt-2 text-[14px] text-cocoa-500">Your account is not authorized to view AI generation traces.</p></section> : missing ? <section className={CARD}><h1 className={CARD_HEADING}>Trace details unavailable or expired</h1></section> : trace.error ? <section className={CARD}><h1 className={CARD_HEADING}>Could not load trace details</h1><p className="mt-2 text-[14px] text-cocoa-500">Use Refresh to retry.</p></section> : trace.data ? <AIGenerationTraceDetail detail={trace.data} /> : <p className="text-[14px] text-cocoa-400">Loading trace…</p>}</div></div>;
}
