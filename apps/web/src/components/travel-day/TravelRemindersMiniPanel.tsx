"use client";

import { useCompleteTripReminder } from "@/hooks/useTripReminders";
import type { TravelDaySummary } from "@/types/travel-day";

export function TravelRemindersMiniPanel({ tripId, reminders, online }: { tripId: string; reminders: TravelDaySummary["reminders"]; online: boolean }) {
  const mutation = useCompleteTripReminder(tripId);
  const items = [...reminders.overdue, ...reminders.dueToday].slice(0, 4);
  return <section className="rounded-2xl border border-sand-300 bg-white p-4"><h2 className="font-semibold text-cocoa-900">Today’s reminders</h2>{items.length ? <ul className="mt-3 space-y-2">{items.map((reminder) => <li className="flex items-center justify-between gap-3 text-sm" key={reminder.id}><span className="text-cocoa-700">{reminder.title}</span><button className="rounded-full px-2 py-1 text-xs font-semibold text-clay disabled:opacity-50" disabled={!online || mutation.isPending} onClick={() => mutation.mutate({ reminderId: reminder.id, completed: reminder.status !== "completed" })} type="button">{reminder.status === "completed" ? "Reopen" : "Complete"}</button></li>)}</ul> : <p className="mt-2 text-sm text-cocoa-500">No reminders due today.</p>}{!online ? <p className="mt-3 text-xs text-cocoa-500">Reminder changes sync when you’re online.</p> : null}</section>;
}
