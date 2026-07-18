import Link from "next/link";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { AccountMenu } from "@/components/layout/AccountMenu";
import { GlobeIcon, PlusIcon } from "./icons";
import { ScopeSelect } from "./ScopeSelect";

const NAV_BASE =
  "inline-flex h-[38px] items-center rounded-full px-4 text-[14.5px] transition";
const NAV_ACTIVE = "bg-sand-200 font-semibold text-cocoa-900";
const NAV_IDLE = "font-medium text-cocoa-500 hover:bg-sand-200 hover:text-cocoa-900";

/**
 * Redesigned app chrome for the Trips screen. It ships with the slice (AppHeader
 * is suppressed for `/trips`) so the warm palette does not leak onto the app
 * pages that still render the old slate header.
 */
export function TripsHeader() {
  return (
    <header className="sticky top-0 z-40 border-b border-sand-300 bg-sand-50/95 backdrop-blur">
      <div className="mx-auto flex max-w-[1280px] items-center justify-between gap-3 px-4 py-2 sm:gap-6 sm:px-10 sm:py-3">
        <div className="flex min-w-0 items-center gap-6 lg:gap-9">
          <Link aria-label="Travel AI Planner" href="/" className="flex shrink-0 items-center gap-2.5 text-cocoa-900">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-clay text-sand-100">
              <GlobeIcon className="h-[18px] w-[18px]" />
            </span>
            <span className="hidden font-newsreader text-[19px] font-semibold tracking-[-0.01em] sm:inline">
              Travel AI Planner
            </span>
          </Link>
          <nav className="hidden items-center gap-1 md:flex">
            <Link href="/trips" aria-current="page" className={`${NAV_BASE} ${NAV_ACTIVE}`}>
              Trips
            </Link>
            <Link href="/library" className={`${NAV_BASE} ${NAV_IDLE}`}>
              Library
            </Link>
            <Link href="/templates" className={`${NAV_BASE} ${NAV_IDLE}`}>
              Templates
            </Link>
            <Link href="/workspaces" className={`${NAV_BASE} ${NAV_IDLE}`}>
              Workspaces
            </Link>
          </nav>
        </div>
        <div className="flex shrink-0 items-center gap-1 sm:gap-3">
          <div className="hidden sm:block">
            <ScopeSelect />
          </div>
          <NotificationBell />
          <Link
            href="/trips/new"
            className="inline-flex h-11 items-center gap-2 rounded-full bg-clay px-3 text-[14px] font-semibold text-sand-100 transition hover:bg-clay-dark sm:px-[18px]"
          >
            <PlusIcon className="h-[15px] w-[15px]" />
            <span className="hidden sm:inline">New trip</span>
          </Link>
          <AccountMenu />
        </div>
      </div>
    </header>
  );
}
