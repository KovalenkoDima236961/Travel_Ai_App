"use client";

import Link from "next/link";
import { useState, type ReactNode } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { ConfirmDialog } from "@/components/ui";
import type { Trip } from "@/entities/trip/model";
import { trackAlphaEvent } from "@/lib/api/alpha";
import { useFeatureFlags } from "@/lib/feature-flags/useFeatureFlags";
import { archiveTrip, restoreTrip, tripLibraryKeys } from "@/lib/api/library";
import { tripKeys } from "@/lib/api/trips";
import {
  getTripWorkspaceActions,
  type TripWorkspaceAction,
  type TripWorkspaceActionContext,
  type TripWorkspaceActionGroup
} from "@/lib/trip-workspace/actions";

const GROUPS: TripWorkspaceActionGroup[] = ["primary", "share_export", "history", "reuse", "settings"];

export function TripWorkspaceActionsPanel({
  trip,
  online,
  onEditItinerary,
  onSaveTemplate,
  exportControl
}: {
  trip: Trip;
  online: boolean;
  onEditItinerary?: () => void;
  onSaveTemplate?: () => void;
  exportControl?: ReactNode;
}) {
  const t = useTranslations("tripWorkspace.actions");
  const queryClient = useQueryClient();
  const { flags } = useFeatureFlags();
  const [confirmAction, setConfirmAction] = useState<"archive_trip" | "restore_trip" | null>(null);
  const access = trip.access;
  const role = access?.role ?? "owner";
  const context: TripWorkspaceActionContext = {
    tripId: trip.id,
    role,
    canEdit: Boolean(access?.canEdit ?? true),
    canManageCollaborators: Boolean(access?.canManageCollaborators ?? true),
    canManageShare: Boolean(access?.canManageShare ?? true),
    canRestoreVersion: Boolean(access?.canRestoreVersion ?? access?.canEdit ?? true),
    canArchive: role === "owner" && Boolean(access?.canDelete ?? true),
    archived: Boolean(trip.archivedAt) || trip.lifecycle === "archived",
    completed: trip.status === "COMPLETED",
    online,
    hasItinerary: Boolean(trip.itinerary),
    flags
  };
  const actions = getTripWorkspaceActions(context).filter(
    (action) => action.id !== "export_trip" || Boolean(exportControl)
  );
  const lifecycleMutation = useMutation({
    mutationFn: (action: "archive_trip" | "restore_trip") =>
      action === "archive_trip" ? archiveTrip(trip.id) : restoreTrip(trip.id),
    onSuccess: (response) => {
      queryClient.setQueryData<Trip>(tripKeys.detail(trip.id), (current) =>
        current
          ? { ...current, archivedAt: response.archivedAt, lifecycle: response.lifecycle }
          : current
      );
      void queryClient.invalidateQueries({ queryKey: tripKeys.lists() });
      void queryClient.invalidateQueries({ queryKey: tripLibraryKeys.all });
      setConfirmAction(null);
    }
  });

  function track(action: TripWorkspaceAction) {
    trackAlphaEvent({
      eventName: action.analyticsEvent,
      feature: "trip_workspace",
      entityType: "trip",
      entityId: trip.id,
      metadata: { actionId: action.id, role, tripLifecycleState: trip.lifecycle ?? "unknown" }
    });
  }

  function execute(action: TripWorkspaceAction) {
    track(action);
    if (action.command === "open_search") {
      window.dispatchEvent(new CustomEvent("travel-ai:open-command-palette"));
    } else if (action.command === "edit_itinerary") {
      onEditItinerary?.();
    } else if (action.command === "save_template") {
      onSaveTemplate?.();
    } else if (action.command === "archive_trip" || action.command === "restore_trip") {
      setConfirmAction(action.command);
    }
  }

  return (
    <section aria-labelledby="trip-workspace-actions-title" className="rounded-[18px] border border-sand-300 bg-white p-5">
      <h3 className="font-newsreader text-[22px] font-semibold text-cocoa-900" id="trip-workspace-actions-title">
        {t("title")}
      </h3>
      <div className="mt-4 grid gap-5 md:grid-cols-2">
        {GROUPS.map((group) => {
          const groupActions = actions.filter((action) => action.group === group);
          if (groupActions.length === 0) return null;
          return (
            <div key={group}>
              <h4 className="text-xs font-semibold uppercase tracking-[0.08em] text-cocoa-400">
                {t(`groups.${group}`)}
              </h4>
              <div className="mt-2 flex flex-col gap-2">
                {groupActions.map((action) =>
                  action.command === "export_trip" && exportControl ? (
                    <div key={action.id} onClick={() => track(action)}>{exportControl}</div>
                  ) : action.disabledReason ? (
                    <button
                      className="min-h-11 cursor-not-allowed rounded-xl border border-sand-200 bg-sand-50 px-3 text-left text-sm font-semibold text-cocoa-700 opacity-50"
                      disabled
                      key={action.id}
                      title={t(`disabled.${action.disabledReason}`)}
                      type="button"
                    >
                      {t(`labels.${action.labelKey}`)}
                    </button>
                  ) : action.href ? (
                    <Link
                      className="flex min-h-11 items-center rounded-xl border border-sand-200 bg-sand-50 px-3 text-sm font-semibold text-cocoa-700 transition hover:border-sand-400 hover:bg-white"
                      href={action.href}
                      key={action.id}
                      onClick={() => track(action)}
                    >
                      {t(`labels.${action.labelKey}`)}
                    </Link>
                  ) : action.command ? (
                    <button
                      className={`min-h-11 rounded-xl border px-3 text-left text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-50 ${
                        action.destructive
                          ? "border-red-200 bg-red-50 text-red-800 hover:bg-red-100"
                          : "border-sand-200 bg-sand-50 text-cocoa-700 hover:border-sand-400 hover:bg-white"
                      }`}
                      disabled={
                        (action.command === "edit_itinerary" && !onEditItinerary) ||
                        (action.command === "save_template" && !onSaveTemplate)
                      }
                      key={action.id}
                      onClick={() => execute(action)}
                      title={
                        (action.command === "edit_itinerary" && !onEditItinerary) ||
                        (action.command === "save_template" && !onSaveTemplate)
                          ? t("disabled.read_only")
                          : undefined
                      }
                      type="button"
                    >
                      {t(`labels.${action.labelKey}`)}
                    </button>
                  ) : null
                )}
              </div>
            </div>
          );
        })}
      </div>
      <ConfirmDialog
        confirmLabel={confirmAction ? t(`confirm.${confirmAction}.confirm`) : ""}
        description={confirmAction ? t(`confirm.${confirmAction}.description`) : ""}
        error={lifecycleMutation.error instanceof Error ? lifecycleMutation.error.message : null}
        onCancel={() => setConfirmAction(null)}
        onConfirm={() => confirmAction && lifecycleMutation.mutate(confirmAction)}
        open={Boolean(confirmAction)}
        pending={lifecycleMutation.isPending}
        title={confirmAction ? t(`confirm.${confirmAction}.title`) : ""}
        tone={confirmAction === "archive_trip" ? "danger" : "default"}
      />
    </section>
  );
}
