"use client";

import type { NotificationDeliveryMode } from "@/entities/notification-preferences/model";
import { SELECT_CLASS } from "@/components/settings/controls";

const modes: Array<{ value: NotificationDeliveryMode; label: string }> = [
  { value: "instant", label: "Instant" },
  { value: "hourly_digest", label: "Hourly digest" },
  { value: "daily_digest", label: "Daily digest" },
  { value: "weekly_digest", label: "Weekly digest" },
  { value: "muted", label: "Muted" }
];

export function CategoryDeliveryModeSelect({ value, onChange, label, disabled }: {
  value: NotificationDeliveryMode;
  onChange: (value: NotificationDeliveryMode) => void;
  label: string;
  disabled?: boolean;
}) {
  return (
    <select aria-label={label} className={`${SELECT_CLASS} h-10 min-w-[142px] text-[13px]`} disabled={disabled} value={value} onChange={(event) => onChange(event.target.value as NotificationDeliveryMode)}>
      {modes.map((mode) => <option key={mode.value} value={mode.value}>{mode.label}</option>)}
    </select>
  );
}
