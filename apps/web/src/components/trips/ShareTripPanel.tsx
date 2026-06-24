"use client";

import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import {
  createTripShare,
  disableTripShare,
  getTripShare,
  tripKeys
} from "@/lib/api/trips";
import { getErrorMessage } from "@/lib/utils";

type ShareTripPanelProps = {
  tripId: string;
};

export function ShareTripPanel({ tripId }: ShareTripPanelProps) {
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const shareQuery = useQuery({
    queryKey: tripKeys.share(tripId),
    queryFn: () => getTripShare(tripId),
    enabled: Boolean(tripId)
  });

  const createMutation = useMutation({
    mutationFn: () => createTripShare(tripId),
    onSuccess: (share) => {
      queryClient.setQueryData(tripKeys.share(tripId), share);
      setMessage("Share link is ready.");
      setError(null);
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not create share link."));
      setMessage(null);
    }
  });

  const disableMutation = useMutation({
    mutationFn: () => disableTripShare(tripId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: tripKeys.share(tripId) });
      setMessage("Share link disabled.");
      setError(null);
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not disable share link."));
      setMessage(null);
    }
  });

  const share = shareQuery.data;
  const shareUrl = share?.shareUrl || "";
  const active = Boolean(share?.enabled && shareUrl);
  const busy = createMutation.isPending || disableMutation.isPending || shareQuery.isPending;

  async function copyShareLink() {
    if (!shareUrl) {
      return;
    }

    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(shareUrl);
        setMessage("Copied.");
        setError(null);
        return;
      }
    } catch {
      // Fall back to selecting the input below.
    }

    inputRef.current?.focus();
    inputRef.current?.select();
    setMessage("Link selected. Copy it from the field.");
    setError(null);
  }

  function disableShareLink() {
    if (!window.confirm("Disable this public share link?")) {
      return;
    }
    disableMutation.mutate();
  }

  return (
    <Card>
      <div className="flex flex-col gap-4">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Share itinerary</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Anyone with this link can view a read-only version of your itinerary.
          </p>
          <p className="mt-1 text-sm leading-6 text-slate-600">
            They cannot edit, regenerate, or see your private preferences.
          </p>
        </div>

        {shareQuery.isError ? (
          <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
            {getErrorMessage(shareQuery.error, "Could not load share status.")}
          </div>
        ) : null}

        {active ? (
          <div className="space-y-3">
            <label className="block text-sm font-medium text-slate-700" htmlFor="trip-share-url">
              Public link
            </label>
            <input
              ref={inputRef}
              className="w-full rounded-md border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-900 shadow-sm focus:border-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-600/20"
              id="trip-share-url"
              readOnly
              value={shareUrl}
            />
            <div className="flex flex-wrap gap-2">
              <Button disabled={busy} onClick={copyShareLink} size="sm" type="button">
                Copy link
              </Button>
              <Button
                disabled={busy}
                onClick={disableShareLink}
                size="sm"
                type="button"
                variant="danger"
              >
                {disableMutation.isPending ? "Disabling..." : "Disable link"}
              </Button>
            </div>
          </div>
        ) : (
          <Button
            disabled={busy}
            onClick={() => createMutation.mutate()}
            type="button"
            variant="secondary"
          >
            {createMutation.isPending ? "Creating..." : "Create share link"}
          </Button>
        )}

        {message ? (
          <p className="text-sm font-medium text-emerald-700">{message}</p>
        ) : null}
        {error ? <p className="text-sm font-medium text-red-700">{error}</p> : null}
      </div>
    </Card>
  );
}
