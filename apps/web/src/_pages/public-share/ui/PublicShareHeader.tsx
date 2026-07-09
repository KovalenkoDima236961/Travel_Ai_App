import Link from "next/link";
import type { ExportTrip } from "@/lib/export/trip-export-adapter";
import { PublicShareExportButton } from "./PublicShareExportButton";
import { GlobeIcon } from "./icons";

type PublicShareHeaderProps = {
  /** Present once the shared trip has loaded; drives the Export control. */
  exportTrip?: ExportTrip | null;
};

export function PublicShareHeader({ exportTrip }: PublicShareHeaderProps) {
  return (
    <header className="border-b border-sand-300 bg-sand-50">
      <div className="mx-auto flex max-w-[1080px] items-center justify-between gap-6 px-6 py-4 sm:px-10">
        <Link href="/" className="flex items-center gap-2.5 text-cocoa-900">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-clay text-sand-100">
            <GlobeIcon className="h-[18px] w-[18px]" />
          </span>
          <span className="font-newsreader text-[19px] font-semibold tracking-[-0.01em]">
            Travel AI Planner
          </span>
        </Link>
        <div className="flex items-center gap-3">
          {exportTrip ? <PublicShareExportButton exportTrip={exportTrip} /> : null}
          <Link
            href="/trips/new"
            className="inline-flex h-10 items-center rounded-full bg-clay px-[18px] text-[13.5px] font-semibold text-sand-100 transition hover:bg-clay-dark"
          >
            Plan your own
          </Link>
        </div>
      </div>
    </header>
  );
}
