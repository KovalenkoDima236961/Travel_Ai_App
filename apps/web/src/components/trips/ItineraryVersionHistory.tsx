"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ItineraryView } from "@/components/trips/ItineraryView";
import { Button } from "@/components/ui/Button";
import {
  getItineraryVersion,
  listItineraryVersions,
  restoreItineraryVersion,
  tripKeys
} from "@/lib/api/trips";
import { formatDate } from "@/lib/utils";
import type {
  ItineraryVersionDetail,
  ItineraryVersionSource,
  ItineraryVersionSummary
} from "@/types/itinerary-version";
import type { Trip } from "@/types/trip";

type ItineraryVersionHistoryProps = {
  tripId: string;
  currency?: string;
  restoreDisabled?: boolean;
  onRestored?: (trip: Trip) => void;
};

const sourceLabels: Record<ItineraryVersionSource, string> = {
  GENERATED: "Generated",
  MANUAL_EDIT: "Manual edit",
  REGENERATE_DAY: "Regenerated day",
  REGENERATE_ITEM: "Regenerated item",
  RESTORED: "Restored"
};

export function ItineraryVersionHistory({
  tripId,
  currency = "EUR",
  restoreDisabled = false,
  onRestored
}: ItineraryVersionHistoryProps) {
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const [preview, setPreview] = useState<ItineraryVersionDetail | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const versionsQuery = useQuery({
    queryKey: tripKeys.itineraryVersions(tripId),
    queryFn: () => listItineraryVersions(tripId),
    enabled: isOpen && Boolean(tripId)
  });

  const previewMutation = useMutation({
    mutationFn: (versionId: string) => getItineraryVersion(tripId, versionId)
  });

  const restoreMutation = useMutation({
    mutationFn: (versionId: string) => restoreItineraryVersion(tripId, versionId)
  });

  async function viewVersion(version: ItineraryVersionSummary) {
    try {
      setErrorMessage(null);
      setSuccessMessage(null);
      const detail = await previewMutation.mutateAsync(version.id);
      setPreview(detail);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Could not load version.");
    }
  }

  async function restoreVersion(version: ItineraryVersionSummary) {
    const confirmed = window.confirm(
      `Restore Version ${version.versionNumber}? Your current itinerary will be replaced, but a new version will be created so you can undo this later.`
    );
    if (!confirmed) {
      return;
    }

    try {
      setErrorMessage(null);
      setSuccessMessage(null);
      const updatedTrip = await restoreMutation.mutateAsync(version.id);
      queryClient.setQueryData(tripKeys.detail(tripId), updatedTrip);
      await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) });
      setPreview(null);
      setSuccessMessage(`Version ${version.versionNumber} restored.`);
      onRestored?.(updatedTrip);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Could not restore version.");
    }
  }

  const versions = versionsQuery.data?.items ?? [];

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Version History</h2>
          {restoreDisabled ? (
            <p className="mt-1 text-sm text-slate-500">Finish editing before restoring a version.</p>
          ) : null}
        </div>
        <Button
          onClick={() => {
            setIsOpen((value) => !value);
            setErrorMessage(null);
            setSuccessMessage(null);
          }}
          type="button"
          variant="secondary"
        >
          {isOpen ? "Hide history" : "Show history"}
        </Button>
      </div>

      {isOpen ? (
        <div className="mt-5 space-y-4">
          {successMessage ? (
            <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">
              {successMessage}
            </div>
          ) : null}

          {errorMessage ? (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {errorMessage}
            </div>
          ) : null}

          {versionsQuery.isPending ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              Loading versions...
            </div>
          ) : null}

          {versionsQuery.isError ? (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {versionsQuery.error instanceof Error
                ? versionsQuery.error.message
                : "Could not load versions."}
            </div>
          ) : null}

          {!versionsQuery.isPending && !versionsQuery.isError && versions.length === 0 ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              No versions yet. New versions will appear after generation, editing, or regeneration.
            </div>
          ) : null}

          {versions.length > 0 ? (
            <ul className="divide-y divide-slate-100 rounded-lg border border-slate-200">
              {versions.map((version) => (
                <li
                  className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"
                  key={version.id}
                >
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-semibold text-slate-950">
                        Version {version.versionNumber}
                      </p>
                      <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700">
                        {sourceLabel(version.source)}
                      </span>
                    </div>
                    <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm text-slate-500">
                      <span>
                        {formatDate(version.createdAt, {
                          dateStyle: "medium",
                          timeStyle: "short"
                        })}
                      </span>
                      {metadataLabel(version) ? <span>{metadataLabel(version)}</span> : null}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      disabled={previewMutation.isPending}
                      onClick={() => viewVersion(version)}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      {previewMutation.isPending && previewMutation.variables === version.id
                        ? "Loading..."
                        : "View"}
                    </Button>
                    <Button
                      disabled={restoreDisabled || restoreMutation.isPending}
                      onClick={() => restoreVersion(version)}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      {restoreMutation.isPending && restoreMutation.variables === version.id
                        ? "Restoring..."
                        : "Restore"}
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}

      {preview ? (
        <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-950/50 p-4">
          <div className="mx-auto max-w-4xl rounded-lg bg-slate-50 p-4 shadow-xl">
            <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 className="text-lg font-semibold text-slate-950">
                  Previewing Version {preview.versionNumber}
                </h3>
                <p className="text-sm text-slate-500">{sourceLabel(preview.source)}</p>
              </div>
              <div className="flex gap-2">
                <Button
                  disabled={restoreDisabled || restoreMutation.isPending}
                  onClick={() => restoreVersion(preview)}
                  type="button"
                  variant="secondary"
                >
                  {restoreMutation.isPending && restoreMutation.variables === preview.id
                    ? "Restoring..."
                    : "Restore"}
                </Button>
                <Button onClick={() => setPreview(null)} type="button" variant="secondary">
                  Close
                </Button>
              </div>
            </div>
            <ItineraryView currency={currency} disabled itinerary={preview.itinerary} />
          </div>
        </div>
      ) : null}
    </section>
  );
}

function sourceLabel(source: ItineraryVersionSource) {
  return sourceLabels[source] ?? source;
}

function metadataLabel(version: ItineraryVersionSummary) {
  const metadata = version.metadata ?? {};
  if (version.source === "REGENERATE_DAY") {
    const dayNumber = numberValue(metadata.dayNumber);
    return dayNumber == null ? null : `Day ${dayNumber} regenerated`;
  }
  if (version.source === "REGENERATE_ITEM") {
    const dayNumber = numberValue(metadata.dayNumber);
    const itemIndex = numberValue(metadata.itemIndex);
    if (dayNumber == null || itemIndex == null) {
      return null;
    }
    return `Item ${itemIndex + 1} in Day ${dayNumber} regenerated`;
  }
  if (version.source === "RESTORED") {
    const restoredFrom = numberValue(metadata.restoredFromVersionNumber);
    return restoredFrom == null ? null : `Restored from Version ${restoredFrom}`;
  }
  return null;
}

function numberValue(value: unknown) {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}
