import Link from "next/link";
import { AccountMenu } from "@/components/layout/AccountMenu";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { GlobeIcon } from "./icons";

const NAV_BASE =
  "inline-flex h-[38px] items-center rounded-full px-4 text-[14.5px] transition";
const NAV_IDLE = "font-medium text-cocoa-500 hover:bg-sand-200 hover:text-cocoa-900";

/**
 * Redesigned app chrome for the Notifications screen. Like TripsHeader it ships
 * with the slice (AppHeader is suppressed for `/notifications`) so the warm
 * palette does not leak onto pages that still render the old slate header. The
 * live `NotificationBell` and `AccountMenu` chrome widgets are reused as-is.
 */
export function NotificationsHeader() {
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
            <Link href="/trips" className={`${NAV_BASE} ${NAV_IDLE}`}>
              Trips
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
          <NotificationBell />
          <AccountMenu />
        </div>
      </div>
    </header>
  );
}
