import type { ComponentType } from "react";
import { AvailabilityCard } from "@/features/availability-search";
import { formatMoney, getCostAmount } from "@/entities/budget/model";
import { transportModeLabel } from "@/components/routes/route-options";
import { getOpeningStatus } from "@/entities/itinerary/model/opening-hours-utils";
import type {
  AvailabilityOption,
  AvailabilitySearchResponse
} from "@/entities/availability/model";
import type { Itinerary, ItineraryItem, Trip } from "@/entities/trip/model";
import type { CommentControls, RegeneratingTarget } from "@/components/trips/ItineraryView";
import { formatDayDate } from "./tripDetailFormat";
import {
  ArrowPathIcon,
  BuildingLibraryIcon,
  ChatBubbleIcon,
  HeartIcon,
  MapPinIcon,
  PaperAirplaneIcon,
  ScaleIcon,
  SparklesIcon,
  StarIcon,
  TruckIcon
} from "./icons";

type ItineraryTimelineProps = {
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
  onOpenCostSplit?: (dayNumber: number, itemIndex: number) => void;
  comments?: CommentControls;
};

/**
 * Warm slice-local fork of the read-mode ItineraryView, styled to the Trip Detail
 * mock (time gutter + card timeline). All interactive handlers — regenerate day /
 * item, cost split, comments, and provider availability — are preserved and wired
 * exactly as the shared view wires them.
 */
