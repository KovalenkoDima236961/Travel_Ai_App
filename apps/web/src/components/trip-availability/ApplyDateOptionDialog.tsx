"use client";

import { useState } from "react";
import { Button } from "@/shared/ui/button";
import type { TripDateOption } from "@/types/trip-availability";

type ApplyDateOptionDialogProps = {
  hasItinerary: boolean;
  isPending?: boolean;
  open: boolean;
  option: TripDateOption | null;
  onApply: (regenerateItinerary: boolean) => void;
  onOpenChange: (open: boolean) => void;
};

export function ApplyDateOptionDialog({
  hasItinerary,
  isPending = false,
  open,
  option,
  onApply,
  onOpenChange
}: ApplyDateOptionDialogProps) {
  const [regenerateItinerary, setRegenerateItinerary] = useState(false);

  if (!open || !option) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 px-4">
      <div className="w-full max-w-md rounded-[18px] border border-sand-300 bg-white p-5 shadow-xl">
        <h3 className="font-newsreader text-[24px] font-semibold text-cocoa-900">
          Apply dates
        </h3>
        <p className="mt-2 text-[14px] text-cocoa-600">
          This will update the trip dates to {option.startDate} - {option.endDate}.
          Existing itinerary content may become outdated.
        </p>
        {hasItinerary ? (
          <label className="mt-4 flex items-start gap-2 text-[14px] text-cocoa-600">
            <input
              checked={regenerateItinerary}
              className="mt-1 h-4 w-4"
              disabled={isPending}
              onChange={(event) => setRegenerateItinerary(event.target.checked)}
              type="checkbox"
            />
            <span>Regenerate itinerary after applying</span>
          </label>
        ) : null}
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <Button
            disabled={isPending}
            onClick={() => onOpenChange(false)}
            type="button"
            variant="ghost"
          >
            Cancel
          </Button>
          <Button
            disabled={isPending}
            onClick={() => onApply(hasItinerary && regenerateItinerary)}
            type="button"
          >
            {isPending ? "Applying..." : "Apply dates"}
          </Button>
        </div>
      </div>
    </div>
  );
}
