"use client";

import { useState } from "react";
import type { TripRouteLeg } from "@/entities/route/model";
import { useAttachRouteLegTransportOption } from "@/hooks/useAttachRouteLegTransportOption";
import { useRemoveRouteLegTransportOption } from "@/hooks/useRemoveRouteLegTransportOption";
import { useRouteLegTransportSearch } from "@/hooks/useRouteLegTransportSearch";
import { selectedOptionFromTransportOption } from "@/lib/api/transport";
import type { SearchRouteLegTransportInput, TransportModeValue, TransportOption } from "@/types/transport";
import { AttachTransportOptionDialog } from "./AttachTransportOptionDialog";
import { SelectedTransportOptionCard } from "./SelectedTransportOptionCard";
import { TransportSearchDialog } from "./TransportSearchDialog";

type Props = {
  tripId?: string;
  leg: TripRouteLeg;
  currency: string;
  travelers?: number;
  canEdit?: boolean;
  expectedItineraryRevision?: number;
  online?: boolean;
};

const supportedModes = new Set<TransportModeValue>([
  "train",
  "bus",
  "flight",
  "ferry",
  "car",
  "rental_car",
  "public_transport",
  "walk",
  "bike",
  "hiking",
  "boat",
  "other"
]);

export function RouteLegTransportOptions({
  tripId,
  leg,
  currency,
  travelers = 1,
  canEdit = false,
  expectedItineraryRevision,
  online = true
}: Props) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [pendingOption, setPendingOption] = useState<TransportOption | null>(null);
  const search = useRouteLegTransportSearch(tripId ?? "", leg.id);
  const attach = useAttachRouteLegTransportOption(tripId ?? "", leg.id);
  const remove = useRemoveRouteLegTransportOption(tripId ?? "", leg.id);
  const defaultModes = defaultModesForLeg(leg);
  const disabledReason = !tripId
    ? "Save the trip before searching transport."
    : !online
      ? "Transport search is unavailable offline."
      : !canEdit
        ? "View-only access."
        : "";

  function openSearch() {
    if (disabledReason) {
      return;
    }
    setSearchOpen(true);
    if (!search.data && !search.isPending) {
      runSearch({});
    }
  }

  function runSearch(input: SearchRouteLegTransportInput) {
    search.mutate({
      currency,
      travelers,
      modes: defaultModes,
      ...input
    });
  }

  function confirmAttach() {
    if (!pendingOption) {
      return;
    }
    attach.mutate(
      {
        expectedItineraryRevision,
        option: selectedOptionFromTransportOption(pendingOption),
        updateLegMode: true
      },
      {
        onSuccess: () => {
          setPendingOption(null);
          setSearchOpen(false);
        }
      }
    );
  }

  function removeSelectedOption() {
    if (!canEdit || !online) {
      return;
    }
    remove.mutate({
      expectedItineraryRevision,
      resetLegMode: false
    });
  }

  const options = search.data?.options ?? [];
  return (
    <div className="mt-3 space-y-2">
      <SelectedTransportOptionCard
        canEdit={canEdit && online}
        option={leg.selectedTransportOption}
        removing={remove.isPending}
        onRemove={removeSelectedOption}
      />
      {canEdit ? (
        <div className="flex flex-wrap items-center gap-2">
          <button
            className="rounded-md border border-sand-300 bg-white px-3 py-1.5 text-[12.5px] font-semibold text-cocoa-600 transition hover:bg-sand-50 disabled:opacity-60"
            disabled={Boolean(disabledReason)}
            onClick={openSearch}
            title={disabledReason || "Compare transport options"}
            type="button"
          >
            {leg.selectedTransportOption ? "Change option" : "Find transport"}
          </button>
          {remove.error ? (
            <span className="text-[12px] font-medium text-red-700">{remove.error.message}</span>
          ) : null}
        </div>
      ) : null}
      <TransportSearchDialog
        currency={currency}
        defaultModes={defaultModes}
        error={search.error?.message ?? null}
        loading={search.isPending}
        onClose={() => setSearchOpen(false)}
        onSearch={runSearch}
        onSelect={setPendingOption}
        open={searchOpen}
        options={options}
        selectingOptionId={pendingOption?.id ?? null}
        summary={search.data?.summary ?? null}
        travelers={travelers}
      />
      <AttachTransportOptionDialog
        error={attach.error?.message ?? null}
        onClose={() => setPendingOption(null)}
        onConfirm={confirmAttach}
        option={pendingOption}
        pending={attach.isPending}
      />
    </div>
  );
}

function defaultModesForLeg(leg: TripRouteLeg): TransportModeValue[] {
  const normalized = normalizeMode(leg.mode);
  return normalized ? [normalized] : ["train", "bus", "car"];
}

function normalizeMode(mode?: string | null): TransportModeValue | null {
  if (!mode) {
    return null;
  }
  const normalized = mode.replaceAll("-", "_").replaceAll(" ", "_").toLowerCase();
  if (normalized === "walking") {
    return "walk";
  }
  if (normalized === "driving") {
    return "car";
  }
  if (normalized === "cycling") {
    return "bike";
  }
  return supportedModes.has(normalized as TransportModeValue)
    ? (normalized as TransportModeValue)
    : null;
}
