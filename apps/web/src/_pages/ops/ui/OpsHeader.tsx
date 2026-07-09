import Link from "next/link";
import { OUTLINE_BUTTON } from "./opsStyles";
import { RefreshIcon, WrenchScrewdriverIcon } from "./icons";

/**
 * Redesigned chrome for the internal Ops Dashboard (Ops.dc.html). Unlike the
 * other authenticated slice headers this one intentionally does NOT wear the
 * "Travel AI Planner" app chrome — the design gives Ops its own identity (dark
 * wrench logo tile, "Ops Dashboard" name, INTERNAL badge) with no nav and a
 * single Refresh action. It deliberately omits the shared NotificationBell /
 * AccountMenu that the other slice headers reuse; the only navigation affordance
 * is the brand link back to `/`. Refresh invalidates every `ops` query.
 */
export function OpsHeader({ onRefresh }: { onRefresh: () => void }) {
  return (
    <header className="sticky top-0 z-40 border-b border-sand-300 bg-sand-50/95 backdrop-blur">
      <div className="mx-auto flex max-w-[1360px] items-center justify-between gap-6 px-6 py-3 sm:px-10">
        <Link href="/" className="flex items-center gap-2.5 text-cocoa-900">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-cocoa-900 text-sand-100">
            <WrenchScrewdriverIcon className="h-[17px] w-[17px]" />
          </span>
          <span className="font-newsreader text-[19px] font-semibold tracking-[-0.01em]">
            Ops Dashboard
          </span>
          <span className="ml-1 rounded-full bg-sand-200 px-2.5 py-[3px] text-[11px] font-semibold tracking-[0.05em] text-cocoa-400">
            INTERNAL
          </span>
        </Link>
        <button type="button" className={OUTLINE_BUTTON} onClick={onRefresh}>
          <RefreshIcon className="h-[15px] w-[15px]" />
          Refresh
        </button>
      </div>
    </header>
  );
}
