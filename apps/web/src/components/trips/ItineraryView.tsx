import { AvailabilityCard } from "@/components/availability/AvailabilityCard";
import type { Itinerary } from "@/types/trip";
import type { Trip } from "@/types/trip";
import type { AvailabilityOption, AvailabilitySearchResponse } from "@/types/availability";
import type { OpeningHoursInterval } from "@/types/place";
import {
  formatOpeningHoursForDay,
  getDayOfWeekMondayBased,
  getOpeningStatus,
  getTripItemDate
} from "@/lib/itinerary/opening-hours-utils";
import { formatDate, formatInterestLabel, formatMoney, formatPaceLabel } from "@/lib/utils";
import { costBadgeLabel, isManualCost, isProviderCost } from "@/lib/budget/format";
import { Button } from "@/components/ui/Button";
import { CommentButton } from "@/components/comments/CommentButton";
import { makeCommentItemKey } from "@/lib/comments/comment-counts";

export type RegeneratingTarget =
  | { type: "day"; dayNumber: number }
  | { type: "item"; dayNumber: number; itemIndex: number };

// CommentControls wires per-item comment badges/buttons into the read-only
// itinerary view. It is optional and never passed on the public share page, so
// comments stay a private, authenticated feature.
export type CommentControls = {
  countByKey: Record<string, number>;
  onOpenItem: (dayNumber: number, itemIndex: number) => void;
  disabled?: boolean;
};

type ItineraryViewProps = {
  itinerary: Itinerary;
  currency?: string;
  startDate?: string | null;
  trip?: Trip;
  disabled?: boolean;
  regeneratingTarget?: RegeneratingTarget | null;
  onRegenerateDay?: (dayNumber: number, instruction?: string) => void;
  onRegenerateItem?: (dayNumber: number, itemIndex: number, instruction?: string) => void;
  onAvailabilityResult?: (
    dayNumber: number,
    itemIndex: number,
    result: AvailabilitySearchResponse
  ) => void;
  onApplyAvailabilityPrice?: (
    dayNumber: number,
    itemIndex: number,
    option: AvailabilityOption,
    result: AvailabilitySearchResponse
  ) => Promise<void>;
  comments?: CommentControls;
};

