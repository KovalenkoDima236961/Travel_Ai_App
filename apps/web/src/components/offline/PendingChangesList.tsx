"use client";

import { Button } from "@/shared/ui/button";
import type { PendingOfflineMutation } from "@/lib/offline/types";

type PendingChangesListProps = {
  mutations: PendingOfflineMutation[];
  onDiscard: (mutation: PendingOfflineMutation) => Promise<void> | void;
};

export function PendingChangesList({ mutations, onDiscard }: PendingChangesListProps) {
  if (mutations.length === 0) {
    return null;
  }

  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <h3 className="text-base font-semibold text-slate-950">Pending changes</h3>
      <ul className="mt-3 divide-y divide-slate-100">
        {mutations.map((mutation) => (
          <li className="flex items-center justify-between gap-3 py-3 text-sm" key={mutation.mutationId}>
            <div>
              <p className="font-medium text-slate-900">{labelForMutation(mutation)}</p>
              <p className="mt-1 text-xs text-slate-500">
                {mutation.status} · {formatDateTime(mutation.createdAt)}
              </p>
              {mutation.errorMessage ? (
                <p className="mt-1 text-xs text-red-700">{mutation.errorMessage}</p>
              ) : null}
            </div>
            <Button onClick={() => void onDiscard(mutation)} size="sm" type="button" variant="ghost">
              Discard
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function labelForMutation(mutation: PendingOfflineMutation) {
  return mutation.type
    .replaceAll("_", " ")
    .replace(/^./, (value) => value.toUpperCase());
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
