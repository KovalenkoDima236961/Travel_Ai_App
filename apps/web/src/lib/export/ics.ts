import { costBadgeLabel } from "@/entities/budget/model";
import { slugifyForFilename } from "@/lib/export/export-filenames";
import type { ExportTrip } from "@/lib/export/trip-export-adapter";
import type { ItineraryItem } from "@/entities/trip/model";

type ParsedTime = {
  hour: number;
  minute: number;
};

type IcsEvent = {
  uid: string;
  start: string;
  end: string;
  summary: string;
  location: string;
  description: string;
};

export function generateTripIcs(exportTrip: ExportTrip): string {
  const dtstamp = formatIcsTimestampUtc(new Date());
  const events = buildIcsEvents(exportTrip);
  const lines = [
    "BEGIN:VCALENDAR",
    "VERSION:2.0",
    "PRODID:-//AI Travel Planner//Export v1//EN",
    "CALSCALE:GREGORIAN",
    "METHOD:PUBLISH",
    `X-WR-CALNAME:${escapeIcsText(`${exportTrip.destination} itinerary`)}`
  ];

  for (const event of events) {
    lines.push(
      "BEGIN:VEVENT",
      `UID:${escapeIcsText(event.uid)}`,
      `DTSTAMP:${dtstamp}`,
      `DTSTART:${event.start}`,
      `DTEND:${event.end}`,
      `SUMMARY:${escapeIcsText(event.summary)}`,
      `LOCATION:${escapeIcsText(event.location)}`,
      `DESCRIPTION:${escapeIcsText(event.description)}`,
      "END:VEVENT"
    );
  }

  lines.push("END:VCALENDAR");

  return lines.map(foldIcsLine).join("\r\n") + "\r\n";
}

export function getTripIcsEventCount(exportTrip: ExportTrip): number {
  return buildIcsEvents(exportTrip).length;
}

export function parseItemTime(time: string): ParsedTime | null {
  const normalized = time.trim().replace(/\./g, "").toUpperCase();
  const match = normalized.match(/^(\d{1,2})(?::(\d{2}))?\s*(AM|PM)?$/);

  if (!match) {
    return null;
  }

  let hour = Number(match[1]);
  const minute = match[2] == null ? 0 : Number(match[2]);
  const meridiem = match[3];

  if (!Number.isInteger(hour) || !Number.isInteger(minute) || minute < 0 || minute > 59) {
    return null;
  }

  if (meridiem) {
    if (hour < 1 || hour > 12) {
      return null;
    }
    if (meridiem === "AM") {
      hour = hour === 12 ? 0 : hour;
    } else {
      hour = hour === 12 ? 12 : hour + 12;
    }
  } else if (hour < 0 || hour > 23) {
    return null;
  }

  return { hour, minute };
}

export function addDaysToDate(startDate: string, offset: number): Date {
  const isoDate = startDate.match(/^(\d{4})-(\d{2})-(\d{2})/);

  if (isoDate) {
    const year = Number(isoDate[1]);
    const month = Number(isoDate[2]) - 1;
    const day = Number(isoDate[3]);
    const date = new Date(Date.UTC(year, month, day));
    date.setUTCDate(date.getUTCDate() + offset);
    return date;
  }

  const date = new Date(startDate);
  if (Number.isNaN(date.getTime())) {
    return new Date(Number.NaN);
  }
  date.setUTCDate(date.getUTCDate() + offset);
  return date;
}

export function formatIcsDateTimeLocal(date: Date, hour: number, minute: number): string {
  return [
    pad(date.getUTCFullYear(), 4),
    pad(date.getUTCMonth() + 1),
    pad(date.getUTCDate()),
    "T",
    pad(hour),
    pad(minute),
    "00"
  ].join("");
}

function buildIcsEvents(exportTrip: ExportTrip): IcsEvent[] {
  if (!exportTrip.startDate || !exportTrip.itinerary?.days?.length) {
    return [];
  }

  const destinationSlug = slugifyForFilename(exportTrip.destination);

  return exportTrip.itinerary.days.flatMap((day, dayIndex) => {
    const dayNumber = day.day || dayIndex + 1;
    const eventDate = addDaysToDate(exportTrip.startDate as string, dayNumber - 1);

    if (Number.isNaN(eventDate.getTime())) {
      return [];
    }

    return (day.items ?? []).flatMap((item, itemIndex) => {
      const timeRange = parseItemTimeRange(item.time);
      if (!timeRange) {
        return [];
      }
      if (!timeRange.end && item.endTime) {
        timeRange.end = parseItemTime(item.endTime);
      }

      const startMinutes = timeToMinutes(timeRange.start);
      const explicitEndMinutes = timeRange.end ? timeToMinutes(timeRange.end) : null;
      const endMinutes =
        explicitEndMinutes != null && explicitEndMinutes > startMinutes
          ? explicitEndMinutes
          : startMinutes + durationMinutesForItem(item);
      const endDate = addDaysToDate(exportTrip.startDate as string, dayNumber - 1);
      endDate.setUTCDate(endDate.getUTCDate() + Math.floor(endMinutes / (24 * 60)));
      const endMinutesInDay = endMinutes % (24 * 60);

      return [
        {
          uid: `trip-${destinationSlug}-day-${dayNumber}-item-${itemIndex + 1}@app-name.local`,
          start: formatIcsDateTimeLocal(eventDate, timeRange.start.hour, timeRange.start.minute),
          end: formatIcsDateTimeLocal(
            endDate,
            Math.floor(endMinutesInDay / 60),
            endMinutesInDay % 60
          ),
          summary: formatEventSummary(item, exportTrip.destination),
          location: formatEventLocation(item, exportTrip.destination),
          description: buildDescription(item, exportTrip.budgetCurrency)
        }
      ];
    });
  });
}

