"use client";

import type { TripRoute } from "@/entities/route/model";

type RouteAlternativeMapPreviewProps = {
  route: TripRoute;
  selected?: boolean;
};

export function RouteAlternativeMapPreview({ route, selected = false }: RouteAlternativeMapPreviewProps) {
  const stops = route.stops ?? [];
  if (stops.length === 0) {
    return null;
  }

  return (
    <div
      className={
        selected
          ? "rounded-[14px] border border-clay bg-clay-tint/40 p-3"
          : "rounded-[14px] border border-sand-300 bg-sand-50 p-3"
      }
    >
      <div className="relative min-h-[88px] overflow-hidden rounded-[10px] bg-[#F7F1E8] px-4 py-5">
        <div className="absolute inset-x-7 top-1/2 h-px -translate-y-1/2 bg-cocoa-300/60" />
        <ol
          className="relative grid items-center gap-2"
          style={{ gridTemplateColumns: `repeat(${Math.max(1, stops.length)}, minmax(0, 1fr))` }}
        >
          {stops.map((stop, index) => (
            <li key={stop.id || `${stop.destination}-${index}`} className="min-w-0 text-center">
              <span className="mx-auto flex h-8 w-8 items-center justify-center rounded-full bg-cocoa-900 text-[12px] font-semibold text-sand-100 shadow-sm">
                {index + 1}
              </span>
              <span className="mt-2 block truncate text-[11px] font-semibold text-cocoa-700">
                {stop.city || stop.destination}
              </span>
            </li>
          ))}
        </ol>
      </div>
    </div>
  );
}
