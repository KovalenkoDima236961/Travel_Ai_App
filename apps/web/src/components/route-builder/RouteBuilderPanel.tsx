"use client";

import { useEffect, useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError } from "@/shared/api/client";
import { TransportModeSelector } from "@/components/routes/TransportModeSelector";
import type { TransportMode, TripRoute, TripRouteStop } from "@/entities/route/model";
import type { Trip } from "@/entities/trip/model";
import { useUpdateTripRoute } from "@/lib/api/trip-routes";
import {
  cloneRoute,
  createRouteDraft,
  markSelectedTransportStale,
  rebuildRouteLegs,
  removeRouteStop,
  reorderRouteStops,
  updateRouteDraft,
  updateRouteStop
} from "@/lib/route-builder/route-draft";
import { getRouteBuilderIssues, type RouteBuilderIssue } from "@/lib/route-builder/route-validation";
import { getErrorMessage } from "@/lib/utils";
import type { TripHealth } from "@/types/trip-health";
import { RouteImpactPreviewDialog } from "./RouteImpactPreviewDialog";
import { RouteMetricsCard } from "./RouteMetricsCard";
import { RouteTimeline } from "./RouteTimeline";
import { RouteValidationPanel } from "./RouteValidationPanel";
import { SortableRouteStopList } from "./SortableRouteStopList";
import { StopDayMapping } from "./StopDayMapping";

type RouteBuilderPanelProps = {
  trip: Trip;
  health?: TripHealth | null;
  canEdit?: boolean;
  online?: boolean;
};

const INPUT =
  "mt-1.5 h-10 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3 text-[13px] text-cocoa-900 outline-none transition focus:border-clay focus:ring-[3px] focus:ring-clay-tint";

