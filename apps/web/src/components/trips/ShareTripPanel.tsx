"use client";

import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import {
  createTripShare,
  disableTripShare,
  getTripShare,
  tripKeys,
  updateTripShare
} from "@/lib/api/trips";
import { formatDate, getErrorMessage } from "@/lib/utils";
import type { TripShareInfo, UpdateTripShareRequest } from "@/types/share";

type ShareTripPanelProps = {
  tripId: string;
};

type ExpirationPreset = "never" | "7_days" | "30_days" | "custom";

export function ShareTripPanel({ tripId }: ShareTripPanelProps) {
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expirationPreset, setExpirationPreset] = useState<ExpirationPreset>("never");
  const [customExpiresAt, setCustomExpiresAt] = useState("");
  const [requirePassword, setRequirePassword] = useState(false);
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const shareQuery = useQuery({
    queryKey: tripKeys.share(tripId),
    queryFn: () => getTripShare(tripId),
    enabled: Boolean(tripId)
  });

  const createMutation = useMutation({
    mutationFn: (body?: UpdateTripShareRequest) => createTripShare(tripId, body),
    onSuccess: (share) => {
      queryClient.setQueryData(tripKeys.share(tripId), share);
      resetPasswordInputs();
      setMessage("Share link is ready.");
      setError(null);
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not create share link."));
      setMessage(null);
    }
  });

  const updateMutation = useMutation({
    mutationFn: (body: UpdateTripShareRequest) => updateTripShare(tripId, body),
    onSuccess: async (share) => {
      queryClient.setQueryData(tripKeys.share(tripId), share);
      await queryClient.invalidateQueries({ queryKey: tripKeys.share(tripId) });
      resetPasswordInputs();
      setMessage("Share settings saved.");
      setError(null);
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not save share settings."));
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

  useEffect(() => {
    if (!share) {
      return;
    }
    setRequirePassword(Boolean(share.passwordRequired));
    if (share.expiresAt) {
      setExpirationPreset("custom");
      setCustomExpiresAt(toDatetimeLocal(share.expiresAt));
    } else {
      setExpirationPreset("never");
      setCustomExpiresAt("");
    }
    resetPasswordInputs();
  }, [share?.expiresAt, share?.passwordRequired]);

  const shareUrl = share?.shareUrl || "";
  const active = Boolean(share?.enabled && shareUrl);
  const busy =
    createMutation.isPending ||
    disableMutation.isPending ||
    updateMutation.isPending ||
    shareQuery.isPending;

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
      inputRef.current?.focus();
      inputRef.current?.select();
      setMessage("Link selected. Copy it from the field.");
      setError(null);
      return;
    }

    inputRef.current?.focus();
    inputRef.current?.select();
    setMessage("Link selected. Copy it from the field.");
    setError(null);
  }

  function createShareLink() {
    const payload = buildSettingsPayload(share, true);
    if (!payload) {
      return;
    }
    createMutation.mutate(Object.keys(payload).length > 0 ? payload : undefined);
  }

  function saveSettings() {
    const payload = buildSettingsPayload(share, false);
    if (!payload) {
      return;
    }
    updateMutation.mutate(payload);
  }

  function removePassword() {
    updateMutation.mutate({ clearPassword: true });
  }

  function disableShareLink() {
    if (!window.confirm("Disable this public share link?")) {
      return;
    }
    disableMutation.mutate();
  }

  function buildSettingsPayload(
    currentShare: TripShareInfo | undefined,
    forCreate: boolean
  ): UpdateTripShareRequest | null {
    const payload: UpdateTripShareRequest = {};

    if (expirationPreset === "never") {
      if (!forCreate) {
        payload.clearExpiration = true;
      }
    } else if (expirationPreset === "7_days" || expirationPreset === "30_days") {
      const days = expirationPreset === "7_days" ? 7 : 30;
      payload.expiresAt = new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString();
    } else {
      const customDate = new Date(customExpiresAt);
      if (!customExpiresAt || Number.isNaN(customDate.getTime())) {
        setError("Choose a valid expiration date and time.");
        setMessage(null);
        return null;
      }
      if (customDate.getTime() <= Date.now()) {
        setError("Expiration must be in the future.");
        setMessage(null);
        return null;
      }
      payload.expiresAt = customDate.toISOString();
    }

    const alreadyProtected = Boolean(currentShare?.passwordRequired);
    const wantsPasswordChange = password.length > 0 || confirmPassword.length > 0;
    if (requirePassword) {
      if (!alreadyProtected || wantsPasswordChange) {
        if (password.length < 6) {
          setError("Password must be at least 6 characters.");
          setMessage(null);
          return null;
        }
        if (password.length > 128) {
          setError("Password must be 128 characters or fewer.");
          setMessage(null);
          return null;
        }
        if (password !== confirmPassword) {
          setError("Passwords do not match.");
          setMessage(null);
          return null;
        }
        payload.password = password;
      }
    } else if (alreadyProtected && !forCreate) {
      payload.clearPassword = true;
    }

    setError(null);
    return payload;
  }

  function resetPasswordInputs() {
    setPassword("");
    setConfirmPassword("");
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
        ) : null}

        <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
          <div className="space-y-2 text-sm">
            <StatusRow label="Expiration" value={expirationStatus(share)} />
            <StatusRow
              label="Password"
              value={share?.passwordRequired ? "Password protected" : "No password required"}
            />
          </div>
          {share?.expired ? (
            <p className="mt-3 text-sm font-medium text-red-700">This link has expired.</p>
          ) : null}
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700" htmlFor="share-expiration">
              Expiration
            </label>
            <Select
              disabled={busy}
              id="share-expiration"
              onChange={(event) => setExpirationPreset(event.target.value as ExpirationPreset)}
              value={expirationPreset}
            >
              <option value="never">Never</option>
              <option value="7_days">7 days</option>
              <option value="30_days">30 days</option>
              <option value="custom">Custom date/time</option>
            </Select>
          </div>

          {expirationPreset === "custom" ? (
            <div>
              <label className="block text-sm font-medium text-slate-700" htmlFor="share-custom-expiration">
                Custom expiration
              </label>
              <Input
                disabled={busy}
                id="share-custom-expiration"
                min={toDatetimeLocal(new Date(Date.now() + 60 * 1000).toISOString())}
                onChange={(event) => setCustomExpiresAt(event.target.value)}
                type="datetime-local"
                value={customExpiresAt}
              />
            </div>
          ) : null}

          <div className="space-y-3">
            <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
              <input
                checked={requirePassword}
                className="h-4 w-4 rounded border-slate-300 text-primary-600 focus:ring-primary-600"
                disabled={busy}
                onChange={(event) => setRequirePassword(event.target.checked)}
                type="checkbox"
              />
              Require password
            </label>

            {requirePassword ? (
              <div className="space-y-3">
                {share?.passwordRequired ? (
                  <p className="text-sm text-slate-600">Leave password fields blank to keep the current password.</p>
                ) : null}
                <div>
                  <label className="block text-sm font-medium text-slate-700" htmlFor="share-password">
                    Password
                  </label>
                  <Input
                    autoComplete="new-password"
                    disabled={busy}
                    id="share-password"
                    maxLength={128}
                    minLength={6}
                    onChange={(event) => setPassword(event.target.value)}
                    type="password"
                    value={password}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700" htmlFor="share-confirm-password">
                    Confirm password
                  </label>
                  <Input
                    autoComplete="new-password"
                    disabled={busy}
                    id="share-confirm-password"
                    maxLength={128}
                    minLength={6}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                    type="password"
                    value={confirmPassword}
                  />
                </div>
                {share?.passwordRequired ? (
                  <Button
                    disabled={busy}
                    onClick={removePassword}
                    size="sm"
                    type="button"
                    variant="secondary"
                  >
                    Remove password
                  </Button>
                ) : null}
              </div>
            ) : null}
          </div>
        </div>

        {active ? (
          <Button disabled={busy} onClick={saveSettings} type="button" variant="secondary">
            {updateMutation.isPending ? "Saving..." : "Save settings"}
          </Button>
        ) : (
          <Button disabled={busy} onClick={createShareLink} type="button" variant="secondary">
            {createMutation.isPending ? "Creating..." : "Create share link"}
          </Button>
        )}

        {message ? <p className="text-sm font-medium text-emerald-700">{message}</p> : null}
        {error ? <p className="text-sm font-medium text-red-700">{error}</p> : null}
      </div>
    </Card>
  );
}

function StatusRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-3">
      <span className="text-slate-500">{label}</span>
      <span className="text-right font-medium text-slate-800">{value}</span>
    </div>
  );
}

function expirationStatus(share: TripShareInfo | undefined) {
  if (!share?.expiresAt) {
    return "Never expires";
  }
  if (share.expired) {
    return "Expired";
  }
  return `Expires ${formatDate(share.expiresAt, {
    dateStyle: "medium",
    timeStyle: "short"
  })}`;
}

function toDatetimeLocal(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
