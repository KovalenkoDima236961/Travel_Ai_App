import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import type { TripReminderPriority, TripReminderStatus } from "@/entities/trip-reminder/model";

export function ReminderStatusBadge({
  children,
  tone = "neutral"
}: {
  children: ReactNode;
  tone?: "neutral" | "warning" | "danger" | "ok" | "info";
}) {
  return (
    <span
      className={cn(
        "rounded-full px-2 py-0.5 text-xs font-medium",
        tone === "neutral" && "bg-slate-100 text-slate-700",
        tone === "warning" && "bg-amber-100 text-amber-800",
        tone === "danger" && "bg-red-100 text-red-800",
        tone === "ok" && "bg-emerald-100 text-emerald-800",
        tone === "info" && "bg-sky-100 text-sky-800"
      )}
    >
      {children}
    </span>
  );
}

export function reminderStatusTone(status: TripReminderStatus) {
  if (status === "completed") {
    return "ok";
  }
  if (status === "disabled" || status === "cancelled") {
    return "neutral";
  }
  if (status === "failed") {
    return "danger";
  }
  if (status === "sent") {
    return "info";
  }
  return "warning";
}

export function reminderPriorityTone(priority: TripReminderPriority) {
  if (priority === "critical") {
    return "danger";
  }
  if (priority === "high") {
    return "warning";
  }
  if (priority === "low") {
    return "ok";
  }
  return "neutral";
}
