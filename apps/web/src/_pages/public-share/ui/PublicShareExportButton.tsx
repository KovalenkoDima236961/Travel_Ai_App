"use client";

import { useEffect, useRef, useState } from "react";
import { downloadTextFile } from "@/lib/export/download";
import { buildIcsFilename } from "@/lib/export/export-filenames";
import { generateTripIcs, getTripIcsEventCount } from "@/lib/export/ics";
import { downloadTripPdf } from "@/lib/export/pdf";
import type { ExportTrip } from "@/lib/export/trip-export-adapter";
import { CalendarIcon, DocumentIcon, ExportIcon } from "./icons";

type PublicShareExportButtonProps = {
  exportTrip: ExportTrip;
};

/**
 * Slice-local Export control matching the mock's single pill. Reuses the shared
 * export library directly (PDF + ICS), so it preserves every behavior of the
 * shared ExportTripMenu — PDF loading state, the PDF-fail → "Print page"
 * fallback, and the "no timed items" calendar message — without restyling that
 * shared component.
 */
export function PublicShareExportButton({ exportTrip }: PublicShareExportButtonProps) {
  const [open, setOpen] = useState(false);
  const [isPdfLoading, setIsPdfLoading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [pdfFailed, setPdfFailed] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const hasItinerary = Boolean(exportTrip.itinerary?.days?.length);
  const pdfDisabled = !hasItinerary || isPdfLoading;
  const calendarDisabled = !hasItinerary || !exportTrip.startDate;

  useEffect(() => {
    if (!open) {
      return;
    }
    function handlePointer(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    function handleKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handlePointer);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handlePointer);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open]);

  async function handlePdfDownload() {
    try {
      setMessage(null);
      setPdfFailed(false);
      setIsPdfLoading(true);
      await downloadTripPdf(exportTrip);
      setOpen(false);
    } catch {
      setPdfFailed(true);
      setMessage(
        "PDF export failed. You can still use your browser's Print option from this page."
      );
    } finally {
      setIsPdfLoading(false);
    }
  }

  function handleCalendarDownload() {
    setMessage(null);
    setPdfFailed(false);

    if (getTripIcsEventCount(exportTrip) === 0) {
      setMessage("No timed itinerary items found to export.");
      return;
    }

    downloadTextFile(
      generateTripIcs(exportTrip),
      buildIcsFilename(exportTrip),
      "text/calendar;charset=utf-8"
    );
    setOpen(false);
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        className="inline-flex h-10 items-center gap-2 rounded-full border border-sand-400 bg-white px-4 text-[13.5px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
      >
        <ExportIcon className="h-[15px] w-[15px]" />
        {isPdfLoading ? "Preparing…" : "Export"}
      </button>

      {open ? (
        <div
          role="menu"
          className="absolute right-0 z-20 mt-2 w-[240px] rounded-[16px] border border-sand-300 bg-white p-1.5 shadow-[0_18px_44px_rgba(34,26,20,0.16)]"
        >
          <button
            type="button"
            role="menuitem"
            disabled={pdfDisabled}
            onClick={handlePdfDownload}
            className="flex w-full items-center gap-3 rounded-[11px] px-3 py-2.5 text-left text-[13.5px] font-medium text-cocoa-700 transition hover:bg-sand-100 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-transparent"
          >
            <DocumentIcon className="h-[17px] w-[17px] text-[#A08D78]" />
            {isPdfLoading ? "Preparing PDF…" : "Download PDF"}
          </button>
          <button
            type="button"
            role="menuitem"
            disabled={calendarDisabled}
            onClick={handleCalendarDownload}
            className="flex w-full items-center gap-3 rounded-[11px] px-3 py-2.5 text-left text-[13.5px] font-medium text-cocoa-700 transition hover:bg-sand-100 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-transparent"
          >
            <CalendarIcon className="h-[17px] w-[17px] text-[#A08D78]" />
            Download calendar (.ics)
          </button>
          {message ? (
            <p className="mx-1 mt-1 rounded-[10px] bg-[#FDF0E3] px-3 py-2 text-[12.5px] leading-[1.5] text-[#96682A]">
              {message}
            </p>
          ) : null}
          {pdfFailed ? (
            <button
              type="button"
              role="menuitem"
              onClick={() => window.print()}
              className="mt-1 flex w-full items-center gap-3 rounded-[11px] px-3 py-2.5 text-left text-[13.5px] font-medium text-clay-deep transition hover:bg-sand-100"
            >
              Print page
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
