"use client";

import { useMemo, useState } from "react";
import type {
  SearchRouteLegTransportInput,
  TransportModeValue,
  TransportOption,
  TransportSearchSummary,
  TransportTimePreference
} from "@/types/transport";
import { transportModeOptions } from "@/components/routes/route-options";
import { CompareTransportModesTable } from "./CompareTransportModesTable";
import { TransportOptionCard } from "./TransportOptionCard";
import { TransportWarningsList } from "./TransportWarningsList";

type Props = {
  open: boolean;
  currency: string;
  travelers?: number;
  defaultModes: TransportModeValue[];
  options: TransportOption[];
  summary?: TransportSearchSummary | null;
  loading?: boolean;
  error?: string | null;
  selectingOptionId?: string | null;
  onClose: () => void;
  onSearch: (input: SearchRouteLegTransportInput) => void;
  onSelect: (option: TransportOption) => void;
};

const searchableModes = transportModeOptions
  .map((option) => option.value)
  .filter((mode): mode is TransportModeValue => !["walking", "driving", "cycling"].includes(mode));

export function TransportSearchDialog({
  open,
  currency,
  travelers = 1,
  defaultModes,
  options,
  summary,
  loading = false,
  error,
  selectingOptionId,
  onClose,
  onSearch,
  onSelect
}: Props) {
  const initialModes = useMemo<TransportModeValue[]>(
    () => (defaultModes.length > 0 ? defaultModes : ["train", "bus", "car"]),
    [defaultModes]
  );
  const [modes, setModes] = useState<TransportModeValue[]>(initialModes);
  const [date, setDate] = useState("");
  const [time, setTime] = useState("");
  const [timePreference, setTimePreference] = useState<TransportTimePreference | "">("depart_after");
  const [travelerCount, setTravelerCount] = useState(String(travelers));
  const [searchCurrency, setSearchCurrency] = useState(currency);
  const [maxDuration, setMaxDuration] = useState("");
  const [maxPrice, setMaxPrice] = useState("");
  const [sortBy, setSortBy] = useState<"recommended" | "fastest" | "cheapest" | "fewest_transfers">("recommended");
  const sortedOptions = useMemo(() => sortOptions(options, sortBy), [options, sortBy]);

  if (!open) {
    return null;
  }

  function toggleMode(mode: TransportModeValue) {
    setModes((current) =>
      current.includes(mode)
        ? current.filter((item) => item !== mode)
        : [...current, mode]
    );
  }

  function submitSearch() {
    onSearch({
      currency: searchCurrency,
      travelers: Number(travelerCount) > 0 ? Number(travelerCount) : travelers,
      date,
      time,
      timePreference,
      modes: modes.length > 0 ? modes : initialModes,
      constraints: {
        ...(Number(maxDuration) > 0 ? { maxDurationMinutes: Number(maxDuration) } : {}),
        ...(Number(maxPrice) > 0 ? { maxPriceAmount: Number(maxPrice) } : {})
      }
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/40 px-4 py-8" role="dialog" aria-modal="true">
      <div className="w-full max-w-4xl rounded-lg bg-[#FFFDFA] p-5 shadow-xl">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Transport search
            </p>
            <h3 className="mt-1 text-[20px] font-semibold text-cocoa-900">
              Compare route leg options
            </h3>
            {summary ? (
              <p className="mt-1 text-[13px] text-cocoa-500">
                {summary.origin} to {summary.destination}
                {summary.cached ? " | cached" : ""}
                {summary.fallbackUsed ? " | fallback estimates" : ""}
              </p>
            ) : null}
          </div>
          <button
            className="rounded-md px-2 py-1 text-[13px] font-semibold text-cocoa-500 transition hover:bg-sand-100"
            onClick={onClose}
            type="button"
          >
            Close
          </button>
        </div>

        <div className="mt-4 grid gap-3 md:grid-cols-[1.2fr_0.8fr]">
          <div className="rounded-lg border border-sand-300 bg-white p-3">
            <p className="text-[13px] font-semibold text-cocoa-700">Modes</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {searchableModes.map((mode) => (
                <label
                  key={mode}
                  className="inline-flex items-center gap-2 rounded-md border border-sand-300 bg-sand-50 px-2.5 py-1.5 text-[12.5px] font-semibold text-cocoa-600"
                >
                  <input
                    checked={modes.includes(mode)}
                    onChange={() => toggleMode(mode)}
                    type="checkbox"
                  />
                  {transportModeOptions.find((option) => option.value === mode)?.label ?? mode}
                </label>
              ))}
            </div>
          </div>
          <div className="rounded-lg border border-sand-300 bg-white p-3">
            <div className="grid grid-cols-2 gap-2">
              <label className="text-[12.5px] font-semibold text-cocoa-600">
                Date
                <input
                  className="mt-1 h-9 w-full rounded-md border border-sand-300 px-2 text-[13px]"
                  onChange={(event) => setDate(event.target.value)}
                  type="date"
                  value={date}
                />
              </label>
              <label className="text-[12.5px] font-semibold text-cocoa-600">
                Time
                <input
                  className="mt-1 h-9 w-full rounded-md border border-sand-300 px-2 text-[13px]"
                  onChange={(event) => setTime(event.target.value)}
                  type="time"
                  value={time}
                />
              </label>
              <label className="text-[12.5px] font-semibold text-cocoa-600">
                Travelers
                <input
                  className="mt-1 h-9 w-full rounded-md border border-sand-300 px-2 text-[13px]"
                  min={1}
                  onChange={(event) => setTravelerCount(event.target.value)}
                  type="number"
                  value={travelerCount}
                />
              </label>
              <label className="text-[12.5px] font-semibold text-cocoa-600">
                Currency
                <input
                  className="mt-1 h-9 w-full rounded-md border border-sand-300 px-2 text-[13px] uppercase"
                  maxLength={3}
                  onChange={(event) => setSearchCurrency(event.target.value.toUpperCase())}
                  value={searchCurrency}
                />
              </label>
              <label className="text-[12.5px] font-semibold text-cocoa-600">
                Max minutes
                <input
                  className="mt-1 h-9 w-full rounded-md border border-sand-300 px-2 text-[13px]"
                  min={1}
                  onChange={(event) => setMaxDuration(event.target.value)}
                  type="number"
                  value={maxDuration}
                />
              </label>
              <label className="text-[12.5px] font-semibold text-cocoa-600">
                Max price
                <input
                  className="mt-1 h-9 w-full rounded-md border border-sand-300 px-2 text-[13px]"
                  min={1}
                  onChange={(event) => setMaxPrice(event.target.value)}
                  type="number"
                  value={maxPrice}
                />
              </label>
            </div>
            <select
              className="mt-2 h-9 w-full rounded-md border border-sand-300 px-2 text-[13px] text-cocoa-700"
              onChange={(event) => setTimePreference(event.target.value as TransportTimePreference)}
              value={timePreference}
            >
              <option value="depart_after">Depart after</option>
              <option value="arrive_before">Arrive before</option>
              <option value="flexible">Flexible</option>
            </select>
            <button
              className="mt-2 h-9 w-full rounded-md bg-cocoa-900 px-3 text-[13px] font-semibold text-white transition hover:bg-cocoa-700 disabled:opacity-60"
              disabled={loading}
              onClick={submitSearch}
              type="button"
            >
              {loading ? "Searching" : "Search"}
            </button>
          </div>
        </div>

        {error ? (
          <p className="mt-3 rounded-md bg-red-50 px-3 py-2 text-[13px] font-medium text-red-700">
            {error}
          </p>
        ) : null}
        {summary?.warnings?.length ? (
          <div className="mt-3 rounded-lg bg-amber-50 p-3">
            <TransportWarningsList warnings={summary.warnings} />
          </div>
        ) : null}

        <div className="mt-4">
          <CompareTransportModesTable options={options} />
        </div>

        {options.length > 0 ? (
          <div className="mt-3 flex justify-end">
            <select
              className="h-9 rounded-md border border-sand-300 bg-white px-2 text-[13px] text-cocoa-700"
              onChange={(event) => setSortBy(event.target.value as typeof sortBy)}
              value={sortBy}
            >
              <option value="recommended">Recommended</option>
              <option value="fastest">Fastest</option>
              <option value="cheapest">Cheapest</option>
              <option value="fewest_transfers">Fewest transfers</option>
            </select>
          </div>
        ) : null}

        <div className="mt-4 grid gap-3 md:grid-cols-2">
          {sortedOptions.map((option) => (
            <TransportOptionCard
              disabled={Boolean(selectingOptionId)}
              key={option.id}
              option={option}
              selecting={selectingOptionId === option.id}
              onSelect={onSelect}
            />
          ))}
        </div>
        {!loading && options.length === 0 ? (
          <p className="mt-4 rounded-lg border border-dashed border-sand-300 bg-white px-3 py-4 text-center text-[13px] text-cocoa-500">
            No transport options returned for this search.
          </p>
        ) : null}
      </div>
    </div>
  );
}

function sortOptions(
  options: TransportOption[],
  sortBy: "recommended" | "fastest" | "cheapest" | "fewest_transfers"
) {
  const ranked = [...options];
  switch (sortBy) {
    case "fastest":
      return ranked.sort((a, b) => a.durationMinutes - b.durationMinutes);
    case "cheapest":
      return ranked.sort(
        (a, b) =>
          (a.estimatedPrice?.amount ?? Number.MAX_SAFE_INTEGER) -
          (b.estimatedPrice?.amount ?? Number.MAX_SAFE_INTEGER)
      );
    case "fewest_transfers":
      return ranked.sort((a, b) => a.transfers - b.transfers || a.durationMinutes - b.durationMinutes);
    case "recommended":
    default:
      return ranked.sort((a, b) => confidenceRank(b.confidence) - confidenceRank(a.confidence) || a.durationMinutes - b.durationMinutes);
  }
}

function confidenceRank(confidence: string) {
  switch (confidence) {
    case "high":
      return 3;
    case "medium":
      return 2;
    default:
      return 1;
  }
}
