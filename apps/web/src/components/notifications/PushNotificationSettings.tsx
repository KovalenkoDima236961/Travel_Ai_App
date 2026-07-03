"use client";

import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { useWebPushNotifications } from "@/hooks/useWebPushNotifications";

export function PushNotificationSettings() {
  const {
    supported,
    permission,
    enabled,
    loading,
    error,
    activeSubscriptions,
    enablePush,
    disablePush
  } = useWebPushNotifications();

  const blocked = permission === "denied";

  return (
    <Card>
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Push notifications</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">{statusText()}</p>
          {typeof activeSubscriptions === "number" && activeSubscriptions > 0 ? (
            <p className="mt-1 text-sm text-slate-500">
              Active devices: {activeSubscriptions}
            </p>
          ) : null}
        </div>

        <div className="flex shrink-0 gap-2">
          {supported && !blocked && !enabled ? (
            <Button disabled={loading} onClick={() => void enablePush()}>
              {loading ? "Enabling..." : "Enable push notifications"}
            </Button>
          ) : null}
          {supported && enabled ? (
            <Button disabled={loading} variant="secondary" onClick={() => void disablePush()}>
              {loading ? "Disabling..." : "Disable on this device"}
            </Button>
          ) : null}
        </div>
      </div>

      {error ? (
        <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
          {error}
        </div>
      ) : null}
    </Card>
  );

  function statusText() {
    if (!supported) {
      return "Push notifications are not supported in this browser.";
    }
    if (blocked) {
      return "Notifications are blocked in your browser settings. Enable them in site settings to use push notifications.";
    }
    if (enabled) {
      return "Push notifications are enabled on this device.";
    }
    if (permission === "granted") {
      return "Enable browser push notifications to receive updates when the app is closed.";
    }
    return "Enable browser push notifications to receive updates when the app is closed.";
  }
}
