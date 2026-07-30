import { getCostAmount } from "@/entities/budget/model";
import type { Itinerary, ItineraryDay, ItineraryItem, SchedulingStatus } from "@/entities/trip/model";

export type ScheduleViewMode = "agenda" | "timeline" | "calendar";

export type ScheduleConflictSeverity = "blocking" | "warning" | "suggestion";

export type ScheduleConflict = {
  id: string;
  severity: ScheduleConflictSeverity;
  title: string;
  description: string;
  dayNumber?: number;
  itemIndex?: number;
};

export type ScheduledItemRef = {
  day: ItineraryDay;
  dayIndex: number;
  dayNumber: number;
  item: ItineraryItem;
  itemIndex: number;
  startMinutes: number;
  endMinutes: number;
  durationMinutes: number;
};

export type TravelBlock = {
  id: string;
  dayNumber: number;
  afterItemIndex: number;
  mode: string;
  durationMinutes: number;
  distanceKm?: number | null;
  label: string;
};

export type DailyScheduleSummary = {
  dayNumber: number;
  activityCount: number;
  mealCount: number;
  travelDurationMinutes: number;
  walkingDistanceKm: number;
  estimatedCost: number;
  freeTimeMinutes: number;
  warningCount: number;
};

const statusValues: SchedulingStatus[] = ["Scheduled", "Unscheduled", "Conflict", "NeedsReview"];

const defaultDurations: Record<string, number> = {
  accommodation: 30,
  activity: 75,
  food: 60,
  meal: 60,
  place: 90,
  rest: 30,
  transfer: 45,
  transport: 45
};

export function normalizeSchedulingStatus(
  value: string | null | undefined,
  item?: Pick<ItineraryItem, "time" | "startTime" | "allDay">
): SchedulingStatus {
  const normalized = statusValues.find((status) => status.toLowerCase() === value?.toLowerCase());
  if (normalized) {
    return normalized;
  }
  if (item?.allDay) {
    return "Scheduled";
  }
  return getItemStartTime(item ?? { time: "" }) ? "Scheduled" : "Unscheduled";
}

export function getItemStartTime(item: Pick<ItineraryItem, "time" | "startTime">): string {
  return (item.startTime || item.time || "").trim();
}

export function isUnscheduledItem(item: ItineraryItem) {
  return normalizeSchedulingStatus(item.schedulingStatus, item) === "Unscheduled";
}

export function parseTimeToMinutes(value: string | null | undefined): number | null {
  const trimmed = (value ?? "").trim();
  if (!/^\d{2}:\d{2}$/.test(trimmed)) {
    return null;
  }
  const [hoursRaw, minutesRaw] = trimmed.split(":");
  const hours = Number(hoursRaw);
  const minutes = Number(minutesRaw);
  if (!Number.isInteger(hours) || !Number.isInteger(minutes) || hours > 23 || minutes > 59) {
    return null;
  }
  return hours * 60 + minutes;
}

