"use client";

import { FormEvent, useEffect, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import type { RouteAlternative } from "@/types/route-alternatives";

type ApplyRouteAlternativeDialogProps = {
  alternative: RouteAlternative | null;
  currentRevision?: number;
  isPending?: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: (input: { expectedItineraryRevision?: number; regenerateItinerary: boolean }) => void;
};

export function ApplyRouteAlternativeDialog({
  alternative,
  currentRevision,
  isPending = false,
  error = null,
  onClose,
  onConfirm
}: ApplyRouteAlternativeDialogProps) {
  const [expectedRevision, setExpectedRevision] = useState("");
  const [regenerateItinerary, setRegenerateItinerary] = useState(false);

  useEffect(() => {
    setExpectedRevision(currentRevision != null ? String(currentRevision) : "");
  }, [currentRevision, alternative?.id]);

  if (!alternative) {
    return null;
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const revision = expectedRevision === "" ? undefined : Number(expectedRevision);
    onConfirm({
      expectedItineraryRevision: revision != null && Number.isFinite(revision) ? revision : undefined,
      regenerateItinerary
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-cocoa-900/55 p-4 backdrop-blur-sm">
      <div className="w-full max-w-lg rounded-[20px] bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-[11px] font-bold uppercase tracking-[0.12em] text-clay">
              Apply route
            </p>
            <h2 className="mt-1 font-newsreader text-[27px] font-semibold text-cocoa-900">
              {alternative.title}
            </h2>
          </div>
          <Button type="button" variant="ghost" onClick={onClose}>
            Close
          </Button>
        </div>

        <p className="mt-4 rounded-[12px] border border-amber-200 bg-amber-50 px-3 py-2 text-[13px] leading-5 text-amber-900">
          This replaces the current trip route. Existing itinerary details may no longer match the route.
        </p>

        <form className="mt-5 space-y-4" onSubmit={submit}>
          <label className="block text-sm font-semibold text-cocoa-700">
            Expected itinerary revision
            <Input
              type="number"
              min={0}
              value={expectedRevision}
              onChange={(event) => setExpectedRevision(event.target.value)}
            />
          </label>
          <label className="flex items-center gap-2 text-sm font-medium text-cocoa-700">
            <input
              checked={regenerateItinerary}
              onChange={(event) => setRegenerateItinerary(event.target.checked)}
              type="checkbox"
            />
            Regenerate itinerary after applying
          </label>
          {error ? (
            <div className="rounded-[12px] border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">
              {error}
            </div>
          ) : null}
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={onClose} disabled={isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Applying..." : "Apply route"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
