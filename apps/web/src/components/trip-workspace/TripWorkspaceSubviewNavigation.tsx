"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { trackAlphaEvent } from "@/lib/api/alpha";
import {
  buildTripWorkspaceHref,
  type TripWorkspaceSectionId
} from "@/lib/trip-workspace/navigation";
import { cn } from "@/lib/utils";

const VIEWS: Readonly<Record<TripWorkspaceSectionId, readonly string[]>> = {
  overview: [],
  plan: ["itinerary", "agenda", "timeline", "calendar", "route", "stay", "map", "verification"],
  money: ["overview", "budget", "expenses", "receipts", "settlements", "splits"],
  group: ["people", "discussion", "decisions", "availability", "approvals", "activity"],
  prepare: ["checklist", "reminders", "offline"],
  more: ["tools", "sharing", "exports", "versions", "health", "policy", "settings"]
};

export function TripWorkspaceSubviewNavigation({
  section,
  view,
  tripId,
  role,
  hasUnsavedChanges = false
}: {
  section: TripWorkspaceSectionId;
  view: string;
  tripId: string;
  role: "owner" | "editor" | "viewer";
  hasUnsavedChanges?: boolean;
}) {
  const t = useTranslations("tripWorkspace");
  const router = useRouter();
  const views = VIEWS[section];
  if (views.length === 0) {
    return null;
  }

  function hrefFor(nextView: string) {
    return buildTripWorkspaceHref(tripId, section, { view: nextView });
  }

  function track(nextView: string) {
    trackAlphaEvent({
      eventName: "trip_workspace_subview_opened",
      feature: "trip_workspace",
      entityType: "trip",
      entityId: tripId,
      metadata: { section, subview: nextView, role }
    });
  }

  return (
    <>
      <div className="mt-3 md:hidden">
        <label className="sr-only" htmlFor="trip-workspace-subview">
          {t("subviewLabel")}
        </label>
        <select
          className="min-h-11 w-full rounded-xl border border-sand-300 bg-white px-3 text-sm font-semibold text-cocoa-800"
          id="trip-workspace-subview"
          onChange={(event) => {
            if (hasUnsavedChanges && !window.confirm(t("unsavedChangesWarning"))) {
              event.target.value = view;
              return;
            }
            track(event.target.value);
            router.push(hrefFor(event.target.value));
          }}
          value={view}
        >
          {views.map((candidate) => (
            <option key={candidate} value={candidate}>
              {t(`views.${candidate}`)}
            </option>
          ))}
        </select>
      </div>
      <nav aria-label={t("subviewLabel")} className="mt-3 hidden flex-wrap gap-2 md:flex">
        {views.map((candidate) => {
          const active = candidate === view;
          return (
            <Link
              aria-current={active ? "page" : undefined}
              className={cn(
                "inline-flex min-h-10 items-center rounded-full border px-4 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-primary-600",
                active
                  ? "border-cocoa-900 bg-cocoa-900 text-white"
                  : "border-sand-300 bg-white text-cocoa-600 hover:border-sand-500 hover:text-cocoa-900"
              )}
              href={hrefFor(candidate)}
              key={candidate}
              onClick={(event) => {
                if (hasUnsavedChanges && !window.confirm(t("unsavedChangesWarning"))) {
                  event.preventDefault();
                  return;
                }
                track(candidate);
              }}
            >
              {t(`views.${candidate}`)}
            </Link>
          );
        })}
      </nav>
    </>
  );
}
