"use client";

import {
  GhostButton,
  PrimaryButton,
  SaveNotice,
  SectionHeading,
  SettingsCard
} from "@/components/settings/controls";
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
    <SettingsCard>
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <SectionHeading title="Push notifications" />
          <p className="mt-2 text-[14px] leading-relaxed text-cocoa-500">{statusText()}</p>
          {typeof activeSubscriptions === "number" && activeSubscriptions > 0 ? (
            <p className="mt-1 text-[13px] text-cocoa-400">Active devices: {activeSubscriptions}</p>
          ) : null}
        </div>

        <div className="flex shrink-0 gap-2">
          {supported && !blocked && !enabled ? (
            <PrimaryButton disabled={loading} onClick={() => void enablePush()}>
              {loading ? "Enabling…" : "Enable push notifications"}
            </PrimaryButton>
          ) : null}
          {supported && enabled ? (
            <GhostButton disabled={loading} onClick={() => void disablePush()}>
              {loading ? "Disabling…" : "Disable on this device"}
            </GhostButton>
          ) : null}
        </div>
      </div>

      {error ? (
        <div className="mt-4">
          <SaveNotice errorMessage={error} />
        </div>
      ) : null}
    </SettingsCard>
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
