"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { useApplyRouteAlternative } from "@/hooks/useApplyRouteAlternative";
import { useCreateTripFromRouteAlternative } from "@/hooks/useCreateTripFromRouteAlternative";
import { useRefineRouteAlternatives } from "@/hooks/useRefineRouteAlternatives";
import { useSuggestRouteAlternatives } from "@/hooks/useSuggestRouteAlternatives";
import { useSuggestTripRouteAlternatives } from "@/hooks/useSuggestTripRouteAlternatives";
import { useFeatureFlag } from "@/lib/feature-flags/useFeatureFlags";
import { getErrorMessage } from "@/lib/utils";
import type { Trip } from "@/entities/trip/model";
import type {
  RouteAlternative,
  RouteAlternativeSession,
  SuggestRouteAlternativesInput
} from "@/types/route-alternatives";
import { ApplyRouteAlternativeDialog } from "./ApplyRouteAlternativeDialog";
import { CreateRoutePollButton } from "./CreateRoutePollButton";
import { CreateTripFromRouteAlternativeDialog } from "./CreateTripFromRouteAlternativeDialog";
import { RouteAlternativeCard } from "./RouteAlternativeCard";
import { RouteAlternativeComparisonTable } from "./RouteAlternativeComparisonTable";
import { RouteAlternativeRefineBar } from "./RouteAlternativeRefineBar";
import { RouteAlternativeVoteControls } from "./RouteAlternativeVoteControls";

type RouteAlternativesPanelProps = {
  trip?: Trip | null;
  preTripDefaults?: Partial<Omit<SuggestRouteAlternativesInput, "prompt">>;
  defaultPrompt?: string;
  canCreateTrip?: boolean;
  canApply?: boolean;
  canCreatePoll?: boolean;
  onTripCreated?: (trip: Trip) => void;
  onRouteApplied?: (trip: Trip) => void;
  className?: string;
};

