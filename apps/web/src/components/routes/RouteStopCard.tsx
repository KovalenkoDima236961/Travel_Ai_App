"use client";

import type { TripRouteStop } from "@/entities/route/model";

type RouteStopCardProps = {
  stop: TripRouteStop;
  index: number;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onChange: (stop: TripRouteStop) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onRemove: () => void;
};

const INPUT =
  "h-10 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3 text-[13.5px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint";
const LABEL = "block text-[12.5px] font-semibold text-cocoa-600";

export function RouteStopCard({
  stop,
  index,
  canMoveUp,
  canMoveDown,
  onChange,
  onMoveUp,
  onMoveDown,
  onRemove
}: RouteStopCardProps) {
  return (
    <div className="rounded-[16px] border border-sand-300 bg-white p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Stop {index + 1}
          </p>
          <p className="mt-1 text-[15px] font-semibold text-cocoa-900">
            {stop.destination || "New stop"}
          </p>
        </div>
        <div className="flex gap-1.5">
          <button
            type="button"
            disabled={!canMoveUp}
            onClick={onMoveUp}
            className="h-8 rounded-full px-3 text-[12.5px] font-semibold text-cocoa-500 transition hover:bg-sand-200 disabled:cursor-not-allowed disabled:opacity-40"
          >
            Up
          </button>
          <button
            type="button"
            disabled={!canMoveDown}
            onClick={onMoveDown}
            className="h-8 rounded-full px-3 text-[12.5px] font-semibold text-cocoa-500 transition hover:bg-sand-200 disabled:cursor-not-allowed disabled:opacity-40"
          >
            Down
          </button>
          <button
            type="button"
            onClick={onRemove}
            className="h-8 rounded-full px-3 text-[12.5px] font-semibold text-clay-deep transition hover:bg-clay-tint"
          >
            Remove
          </button>
        </div>
      </div>

      <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label className={LABEL}>
          Destination
          <input
            className={`${INPUT} mt-1.5`}
            value={stop.destination}
            onChange={(event) => onChange({ ...stop, destination: event.target.value })}
            placeholder="Vienna"
          />
        </label>
        <label className={LABEL}>
          Country
          <input
            className={`${INPUT} mt-1.5`}
            value={stop.country ?? ""}
            onChange={(event) => onChange({ ...stop, country: event.target.value })}
            placeholder="Austria"
          />
        </label>
        <label className={LABEL}>
          Arrival
          <input
            type="date"
            className={`${INPUT} mt-1.5`}
            value={stop.arrivalDate ?? ""}
            onChange={(event) => onChange({ ...stop, arrivalDate: event.target.value || null })}
          />
        </label>
        <label className={LABEL}>
          Nights
          <input
            type="number"
            min={0}
            className={`${INPUT} mt-1.5`}
            value={stop.nights ?? 1}
            onChange={(event) => onChange({ ...stop, nights: Number(event.target.value || 0) })}
          />
        </label>
      </div>
    </div>
  );
}
