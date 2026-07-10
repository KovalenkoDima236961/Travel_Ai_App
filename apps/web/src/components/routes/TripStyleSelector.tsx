"use client";

import type { TripStyle } from "@/entities/route/model";
import { tripStyleOptions } from "./route-options";

type TripStyleSelectorProps = {
  value: TripStyle[];
  onChange: (value: TripStyle[]) => void;
};

export function TripStyleSelector({ value, onChange }: TripStyleSelectorProps) {
  function toggle(style: TripStyle) {
    onChange(value.includes(style) ? value.filter((item) => item !== style) : [...value, style]);
  }

  return (
    <div className="flex flex-wrap gap-2" role="group" aria-label="Trip styles">
      {tripStyleOptions.map((option) => {
        const selected = value.includes(option.value);
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={selected}
            onClick={() => toggle(option.value)}
            className={
              selected
                ? "h-9 rounded-full border border-cocoa-900 bg-cocoa-900 px-3.5 text-[13px] font-semibold text-sand-100"
                : "h-9 rounded-full border border-sand-400 bg-white px-3.5 text-[13px] font-medium text-cocoa-500 transition hover:border-sand-600 hover:text-cocoa-900"
            }
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
