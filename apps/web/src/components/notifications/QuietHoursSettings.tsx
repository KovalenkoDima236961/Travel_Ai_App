"use client";

import type { NotificationSettings } from "@/entities/notification-preferences/model";
import { FIELD_LABEL_CLASS, INPUT_CLASS, Switch } from "@/components/settings/controls";

export function QuietHoursSettings({ value, onChange, disabled }: { value: NotificationSettings; onChange: (value: NotificationSettings) => void; disabled?: boolean }) {
  const patch = (next: Partial<NotificationSettings>) => onChange({ ...value, ...next });
  return (
    <section className="rounded-2xl border border-sand-300 bg-sand-50/60 p-5">
      <div className="flex items-start justify-between gap-4">
        <div><h3 className="text-[14px] font-semibold text-cocoa-900">Quiet hours</h3><p className="mt-1 text-[12.5px] text-cocoa-400">Email and push wait until quiet hours end. Urgent alerts can still break through.</p></div>
        <Switch label="Enable quiet hours" checked={value.quietHoursEnabled} disabled={disabled} onChange={(quietHoursEnabled) => patch({ quietHoursEnabled })} />
      </div>
      <div className="mt-4 grid gap-4 sm:grid-cols-3">
        <label><span className={FIELD_LABEL_CLASS}>Start</span><input className={`${INPUT_CLASS} mt-1.5`} type="time" disabled={disabled || !value.quietHoursEnabled} value={value.quietHoursStart} onChange={(event) => patch({ quietHoursStart: event.target.value })} /></label>
        <label><span className={FIELD_LABEL_CLASS}>End</span><input className={`${INPUT_CLASS} mt-1.5`} type="time" disabled={disabled || !value.quietHoursEnabled} value={value.quietHoursEnd} onChange={(event) => patch({ quietHoursEnd: event.target.value })} /></label>
        <label><span className={FIELD_LABEL_CLASS}>Timezone</span><input className={`${INPUT_CLASS} mt-1.5`} disabled={disabled || !value.quietHoursEnabled} value={value.quietHoursTimezone} placeholder="Europe/Bratislava" onChange={(event) => patch({ quietHoursTimezone: event.target.value })} /></label>
      </div>
      <label className="mt-4 flex items-center justify-between gap-4 text-[13px] text-cocoa-600"><span>Allow urgent notifications during quiet hours</span><Switch label="Urgent notifications bypass quiet hours" checked={value.urgentBypassesQuietHours} disabled={disabled || !value.quietHoursEnabled} onChange={(urgentBypassesQuietHours) => patch({ urgentBypassesQuietHours })} /></label>
    </section>
  );
}