export function formatMinutesAsTime(minutes: number): string {
  const clamped = Math.max(0, Math.min(23 * 60 + 59, Math.round(minutes)));
  const hours = Math.floor(clamped / 60);
  const remainder = clamped % 60;
  return `${String(hours).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}

export function getDefaultDuration(item: Pick<ItineraryItem, "type" | "category" | "transfer">) {
  if (item.transfer?.estimatedDurationMinutes && item.transfer.estimatedDurationMinutes > 0) {
    return item.transfer.estimatedDurationMinutes;
  }
  const token = `${item.type ?? ""} ${item.category ?? ""}`.toLowerCase();
  const match = Object.entries(defaultDurations).find(([key]) => token.includes(key));
  return match?.[1] ?? 60;
}

export function getItemDurationMinutes(item: ItineraryItem, startMinutes?: number | null) {
  if (typeof item.durationMinutes === "number" && Number.isFinite(item.durationMinutes) && item.durationMinutes > 0) {
    return Math.round(item.durationMinutes);
  }
  const start = startMinutes ?? parseTimeToMinutes(getItemStartTime(item));
  const end = parseTimeToMinutes(item.endTime);
  if (start != null && end != null && end > start) {
    return end - start;
  }
  return getDefaultDuration(item);
}

export function getItemEndTime(item: ItineraryItem): string {
  const start = parseTimeToMinutes(getItemStartTime(item));
  if (start == null) {
    return item.endTime?.trim() || "";
  }
  const duration = getItemDurationMinutes(item, start);
  return formatMinutesAsTime(start + duration);
}

export function scheduledItemsForDay(day: ItineraryDay, dayIndex: number): ScheduledItemRef[] {
  const dayNumber = day.day || dayIndex + 1;
  return (day.items ?? [])
    .map((item, itemIndex): ScheduledItemRef | null => {
      if (item.allDay || isUnscheduledItem(item)) {
        return null;
      }
      const startMinutes = parseTimeToMinutes(getItemStartTime(item));
      if (startMinutes == null) {
        return null;
      }
      const durationMinutes = getItemDurationMinutes(item, startMinutes);
      return {
        day,
        dayIndex,
        dayNumber,
        item,
        itemIndex,
        startMinutes,
        endMinutes: startMinutes + durationMinutes,
        durationMinutes
      };
    })
    .filter((value): value is ScheduledItemRef => value != null)
    .sort((left, right) => left.startMinutes - right.startMinutes || left.itemIndex - right.itemIndex);
}

export function unscheduledItemsForItinerary(itinerary: Itinerary) {
  return (itinerary.days ?? []).flatMap((day, dayIndex) => {
    const dayNumber = day.day || dayIndex + 1;
    return (day.items ?? [])
      .map((item, itemIndex) => ({ day, dayIndex, dayNumber, item, itemIndex }))
      .filter(({ item }) => isUnscheduledItem(item) || (!item.allDay && !getItemStartTime(item)));
  });
}

export function updateItineraryItem(
  itinerary: Itinerary,
  dayIndex: number,
  itemIndex: number,
  updater: (item: ItineraryItem) => ItineraryItem
): Itinerary {
  return {
    ...itinerary,
    days: (itinerary.days ?? []).map((day, currentDayIndex) =>
      currentDayIndex === dayIndex
        ? {
            ...day,
            items: day.items.map((item, currentItemIndex) =>
              currentItemIndex === itemIndex ? updater(item) : item
            )
          }
        : day
    )
  };
}

export function applyItemSchedule(
  item: ItineraryItem,
  updates: {
    startTime?: string | null;
    durationMinutes?: number | null;
    endTime?: string | null;
    allDay?: boolean;
    timezone?: string | null;
    schedulingStatus?: SchedulingStatus;
  }
): ItineraryItem {
  const allDay = updates.allDay ?? item.allDay ?? false;
  const startTime = allDay ? "" : (updates.startTime ?? getItemStartTime(item)).trim();
  const durationMinutes =
    updates.durationMinutes === undefined
      ? item.durationMinutes ?? null
      : updates.durationMinutes === null
        ? null
        : Math.max(5, Math.min(24 * 60, Math.round(updates.durationMinutes)));
  const startMinutes = parseTimeToMinutes(startTime);
  const computedDuration = durationMinutes ?? getItemDurationMinutes(item);
  const endTime =
    updates.endTime !== undefined
      ? updates.endTime == null
        ? null
        : updates.endTime.trim()
      : startMinutes != null
        ? formatMinutesAsTime(startMinutes + computedDuration)
        : item.endTime ?? null;
  const schedulingStatus =
    updates.schedulingStatus ?? (allDay || startTime ? "Scheduled" : "Unscheduled");

  return {
    ...item,
    time: startTime,
    startTime: startTime || null,
    endTime: endTime || null,
    durationMinutes,
    allDay,
    timezone: updates.timezone ?? item.timezone ?? null,
    schedulingStatus
  };
}

export function unscheduleItem(item: ItineraryItem): ItineraryItem {
  return {
    ...item,
    time: "",
    startTime: null,
    endTime: null,
    allDay: false,
    schedulingStatus: "Unscheduled"
  };
}

export function moveScheduledItem(
  itinerary: Itinerary,
  fromDayIndex: number,
  itemIndex: number,
  toDayIndex: number,
  startTime: string
): Itinerary {
  const days = itinerary.days ?? [];
  const sourceDay = days[fromDayIndex];
  const targetDay = days[toDayIndex];
  const item = sourceDay?.items[itemIndex];
  if (!sourceDay || !targetDay || !item) {
    return itinerary;
  }

  const scheduled = applyItemSchedule(item, {
    startTime,
    durationMinutes: getItemDurationMinutes(item),
    schedulingStatus: "Scheduled"
  });

  if (fromDayIndex === toDayIndex) {
    return updateItineraryItem(itinerary, fromDayIndex, itemIndex, () => scheduled);
  }

  const nextDays = days.map((day, dayIndex) => {
    if (dayIndex === fromDayIndex) {
      return {
        ...day,
        items: day.items.filter((_, currentIndex) => currentIndex !== itemIndex)
      };
    }
    if (dayIndex === toDayIndex) {
      return {
        ...day,
        items: [...day.items, scheduled]
      };
    }
    return day;
  });

  return { ...itinerary, days: nextDays };
}

export function adjustItemStart(item: ItineraryItem, deltaMinutes: number): ItineraryItem {
  const start = parseTimeToMinutes(getItemStartTime(item));
  if (start == null) {
    return applyItemSchedule(item, { startTime: "09:00", durationMinutes: getItemDurationMinutes(item) });
  }
  const duration = getItemDurationMinutes(item, start);
  const nextStart = Math.max(0, Math.min(23 * 60, start + deltaMinutes));
  return applyItemSchedule(item, { startTime: formatMinutesAsTime(nextStart), durationMinutes: duration });
}

export function adjustItemDuration(item: ItineraryItem, deltaMinutes: number): ItineraryItem {
  const duration = Math.max(15, Math.min(24 * 60, getItemDurationMinutes(item) + deltaMinutes));
  return applyItemSchedule(item, { durationMinutes: duration });
}

export function detectScheduleConflicts(itinerary: Itinerary): ScheduleConflict[] {
  const conflicts: ScheduleConflict[] = [];
  const seenScheduleKeys = new Set<string>();

  (itinerary.days ?? []).forEach((day, dayIndex) => {
    const dayNumber = day.day || dayIndex + 1;
    const scheduled = scheduledItemsForDay(day, dayIndex);
    let busyMinutes = 0;

    (day.items ?? []).forEach((item, itemIndex) => {
      const start = getItemStartTime(item);
      const status = normalizeSchedulingStatus(item.schedulingStatus, item);
      const startMinutes = parseTimeToMinutes(start);
      const endMinutes = parseTimeToMinutes(item.endTime);

      if (status !== "Unscheduled" && !item.allDay && !start) {
        conflicts.push(conflict("missing-start", "blocking", "Missing start time", `${item.name} needs a start time or should be unscheduled.`, dayNumber, itemIndex));
      }
      if (start && startMinutes == null) {
        conflicts.push(conflict("bad-start", "blocking", "Invalid start time", `${item.name} must use HH:mm time.`, dayNumber, itemIndex));
      }
      if (item.endTime && endMinutes == null) {
        conflicts.push(conflict("bad-end", "blocking", "Invalid end time", `${item.name} end time must use HH:mm time.`, dayNumber, itemIndex));
      }
      if (startMinutes != null && endMinutes != null && endMinutes <= startMinutes) {
        conflicts.push(conflict("end-before-start", "blocking", "End before start", `${item.name} ends before it starts.`, dayNumber, itemIndex));
      }
      if (typeof item.durationMinutes === "number" && item.durationMinutes <= 0) {
        conflicts.push(conflict("bad-duration", "blocking", "Invalid duration", `${item.name} duration must be positive.`, dayNumber, itemIndex));
      }
      if (status === "Conflict") {
        conflicts.push(conflict("status-conflict", "blocking", "Marked as conflict", `${item.name} is marked as a schedule conflict.`, dayNumber, itemIndex));
      }
      if (status === "NeedsReview") {
        conflicts.push(conflict("status-review", "warning", "Needs review", `${item.name} needs schedule review.`, dayNumber, itemIndex));
      }
      const duplicateKey = start ? `${dayNumber}:${start}:${item.name.trim().toLowerCase()}` : "";
      if (duplicateKey) {
        if (seenScheduleKeys.has(duplicateKey)) {
          conflicts.push(conflict("duplicate-schedule", "blocking", "Duplicate schedule", `${item.name} appears twice at ${start}.`, dayNumber, itemIndex));
        }
        seenScheduleKeys.add(duplicateKey);
      }
    });

    scheduled.forEach((current, index) => {
      busyMinutes += current.durationMinutes;
      const previous = scheduled[index - 1];
      if (!previous) {
        return;
      }
      if (current.startMinutes < previous.endMinutes) {
        conflicts.push(conflict("overlap", "blocking", "Overlapping activities", `${current.item.name} overlaps ${previous.item.name}.`, dayNumber, current.itemIndex));
      }
      const gap = current.startMinutes - previous.endMinutes;
      if (gap > 0 && gap < 15 && !isTravelLike(previous.item) && !isTravelLike(current.item)) {
        conflicts.push(conflict("short-transfer", "warning", "Very short transfer", `${current.item.name} starts ${gap} minutes after ${previous.item.name}.`, dayNumber, current.itemIndex));
      }
      if (gap <= 0) {
        return;
      }
      if (previous.item.place && current.item.place && gap < 30) {
        conflicts.push(conflict("missing-travel", "warning", "Missing travel time", `Add travel time between ${previous.item.name} and ${current.item.name}.`, dayNumber, current.itemIndex));
      }
    });

    if (busyMinutes > 12 * 60) {
      conflicts.push(conflict("long-day", "warning", "Daily duration is high", `Day ${dayNumber} has more than 12 scheduled hours.`, dayNumber));
    }
  });

  return conflicts;
}

export function dailyScheduleSummary(
  day: ItineraryDay,
  dayIndex: number,
  conflicts: ScheduleConflict[] = []
): DailyScheduleSummary {
  const dayNumber = day.day || dayIndex + 1;
  const scheduled = scheduledItemsForDay(day, dayIndex);
  const travelBlocks = deriveTravelBlocks(day, dayIndex);
  const activeWindowStart = scheduled[0]?.startMinutes ?? 0;
  const activeWindowEnd = scheduled[scheduled.length - 1]?.endMinutes ?? activeWindowStart;
  const busyMinutes = scheduled.reduce((total, item) => total + item.durationMinutes, 0);
  const estimatedCost = (day.items ?? []).reduce((total, item) => total + (getCostAmount(item.estimatedCost) ?? 0), 0);
  const walkingDistanceKm = (day.items ?? []).reduce(
    (total, item) => total + (typeof item.walkingDistanceKm === "number" ? item.walkingDistanceKm : 0),
    0
  );

  return {
    dayNumber,
    activityCount: (day.items ?? []).filter((item) => !isTravelLike(item) && !isUnscheduledItem(item)).length,
    mealCount: (day.items ?? []).filter(isMealLike).length,
    travelDurationMinutes: travelBlocks.reduce((total, block) => total + block.durationMinutes, 0),
    walkingDistanceKm,
    estimatedCost,
    freeTimeMinutes: Math.max(0, activeWindowEnd - activeWindowStart - busyMinutes),
    warningCount: conflicts.filter((item) => item.dayNumber === dayNumber).length
  };
}

export function deriveTravelBlocks(day: ItineraryDay, dayIndex: number): TravelBlock[] {
  const dayNumber = day.day || dayIndex + 1;
  const blocks: TravelBlock[] = [];
  (day.items ?? []).forEach((item, itemIndex) => {
    if (item.transfer) {
      blocks.push({
        id: `${dayNumber}-${itemIndex}-transfer`,
        dayNumber,
        afterItemIndex: itemIndex,
        mode: item.transfer.mode,
        durationMinutes: item.transfer.estimatedDurationMinutes ?? item.durationMinutes ?? 45,
        distanceKm: item.transfer.estimatedDistanceKm,
        label: `${item.transfer.from} to ${item.transfer.to}`
      });
      return;
    }
    if (isTravelLike(item)) {
      blocks.push({
        id: `${dayNumber}-${itemIndex}-travel`,
        dayNumber,
        afterItemIndex: itemIndex,
        mode: item.transportMode || item.type || "travel",
        durationMinutes: item.durationMinutes ?? 30,
        distanceKm: item.walkingDistanceKm,
        label: item.name
      });
    }
  });
  return blocks;
}

export function getTripDayDate(startDate: string | null | undefined, dayNumber: number) {
  if (!startDate) {
    return null;
  }
  const parsed = new Date(`${startDate}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  parsed.setDate(parsed.getDate() + dayNumber - 1);
  return parsed.toISOString().slice(0, 10);
}

