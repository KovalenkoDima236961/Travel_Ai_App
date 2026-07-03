"use client";

import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { formatDate } from "@/lib/utils";
import type { PendingItineraryMutation } from "@/lib/offline/types";

type PendingOfflineChangesPanelProps = {
  mutation: PendingItineraryMutation;
  online: boolean;
  syncing: boolean;
  onSyncNow: () => Promise<void> | void;
  onReview: () => void;
  onDiscard: () => Promise<void> | void;
};

export function PendingOfflineChangesPanel({
  mutation,
  online,
  syncing,
  onSyncNow,
  onReview,
  onDiscard
}: PendingOfflineChangesPanelProps) {
  const isConflict = mutation.status === "conflict";

  return (
    <Card className="border-amber-200 bg-amber-50">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-amber-950">
            Pending offline itinerary changes
          </h2>
          <dl className="mt-3 grid gap-2 text-sm text-amber-900 sm:grid-cols-2">
            <MetadataRow label="Status" value={formatStatus(mutation.status)} />
            <MetadataRow label="Base revision" value={String(mutation.baseRevision)} />
            <MetadataRow label="Created" value={formatDateTime(mutation.createdAt)} />
            <MetadataRow label="Updated" value={formatDateTime(mutation.updatedAt)} />
          </dl>
          {mutation.errorMessage ? (
            <p className="mt-3 text-sm leading-6 text-amber-900">{mutation.errorMessage}</p>
          ) : null}
        </div>

        <div className="flex flex-col gap-2 sm:min-w-40">
          <Button
            disabled={!online || syncing || isConflict}
            onClick={onSyncNow}
            size="sm"
            title={!online ? "This action requires an internet connection." : undefined}
            type="button"
          >
            {syncing ? "Syncing..." : "Sync now"}
          </Button>
          <Button onClick={onReview} size="sm" type="button" variant="secondary">
            {isConflict ? "Review conflict" : "Review local draft"}
          </Button>
          <Button onClick={onDiscard} size="sm" type="button" variant="ghost">
            Discard offline changes
          </Button>
        </div>
      </div>
    </Card>
  );
}

function MetadataRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-semibold uppercase tracking-wide text-amber-800">{label}</dt>
      <dd className="mt-0.5 text-amber-950">{value}</dd>
    </div>
  );
}

function formatDateTime(value: string) {
  return formatDate(value, { dateStyle: "medium", timeStyle: "short" });
}

function formatStatus(status: PendingItineraryMutation["status"]) {
  return status.replace(/_/g, " ");
}