export function ItineraryView({
  itinerary,
  currency = "EUR",
  startDate,
  trip,
  disabled = false,
  regeneratingTarget = null,
  onRegenerateDay,
  onRegenerateItem,
  onAvailabilityResult,
  onApplyAvailabilityPrice,
  comments
}: ItineraryViewProps) {
  if (!itinerary.days || itinerary.days.length === 0) {
    return (
      <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
        No itinerary days were returned.
      </div>
    );
  }

  const displayCurrency = itinerary.currency || currency;
  const regenerationDisabled = disabled || Boolean(regeneratingTarget);

  function requestInstruction() {
    const value = window.prompt("Optional instruction for AI regeneration", "");
    if (value == null) {
      return null;
    }
    return value.trim() || undefined;
  }

  function regenerateDay(dayNumber: number) {
    const instruction = requestInstruction();
    if (instruction === null) {
      return;
    }
    onRegenerateDay?.(dayNumber, instruction);
  }

  function regenerateItem(dayNumber: number, itemIndex: number) {
    const instruction = requestInstruction();
    if (instruction === null) {
      return;
    }
    onRegenerateItem?.(dayNumber, itemIndex, instruction);
  }

  return (
    <div className="space-y-5">
      <div className="rounded-lg border border-slate-200 bg-white p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Generated itinerary</h2>
            {itinerary.summary ? (
              <p className="mt-2 text-sm leading-6 text-slate-600">{itinerary.summary}</p>
            ) : null}
          </div>
          {itinerary.totalBudget != null ? (
            <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm">
              <p className="text-xs font-medium text-slate-500">Budget</p>
              <p className="font-semibold text-slate-900">
                {formatMoney(itinerary.totalBudget, displayCurrency)}
              </p>
            </div>
          ) : null}
        </div>

        <div className="mt-5 flex flex-wrap gap-x-5 gap-y-2 text-sm text-slate-600">
          {itinerary.destination ? <span>{itinerary.destination}</span> : null}
          {itinerary.travelers ? (
            <span>
              {itinerary.travelers} {itinerary.travelers === 1 ? "traveler" : "travelers"}
            </span>
          ) : null}
          {itinerary.pace ? <span>{formatPaceLabel(itinerary.pace)} pace</span> : null}
          {itinerary.generatedAt ? <span>Generated {formatDate(itinerary.generatedAt)}</span> : null}
        </div>
      </div>

      {itinerary.days.map((day, dayIndex) => {
        const dayNumber = day.day || dayIndex + 1;

        return (
          <section key={dayNumber} className="rounded-lg border border-slate-200 bg-white p-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <h3 className="text-lg font-semibold text-slate-950">
                Day {dayNumber} — {day.title}
              </h3>
              {onRegenerateDay ? (
                <Button
                  disabled={regenerationDisabled}
                  onClick={() => regenerateDay(dayNumber)}
                  size="sm"
                  type="button"
                  variant="secondary"
                >
                  {regeneratingTarget?.type === "day" && regeneratingTarget.dayNumber === dayNumber
                    ? "Regenerating..."
                    : "Regenerate day"}
                </Button>
              ) : null}
            </div>
            <ol className="mt-5 divide-y divide-slate-100">
              {day.items.map((item, index) => (
                <li
                  key={`${dayNumber}-${item.time}-${item.name}-${index}`}
                  className="grid gap-3 py-4 first:pt-0 last:pb-0 sm:grid-cols-[6.5rem_minmax(0,1fr)_9rem]"
                >
                  <div className="text-sm font-semibold text-slate-900">{item.time}</div>
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700">
                        {formatInterestLabel(item.type)}
                      </span>
                      <p className="font-semibold text-slate-950">{item.name}</p>
                    </div>
                    {item.note ? (
                      <p className="mt-2 text-sm leading-6 text-slate-600">{item.note}</p>
                    ) : null}
                    {item.place ? (
                      <div className="mt-2 space-y-1 text-sm text-slate-600">
                        <p>{item.place.address}</p>
                        <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs font-medium text-slate-500">
                          <span>Provider: {formatPlaceCategory(item.place.provider || "unknown")}</span>
                          {item.placeEnrichment?.status === "matched" ? (
                            <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-emerald-700">
                              Auto-matched place
                              {formatConfidence(item.placeEnrichment.confidence)}
                            </span>
                          ) : null}
                          {item.place.rating != null ? (
                            <span>
                              Rating {item.place.rating}
                              {item.place.ratingCount != null
                                ? ` (${item.place.ratingCount.toLocaleString()})`
                                : ""}
                            </span>
                          ) : null}
                          {item.place.category ? (
                            <span>{formatPlaceCategory(item.place.category)}</span>
                          ) : null}
                          {item.place.mapUrl ? (
                            <a
                              className="text-primary-700 hover:text-primary-600"
                              href={item.place.mapUrl}
                              rel="noreferrer"
                              target="_blank"
                            >
                              Open map
                            </a>
                          ) : null}
                        </div>
                        <OpeningHoursStatus
                          dayNumber={dayNumber}
                          itemTime={item.time}
                          openingHours={item.place.openingHours}
                          startDate={startDate}
                        />
                      </div>
                    ) : null}
                    {trip && isLikelyBookableItem(item) ? (
                      <AvailabilityCard
                        currency={displayCurrency}
                        dayNumber={dayNumber}
                        disabled={disabled}
                        item={item}
                        itemIndex={index}
                        onApplyPrice={
                          onApplyAvailabilityPrice
                            ? (option, result) =>
                                onApplyAvailabilityPrice(dayNumber, index, option, result)
                            : undefined
                        }
                        onResult={onAvailabilityResult}
                        trip={trip}
                      />
                    ) : null}
                  </div>
                  <div className="flex items-start justify-between gap-3 sm:flex-col sm:items-end">
                    {costBadgeLabel(item.estimatedCost, displayCurrency) ? (
                      <div
                        className="text-sm font-semibold text-slate-900"
                        title={isManualCost(item.estimatedCost) ? "Manually edited cost" : undefined}
                      >
                        {costBadgeLabel(item.estimatedCost, displayCurrency)}
                        {isManualCost(item.estimatedCost) ? (
                          <span className="ml-1 text-xs font-normal text-slate-400">manual</span>
                        ) : null}
                        {isProviderCost(item.estimatedCost) ? (
                          <span className="ml-1 text-xs font-normal text-slate-400">provider estimate</span>
                        ) : null}
                      </div>
                    ) : item.priceEnrichment?.status === "no_match" ? (
                      <div className="text-xs font-medium text-slate-400">No ticket estimate</div>
                    ) : (
                      <span className="hidden sm:block" />
                    )}
                    {onRegenerateItem ? (
                      <Button
                        disabled={regenerationDisabled}
                        onClick={() => regenerateItem(dayNumber, index)}
                        size="sm"
                        type="button"
                        variant="secondary"
                      >
                        {regeneratingTarget?.type === "item" &&
                        regeneratingTarget.dayNumber === dayNumber &&
                        regeneratingTarget.itemIndex === index
                          ? "Regenerating..."
                          : "Regenerate item"}
                      </Button>
                    ) : null}
                    {comments ? (
                      <CommentButton
                        count={comments.countByKey[makeCommentItemKey(dayNumber, index)] ?? 0}
                        disabled={comments.disabled}
                        onClick={() => comments.onOpenItem(dayNumber, index)}
                      />
                    ) : null}
                  </div>
                </li>
              ))}
            </ol>
          </section>
        );
      })}
    </div>
  );
}