export function formatDurationLabel(minutes: number) {
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min`;
}

export function isTravelLike(item: ItineraryItem) {
  const token = `${item.type ?? ""} ${item.transportMode ?? ""} ${item.name ?? ""}`.toLowerCase();
  return Boolean(item.transfer) || ["transfer", "transport", "walk", "taxi", "train", "bus", "metro", "drive", "flight"].some((part) => token.includes(part));
}

export function isAccommodationLike(item: ItineraryItem) {
  const token = `${item.type ?? ""} ${item.category ?? ""} ${item.name ?? ""}`.toLowerCase();
  return ["accommodation", "hotel", "check-in", "check in", "check-out", "check out", "luggage"].some((part) => token.includes(part));
}

function isMealLike(item: ItineraryItem) {
  const token = `${item.type ?? ""} ${item.category ?? ""} ${item.name ?? ""}`.toLowerCase();
  return ["food", "meal", "breakfast", "lunch", "dinner", "restaurant", "cafe"].some((part) => token.includes(part));
}

function conflict(
  key: string,
  severity: ScheduleConflictSeverity,
  title: string,
  description: string,
  dayNumber?: number,
  itemIndex?: number
): ScheduleConflict {
  return {
    id: `${key}:${dayNumber ?? "trip"}:${itemIndex ?? "day"}`,
    severity,
    title,
    description,
    dayNumber,
    itemIndex
  };
}