export function RouteBuilderPanel({
  trip,
  health,
  canEdit = false,
  online = true
}: RouteBuilderPanelProps) {
  const t = useTranslations("route");
  const baseline = useMemo(() => trip.route ?? emptyRoute(), [trip.route]);
  const [draft, setDraft] = useState(() => createRouteDraft(baseline, baseline, trip.itinerary));
  const [editing, setEditing] = useState(false);
  const [editingStopId, setEditingStopId] = useState<string | null>(null);
  const [impactOpen, setImpactOpen] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const mutation = useUpdateTripRoute(trip.id);

  useEffect(() => {
    if (!draft.dirty) {
      setDraft(createRouteDraft(baseline, baseline, trip.itinerary));
    }
  }, [baseline, trip.itinerary, trip.itineraryRevision]);

  useUnsavedRouteWarning(draft.dirty);

  const visibleRoute = editing ? draft.draftRoute : trip.route;
  const issues = useMemo(
    () =>
      getRouteBuilderIssues({
        route: visibleRoute ?? emptyRoute(),
        itinerary: trip.itinerary,
        healthIssues: health?.issues,
        totalDays: trip.days,
        tripId: trip.id,
        multiDestination: trip.tripType === "multi_destination"
      }),
    [health?.issues, trip.days, trip.id, trip.itinerary, trip.tripType, visibleRoute]
  );
  const blockingIssues = issues.filter(
    (issue) =>
      issue.source === "draft" &&
      issue.severity === "error" &&
      (issue.id.startsWith("route_missing") ||
        issue.id.startsWith("missing_stop_name") ||
        issue.id.startsWith("duplicate_stop") ||
        issue.id.startsWith("route_incomplete") ||
        issue.id.startsWith("missing_leg_mode"))
  );

  function beginEditing() {
    if (!canEdit || !online) {
      return;
    }
    const start = trip.route ?? initialRouteForTrip(trip);
    setDraft(createRouteDraft(baseline, start, trip.itinerary));
    setEditing(true);
    setSaveError(null);
  }

  function changeDraft(nextRoute: TripRoute) {
    setDraft((current) => updateRouteDraft(current, nextRoute, trip.itinerary));
    setSaveError(null);
  }

  function discardChanges() {
    setDraft(createRouteDraft(baseline, baseline, trip.itinerary));
    setEditing(false);
    setEditingStopId(null);
    setImpactOpen(false);
    setSaveError(null);
  }

  function previewSave() {
    if (blockingIssues.length > 0) {
      setSaveError(t("resolveValidationBeforeSaving"));
      document.getElementById("route-validation")?.scrollIntoView({ block: "start" });
      return;
    }
    setImpactOpen(true);
  }

  function saveRoute() {
    mutation.mutate(
      {
        expectedItineraryRevision: trip.itineraryRevision,
        route: draft.draftRoute
      },
      {
        onSuccess: (updatedTrip) => {
          const savedRoute = updatedTrip.route ?? draft.draftRoute;
          setDraft(createRouteDraft(savedRoute, savedRoute, updatedTrip.itinerary));
          setEditing(false);
          setEditingStopId(null);
          setImpactOpen(false);
          setSaveError(null);
        },
        onError: (error) => {
          if (error instanceof ApiError && error.status === 409) {
            setSaveError(t("routeConflict"));
            return;
          }
          setSaveError(getErrorMessage(error, t("saveFailed")));
        }
      }
    );
  }

  function addStop() {
    const stop: TripRouteStop = {
      id: `stop_${Date.now().toString(36)}`,
      destination: "",
      country: "",
      nights: 1,
      accommodationHint: "unknown"
    };
    changeDraft(rebuildRouteLegs(draft.draftRoute, [...draft.draftRoute.stops, stop]));
    setEditingStopId(stop.id);
  }

  function updateStop(nextStop: TripRouteStop) {
    changeDraft(updateRouteStop(draft.draftRoute, nextStop.id, nextStop));
  }

  function updateLegMode(index: number, mode: TransportMode) {
    changeDraft({
      ...cloneRoute(draft.draftRoute),
      legs: (draft.draftRoute.legs ?? []).map((leg, legIndex) =>
        legIndex === index
          ? markSelectedTransportStale({ ...leg, mode }, "The transport mode changed.")
          : leg
      )
    });
  }

  function handleIssueAction(issue: RouteBuilderIssue) {
    if (issue.action?.type === "edit_stop" && issue.stopId) {
      if (!editing) {
        beginEditing();
      }
      setEditingStopId(issue.stopId);
      window.requestAnimationFrame(() => document.getElementById("route-stop-editor")?.scrollIntoView({ block: "center" }));
      return;
    }
    if (issue.action?.type === "find_transport" && issue.legId) {
      if (draft.dirty) {
        setSaveError(t("saveBeforeTransportSearch"));
        return;
      }
      const anchor = document.getElementById(`route-leg-${issue.legId}`);
      anchor?.scrollIntoView({ block: "center" });
      window.requestAnimationFrame(() => {
        anchor?.querySelector<HTMLButtonElement>("[data-route-transport-trigger]")?.click();
      });
      return;
    }
    if (issue.action?.href) {
      if (draft.dirty && !window.confirm("Discard unsaved route changes and leave this page?")) {
        return;
      }
      window.location.assign(issue.action.href);
    }
  }

  const selectedStop = draft.draftRoute.stops.find((stop) => stop.id === editingStopId);

  return (
    <section className="w-full space-y-4">
      <div className="rounded-[20px] border border-sand-300 bg-white p-5 shadow-[0_1px_2px_rgba(34,26,20,0.04)]">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">{t("builder")}</p>
            <h2 className="mt-1 font-newsreader text-[25px] font-semibold text-cocoa-900">{t("timeline")}</h2>
            <p className="mt-1 max-w-[640px] text-[13.5px] leading-6 text-cocoa-500">{t("builderDescription")}</p>
          </div>
          {canEdit ? (
            editing ? null : (
              <button
                className="h-10 rounded-full border border-sand-400 bg-sand-50 px-4 text-[13px] font-semibold text-cocoa-700 transition hover:border-clay hover:text-clay-deep disabled:cursor-not-allowed disabled:opacity-50"
                disabled={!online}
                onClick={beginEditing}
                title={!online ? t("editingRequiresInternet") : undefined}
                type="button"
              >
                {trip.route ? t("editRoute") : t("createRoute")}
              </button>
            )
          ) : (
            <span className="rounded-full bg-sand-200 px-3 py-1 text-[12px] font-semibold text-cocoa-500">{t("readOnly")}</span>
          )}
        </div>

        {!online ? (
          <div className="mt-4 rounded-[14px] border border-amber-300 bg-amber-50 p-3 text-[13px] font-medium text-amber-900">
            {t("offlineEditingDisabled")}
          </div>
        ) : null}

        {editing ? (
          <div className="mt-5 space-y-4 rounded-[16px] border border-clay/25 bg-clay-tint/25 p-4">
            <SortableRouteStopList
              onReorder={(fromIndex, toIndex) => changeDraft(reorderRouteStops(draft.draftRoute, fromIndex, toIndex))}
              stops={draft.draftRoute.stops}
            />
            <button
              className="h-10 w-full rounded-full border border-dashed border-sand-500 bg-white px-4 text-[13px] font-semibold text-cocoa-600 transition hover:border-clay hover:text-clay-deep"
              onClick={addStop}
              type="button"
            >
              + {t("addStop")}
            </button>
            {(draft.draftRoute.legs ?? []).length > 0 ? (
              <div>
                <p className="mb-2 text-[13px] font-semibold text-cocoa-800">{t("legModes")}</p>
                <div className="grid gap-3 lg:grid-cols-2">
                  {(draft.draftRoute.legs ?? []).map((leg, index) => (
                    <div className="rounded-[14px] border border-sand-300 bg-white p-3" key={leg.id}>
                      <p className="mb-2 text-[12.5px] font-semibold text-cocoa-700">
                        {leg.fromName || t("origin")} → {leg.toName || draft.draftRoute.stops[index]?.destination}
                      </p>
                      <TransportModeSelector value={leg.mode} onChange={(mode) => updateLegMode(index, mode)} />
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
        ) : null}

        <div className="mt-5">
          <RouteTimeline
            canEdit={editing}
            canEditTransport={canEdit && online && !editing}
            currency={trip.budgetCurrency}
            expectedItineraryRevision={trip.itineraryRevision}
            issues={issues}
            itinerary={trip.itinerary}
            onEditStop={editing ? setEditingStopId : undefined}
            onRemoveStop={editing ? (stopId) => changeDraft(removeRouteStop(draft.draftRoute, stopId)) : undefined}
            online={online}
            route={visibleRoute}
            travelers={trip.travelers}
            tripId={trip.id}
          />
        </div>

        {editing && selectedStop ? (
          <div id="route-stop-editor" className="mt-4 scroll-mt-28 rounded-[16px] border border-clay/30 bg-[#FFFDFA] p-4">
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-[15px] font-semibold text-cocoa-900">{t("editStop")}</h3>
              <button className="text-[12px] font-semibold text-cocoa-500" onClick={() => setEditingStopId(null)} type="button">{t("done")}</button>
            </div>
            <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <StopField label={t("destination")}>
                <input className={INPUT} value={selectedStop.destination} onChange={(event) => updateStop({ ...selectedStop, destination: event.target.value })} />
              </StopField>
              <StopField label={t("city")}>
                <input className={INPUT} value={selectedStop.city ?? ""} onChange={(event) => updateStop({ ...selectedStop, city: event.target.value || null })} />
              </StopField>
              <StopField label={t("country")}>
                <input className={INPUT} value={selectedStop.country ?? ""} onChange={(event) => updateStop({ ...selectedStop, country: event.target.value || null })} />
              </StopField>
              <StopField label={t("arrivalDate")}>
                <input className={INPUT} type="date" value={selectedStop.arrivalDate ?? ""} onChange={(event) => updateStop({ ...selectedStop, arrivalDate: event.target.value || null })} />
              </StopField>
              <StopField label={t("departureDate")}>
                <input className={INPUT} type="date" value={selectedStop.departureDate ?? ""} onChange={(event) => updateStop({ ...selectedStop, departureDate: event.target.value || null })} />
              </StopField>
              <StopField label={t("nights")}>
                <input className={INPUT} min={0} type="number" value={selectedStop.nights ?? 0} onChange={(event) => updateStop({ ...selectedStop, nights: Number(event.target.value || 0) })} />
              </StopField>
            </div>
          </div>
        ) : null}

        {saveError ? <p role="alert" className="mt-4 text-[13px] font-medium text-red-700">{saveError}</p> : null}
      </div>

      {visibleRoute ? <RouteMetricsCard currency={trip.budgetCurrency} route={visibleRoute} totalDays={trip.days} /> : null}
      {visibleRoute ? <StopDayMapping itinerary={trip.itinerary} route={visibleRoute} /> : null}
      <div id="route-validation" className="scroll-mt-28"><RouteValidationPanel issues={issues} onAction={handleIssueAction} /></div>

      {editing ? (
        <div className="sticky bottom-3 z-30 flex flex-col gap-2 rounded-[16px] border border-sand-300 bg-white/95 p-3 shadow-xl backdrop-blur sm:flex-row sm:items-center sm:justify-between">
          <p className="text-[12.5px] font-medium text-cocoa-500">
            {draft.dirty ? t("unsavedChanges") : t("noUnsavedChanges")}
          </p>
          <div className="flex gap-2">
            <button className="h-10 flex-1 rounded-full border border-sand-400 px-4 text-[13px] font-semibold text-cocoa-700 sm:flex-none" onClick={discardChanges} type="button">
              {t("discardChanges")}
            </button>
            <button
              className="h-10 flex-1 rounded-full bg-cocoa-900 px-4 text-[13px] font-semibold text-white disabled:cursor-not-allowed disabled:opacity-45 sm:flex-none"
              disabled={!draft.dirty || mutation.isPending}
              onClick={previewSave}
              type="button"
            >
              {t("reviewAndSave")}
            </button>
          </div>
        </div>
      ) : null}

      <RouteImpactPreviewDialog
        error={saveError}
        impact={draft}
        onCancel={() => setImpactOpen(false)}
        onConfirm={saveRoute}
        open={impactOpen}
        pending={mutation.isPending}
      />
    </section>
  );
}

function StopField({ label, children }: { label: string; children: React.ReactNode }) {
  return <label className="text-[12.5px] font-semibold text-cocoa-600">{label}{children}</label>;
}

function emptyRoute(): TripRoute {
  return { origin: { name: "" }, stops: [], legs: [], preferences: {} };
}

function initialRouteForTrip(trip: Trip): TripRoute {
  const stop: TripRouteStop = {
    id: `stop_${Date.now().toString(36)}`,
    destination: trip.destination,
    arrivalDate: trip.startDate,
    nights: Math.max(0, trip.days - 1),
    accommodationHint: "unknown"
  };
  return rebuildRouteLegs(
    {
      origin: { name: "" },
      stops: [],
      legs: [],
      preferences: { preferredModes: ["train"] }
    },
    [stop]
  );
}

function useUnsavedRouteWarning(dirty: boolean) {
  useEffect(() => {
    if (!dirty) {
      return;
    }
    const beforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    const interceptLink = (event: MouseEvent) => {
      const link = (event.target as HTMLElement | null)?.closest<HTMLAnchorElement>("a[href]");
      if (!link || link.target === "_blank" || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return;
      }
      const targetUrl = new URL(link.href, window.location.href);
      if (
        targetUrl.pathname === window.location.pathname &&
        targetUrl.search === window.location.search &&
        targetUrl.hash
      ) {
        return;
      }
      if (!window.confirm("Discard unsaved route changes and leave this page?")) {
        event.preventDefault();
        event.stopPropagation();
      }
    };
    window.addEventListener("beforeunload", beforeUnload);
    document.addEventListener("click", interceptLink, true);
    return () => {
      window.removeEventListener("beforeunload", beforeUnload);
      document.removeEventListener("click", interceptLink, true);
    };
  }, [dirty]);
}
