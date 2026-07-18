"use client";

import { useState } from "react";
import { Button } from "@/shared/ui/button";
import { downloadTextFile } from "@/lib/export/download";
import { buildIcsFilename } from "@/lib/export/export-filenames";
import { generateTripIcs, getTripIcsEventCount } from "@/lib/export/ics";
import type { ExportTrip } from "@/lib/export/trip-export-adapter";
import { cn } from "@/lib/utils";

type ExportTripMenuProps = {
  exportTrip: ExportTrip;
  disabled?: boolean;
  className?: string;
};

export function ExportTripMenu({ exportTrip, disabled = false, className }: ExportTripMenuProps) {
  const [isPdfLoading, setIsPdfLoading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const hasItinerary = Boolean(exportTrip.itinerary?.days?.length);
  const pdfDisabled = disabled || !hasItinerary || isPdfLoading;
  const calendarDisabled = disabled || !hasItinerary || !exportTrip.startDate;

  async function handlePdfDownload() {
    try {
      setMessage(null);
      setIsPdfLoading(true);
      // The PDF line builder is only useful after the explicit export action.
      const { downloadTripPdf } = await import("@/lib/export/pdf");
      await downloadTripPdf(exportTrip);
    } catch {
      setMessage("PDF export failed. You can still use your browser's Print option from this page.");
    } finally {
      setIsPdfLoading(false);
    }
  }

  function handleCalendarDownload() {
    setMessage(null);

    if (getTripIcsEventCount(exportTrip) === 0) {
      setMessage("No timed itinerary items found to export.");
      return;
    }

    downloadTextFile(
      generateTripIcs(exportTrip),
      buildIcsFilename(exportTrip),
      "text/calendar;charset=utf-8"
    );
  }

  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        <Button disabled={pdfDisabled} onClick={handlePdfDownload} type="button" variant="secondary">
          {isPdfLoading ? "Preparing PDF..." : "Download PDF"}
        </Button>
        <Button
          disabled={calendarDisabled}
          onClick={handleCalendarDownload}
          type="button"
          variant="secondary"
        >
          Download calendar (.ics)
        </Button>
        {message?.startsWith("PDF export failed") ? (
          <Button onClick={() => window.print()} type="button" variant="ghost">
            Print page
          </Button>
        ) : null}
      </div>
      {message ? (
        <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
          {message}
        </p>
      ) : null}
    </div>
  );
}
