"use client";

import Link from "next/link";
import { useEffect, type MouseEvent, type SyntheticEvent } from "react";
import { useTranslations } from "next-intl";
import { trackAlphaEvent } from "@/lib/api/alpha";
import {
  buildTripWorkspaceHref,
  TRIP_WORKSPACE_SECTIONS,
  type TripWorkspaceSectionId
} from "@/lib/trip-workspace/navigation";
import { cn } from "@/lib/utils";

export function TripWorkspaceNavigation({
  activeSection,
  tripId,
  role,
  lifecycle,
  mobileNavigation = true,
  hasUnsavedChanges = false
}: {
  activeSection: TripWorkspaceSectionId;
  tripId: string;
  role: "owner" | "editor" | "viewer";
  lifecycle?: string | null;
  mobileNavigation?: boolean;
  hasUnsavedChanges?: boolean;
}) {
  const t = useTranslations("tripWorkspace");

  useEffect(() => {
    const metadata = {
      section: activeSection,
      role,
      tripLifecycleState: lifecycle ?? "unknown",
      deviceCategory: window.matchMedia("(max-width: 767px)").matches ? "mobile" : "desktop"
    };
    trackAlphaEvent({
      eventName: "trip_workspace_opened",
      feature: "trip_workspace",
      entityType: "trip",
      entityId: tripId,
      metadata
    });
    trackAlphaEvent({
      eventName: "trip_workspace_section_opened",
      feature: "trip_workspace",
      entityType: "trip",
      entityId: tripId,
      metadata
    });
  }, [activeSection, lifecycle, role, tripId]);

  function trackNavigation(section: TripWorkspaceSectionId, source: "desktop" | "mobile") {
    trackAlphaEvent({
      eventName: "trip_workspace_section_opened",
      feature: "trip_workspace",
      entityType: "trip",
      entityId: tripId,
      metadata: {
        section,
        role,
        tripLifecycleState: lifecycle ?? "unknown",
        deviceCategory: source
      }
    });
  }

  function guardNavigation(event: MouseEvent<HTMLAnchorElement>) {
    if (hasUnsavedChanges && !window.confirm(t("unsavedChangesWarning"))) {
      event.preventDefault();
    }
  }

  function trackMobilePicker(event: SyntheticEvent<HTMLDetailsElement>) {
    if (!event.currentTarget.open) {
      return;
    }
    trackAlphaEvent({
      eventName: "trip_workspace_mobile_section_picker_opened",
      feature: "trip_workspace",
      entityType: "trip",
      entityId: tripId,
      metadata: {
        section: activeSection,
        role,
        tripLifecycleState: lifecycle ?? "unknown",
        deviceCategory: "mobile"
      }
    });
  }

  const activeLabel = t(`sections.${activeSection}`);
  const mobileSections = TRIP_WORKSPACE_SECTIONS.filter((section) => section.mobilePrimary);

  return (
    <>
      <a
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-2 focus:z-50 focus:rounded-md focus:bg-white focus:px-4 focus:py-2 focus:text-cocoa-900 focus:ring-2 focus:ring-primary-600"
        href="#trip-workspace-content"
      >
        {t("skipToContent")}
      </a>
      <nav
        aria-label={t("navigationLabel")}
        className="sticky top-[57px] z-30 mt-6 hidden rounded-[16px] border border-sand-300 bg-sand-50/95 p-1.5 backdrop-blur md:flex"
      >
        {TRIP_WORKSPACE_SECTIONS.map((section) => {
          const active = activeSection === section.id;
          return (
            <Link
              aria-current={active ? "page" : undefined}
              className={cn(
                "inline-flex min-h-11 flex-1 items-center justify-center rounded-xl px-3 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2",
                active
                  ? "bg-cocoa-900 text-sand-50 shadow-sm"
                  : "text-cocoa-500 hover:bg-sand-200 hover:text-cocoa-900"
              )}
              href={buildTripWorkspaceHref(tripId, section.id)}
              key={section.id}
              onClick={(event) => {
                guardNavigation(event);
                if (!event.defaultPrevented) trackNavigation(section.id, "desktop");
              }}
            >
              {t(`sections.${section.labelKey}`)}
            </Link>
          );
        })}
      </nav>

      {mobileNavigation ? (
        <div className="sticky top-[57px] z-30 -mx-4 mt-4 border-y border-sand-300 bg-sand-50/95 px-4 py-2 backdrop-blur md:hidden">
          <div className="flex min-h-11 items-center justify-between gap-3">
            <details className="group relative min-w-0 flex-1" onToggle={trackMobilePicker}>
              <summary
                aria-label={t("chooseSection", { section: activeLabel })}
                className="flex min-h-11 cursor-pointer list-none items-center justify-between rounded-xl border border-sand-300 bg-white px-3 text-sm font-semibold text-cocoa-900 marker:content-none"
              >
                <span className="min-w-0 truncate">
                  <span className="block text-[10px] font-semibold uppercase tracking-[0.08em] text-cocoa-400">
                    {t("currentSection")}
                  </span>
                  {activeLabel}
                </span>
                <span aria-hidden="true" className="ml-2 text-cocoa-400 transition group-open:rotate-180">
                  ⌄
                </span>
              </summary>
              <div className="absolute left-0 top-[calc(100%+0.5rem)] z-40 grid w-[min(22rem,calc(100vw-2rem))] grid-cols-2 gap-2 rounded-2xl border border-sand-300 bg-white p-3 shadow-xl">
                {TRIP_WORKSPACE_SECTIONS.map((section) => {
                  const active = activeSection === section.id;
                  return (
                    <Link
                      aria-current={active ? "page" : undefined}
                      className={cn(
                        "flex min-h-12 items-center rounded-xl px-3 text-sm font-semibold",
                        active ? "bg-cocoa-900 text-white" : "bg-sand-50 text-cocoa-700"
                      )}
                      href={buildTripWorkspaceHref(tripId, section.id)}
                      key={section.id}
                      onClick={(event) => {
                        guardNavigation(event);
                        if (!event.defaultPrevented) trackNavigation(section.id, "mobile");
                      }}
                    >
                      {t(`sections.${section.labelKey}`)}
                    </Link>
                  );
                })}
              </div>
            </details>
            <button
              aria-label={t("openSearch")}
              className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-sand-300 bg-white text-cocoa-700 focus:outline-none focus:ring-2 focus:ring-primary-600"
              onClick={() => window.dispatchEvent(new CustomEvent("travel-ai:open-command-palette"))}
              type="button"
            >
              <SearchIcon />
            </button>
          </div>
        </div>
      ) : null}

      {mobileNavigation ? (
        <nav
          aria-label={t("mobileNavigationLabel")}
          className="fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-sand-300 bg-white/95 px-1 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
        >
          {mobileSections.map((section) => {
            const active = activeSection === section.id || (activeSection === "group" && section.id === "more");
            return (
              <Link
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex min-h-14 items-center justify-center px-1 text-center text-[11px] font-semibold",
                  active ? "text-clay-deep" : "text-cocoa-400"
                )}
                href={buildTripWorkspaceHref(tripId, section.id)}
                key={section.id}
                onClick={(event) => {
                  guardNavigation(event);
                  if (!event.defaultPrevented) trackNavigation(section.id, "mobile");
                }}
              >
                {t(`sections.${section.labelKey}`)}
              </Link>
            );
          })}
        </nav>
      ) : null}
    </>
  );
}

function SearchIcon() {
  return (
    <svg aria-hidden="true" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <circle cx="11" cy="11" r="7" />
      <path d="m16.5 16.5 4 4" />
    </svg>
  );
}
