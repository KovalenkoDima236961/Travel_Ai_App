"use client";

import type { TransportMode } from "@/entities/route/model";
import { transportModeOptions } from "./route-options";

type TransportModeSelectorProps = {
  value: TransportMode;
  onChange: (value: TransportMode) => void;
};

export function TransportModeSelector({ value, onChange }: TransportModeSelectorProps) {
  return (
    <div className="flex flex-wrap gap-2" role="group" aria-label="Transport mode">
      {transportModeOptions.map((option) => (
        <button
          key={option.value}
          type="button"
          aria-pressed={value === option.value}
          onClick={() => onChange(option.value)}
          className={
            value === option.value
              ? "h-9 rounded-full border border-clay bg-clay-tint px-3.5 text-[13px] font-semibold text-clay-deep"
              : "h-9 rounded-full border border-sand-400 bg-white px-3.5 text-[13px] font-medium text-cocoa-500 transition hover:border-sand-600 hover:text-cocoa-900"
          }
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