export function ItineraryTimeline({
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
  onOpenCostSplit,
  comments
}: ItineraryTimelineProps) {
  if (!itinerary.days || itinerary.days.length === 0) {
    return (
      <div className="rounded-[18px] border border-sand-300 bg-white p-6 text-[14px] text-cocoa-500">
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
    <div id="itinerary" className="flex scroll-mt-24 flex-col gap-9">
      {itinerary.days.map((day, dayIndex) => {
        const dayNumber = day.day || dayIndex + 1;
        const dayRegenerating =
          regeneratingTarget?.type === "day" && regeneratingTarget.dayNumber === dayNumber;

        return (
          <section id={`day-${dayNumber}`} key={dayNumber} className="scroll-mt-24">
            <div className="flex items-baseline justify-between gap-4">
              <h2 className="font-newsreader text-[27px] font-semibold tracking-[-0.01em] text-cocoa-900">
                Day {dayNumber} <span className="font-normal text-[#A08D78]">·</span>{" "}
                <em className="font-medium not-italic">{day.title}</em>
              </h2>
              <div className="flex items-center gap-3">
                <span className="text-[13.5px] font-medium text-cocoa-400">
                  {formatDayDate(startDate, dayNumber)}
                </span>
                {onRegenerateDay ? (
                  <button
                    type="button"
                    title="Regenerate day"
                    disabled={regenerationDisabled}
                    onClick={() => regenerateDay(dayNumber)}
                    className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-sand-300 bg-white text-cocoa-400 transition hover:border-[#E5C3B6] hover:text-clay disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <ArrowPathIcon className={`h-[15px] w-[15px] ${dayRegenerating ? "animate-spin" : ""}`} />
                  </button>
                ) : null}
              </div>
            </div>

            <div className="mt-[18px] flex flex-col gap-3">
              {day.items.map((item, index) => (
                <TimelineItem
                  key={`${dayNumber}-${item.time}-${item.name}-${index}`}
                  item={item}
                  dayNumber={dayNumber}
                  itemIndex={index}
                  currency={displayCurrency}
                  startDate={startDate}
                  trip={trip}
                  disabled={disabled}
                  regenerationDisabled={regenerationDisabled}
                  isRegenerating={
                    regeneratingTarget?.type === "item" &&
                    regeneratingTarget.dayNumber === dayNumber &&
                    regeneratingTarget.itemIndex === index
                  }
                  onRegenerateItem={onRegenerateItem ? regenerateItem : undefined}
                  onOpenCostSplit={onOpenCostSplit}
                  onAvailabilityResult={onAvailabilityResult}
                  onApplyAvailabilityPrice={onApplyAvailabilityPrice}
                  comments={comments}
                />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

type TimelineItemProps = {
  item: ItineraryItem;
  dayNumber: number;
  itemIndex: number;
  currency: string;
  startDate?: string | null;
  trip?: Trip;
  disabled: boolean;
  regenerationDisabled: boolean;
  isRegenerating: boolean;
  onRegenerateItem?: (dayNumber: number, itemIndex: number) => void;
  onOpenCostSplit?: (dayNumber: number, itemIndex: number) => void;
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

function TimelineItem({
  item,
  dayNumber,
  itemIndex,
  currency,
  startDate,
  trip,
  disabled,
  regenerationDisabled,
  isRegenerating,
  onRegenerateItem,
  onOpenCostSplit,
  onAvailabilityResult,
  onApplyAvailabilityPrice,
  comments
}: TimelineItemProps) {
  const visual = getItemVisual(item.type);
  const IconComponent = visual.icon;
  const amount = getCostAmount(item.estimatedCost);
  const openingStatus = item.place
    ? getOpeningStatus({
        startDate,
        dayNumber,
        itemTime: item.time,
        openingHours: item.place.openingHours
      })
    : null;
  const commentKey = comments ? comments.countByKey[`${dayNumber}:${itemIndex}`] ?? 0 : 0;
  const showAvailability = Boolean(trip) && isLikelyBookableItem(item);

  return (
    <div id={`day-${dayNumber}-item-${itemIndex}`} className="grid scroll-mt-24 grid-cols-[52px_minmax(0,1fr)] gap-4 sm:grid-cols-[64px_minmax(0,1fr)]">
      <div className="pt-5 text-right">
        <span className="text-[13px] font-bold text-cocoa-900">{item.time}</span>
      </div>
      <div className="rounded-[18px] border border-sand-300 bg-white px-[22px] py-[18px] shadow-[0_1px_2px_rgba(34,26,20,0.03)] transition hover:border-sand-400 hover:shadow-[0_8px_24px_rgba(34,26,20,0.08)]">
        <div className="flex items-start justify-between gap-4">
          <div className="flex min-w-0 items-start gap-3.5">
            <span
              className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${visual.tile}`}
            >
              <IconComponent className="h-[19px] w-[19px]" />
            </span>
            <div className="min-w-0">
              <p className="text-[16px] font-semibold text-cocoa-900">{item.name}</p>
              {item.note ? (
                <p className="mt-1.5 text-[13.5px] leading-[1.55] text-cocoa-500">{item.note}</p>
              ) : null}
              {item.type === "transfer" && item.transfer ? (
                <TransferDetails item={item} currency={currency} />
              ) : null}
              <div className="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-2 text-[12.5px] text-cocoa-400">
                {openingStatus ? <OpeningPill status={openingStatus} /> : null}
                {item.place?.rating != null ? (
                  <span className="inline-flex items-center gap-1">
                    <StarIcon className="h-3 w-3 text-[#D9A441]" />
                    {item.place.rating}
                    {item.place.ratingCount != null
                      ? ` · ${compactNumber(item.place.ratingCount)} reviews`
                      : ""}
                  </span>
                ) : null}
                {item.place?.address ? (
                  <span className="inline-flex items-center gap-1">
                    <MapPinIcon className="h-[13px] w-[13px]" />
                    {item.place.address}
                  </span>
                ) : null}
                {item.place?.mapUrl ? (
                  <a
                    href={item.place.mapUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="font-semibold text-clay-deep transition hover:text-clay"
                  >
                    Open map
                  </a>
                ) : null}
              </div>
              {showAvailability && trip ? (
                <div className="mt-3">
                  <AvailabilityCard
                    currency={currency}
                    dayNumber={dayNumber}
                    disabled={disabled}
                    item={item}
                    itemIndex={itemIndex}
                    onApplyPrice={
                      onApplyAvailabilityPrice
                        ? (option, result) =>
                            onApplyAvailabilityPrice(dayNumber, itemIndex, option, result)
                        : undefined
                    }
                    onResult={onAvailabilityResult}
                    trip={trip}
                  />
                </div>
              ) : null}
            </div>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-2.5">
            {amount != null ? (
              <span className="font-newsreader text-[19px] font-semibold text-cocoa-900">
                {formatMoney(amount, currency)}
              </span>
            ) : null}
            <div className="flex gap-1">
              {onRegenerateItem ? (
                <IconButton
                  title="Regenerate item"
                  disabled={regenerationDisabled}
                  onClick={() => onRegenerateItem(dayNumber, itemIndex)}
                  icon={ArrowPathIcon}
                  spinning={isRegenerating}
                />
              ) : null}
              {onOpenCostSplit ? (
                <IconButton
                  title={amount == null ? "Add a cost before splitting" : "Split cost"}
                  disabled={disabled || amount == null}
                  onClick={() => onOpenCostSplit(dayNumber, itemIndex)}
                  icon={ScaleIcon}
                />
              ) : null}
              {comments ? (
                <IconButton
                  title="Comments"
                  disabled={comments.disabled}
                  onClick={() => comments.onOpenItem(dayNumber, itemIndex)}
                  icon={ChatBubbleIcon}
                  badge={commentKey > 0 ? commentKey : undefined}
                />
              ) : null}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function TransferDetails({ item, currency }: { item: ItineraryItem; currency: string }) {
  const transfer = item.transfer;
  if (!transfer) {
    return null;
  }
  const transferCost = getCostAmount(transfer.estimatedCost ?? item.estimatedCost);
  const duration = transfer.estimatedDurationMinutes ?? item.durationMinutes;

  return (
    <div className="mt-3 rounded-[14px] border border-sand-300 bg-sand-50 p-3">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[13px] font-semibold text-cocoa-800">
        <span>{transfer.from}</span>
        <span className="text-cocoa-300">to</span>
        <span>{transfer.to}</span>
      </div>
      <div className="mt-2 flex flex-wrap gap-2 text-[12.5px] font-medium text-cocoa-500">
        <span className="rounded-full bg-white px-2.5 py-1">
          {transportModeLabel(transfer.mode)}
        </span>
        {duration ? <span className="rounded-full bg-white px-2.5 py-1">{formatDuration(duration)}</span> : null}
        {transfer.estimatedDistanceKm ? (
          <span className="rounded-full bg-white px-2.5 py-1">
            {Math.round(transfer.estimatedDistanceKm)} km
          </span>
        ) : null}
        {transferCost != null ? (
          <span className="rounded-full bg-white px-2.5 py-1">
            {formatMoney(transferCost, transfer.estimatedCost?.currency ?? currency)}
          </span>
        ) : null}
      </div>
      <p className="mt-2 text-[12.5px] leading-[1.45] text-[#96682A]">
        Verify schedules before travel.
      </p>
    </div>
  );
}

function IconButton({
  title,
  disabled,
  onClick,
  icon: IconComponent,
  spinning,
  badge
}: {
  title: string;
  disabled?: boolean;
  onClick: () => void;
  icon: ComponentType<{ className?: string }>;
  spinning?: boolean;
  badge?: number;
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className="relative inline-flex h-[30px] w-[30px] items-center justify-center rounded-full text-[#B09E8A] transition hover:bg-sand-200 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent"
    >
      <IconComponent className={`h-3.5 w-3.5 ${spinning ? "animate-spin" : ""}`} />
      {badge ? (
        <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-clay px-1 text-[10px] font-semibold text-sand-100">
          {badge}
        </span>
      ) : null}
    </button>
  );
}

function OpeningPill({ status }: { status: ReturnType<typeof getOpeningStatus> }) {
  if (status.status === "open") {
    const { open, close } = status.matchingInterval;
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-[#EDF3EA] px-2.5 py-[3px] font-semibold text-[#2F7A57]">
        Open · {open}–{close}
      </span>
    );
  }
  if (status.status === "closed") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-[#FDF0E3] px-2.5 py-[3px] font-semibold text-[#96682A]">
        {status.label}
      </span>
    );
  }
  return null;
}

type ItemVisual = {
  icon: ComponentType<{ className?: string }>;
  tile: string;
};

function getItemVisual(type: string | null | undefined): ItemVisual {
  const value = (type ?? "").toLowerCase();
  if (value.includes("food") || value.includes("meal") || value.includes("restaurant")) {
    return { icon: HeartIcon, tile: "bg-[#FDF0E3] text-[#B57F24]" };
  }
  if (value.includes("activity") || value.includes("tour") || value.includes("experience")) {
    return { icon: SparklesIcon, tile: "bg-[#F7E4DB] text-[#A84A2E]" };
  }
  if (value.includes("transport") || value.includes("flight") || value.includes("transfer")) {
    return { icon: TruckIcon, tile: "bg-sand-200 text-cocoa-500" };
  }
  if (value.includes("rest") || value.includes("break")) {
    return { icon: PaperAirplaneIcon, tile: "bg-sand-200 text-cocoa-500" };
  }
  return { icon: BuildingLibraryIcon, tile: "bg-[#F7E4DB] text-[#A84A2E]" };
}

function compactNumber(value: number): string {
  return new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 }).format(
    value
  );
}

function formatDuration(minutes: number) {
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min`;
}

// Mirrors ItineraryView.isLikelyBookableItem so the availability search shows up
// on exactly the same items in both the shared and forked views.
function isLikelyBookableItem(item: ItineraryItem) {
  const type = (item.type ?? "").toLowerCase();
  const placeCategory = (item.place?.category ?? "").toLowerCase();
  const text = `${type} ${placeCategory} ${item.name ?? ""} ${item.note ?? ""}`.toLowerCase();

  if (
    [
      "rest",
      "break",
      "walk",
      "walking",
      "transfer",
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
