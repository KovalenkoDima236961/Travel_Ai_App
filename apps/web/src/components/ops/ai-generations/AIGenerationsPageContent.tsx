"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useDocumentVisibility } from "@/hooks/useDocumentVisibility";
import { ApiError } from "@/shared/api/client";
import { getAIGenerationTraces, opsKeys, type AIGenerationTraceFilters as TraceFilters } from "@/lib/api/ops";
import { invalidateOps } from "@/_pages/ops/model/opsPageModel";
import { instrumentSans, jetBrainsMono, newsreader } from "@/_pages/ops/ui/fonts";
import { OpsHeader } from "@/_pages/ops/ui/OpsHeader";
import { CARD, CARD_HEADING } from "@/_pages/ops/ui/opsStyles";
import { cn } from "@/shared/lib/cn";
import { AIGenerationTraceFilters } from "./AIGenerationTraceFilters";
import { AIGenerationTraceList } from "./AIGenerationTraceList";

export function AIGenerationsPageContent() {
  const queryClient = useQueryClient();
  const documentVisible = useDocumentVisibility();
  const [filters, setFilters] = useState<TraceFilters>({ limit: 50 });
  const traces = useQuery({ queryKey: opsKeys.aiGenerations(filters), queryFn: () => getAIGenerationTraces(filters), staleTime: 10_000, refetchInterval: documentVisible ? 20_000 : false, refetchIntervalInBackground: false });
  const denied = traces.error instanceof ApiError && traces.error.status === 403;
  const unavailable = traces.error instanceof ApiError && (traces.error.status === 404 || traces.error.status === 503);
  return <div className={cn(newsreader.variable, instrumentSans.variable, jetBrainsMono.variable, "min-h-screen bg-sand-50 font-instrument text-cocoa-700")}><OpsHeader onRefresh={() => invalidateOps(queryClient)} /><div className="mx-auto max-w-[1360px] px-6 pb-[72px] pt-8 sm:px-10"><h1 className="font-newsreader text-[34px] font-medium tracking-[-0.02em] text-cocoa-900">AI Generations</h1><p className="mt-2 text-[14.5px] text-cocoa-400">Safe traces for diagnosing generation quality, validation, repairs, and failures.</p>{denied ? <section className={cn(CARD, "mt-7")}><h2 className={CARD_HEADING}>Ops access required</h2><p className="mt-2 text-[14px] text-cocoa-500">Your account is not authorized to view AI generation traces.</p></section> : unavailable ? <section className={cn(CARD, "mt-7")}><h2 className={CARD_HEADING}>Trace details unavailable or expired</h2><p className="mt-2 text-[14px] text-cocoa-500">AI observability is not enabled for this environment.</p></section> : <section className={cn(CARD, "mt-7")}><AIGenerationTraceFilters filters={filters} onChange={setFilters} /><div className="mt-5">{traces.error ? <p className="text-[13px] text-[#B3402E]">Could not load traces. Use Refresh to retry.</p> : <AIGenerationTraceList items={traces.data?.items ?? []} />}</div></section>}</div></div>;
}