export function RouteAlternativesPanel({
  trip = null,
  preTripDefaults,
  defaultPrompt = "",
  canCreateTrip = false,
  canApply = false,
  canCreatePoll = false,
  onTripCreated,
  onRouteApplied,
  className = ""
}: RouteAlternativesPanelProps) {
  const routeAlternativesEnabled = useFeatureFlag("route_alternatives_enabled");
  const t = useTranslations("routeAlternatives");
  const { editableWorkspaces } = useWorkspaces();
  const [prompt, setPrompt] = useState(defaultPrompt);
  const [session, setSession] = useState<RouteAlternativeSession | null>(null);
  const [selected, setSelected] = useState<RouteAlternative | null>(null);
  const [createTripAlternative, setCreateTripAlternative] = useState<RouteAlternative | null>(null);
  const [applyAlternative, setApplyAlternative] = useState<RouteAlternative | null>(null);
  const suggestPreTripMutation = useSuggestRouteAlternatives();
  const suggestTripMutation = useSuggestTripRouteAlternatives(trip?.id);
  const refineMutation = useRefineRouteAlternatives(session?.id);
  const createTripMutation = useCreateTripFromRouteAlternative(session?.id, createTripAlternative?.id);
  const applyMutation = useApplyRouteAlternative(trip?.id, session?.id, applyAlternative?.id);

  const alternatives = session?.alternatives ?? [];
  const activeError =
    suggestPreTripMutation.error ??
    suggestTripMutation.error ??
    refineMutation.error ??
    null;
  const isGenerating =
    suggestPreTripMutation.isPending || suggestTripMutation.isPending || refineMutation.isPending;
  const initialPrompt = useMemo(() => {
    if (defaultPrompt.trim()) {
      return defaultPrompt.trim();
    }
    if (trip) {
      return "Find better route alternatives for this trip.";
    }
    return "A 5-day Austria trip with nature, old towns, and train travel.";
  }, [defaultPrompt, trip]);

  if (!routeAlternativesEnabled) return null;

  async function generate() {
    const nextPrompt = prompt.trim() || initialPrompt;
    if (trip?.id) {
      const nextSession = await suggestTripMutation.mutateAsync({
        prompt: nextPrompt,
        suggestionCount: 3,
        useCurrentRouteAsBaseline: true
      });
      acceptSession(nextSession);
      return;
    }
    const nextSession = await suggestPreTripMutation.mutateAsync({
      durationDays: 5,
      travelers: 2,
      outputLanguage: "en",
      suggestionCount: 3,
      ...preTripDefaults,
      prompt: nextPrompt
    });
    acceptSession(nextSession);
  }

  function acceptSession(nextSession: RouteAlternativeSession) {
    setSession(nextSession);
    setSelected(nextSession.alternatives[0] ?? null);
  }

  async function refine(instruction: string, alternative?: RouteAlternative) {
    const nextSession = await refineMutation.mutateAsync({
      instruction,
      selectedAlternativeId: alternative?.id
    });
    acceptSession(nextSession);
  }

  async function createTrip(input: Parameters<typeof createTripMutation.mutateAsync>[0]) {
    const result = await createTripMutation.mutateAsync(input);
    setCreateTripAlternative(null);
    onTripCreated?.(result.trip);
  }

  async function applyRoute(input: Parameters<typeof applyMutation.mutateAsync>[0]) {
    const updated = await applyMutation.mutateAsync(input);
    setApplyAlternative(null);
    onRouteApplied?.(updated);
  }

  return (
    <section className={`space-y-5 rounded-[18px] border border-sand-300 bg-white p-5 ${className}`}>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-clay">
            {t("eyebrow")}
          </p>
          <h2 className="mt-1 font-newsreader text-[28px] font-semibold text-cocoa-900">
            {t("title")}
          </h2>
        </div>
        {trip?.id && canCreatePoll ? (
          <CreateRoutePollButton tripId={trip.id} session={session} disabled={isGenerating} />
        ) : null}
      </div>

      <div className="grid gap-3 sm:grid-cols-[1fr_auto]">
        <textarea
          value={prompt}
          onChange={(event) => setPrompt(event.target.value)}
          placeholder={initialPrompt}
          rows={3}
          className="min-h-[92px] rounded-[14px] border border-sand-400 bg-[#FFFDFA] px-4 py-3 text-[14px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint"
        />
        <button
          type="button"
          disabled={isGenerating}
          onClick={() => void generate()}
          className="h-12 self-end rounded-full bg-clay px-5 text-[14px] font-semibold text-sand-100 shadow-[0_8px_20px_rgba(192,91,59,0.18)] transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isGenerating ? t("generating") : t("generate")}
        </button>
      </div>

      {activeError ? (
        <div role="alert" className="rounded-[14px] border border-red-200 bg-red-50 px-4 py-3 text-[13px] text-red-800">
          {getErrorMessage(activeError, "Could not generate route alternatives.")}
        </div>
      ) : null}

      {session ? (
        <div className="space-y-5">
          <div className="rounded-[12px] border border-amber-200 bg-amber-50 px-3 py-2 text-[13px] leading-5 text-amber-900">
            {(session.warnings.length > 0 ? session.warnings : [t("approximateWarning")]).join(" ")}
          </div>

          <RouteAlternativeComparisonTable
            alternatives={alternatives}
            summary={session.comparisonSummary}
            selectedId={selected?.id}
            onSelect={setSelected}
          />

          <div className="grid gap-4 lg:grid-cols-2">
            {alternatives.map((alternative) => (
              <RouteAlternativeCard
                key={alternative.id}
                alternative={alternative}
                selected={selected?.id === alternative.id}
                canCreateTrip={canCreateTrip}
                canApply={Boolean(trip?.id && canApply)}
                canCreatePoll={Boolean(trip?.id && canCreatePoll)}
                onSelect={setSelected}
                onCreateTrip={setCreateTripAlternative}
                onApply={setApplyAlternative}
                onMoreLikeThis={(item) => void refine(`More like ${item.title}`, item)}
              />
            ))}
          </div>

          {trip?.id ? <RouteAlternativeVoteControls pollAvailable={false} /> : null}

          <RouteAlternativeRefineBar
            disabled={!session}
            isPending={refineMutation.isPending}
            onRefine={(instruction) => void refine(instruction, selected ?? undefined)}
          />
        </div>
      ) : null}

      <CreateTripFromRouteAlternativeDialog
        alternative={createTripAlternative}
        sessionWorkspaceId={session?.workspaceId}
        workspaces={editableWorkspaces}
        isPending={createTripMutation.isPending}
        error={
          createTripMutation.isError
            ? getErrorMessage(createTripMutation.error, "Could not create trip.")
            : null
        }
        onClose={() => setCreateTripAlternative(null)}
        onConfirm={(input) => void createTrip(input)}
      />
      <ApplyRouteAlternativeDialog
        alternative={applyAlternative}
        currentRevision={trip?.itineraryRevision}
        isPending={applyMutation.isPending}
        error={
          applyMutation.isError
            ? getErrorMessage(applyMutation.error, "Could not apply route.")
            : null
        }
        onClose={() => setApplyAlternative(null)}
        onConfirm={(input) => void applyRoute(input)}
      />
    </section>
  );
}
