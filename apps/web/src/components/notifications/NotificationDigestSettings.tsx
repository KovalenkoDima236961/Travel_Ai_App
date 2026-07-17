"use client";

import type { NotificationSettings } from "@/entities/notification-preferences/model";
import { FIELD_LABEL_CLASS, INPUT_CLASS, SELECT_CLASS } from "@/components/settings/controls";

export function NotificationDigestSettings({ value, onChange, disabled }: { value: NotificationSettings; onChange: (value: NotificationSettings) => void; disabled?: boolean }) {
  const patch = (next: Partial<NotificationSettings>) => onChange({ ...value, ...next });
  return (
    <section className="rounded-2xl border border-sand-300 bg-sand-50/60 p-5">
      <h3 className="text-[14px] font-semibold text-cocoa-900">Digest schedule</h3>
      <p className="mt-1 text-[12.5px] text-cocoa-400">Hourly digests run at the next hour. Choose your local daily and weekly delivery times.</p>
      <div className="mt-4 grid gap-4 sm:grid-cols-3">
        <label><span className={FIELD_LABEL_CLASS}>Daily time</span><input className={`${INPUT_CLASS} mt-1.5`} type="time" disabled={disabled} value={value.dailyDigestTime} onChange={(event) => patch({ dailyDigestTime: event.target.value })} /></label>
        <label><span className={FIELD_LABEL_CLASS}>Weekly day</span><select className={`${SELECT_CLASS} mt-1.5`} disabled={disabled} value={value.weeklyDigestDay} onChange={(event) => patch({ weeklyDigestDay: Number(event.target.value) })}>{["Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"].map((day,index)=><option key={day} value={index}>{day}</option>)}</select></label>
        <label><span className={FIELD_LABEL_CLASS}>Weekly time</span><input className={`${INPUT_CLASS} mt-1.5`} type="time" disabled={disabled} value={value.weeklyDigestTime} onChange={(event) => patch({ weeklyDigestTime: event.target.value })} /></label>
      </div>
    </section>
  );
}
