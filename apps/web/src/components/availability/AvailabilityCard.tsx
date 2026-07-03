"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button, buttonStyles } from "@/components/ui/Button";
import { availabilityKeys, searchAvailability } from "@/lib/api/availability";
import {
  formatMoney,
  getCostAmount,
  getCostCurrency,
  isManualCost
} from "@/lib/budget/format";
import { getTripItemDate } from "@/lib/itinerary/opening-hours-utils";
import { cn } from "@/lib/utils";
import type {
  AvailabilityOption,
  AvailabilitySearchRequest,
  AvailabilitySearchResponse,
  AvailabilityStatus
} from "@/types/availability";
import type { ItineraryItem, Trip } from "@/types/trip";

type AvailabilityCardProps = {
  trip: Trip;
  dayNumber: number;
  itemIndex: number;
  item: ItineraryItem;
  currency: string;
  travelers?: { adults?: number; children?: number };
  disabled?: boolean;
  onApplyPrice?: (
    option: AvailabilityOption,
    result: AvailabilitySearchResponse
  ) => Promise<void>;
  onResult?: (
    dayNumber: number,
    itemIndex: number,
    result: AvailabilitySearchResponse
  ) => void;
};

const AVAILABILITY_STALE_MS = 15 * 60 * 1000;

export function AvailabilityCard({
  trip,
  dayNumber,
  itemIndex,
  item,
  currency,
  travelers,
  disabled = false,
  onApplyPrice,
  onResult
}: AvailabilityCardProps) {
  const [applyingOptionId, setApplyingOptionId] = useState<string | null>(null);
  const itemDate = trip.startDate
    ? formatDateForAvailability(getTripItemDate(trip.startDate, dayNumber))
    : null;
  const request = useMemo(
    () =>
      itemDate
        ? buildAvailabilityRequest({
            trip,
            item,
            itemDate,
            currency,
            travelers
          })
        : null,
    [currency, item, itemDate, travelers, trip]
  );

  const query = useQuery({
    queryKey: availabilityKeys.search({
      tripId: trip.id,
      dayNumber,
      itemIndex,
      date: itemDate ?? "",
      itemName: item.name
    }),
    queryFn: () => {
      if (!request) {
        throw new Error("Trip start date is required to check availability.");
      }
      return searchAvailability(request);
    },
    enabled: false,
    staleTime: AVAILABILITY_STALE_MS,
    gcTime: AVAILABILITY_STALE_MS * 2
  });

  useEffect(() => {
    if (query.data) {
      onResult?.(dayNumber, itemIndex, query.data);
    }
  }, [dayNumber, itemIndex, onResult, query.data]);

  const result = query.data ?? null;
  const pricedOptions = result?.options.filter((option) => option.price) ?? [];
  const primaryOption = pricedOptions[0] ?? result?.options[0] ?? null;

  async function checkAvailability() {
    await query.refetch();
  }

  async function applyPrice(option: AvailabilityOption) {
    if (!onApplyPrice || !result || !option.price) {
      return;
    }
    if (isManualCost(item.estimatedCost)) {
      const confirmed = window.confirm(
        "This item already has a manually edited cost. Replace it?"
      );
      if (!confirmed) {
        return;
      }
    }
    setApplyingOptionId(option.id);
    try {
      await onApplyPrice(option, result);
    } finally {
      setApplyingOptionId(null);
    }
  }

  return (
    <div
      className="mt-3 rounded-md border border-slate-200 bg-slate-50 p-3"
      id={`day-${dayNumber}-item-${itemIndex}-availability`}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-semibold text-slate-900">Availability</p>
            {result ? (
              <StatusBadge fallback={result.fallbackUsed} status={result.status} />
            ) : null}
            {result?.cached ? (
              <span className="rounded-full border border-slate-200 bg-white px-2 py-0.5 text-xs font-medium text-slate-500">
                Cached
              </span>
            ) : null}
          </div>
          {!result && !query.isError ? (
            <p className="mt-1 text-xs leading-5 text-slate-500">
              Availability and prices may change on the provider website.
            </p>
          ) : null}
        </div>
        <Button
          disabled={disabled || query.isFetching || !request}
          onClick={checkAvailability}
          size="sm"
          type="button"
          variant="secondary"
        >
          {query.isFetching ? "Checking..." : result ? "Check again" : "Check availability"}
        </Button>
      </div>

      {!request ? (
        <p className="mt-3 text-sm text-amber-800">Add a trip start date to check availability.</p>
      ) : null}

      {query.isError ? (
        <p className="mt-3 text-sm text-red-700">
          {query.error instanceof Error ? query.error.message : "Could not check availability."}
        </p>
      ) : null}

      {result ? (
        <div className="mt-3 space-y-3">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
            <span>{result.providerDisplayName || result.provider}</span>
            <span>{checkedLabel(result.checkedAt)}</span>
            {result.match?.matched ? (
              <span>{Math.round(result.match.confidence * 100)}% match</span>
            ) : null}
          </div>

          {primaryOption ? (
            <OptionSummary
              currency={currency}
              currentCost={item.estimatedCost}
              disabled={disabled || applyingOptionId === primaryOption.id}
              isApplying={applyingOptionId === primaryOption.id}
              onApplyPrice={onApplyPrice ? () => applyPrice(primaryOption) : undefined}
              option={primaryOption}
            />
          ) : (
            <p className="text-sm text-slate-600">
              No bookable option was found for this date.
            </p>
          )}

          {result.options.length > 1 ? (
            <div className="space-y-2">
              {result.options.slice(1, 3).map((option) => (
                <OptionSummary
                  compact
                  currency={currency}
                  currentCost={item.estimatedCost}
                  disabled={disabled || applyingOptionId === option.id}
                  isApplying={applyingOptionId === option.id}
                  key={option.id}
                  onApplyPrice={onApplyPrice ? () => applyPrice(option) : undefined}
                  option={option}
                />
              ))}
            </div>
          ) : null}

          {(result.warnings ?? []).map((warning) => (
            <p
              className={cn(
                "text-xs leading-5",
                result.fallbackUsed ? "text-amber-800" : "text-slate-500"
              )}
              key={warning}
            >
              {warning}
            </p>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function OptionSummary({
  option,
  currentCost,
  currency,
  compact = false,
  disabled,
  isApplying,
  onApplyPrice
}: {
  option: AvailabilityOption;
  currentCost: ItineraryItem["estimatedCost"];
  currency: string;
  compact?: boolean;
  disabled: boolean;
  isApplying: boolean;
  onApplyPrice?: () => void;
}) {
  const currentAmount = getCostAmount(currentCost);
  const currentCurrency = getCostCurrency(currentCost) ?? currency;
  const optionCurrency = option.price?.currency ?? currency;
  const canCompare =
    option.price &&
    currentAmount != null &&
    currentCurrency.toUpperCase() === optionCurrency.toUpperCase();
  const difference = canCompare ? option.price!.amount - (currentAmount ?? 0) : null;
  const higherWarning =
    difference != null &&
    difference > 0 &&
    (difference >= 10 || ((currentAmount ?? 0) > 0 && difference / (currentAmount ?? 1) >= 0.2));
  const bookingUrl = safeBookingUrl(option.bookingUrl);

  return (
    <div className={cn("rounded-md border border-slate-200 bg-white p-3", compact && "p-2")}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-950">{option.title}</p>
          <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-500">
            {option.price ? (
              <span className="font-semibold text-slate-800">
                From {formatMoney(option.price.amount, option.price.currency)}
              </span>
            ) : null}
            <span>{formatPriceType(option.priceType)}</span>
            {option.startTimes?.length ? (
              <span>{option.startTimes.slice(0, 3).join(", ")}</span>
            ) : null}
            {option.durationMinutes ? <span>{option.durationMinutes} min</span> : null}
          </div>
          {currentAmount != null && option.price ? (
            <p className="mt-1 text-xs text-slate-500">
              Current estimate: {formatMoney(currentAmount, currentCurrency)} · Provider option:{" "}
              {formatMoney(option.price.amount, option.price.currency)}
              {difference != null ? ` · Difference: ${formatSignedMoney(difference, optionCurrency)}` : ""}
            </p>
          ) : null}
          {higherWarning ? (
            <p className="mt-1 text-xs font-medium text-amber-800">
              Provider price is higher than current estimate.
            </p>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2 sm:justify-end">
          {bookingUrl ? (
            <a
              className={buttonStyles({ variant: "secondary", size: "sm" })}
              href={bookingUrl}
              rel="noopener noreferrer"
              target="_blank"
            >
              Book externally
            </a>
          ) : null}
          {option.price && onApplyPrice ? (
            <Button
              disabled={disabled}
              onClick={onApplyPrice}
              size="sm"
              type="button"
              variant="secondary"
            >
              {isApplying ? "Updating..." : "Use this price"}
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function StatusBadge({ status, fallback }: { status: AvailabilityStatus; fallback: boolean }) {
  return (
    <span
      className={cn(
        "rounded-full border px-2 py-0.5 text-xs font-medium",
        status === "available" && "border-emerald-200 bg-emerald-50 text-emerald-700",
        status === "limited" && "border-amber-200 bg-amber-50 text-amber-800",
        status === "unavailable" && "border-red-200 bg-red-50 text-red-700",
        status === "unknown" && "border-slate-200 bg-white text-slate-600"
      )}
    >
      {fallback ? "Fallback " : ""}
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}

function buildAvailabilityRequest({
  trip,
  item,
  itemDate,
  currency,
  travelers
}: {
  trip: Trip;
  item: ItineraryItem;
  itemDate: string;
  currency: string;
  travelers?: { adults?: number; children?: number };
}): AvailabilitySearchRequest {
  return {
    destination: trip.destination,
    date: itemDate,
    currency: (item.estimatedCost?.currency || currency || trip.budgetCurrency || "EUR").toUpperCase(),
    item: {
      name: item.name,
      type: item.type,
      description: item.note ?? undefined,
      startTime: item.time,
      estimatedCost: item.estimatedCost ?? null,
      place: item.place
        ? {
            name: item.place.name,
            address: item.place.address,
            lat: item.place.latitude,
            lng: item.place.longitude,
            provider: item.place.provider,
            providerPlaceId: item.place.providerPlaceId
          }
        : null
    },
    travelers: {
      adults: travelers?.adults ?? trip.travelers ?? 1,
      children: travelers?.children ?? 0
    }
  };
}

function checkedLabel(checkedAt: string) {
  const checked = new Date(checkedAt).getTime();
  if (!Number.isFinite(checked)) {
    return "Checked recently";
  }
  const minutes = Math.max(0, Math.round((Date.now() - checked) / 60000));
  if (minutes < 1) {
    return "Checked just now";
  }
  if (minutes === 1) {
    return "Checked 1 minute ago";
  }
  return `Checked ${minutes} minutes ago`;
}

function formatDateForAvailability(value: Date | null) {
  if (!value) {
    return null;
  }
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function formatPriceType(value: string) {
  if (value === "per_person") {
    return "per person";
  }
  if (value === "per_group") {
    return "per group";
  }
  return value.replace(/_/g, " ");
}

function formatSignedMoney(amount: number, currency: string) {
  const prefix = amount > 0 ? "+" : "";
  return `${prefix}${formatMoney(amount, currency)}`;
}

function safeBookingUrl(value: string | null | undefined) {
  if (!value) {
    return null;
  }
  try {
    const parsed = new URL(value);
    return parsed.protocol === "http:" || parsed.protocol === "https:" ? value : null;
  } catch {
    return null;
  }
}
