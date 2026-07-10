"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAppLanguage } from "@/components/i18n/I18nProvider";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import {
  useCreateTripFromSuggestion,
  useRefineTripDiscovery,
  useSurpriseMe,
  useTripDiscoverySessions,
  useTripDiscoverySuggestions
} from "@/hooks/useTripDiscovery";
import { getErrorMessage } from "@/lib/utils";
import type { TripDiscoverySession, TripDiscoverySuggestion } from "@/types/trip-discovery";
import { CreateTripFromSuggestionDialog } from "./CreateTripFromSuggestionDialog";
import { DestinationSuggestionsGrid } from "./DestinationSuggestionsGrid";
import { DiscoverySessionHistory } from "./DiscoverySessionHistory";
import { TripDiscoveryHero } from "./TripDiscoveryHero";
import { TripDiscoveryPromptBox, type DiscoveryDraft } from "./TripDiscoveryPromptBox";
import { TripDiscoveryRefineBar } from "./TripDiscoveryRefineBar";

const initialDraft: DiscoveryDraft = {
  prompt: "",
  chips: [],
  workspaceId: "",
  durationDays: 4,
  budgetAmount: "",
  budgetCurrency: "EUR",
  travelers: 2,
  origin: ""
};

export function TripDiscoveryExperience() {
  const t = useTranslations("tripDiscovery");
  const { language } = useAppLanguage();
  const router = useRouter();
  const { currentScope, currentWorkspace, editableWorkspaces } = useWorkspaces();
  const [draft, setDraft] = useState(initialDraft);
  const [session, setSession] = useState<TripDiscoverySession | null>(null);
  const [selected, setSelected] = useState<TripDiscoverySuggestion | null>(null);
  const suggestionsMutation = useTripDiscoverySuggestions();
  const surpriseMutation = useSurpriseMe();
  const refineMutation = useRefineTripDiscovery(session?.id);
  const createMutation = useCreateTripFromSuggestion(session?.id, selected?.id);
  const history = useTripDiscoverySessions();

  useEffect(() => {
    if (
      currentScope === "workspace" &&
      currentWorkspace &&
      editableWorkspaces.some((workspace) => workspace.id === currentWorkspace.id)
    ) {
      setDraft((current) => ({ ...current, workspaceId: currentWorkspace.id }));
    }
  }, [currentScope, currentWorkspace, editableWorkspaces]);

  const requestContext = useMemo(() => {
    const budgetAmount = draft.budgetAmount === "" ? undefined : Number(draft.budgetAmount);
    return {
      scope: draft.workspaceId ? ("workspace" as const) : ("personal" as const),
      workspaceId: draft.workspaceId || undefined,
      durationDays: draft.durationDays || undefined,
      budget:
        budgetAmount != null && Number.isFinite(budgetAmount)
          ? { amount: budgetAmount, currency: draft.budgetCurrency }
          : undefined,
      travelers: draft.travelers || 1,
      origin: draft.origin.trim() || undefined,
      outputLanguage: language,
      avoidPreviouslyVisited: true,
      preferNovelty: true
    };
  }, [draft, language]);

  const activeError =
    suggestionsMutation.error ?? surpriseMutation.error ?? refineMutation.error ?? null;
  const isFinding =
    suggestionsMutation.isPending || surpriseMutation.isPending || refineMutation.isPending;

  function acceptSession(next: TripDiscoverySession) {
    setSession(next);
    setSelected(null);
    requestAnimationFrame(() =>
      document.getElementById("destination-suggestions")?.scrollIntoView({
        behavior: "smooth",
        block: "start"
      })
    );
  }

  function refine(instruction: string, suggestion?: TripDiscoverySuggestion, feedbackType?: string) {
    refineMutation.mutate(
      {
        instruction,
        selectedSuggestionId: suggestion?.id,
        feedbackType,
        outputLanguage: language
      },
      { onSuccess: acceptSession }
    );
  }

  return (
    <div className="space-y-6">
      <TripDiscoveryHero />
      <TripDiscoveryPromptBox
        value={draft}
        workspaces={editableWorkspaces}
        isPending={suggestionsMutation.isPending}
        isSurprisePending={surpriseMutation.isPending}
        onChange={setDraft}
        onSubmit={() =>
          suggestionsMutation.mutate(
            {
              ...requestContext,
              prompt: draft.prompt,
              quickChips: draft.chips.map((chip) => chip.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`))
            },
            { onSuccess: acceptSession }
          )
        }
        onSurprise={() =>
          surpriseMutation.mutate(
            { ...requestContext, noveltyLevel: "balanced" },
            { onSuccess: acceptSession }
          )
        }
      />

      {activeError ? (
        <div role="alert" className="rounded-[16px] border border-red-200 bg-red-50 px-5 py-4 text-[13.5px] text-red-800">
          <p className="font-semibold">{t("errorTitle")}</p>
          <p className="mt-1">{getErrorMessage(activeError, t("errorBody"))}</p>
        </div>
      ) : null}

      {isFinding ? (
        <div aria-live="polite" className="rounded-[18px] border border-sand-300 bg-white px-6 py-8 text-center text-[14px] text-cocoa-500">
          <span className="mr-2 inline-block animate-pulse text-clay" aria-hidden="true">✦</span>
          {refineMutation.isPending ? t("refining") : t("findingPlaces")}
        </div>
      ) : null}

      {session ? (
        <section id="destination-suggestions" className="scroll-mt-8 space-y-5">
          <div>
            <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-clay">
              {t("suggestionsEyebrow")}
            </p>
            <h2 className="mt-1.5 font-newsreader text-[30px] font-semibold text-cocoa-900">
              {session.response.sessionTitle}
            </h2>
          </div>
          <DestinationSuggestionsGrid
            suggestions={session.response.suggestions}
            onSelect={setSelected}
            onSimilar={(suggestion) => refine(t("refineInstructions.similarPlaces"), suggestion, "similar")}
            onReject={(suggestion) => refine(t("refineInstructions.notThisVibe"), suggestion, "not_for_me")}
          />
          <TripDiscoveryRefineBar
            isPending={refineMutation.isPending}
            onRefine={(instruction) => refine(instruction)}
          />
          <div className="rounded-xl bg-sand-100 px-4 py-3 text-[12px] leading-5 text-cocoa-400">
            {session.response.warnings.join(" ") || t("budgetDisclaimer")}
          </div>
        </section>
      ) : null}

      <DiscoverySessionHistory
        sessions={history.data?.items ?? []}
        onSelect={acceptSession}
      />

      <CreateTripFromSuggestionDialog
        suggestion={selected}
        sessionWorkspaceId={session?.workspaceId}
        workspaces={editableWorkspaces}
        isPending={createMutation.isPending}
        error={
          createMutation.isError
            ? getErrorMessage(createMutation.error, t("createError"))
            : null
        }
        onClose={() => setSelected(null)}
        onConfirm={(input) =>
          createMutation.mutate(input, {
            onSuccess: ({ trip }) => router.push(`/trips/${trip.id}`)
          })
        }
      />
    </div>
  );
}