function parseItemTimeRange(time: string | null | undefined):
  | {
      start: ParsedTime;
      end?: ParsedTime | null;
    }
  | null {
  if (!time?.trim()) {
    return null;
  }

  const normalized = time.replace(/[–—]/g, "-");
  const [startValue, endValue] = normalized.split(/\s+-\s+|\s*-\s*/);
  const start = parseItemTime(startValue ?? "");

  if (!start) {
    return null;
  }

  return {
    start,
    end: endValue ? parseItemTime(endValue) : null
  };
}

function durationMinutesForItem(item: ItineraryItem): number {
  if (item.transfer?.estimatedDurationMinutes != null) {
    return item.transfer.estimatedDurationMinutes;
  }
  if (item.durationMinutes != null) {
    return item.durationMinutes;
  }
  const type = (item.type || "").toLowerCase();

  if (["food", "restaurant", "cafe"].includes(type)) {
    return 90;
  }
  if (type === "transport") {
    return 30;
  }
  if (type === "transfer") {
    return 120;
  }
  if (["rest", "break", "free_time", "free time"].includes(type)) {
    return 60;
  }

  return 60;
}

function formatEventSummary(item: ItineraryItem, fallbackDestination: string): string {
  if (item.transfer) {
    return `Transfer: ${item.transfer.from} -> ${item.transfer.to}`;
  }
  return item.name || `${fallbackDestination} itinerary item`;
}

function formatEventLocation(item: ItineraryItem, fallbackDestination: string): string {
  if (item.transfer) {
    return `${item.transfer.from} -> ${item.transfer.to}`;
  }
  return item.place?.address || item.place?.name || fallbackDestination;
}

function buildDescription(item: ItineraryItem, currency?: string | null): string {
  const lines = [
    item.transfer ? `Transport mode: ${String(item.transfer.mode).replace(/_/g, " ")}` : null,
    item.transfer?.estimatedDurationMinutes != null
      ? `Estimated duration: ${item.transfer.estimatedDurationMinutes} minutes`
      : null,
    item.transfer?.estimatedDistanceKm != null
      ? `Estimated distance: ${item.transfer.estimatedDistanceKm.toFixed(1)} km`
      : null,
    item.transfer ? "Verify schedules before travel." : null,
    ...(item.transfer?.warnings ?? []),
    item.note?.trim(),
    item.place?.name ? `Place: ${item.place.name}` : null,
    item.place?.address ? `Address: ${item.place.address}` : null,
    item.place?.mapUrl ? `Map: ${item.place.mapUrl}` : null,
    costBadgeLabel(item.estimatedCost, currency)
      ? `Estimated cost: ${costBadgeLabel(item.estimatedCost, currency)}`
      : null
  ].filter((line): line is string => Boolean(line));

  return lines.join("\n");
}

function escapeIcsText(value: string): string {
  return value
    .replace(/\\/g, "\\\\")
    .replace(/\r\n/g, "\n")
    .replace(/\r/g, "\n")
    .replace(/\n/g, "\\n")
    .replace(/;/g, "\\;")
    .replace(/,/g, "\\,");
}

function foldIcsLine(line: string): string {
  const limit = 75;
  if (line.length <= limit) {
    return line;
  }

  const folded: string[] = [];
  let remaining = line;

  while (remaining.length > limit) {
    folded.push(remaining.slice(0, limit));
    remaining = ` ${remaining.slice(limit)}`;
  }

  folded.push(remaining);
  return folded.join("\r\n");
}

function formatIcsTimestampUtc(date: Date): string {
  return [
    pad(date.getUTCFullYear(), 4),
    pad(date.getUTCMonth() + 1),
    pad(date.getUTCDate()),
    "T",
    pad(date.getUTCHours()),
    pad(date.getUTCMinutes()),
    pad(date.getUTCSeconds()),
    "Z"
  ].join("");
}

function timeToMinutes(time: ParsedTime): number {
  return time.hour * 60 + time.minute;
}

function pad(value: number, length = 2): string {
  return String(value).padStart(length, "0");
}
