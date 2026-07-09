"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/components/auth/AuthProvider";
import { Button, buttonStyles } from "@/shared/ui/button";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { WorkspaceSwitcher } from "@/components/workspaces/WorkspaceSwitcher";

export function AppHeader() {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, isLoading, logout, user } = useAuth();

  async function handleLogout() {
    await logout();
    router.push("/login");
  }

  // The landing, auth, and redesigned trips/create-trip/templates/notifications/
  // settings/offline/ops/workspaces screens are full-bleed with their own
  // branding/chrome, so suppress the shared app header on those routes. Most paths are matched
  // exactly; the redesigned Trip Detail screen lives at the dynamic `/trips/[id]`
  // route, so it is matched by a single-segment pattern that intentionally
  // excludes `/trips/new` (its own chrome). The redesigned Trip Cost Analytics
  // screen lives at `/trips/[id]/analytics` and ships its own chrome, matched by
  // its own pattern. `/templates` is matched exactly for the list screen; the
  // redesigned Template Detail screen lives at the dynamic `/templates/[id]`
  // route, matched by its own single-segment pattern. `/workspaces` is
  // matched exactly for the list screen; the redesigned Workspace Detail screen
  // lives at the dynamic `/workspaces/[id]` route, matched by the same
  // single-segment pattern that excludes `/workspaces/new` (still the old chrome)
  // and the nested `/workspaces/[id]/settings|analytics|budgets|...` routes.
  const isRedesignedTripDetail =
    /^\/trips\/[^/]+$/.test(pathname) && pathname !== "/trips/new";
  const isRedesignedTripAnalytics = /^\/trips\/[^/]+\/analytics$/.test(pathname);
  const isRedesignedTemplateDetail = /^\/templates\/[^/]+$/.test(pathname);
  const isRedesignedWorkspaceDetail =
    /^\/workspaces\/[^/]+$/.test(pathname) && pathname !== "/workspaces/new";
  // The redesigned Public Share screen lives at `/share/[shareToken]` and ships
  // its own full-bleed chrome (brand + Export + "Plan your own"), so suppress the
  // shared app header there for both anonymous and signed-in viewers.
  const isRedesignedPublicShare = /^\/share\/[^/]+$/.test(pathname);
  if (
    pathname === "/" ||
    pathname === "/login" ||
    pathname === "/register" ||
    pathname === "/trips" ||
    pathname === "/trips/new" ||
    isRedesignedTripDetail ||
    isRedesignedTripAnalytics ||
    pathname === "/templates" ||
    isRedesignedTemplateDetail ||
    pathname === "/notifications" ||
    pathname === "/settings" ||
    pathname === "/offline" ||
    pathname === "/ops" ||
    pathname === "/workspaces" ||
    isRedesignedWorkspaceDetail ||
    isRedesignedPublicShare
  ) {
    return null;
  }

  return (
    <header className="border-b border-slate-200 bg-white/95">
      <div className="mx-auto flex min-h-16 max-w-6xl flex-wrap items-center justify-between gap-4 px-4 py-3 sm:px-6 lg:px-8">
        <Link className="text-base font-semibold text-slate-950" href="/">
          Travel AI Planner
        </Link>
        <nav className="flex flex-wrap items-center justify-end gap-2">
          {!isLoading && isAuthenticated ? (
            <>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/trips">
                Trips
              </Link>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/templates">
                Templates
              </Link>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/offline-trips">
                Offline Trips
              </Link>
              <WorkspaceSwitcher />
              <Link className={buttonStyles({ size: "sm" })} href="/trips/new">
                Create Trip
              </Link>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/settings">
                Settings
              </Link>
              <NotificationBell />
              <span className="hidden max-w-40 truncate text-sm text-slate-600 lg:inline">
                {user?.email}
              </span>
              <Button size="sm" variant="secondary" onClick={handleLogout}>
                Logout
              </Button>
            </>
          ) : null}
          {!isLoading && !isAuthenticated ? (
            <>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/login">
                Login
              </Link>
              <Link className={buttonStyles({ size: "sm" })} href="/register">
                Register
              </Link>
            </>
          ) : null}
        </nav>
      </div>
    </header>
  );
}