function formatPlaceCategory(value: string) {
  return value
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function isLikelyBookableItem(item: Itinerary["days"][number]["items"][number]) {
  const type = (item.type ?? "").toLowerCase();
  const placeCategory = (item.place?.category ?? "").toLowerCase();
  const text = `${type} ${placeCategory} ${item.name ?? ""} ${item.note ?? ""}`.toLowerCase();

  if (
    [
      "rest",
      "break",
      "walk",
      "walking",
      "transport",
      "accommodation",
      "hotel",
      "note",
      "food",
      "meal",
      "restaurant",
      "cafe"
    ].some((term) => text.includes(term))
  ) {
    return false;
  }

  return [
    "attraction",
    "museum",
    "landmark",
    "tour",
    "activity",
    "gallery",
    "palace",
    "castle",
    "zoo",
    "aquarium",
    "theme park",
    "theme_park",
    "ticket",
    "event"
  ].some((term) => text.includes(term));
}

function formatConfidence(value: number | null | undefined) {
  if (value == null || Number.isNaN(value)) {
    return "";
  }
  return ` (${Math.round(value * 100)}%)`;
}

function OpeningHoursStatus({
  openingHours,
  startDate,
  dayNumber,
  itemTime
}: {
  openingHours?: OpeningHoursInterval[] | null;
  startDate?: string | null;
  dayNumber: number;
  itemTime?: string | null;
}) {
  const status = getOpeningStatus({ startDate, dayNumber, itemTime, openingHours });
  const itemDate = startDate ? getTripItemDate(startDate, dayNumber) : null;
  const dayOfWeek = itemDate ? getDayOfWeekMondayBased(itemDate) : null;
  const dailyHours =
    dayOfWeek == null ? "Closed or unknown" : formatOpeningHoursForDay(openingHours, dayOfWeek);
  const badgeLabel =
    status.status === "open"
      ? "Likely open"
      : status.status === "closed"
        ? "May be closed"
        : "Unknown";

  return (
    <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
      <span
        className={
          status.status === "open"
            ? "rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 font-medium text-emerald-700"
            : status.status === "closed"
              ? "rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 font-medium text-amber-800"
              : "rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 font-medium text-slate-500"
        }
      >
        {badgeLabel}
      </span>
      <span className={status.status === "closed" ? "font-medium text-amber-800" : "text-slate-500"}>
        {status.label}
      </span>
      <span className="text-slate-500">Hours: {dailyHours}</span>
    </div>
  );
}
