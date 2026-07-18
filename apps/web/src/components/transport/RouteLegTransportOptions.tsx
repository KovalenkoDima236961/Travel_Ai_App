"use client";

import { useMemo, useState } from "react";
import type { TripRouteLeg } from "@/entities/route/model";
import { useAttachRouteLegTransportOption } from "@/hooks/useAttachRouteLegTransportOption";
import { useRemoveRouteLegTransportOption } from "@/hooks/useRemoveRouteLegTransportOption";
import { useRouteLegTransportSearch } from "@/hooks/useRouteLegTransportSearch";
import { selectedOptionFromTransportOption } from "@/lib/api/transport";
import { useFeatureFlag } from "@/lib/feature-flags/FeatureFlagProvider";
import { isLegTransportStale } from "@/lib/route-builder/route-draft";
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
	const transportSearchEnabled = useFeatureFlag("transport_search_enabled");
  const [searchOpen, setSearchOpen] = useState(false);
  const [pendingOption, setPendingOption] = useState<TransportOption | null>(null);
  const search = useRouteLegTransportSearch(tripId ?? "", leg.id);
  const attach = useAttachRouteLegTransportOption(tripId ?? "", leg.id);
  const remove = useRemoveRouteLegTransportOption(tripId ?? "", leg.id);
	const defaultModes = useMemo(() => defaultModesForLeg(leg.mode), [leg.mode]);
  const defaultDate = leg.departureDate || leg.selectedTransportOption?.departureDate || "";
  const defaultTime = leg.selectedTransportOption?.departureTime || "";
  const disabledReason = !tripId
    ? "Save the trip before searching transport."
	: !transportSearchEnabled
	  ? "Transport search is currently unavailable."
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
      runSearch({ date: defaultDate, time: defaultTime });
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
        stale={isLegTransportStale(leg)}
        onRemove={removeSelectedOption}
      />
		{canEdit && transportSearchEnabled ? (
        <div className="flex flex-wrap items-center gap-2">
          <button
            className="rounded-md border border-sand-300 bg-white px-3 py-1.5 text-[12.5px] font-semibold text-cocoa-600 transition hover:bg-sand-50 disabled:opacity-60"
            data-route-transport-trigger
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
        defaultDate={defaultDate}
        defaultModes={defaultModes}
        defaultTime={defaultTime}
        error={search.error?.message ?? null}
        loading={search.isPending}
        onClose={() => setSearchOpen(false)}
        onSearch={runSearch}
        onSelect={setPendingOption}
			open={transportSearchEnabled && searchOpen}
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

function defaultModesForLeg(mode?: string | null): TransportModeValue[] {
	const normalized = normalizeMode(mode);
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
