"use client";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import type { AvailabilityDateRange } from "@/types/trip-availability";

type DateRangeListEditorProps = {
  disabled?: boolean;
  emptyLabel: string;
  ranges: AvailabilityDateRange[];
  title: string;
  onChange: (ranges: AvailabilityDateRange[]) => void;
};

export function DateRangeListEditor({
  disabled = false,
  emptyLabel,
  ranges,
  title,
  onChange
}: DateRangeListEditorProps) {
  function updateRange(index: number, patch: Partial<AvailabilityDateRange>) {
    onChange(ranges.map((range, idx) => (idx === index ? { ...range, ...patch } : range)));
  }

  function removeRange(index: number) {
    onChange(ranges.filter((_, idx) => idx !== index));
  }

  return (
    <div className="rounded-[14px] border border-sand-300 bg-sand-50 p-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-[13px] font-semibold text-cocoa-800">{title}</h3>
        <Button
          disabled={disabled}
          onClick={() => onChange([...ranges, { startDate: "", endDate: "" }])}
          size="sm"
          type="button"
          variant="secondary"
        >
          Add range
        </Button>
      </div>
      <div className="mt-3 space-y-2">
        {ranges.length > 0 ? (
          ranges.map((range, index) => (
            <div
              key={`range-${index}`}
              className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
            >
              <Input
                aria-label={`${title} start date`}
                disabled={disabled}
                onChange={(event) => updateRange(index, { startDate: event.target.value })}
                type="date"
                value={range.startDate}
              />
              <Input
                aria-label={`${title} end date`}
                disabled={disabled}
                onChange={(event) => updateRange(index, { endDate: event.target.value })}
                type="date"
                value={range.endDate}
              />
              <Button
                disabled={disabled}
                onClick={() => removeRange(index)}
                size="sm"
                type="button"
                variant="ghost"
              >
                Remove
              </Button>
            </div>
          ))
        ) : (
          <p className="rounded-[12px] bg-white px-3 py-2 text-[13px] text-cocoa-400">
            {emptyLabel}
          </p>
        )}
      </div>
    </div>
  );
}
