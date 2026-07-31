"use client";

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type DragEvent,
  type ReactNode
} from "react";
import type { TripAccommodation } from "@/entities/accommodation/model";
import { formatMoney } from "@/entities/budget/model";
import type { Itinerary, ItineraryItem } from "@/entities/trip/model";
import type { WeatherForecast } from "@/entities/weather/model";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { trackAlphaEvent } from "@/lib/api/alpha";
import { cn } from "@/lib/utils";
import {
  adjustItemDuration,
  adjustItemStart,
  dailyScheduleSummary,
  deriveTravelBlocks,
  detectScheduleConflicts,
  formatDurationLabel,
  formatMinutesAsTime,
  getItemDurationMinutes,
  getItemEndTime,
  getTripDayDate,
  isAccommodationLike,
  isTravelLike,
  moveScheduledItem,
  normalizeSchedulingStatus,
  parseTimeToMinutes,
  scheduledItemsForDay,
  type ScheduleConflict,
  type ScheduleViewMode,
  unscheduleItem,
  unscheduledItemsForItinerary,
  updateItineraryItem
} from "../model/schedule";

type FeatureAvailability = Partial<Record<ScheduleViewMode, boolean>>;

type SchedulePlanningWorkspaceProps = {
  itinerary: Itinerary;
  currency: string;
  startDate?: string | null;
  accommodation?: TripAccommodation | null;
  weatherForecast?: WeatherForecast | null;
  editable?: boolean;
  dragDropEnabled?: boolean;
  conflictDetectionEnabled?: boolean;
  featureAvailability?: FeatureAvailability;
  storageKey?: string;
  analyticsEntityId?: string;
  initialView?: ScheduleViewMode | null;
  initialDayNumber?: number | null;
  agendaSlot?: ReactNode;
  onViewChange?: (view: ScheduleViewMode) => void;
  onChange?: (itinerary: Itinerary) => void;
};

type DragPayload = {
  dayIndex: number;
  itemIndex: number;
};

const viewLabels: Record<ScheduleViewMode, string> = {
  agenda: "Agenda",
  timeline: "Timeline",
  calendar: "Calendar"
};

const timelineStartHour = 6;
const timelineEndHour = 24;
const hourHeightPx = 64;

export function SchedulePlanningWorkspace({
  itinerary,
  currency,
  startDate,
  accommodation,
  weatherForecast,
  editable = false,
  dragDropEnabled = true,
  conflictDetectionEnabled = true,
  featureAvailability,
  storageKey = "timeline-planning-view",
  analyticsEntityId,
  initialView,
  initialDayNumber,
  agendaSlot,
  onViewChange,
  onChange
}: SchedulePlanningWorkspaceProps) {
  const [viewMode, setViewMode] = useState<ScheduleViewMode>("agenda");
  const [selectedDayIndex, setSelectedDayIndex] = useState(0);
  const [undoStack, setUndoStack] = useState<Itinerary[]>([]);
  const [search, setSearch] = useState("");
  const [calendarMode, setCalendarMode] = useState<"trip" | "week" | "month">("trip");
  const lastConflictMetricSignature = useRef("");

  const enabledModes = useMemo(() => {
    const availability = {
      agenda: featureAvailability?.agenda ?? true,
      timeline: featureAvailability?.timeline ?? true,
      calendar: featureAvailability?.calendar ?? true
    };
    return (Object.keys(availability) as ScheduleViewMode[]).filter((mode) => availability[mode]);
  }, [featureAvailability]);

  const conflicts = useMemo(
    () => (conflictDetectionEnabled ? detectScheduleConflicts(itinerary) : []),
    [conflictDetectionEnabled, itinerary]
  );
  const unscheduled = useMemo(() => unscheduledItemsForItinerary(itinerary), [itinerary]);
  const days = itinerary.days ?? [];
  const selectedDay = days[selectedDayIndex] ?? days[0];

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const stored = window.localStorage.getItem(storageKey) as ScheduleViewMode | null;
    const mobileDefault = window.matchMedia("(max-width: 767px)").matches ? "agenda" : "timeline";
    const next = initialView && enabledModes.includes(initialView)
      ? initialView
      : stored && enabledModes.includes(stored)
        ? stored
        : mobileDefault;
    setViewMode(enabledModes.includes(next) ? next : enabledModes[0] ?? "agenda");
  }, [enabledModes, initialView, storageKey]);

  useEffect(() => {
    if (initialDayNumber == null) {
      return;
    }
    const index = days.findIndex((day, dayIndex) => (day.day || dayIndex + 1) === initialDayNumber);
    if (index >= 0) {
      setSelectedDayIndex(index);
    }
  }, [days, initialDayNumber]);

  useEffect(() => {
    if (enabledModes.length > 0 && !enabledModes.includes(viewMode)) {
      setViewMode(enabledModes[0]);
    }
  }, [enabledModes, viewMode]);

  function selectView(mode: ScheduleViewMode) {
    setViewMode(mode);
    onViewChange?.(mode);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(storageKey, mode);
    }
    if (analyticsEntityId) {
      trackAlphaEvent({
        eventName: `${mode}_opened`,
        feature: "timeline_planning",
        entityType: "trip",
        entityId: analyticsEntityId
      });
    }
  }

  function commit(next: Itinerary) {
    if (!editable || !onChange) {
      return;
    }
    setUndoStack((current) => [cloneItinerary(itinerary), ...current].slice(0, 12));
    onChange(next);
  }

  function undo() {
    const [previous, ...rest] = undoStack;
    if (!previous || !onChange) {
      return;
    }
    setUndoStack(rest);
    onChange(previous);
  }

  function updateItem(dayIndex: number, itemIndex: number, updater: (item: ItineraryItem) => ItineraryItem) {
    commit(updateItineraryItem(itinerary, dayIndex, itemIndex, updater));
  }

  function moveItem(dayIndex: number, itemIndex: number, toDayIndex: number, startMinutes: number) {
    const sourceDay = days[dayIndex];
    const targetDay = days[toDayIndex];
    if (!sourceDay || !targetDay || (dayIndex !== toDayIndex && sourceDay.items.length <= 1)) {
      return;
    }
    commit(moveScheduledItem(itinerary, dayIndex, itemIndex, toDayIndex, formatMinutesAsTime(startMinutes)));
    setSelectedDayIndex(toDayIndex);
  }

  function handleDragStart(event: DragEvent<HTMLElement>, payload: DragPayload) {
    if (!editable || !dragDropEnabled) {
      return;
    }
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("application/x-itinerary-item", JSON.stringify(payload));
  }

  function handleDrop(event: DragEvent<HTMLElement>, toDayIndex: number, startMinutes: number) {
    event.preventDefault();
    const payload = readDragPayload(event);
    if (!payload) {
      return;
    }
    moveItem(payload.dayIndex, payload.itemIndex, toDayIndex, startMinutes);
    if (analyticsEntityId) {
      trackAlphaEvent({
        eventName: "drag_operations",
        feature: "timeline_planning",
        entityType: "trip",
        entityId: analyticsEntityId,
        metadata: { targetDayIndex: toDayIndex }
      });
    }
  }

  useEffect(() => {
    if (!analyticsEntityId || conflicts.length === 0) {
      return;
    }
    const blocking = conflicts.filter((conflict) => conflict.severity === "blocking").length;
    const signature = `${conflicts.length}:${blocking}`;
    if (lastConflictMetricSignature.current === signature) {
      return;
    }
    lastConflictMetricSignature.current = signature;
    trackAlphaEvent({
      eventName: "conflicts_detected",
      feature: "timeline_planning",
      entityType: "trip",
      entityId: analyticsEntityId,
      metadata: { conflictCount: conflicts.length, blockingCount: blocking }
    });
  }, [analyticsEntityId, conflicts]);

  function jumpToSearchMatch() {
    const needle = search.trim().toLowerCase();
    if (!needle) {
      return;
    }
    for (const [dayIndex, day] of days.entries()) {
      const itemIndex = day.items.findIndex((item) =>
        `${item.name} ${item.note ?? ""} ${item.type}`.toLowerCase().includes(needle)
      );
      if (itemIndex >= 0) {
        setSelectedDayIndex(dayIndex);
        const dayNumber = day.day || dayIndex + 1;
        window.setTimeout(() => {
          document
            .getElementById(`schedule-day-${dayNumber}-item-${itemIndex}`)
            ?.scrollIntoView({ behavior: "smooth", block: "center" });
          document
            .getElementById(`day-${dayNumber}-item-${itemIndex}`)
            ?.scrollIntoView({ behavior: "smooth", block: "center" });
        }, 50);
        return;
      }
    }
  }

  if (enabledModes.length === 0) {
    return agendaSlot ? <>{agendaSlot}</> : null;
  }

  return (
    <section className="scroll-mt-24 space-y-4" aria-label="Itinerary planning workspace">
      <ScheduleToolbar
        editable={editable}
        enabledModes={enabledModes}
        onSearch={jumpToSearchMatch}
        onUndo={undo}
        onViewChange={selectView}
        search={search}
        setSearch={setSearch}
        undoDisabled={undoStack.length === 0}
        viewMode={viewMode}
      />
      <ConflictBanner conflicts={conflicts} />
      <ScheduleSummaryStrip
        accommodation={accommodation}
        conflicts={conflicts}
        currency={currency}
        days={days}
        startDate={startDate}
        weatherForecast={weatherForecast}
      />
      {unscheduled.length > 0 || editable ? (
        <UnscheduledPanel
          editable={editable}
          dragDropEnabled={dragDropEnabled}
          dayCount={days.length}
          items={unscheduled}
          onDragStart={handleDragStart}
          onSchedule={(dayIndex, itemIndex, targetDayIndex, startTime) =>
            moveItem(dayIndex, itemIndex, targetDayIndex, parseTimeToMinutes(startTime) ?? 9 * 60)
          }
          selectedDayIndex={selectedDayIndex}
          setSelectedDayIndex={setSelectedDayIndex}
        />
      ) : null}

      {viewMode === "agenda" ? (
        agendaSlot && !editable ? (
          <>{agendaSlot}</>
        ) : (
          <AgendaView
            conflicts={conflicts}
            currency={currency}
            days={days}
            dragDropEnabled={dragDropEnabled}
            editable={editable}
            onDragStart={handleDragStart}
            onItemUpdate={updateItem}
            startDate={startDate}
            weatherForecast={weatherForecast}
          />
        )
      ) : null}

      {viewMode === "timeline" ? (
        <TimelineView
          conflicts={conflicts}
          days={days}
          editable={editable}
          dragDropEnabled={dragDropEnabled}
          onDragStart={handleDragStart}
          onDrop={handleDrop}
          onItemMove={moveItem}
          onItemUpdate={updateItem}
          selectedDay={selectedDay}
          selectedDayIndex={selectedDayIndex}
          setSelectedDayIndex={setSelectedDayIndex}
        />
      ) : null}

      {viewMode === "calendar" ? (
        <CalendarView
          accommodation={accommodation}
          calendarMode={calendarMode}
          conflicts={conflicts}
          currency={currency}
          days={days}
          setCalendarMode={setCalendarMode}
          setSelectedDayIndex={setSelectedDayIndex}
          startDate={startDate}
          weatherForecast={weatherForecast}
        />
      ) : null}
    </section>
  );
}

function ScheduleToolbar({
  editable,
  enabledModes,
  onSearch,
  onUndo,
  onViewChange,
  search,
  setSearch,
  undoDisabled,
  viewMode
}: {
  editable: boolean;
  enabledModes: ScheduleViewMode[];
  onSearch: () => void;
  onUndo: () => void;
  onViewChange: (mode: ScheduleViewMode) => void;
  search: string;
  setSearch: (value: string) => void;
  undoDisabled: boolean;
  viewMode: ScheduleViewMode;
}) {
  return (
    <div className="rounded-[16px] border border-sand-300 bg-white p-3 shadow-[0_1px_2px_rgba(34,26,20,0.03)]">
      <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
        <div className="inline-flex w-full rounded-md border border-sand-300 bg-sand-50 p-1 sm:w-auto">
          {enabledModes.map((mode) => (
            <button
              key={mode}
              type="button"
              aria-pressed={viewMode === mode}
              onClick={() => onViewChange(mode)}
              className={cn(
                "min-h-10 flex-1 rounded px-3 text-[13px] font-semibold transition sm:min-w-24",
                viewMode === mode
                  ? "bg-white text-cocoa-900 shadow-[0_1px_2px_rgba(34,26,20,0.06)]"
                  : "text-cocoa-500 hover:text-cocoa-900"
              )}
            >
              {viewLabels[mode]}
            </button>
          ))}
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Input
            aria-label="Search itinerary activities"
            className="min-h-10 sm:w-72"
            onChange={(event) => setSearch(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                onSearch();
              }
            }}
            placeholder="Search activities"
            value={search}
          />
          <Button onClick={onSearch} type="button" variant="secondary">
            Jump
          </Button>
          {editable ? (
            <Button disabled={undoDisabled} onClick={onUndo} type="button" variant="secondary">
              Undo
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function ConflictBanner({ conflicts }: { conflicts: ScheduleConflict[] }) {
  if (conflicts.length === 0) {
    return (
      <div className="rounded-[14px] border border-[#DCE8DD] bg-[#F2F7F1] p-4 text-[14px] text-[#38543F]">
        Schedule has no detected conflicts.
      </div>
    );
  }

  const blocking = conflicts.filter((item) => item.severity === "blocking");
  const warnings = conflicts.length - blocking.length;

  return (
    <div className="rounded-[14px] border border-[#EAD9B8] bg-[#FDF7E8] p-4 text-[14px] text-[#7A5727]">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-baseline sm:justify-between">
        <h3 className="font-semibold text-cocoa-900">
          {blocking.length > 0 ? `${blocking.length} blocking conflict${blocking.length === 1 ? "" : "s"}` : "Schedule warnings"}
        </h3>
        <span className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
          {warnings} warning{warnings === 1 ? "" : "s"}
        </span>
      </div>
      <ul className="mt-3 space-y-2">
        {conflicts.slice(0, 4).map((conflict) => (
          <li key={conflict.id} className="flex gap-2">
            <span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-[#B57F24]" />
            <span>
              <span className="font-semibold">{conflict.title}</span>
              <span className="text-cocoa-500"> · {conflict.description}</span>
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function ScheduleSummaryStrip({
  accommodation,
  conflicts,
  currency,
  days,
  startDate,
  weatherForecast
}: {
  accommodation?: TripAccommodation | null;
  conflicts: ScheduleConflict[];
  currency: string;
  days: Itinerary["days"];
  startDate?: string | null;
  weatherForecast?: WeatherForecast | null;
}) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {days.map((day, dayIndex) => {
        const dayNumber = day.day || dayIndex + 1;
        const summary = dailyScheduleSummary(day, dayIndex, conflicts);
        const date = day.date || getTripDayDate(startDate, dayNumber);
        const weather = weatherForecast?.days.find((item) => item.date === date);
        const accommodationLabel = accommodationAnchor(accommodation, date);

        return (
          <div key={dayNumber} className="rounded-[14px] border border-sand-300 bg-white p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
                  Day {dayNumber}
                </p>
                <h3 className="mt-1 text-[15px] font-semibold text-cocoa-900">{day.title}</h3>
              </div>
              <span className="text-[12.5px] font-medium text-cocoa-400">{date ?? "No date"}</span>
            </div>
            <div className="mt-3 grid grid-cols-2 gap-2 text-[12.5px] text-cocoa-500">
              <span>{summary.activityCount} activities</span>
              <span>{summary.mealCount} meals</span>
              <span>{formatDurationLabel(summary.travelDurationMinutes)} travel</span>
              <span>{Math.round(summary.walkingDistanceKm * 10) / 10} km walk</span>
              <span>{formatMoney(summary.estimatedCost, currency)}</span>
              <span>{formatDurationLabel(summary.freeTimeMinutes)} free</span>
            </div>
            {weather || accommodationLabel || summary.warningCount > 0 ? (
              <div className="mt-3 flex flex-wrap gap-2 text-[12px] font-semibold">
                {weather ? (
                  <span className="rounded-full bg-sand-100 px-2.5 py-1 text-cocoa-500">
                    {weather.summary || weather.condition}
                  </span>
                ) : null}
                {accommodationLabel ? (
                  <span className="rounded-full bg-[#EDF3EA] px-2.5 py-1 text-[#38543F]">
                    {accommodationLabel}
                  </span>
                ) : null}
                {summary.warningCount > 0 ? (
                  <span className="rounded-full bg-[#FDF0E3] px-2.5 py-1 text-[#96682A]">
                    {summary.warningCount} warnings
                  </span>
                ) : null}
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function UnscheduledPanel({
  dragDropEnabled,
  dayCount,
  editable,
  items,
  onDragStart,
  onSchedule,
  selectedDayIndex,
  setSelectedDayIndex
}: {
  dragDropEnabled: boolean;
  dayCount: number;
  editable: boolean;
  items: ReturnType<typeof unscheduledItemsForItinerary>;
  onDragStart: (event: DragEvent<HTMLElement>, payload: DragPayload) => void;
  onSchedule: (dayIndex: number, itemIndex: number, targetDayIndex: number, startTime: string) => void;
  selectedDayIndex: number;
  setSelectedDayIndex: (dayIndex: number) => void;
}) {
  if (items.length === 0 && !editable) {
    return null;
  }

  return (
    <div className="rounded-[16px] border border-sand-300 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="text-[15px] font-semibold text-cocoa-900">Unscheduled</h3>
          <p className="text-[13px] text-cocoa-500">
            Activities without a time stay in the itinerary until assigned.
          </p>
        </div>
        {items.length > 0 ? (
          <span className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            {items.length} item{items.length === 1 ? "" : "s"}
          </span>
        ) : null}
      </div>
      {items.length > 0 ? (
        <div className="mt-3 grid gap-2 lg:grid-cols-2">
          {items.map(({ dayIndex, dayNumber, item, itemIndex }) => (
            <div
              key={`${dayNumber}-${itemIndex}-${item.name}`}
              draggable={editable && dragDropEnabled}
              onDragStart={(event) => onDragStart(event, { dayIndex, itemIndex })}
              className="rounded-md border border-dashed border-sand-400 bg-sand-50 p-3"
            >
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0">
                  <p className="truncate text-[14px] font-semibold text-cocoa-900">{item.name}</p>
                  <p className="text-[12px] text-cocoa-500">Day {dayNumber} · {item.type}</p>
                </div>
                {editable ? (
                  <div className="grid grid-cols-[6.5rem_5rem] gap-2">
                    <Select
                      aria-label={`Target day for ${item.name}`}
                      onChange={(event) => setSelectedDayIndex(Number(event.target.value))}
                      value={selectedDayIndex}
                    >
                      {Array.from({ length: dayCount }).map((_, index) => (
                        <option key={index} value={index}>
                          Day {index + 1}
                        </option>
                      ))}
                    </Select>
                    <Button
                      onClick={() => onSchedule(dayIndex, itemIndex, selectedDayIndex, "09:00")}
                      type="button"
                      variant="secondary"
                    >
                      09:00
                    </Button>
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <p className="mt-3 text-[13px] text-cocoa-400">No unscheduled activities.</p>
      )}
    </div>
  );
}

function AgendaView({
  conflicts,
  currency,
  days,
  dragDropEnabled,
  editable,
  onDragStart,
  onItemUpdate,
  startDate,
  weatherForecast
}: {
  conflicts: ScheduleConflict[];
  currency: string;
  days: Itinerary["days"];
  dragDropEnabled: boolean;
  editable: boolean;
  onDragStart: (event: DragEvent<HTMLElement>, payload: DragPayload) => void;
  onItemUpdate: (dayIndex: number, itemIndex: number, updater: (item: ItineraryItem) => ItineraryItem) => void;
  startDate?: string | null;
  weatherForecast?: WeatherForecast | null;
}) {
  return (
    <div className="space-y-5">
      {days.map((day, dayIndex) => {
        const dayNumber = day.day || dayIndex + 1;
        const date = day.date || getTripDayDate(startDate, dayNumber);
        const weather = weatherForecast?.days.find((item) => item.date === date);
        const dayConflicts = conflicts.filter((conflict) => conflict.dayNumber === dayNumber);
        const summary = dailyScheduleSummary(day, dayIndex, conflicts);
        const ordered = [...scheduledItemsForDay(day, dayIndex)];
        const allDayItems = day.items
          .map((item, itemIndex) => ({ item, itemIndex }))
          .filter(({ item }) => item.allDay);

        return (
          <section key={dayNumber} id={`schedule-day-${dayNumber}`} className="scroll-mt-24 rounded-[16px] border border-sand-300 bg-white p-5">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h3 className="font-newsreader text-[25px] font-semibold text-cocoa-900">
                  Day {dayNumber} <span className="font-normal text-[#A08D78]">·</span>{" "}
                  <em className="font-medium not-italic">{day.title}</em>
                </h3>
                <div className="mt-1 flex flex-wrap gap-2 text-[12.5px] font-medium text-cocoa-500">
                  <span>{date ?? "No date"}</span>
                  {weather ? <span>{weather.summary || weather.condition}</span> : null}
                  <span>{formatMoney(summary.estimatedCost, currency)}</span>
                  <span>{formatDurationLabel(summary.freeTimeMinutes)} free</span>
                </div>
              </div>
              {dayConflicts.length > 0 ? (
                <span className="rounded-full bg-[#FDF0E3] px-2.5 py-1 text-[12px] font-semibold text-[#96682A]">
                  {dayConflicts.length} warning{dayConflicts.length === 1 ? "" : "s"}
                </span>
              ) : null}
            </div>
            {allDayItems.length > 0 ? (
              <div className="mt-4 flex flex-wrap gap-2">
                {allDayItems.map(({ item, itemIndex }) => (
                  <span key={`${item.name}-${itemIndex}`} className="rounded-full bg-[#EDF3EA] px-3 py-1 text-[12px] font-semibold text-[#38543F]">
                    All day · {item.name}
                  </span>
                ))}
              </div>
            ) : null}
            <div className="mt-4 divide-y divide-sand-200">
              {ordered.map(({ item, itemIndex, startMinutes, durationMinutes }) => (
                <AgendaItem
                  key={`${itemIndex}-${item.name}`}
                  currency={currency}
                  dayIndex={dayIndex}
                  dayNumber={dayNumber}
                  durationMinutes={durationMinutes}
                  dragDropEnabled={dragDropEnabled}
                  editable={editable}
                  item={item}
                  itemIndex={itemIndex}
                  onDragStart={onDragStart}
                  onItemUpdate={onItemUpdate}
                  startMinutes={startMinutes}
                />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function AgendaItem({
  currency,
  dayIndex,
  dayNumber,
  durationMinutes,
  dragDropEnabled,
  editable,
  item,
  itemIndex,
  onDragStart,
  onItemUpdate,
  startMinutes
}: {
  currency: string;
  dayIndex: number;
  dayNumber: number;
  durationMinutes: number;
  dragDropEnabled: boolean;
  editable: boolean;
  item: ItineraryItem;
  itemIndex: number;
  onDragStart: (event: DragEvent<HTMLElement>, payload: DragPayload) => void;
  onItemUpdate: (dayIndex: number, itemIndex: number, updater: (item: ItineraryItem) => ItineraryItem) => void;
  startMinutes: number;
}) {
  return (
    <div
      id={`schedule-day-${dayNumber}-item-${itemIndex}`}
      draggable={editable && dragDropEnabled && !isTravelLike(item)}
      onDragStart={(event) => onDragStart(event, { dayIndex, itemIndex })}
      className="grid scroll-mt-24 gap-3 py-4 sm:grid-cols-[6rem_minmax(0,1fr)_auto]"
    >
      <div className="text-[13px] font-bold text-cocoa-900">
        {formatMinutesAsTime(startMinutes)}
        <span className="block text-[12px] font-medium text-cocoa-400">
          {formatDurationLabel(durationMinutes)}
        </span>
      </div>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <span className={cn("rounded-full px-2.5 py-1 text-[11px] font-semibold", toneForItem(item))}>
            {isAccommodationLike(item) ? "Accommodation" : isTravelLike(item) ? "Travel" : item.type}
          </span>
          <p className="font-semibold text-cocoa-900">{item.name}</p>
        </div>
        {item.note ? <p className="mt-1 text-[13px] leading-6 text-cocoa-500">{item.note}</p> : null}
        {isTravelLike(item) ? <TravelDetail item={item} currency={currency} /> : null}
      </div>
      {editable ? (
        <ItemScheduleActions
          dayIndex={dayIndex}
          item={item}
          itemIndex={itemIndex}
          onItemUpdate={onItemUpdate}
        />
      ) : null}
    </div>
  );
}

function TimelineView({
  conflicts,
  days,
  dragDropEnabled,
  editable,
  onDragStart,
  onDrop,
  onItemMove,
  onItemUpdate,
  selectedDay,
  selectedDayIndex,
  setSelectedDayIndex
}: {
  conflicts: ScheduleConflict[];
  days: Itinerary["days"];
  dragDropEnabled: boolean;
  editable: boolean;
  onDragStart: (event: DragEvent<HTMLElement>, payload: DragPayload) => void;
  onDrop: (event: DragEvent<HTMLElement>, toDayIndex: number, startMinutes: number) => void;
  onItemMove: (dayIndex: number, itemIndex: number, toDayIndex: number, startMinutes: number) => void;
  onItemUpdate: (dayIndex: number, itemIndex: number, updater: (item: ItineraryItem) => ItineraryItem) => void;
  selectedDay?: Itinerary["days"][number];
  selectedDayIndex: number;
  setSelectedDayIndex: (dayIndex: number) => void;
}) {
  if (!selectedDay) {
    return null;
  }
  const dayNumber = selectedDay.day || selectedDayIndex + 1;
  const scheduled = scheduledItemsForDay(selectedDay, selectedDayIndex);
  const travelBlocks = deriveTravelBlocks(selectedDay, selectedDayIndex);
  const totalHeight = (timelineEndHour - timelineStartHour) * hourHeightPx;
  const now = new Date();
  const nowMinutes = now.getHours() * 60 + now.getMinutes();
  const showNow = nowMinutes >= timelineStartHour * 60 && nowMinutes <= timelineEndHour * 60;

  return (
    <div className="rounded-[16px] border border-sand-300 bg-white p-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex gap-2 overflow-x-auto pb-1">
          {days.map((day, dayIndex) => (
            <button
              key={`${day.day}-${dayIndex}`}
              type="button"
              onClick={() => setSelectedDayIndex(dayIndex)}
              className={cn(
                "h-10 shrink-0 rounded-md border px-3 text-[13px] font-semibold",
                selectedDayIndex === dayIndex
                  ? "border-clay bg-clay text-sand-100"
                  : "border-sand-300 bg-white text-cocoa-500"
              )}
            >
              Day {day.day || dayIndex + 1}
            </button>
          ))}
        </div>
        <div className="text-[13px] font-medium text-cocoa-500">
          Drag blocks onto an hour. Buttons provide keyboard alternatives.
        </div>
      </div>

      {travelBlocks.length > 0 ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {travelBlocks.map((block) => (
            <span key={block.id} className="rounded-full bg-sand-100 px-3 py-1 text-[12px] font-semibold text-cocoa-500">
              {block.label} · {formatDurationLabel(block.durationMinutes)}
            </span>
          ))}
        </div>
      ) : null}

      <div className="mt-5 grid grid-cols-[4rem_minmax(0,1fr)] gap-3">
        <div className="relative" style={{ height: totalHeight }}>
          {Array.from({ length: timelineEndHour - timelineStartHour + 1 }).map((_, index) => (
            <div
              key={index}
              className="absolute right-0 text-[12px] font-semibold text-cocoa-400"
              style={{ top: index * hourHeightPx - 8 }}
            >
              {String(timelineStartHour + index).padStart(2, "0")}:00
            </div>
          ))}
        </div>
        <div className="relative overflow-hidden rounded-[12px] border border-sand-300 bg-sand-50" style={{ height: totalHeight }}>
          {Array.from({ length: timelineEndHour - timelineStartHour }).map((_, index) => {
            const startMinutes = (timelineStartHour + index) * 60;
            return (
              <div
                key={index}
                onDragOver={(event) => editable && dragDropEnabled && event.preventDefault()}
                onDrop={(event) => onDrop(event, selectedDayIndex, startMinutes)}
                className="absolute left-0 right-0 border-t border-sand-300"
                style={{ top: index * hourHeightPx, height: hourHeightPx }}
              />
            );
          })}
          {showNow ? (
            <div
              className="absolute left-0 right-0 z-10 border-t-2 border-clay"
              style={{ top: ((nowMinutes - timelineStartHour * 60) / 60) * hourHeightPx }}
            >
              <span className="absolute -top-3 left-2 rounded-full bg-clay px-2 py-0.5 text-[10px] font-semibold text-sand-100">
                Now
              </span>
            </div>
          ) : null}
          {scheduled.map((entry) => (
            <TimelineBlock
              key={`${entry.itemIndex}-${entry.item.name}`}
              conflicts={conflicts}
              dayIndex={selectedDayIndex}
              dayNumber={dayNumber}
              dragDropEnabled={dragDropEnabled}
              editable={editable}
              entry={entry}
              onDragStart={onDragStart}
              onItemMove={onItemMove}
              onItemUpdate={onItemUpdate}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function TimelineBlock({
  conflicts,
  dayIndex,
  dayNumber,
  dragDropEnabled,
  editable,
  entry,
  onDragStart,
  onItemMove,
  onItemUpdate
}: {
  conflicts: ScheduleConflict[];
  dayIndex: number;
  dayNumber: number;
  dragDropEnabled: boolean;
  editable: boolean;
  entry: ReturnType<typeof scheduledItemsForDay>[number];
  onDragStart: (event: DragEvent<HTMLElement>, payload: DragPayload) => void;
  onItemMove: (dayIndex: number, itemIndex: number, toDayIndex: number, startMinutes: number) => void;
  onItemUpdate: (dayIndex: number, itemIndex: number, updater: (item: ItineraryItem) => ItineraryItem) => void;
}) {
  const top = ((entry.startMinutes - timelineStartHour * 60) / 60) * hourHeightPx;
  const height = Math.max(44, (entry.durationMinutes / 60) * hourHeightPx);
  const itemConflicts = conflicts.filter(
    (conflict) => conflict.dayNumber === dayNumber && conflict.itemIndex === entry.itemIndex
  );
  const travel = isTravelLike(entry.item);

  return (
    <article
      id={`schedule-day-${dayNumber}-item-${entry.itemIndex}`}
      draggable={editable && dragDropEnabled && !travel}
      onDragStart={(event) => onDragStart(event, { dayIndex, itemIndex: entry.itemIndex })}
      className={cn(
        "absolute left-2 right-2 z-20 overflow-hidden rounded-md border p-3 shadow-[0_8px_18px_rgba(34,26,20,0.08)]",
        travel ? "border-sand-400 bg-sand-100" : "border-[#E5C3B6] bg-white",
        itemConflicts.length > 0 ? "ring-2 ring-[#B57F24]" : ""
      )}
      style={{ top, height }}
    >
      <div className="flex h-full flex-col justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-[13px] font-semibold text-cocoa-900">{entry.item.name}</p>
          <p className="text-[11px] font-medium text-cocoa-500">
            {formatMinutesAsTime(entry.startMinutes)} to {getItemEndTime(entry.item)}
          </p>
        </div>
        {editable && !travel ? (
          <div className="flex flex-wrap gap-1">
            <SmallAction label="Earlier" onClick={() => onItemUpdate(dayIndex, entry.itemIndex, (item) => adjustItemStart(item, -15))} />
            <SmallAction label="Later" onClick={() => onItemUpdate(dayIndex, entry.itemIndex, (item) => adjustItemStart(item, 15))} />
            <SmallAction label="Shorter" onClick={() => onItemUpdate(dayIndex, entry.itemIndex, (item) => adjustItemDuration(item, -15))} />
            <SmallAction label="Longer" onClick={() => onItemUpdate(dayIndex, entry.itemIndex, (item) => adjustItemDuration(item, 15))} />
            <SmallAction label="Tomorrow" onClick={() => onItemMove(dayIndex, entry.itemIndex, dayIndex + 1, entry.startMinutes)} />
            <SmallAction label="Yesterday" onClick={() => onItemMove(dayIndex, entry.itemIndex, dayIndex - 1, entry.startMinutes)} />
          </div>
        ) : null}
      </div>
    </article>
  );
}

function CalendarView({
  accommodation,
  calendarMode,
  conflicts,
  currency,
  days,
  setCalendarMode,
  setSelectedDayIndex,
  startDate,
  weatherForecast
}: {
  accommodation?: TripAccommodation | null;
  calendarMode: "trip" | "week" | "month";
  conflicts: ScheduleConflict[];
  currency: string;
  days: Itinerary["days"];
  setCalendarMode: (mode: "trip" | "week" | "month") => void;
  setSelectedDayIndex: (dayIndex: number) => void;
  startDate?: string | null;
  weatherForecast?: WeatherForecast | null;
}) {
  const monthCells = buildMonthCells(startDate, days.length);
  const tripDates = new Map(
    days.map((day, dayIndex) => [day.date || getTripDayDate(startDate, day.day || dayIndex + 1), dayIndex])
  );
  const visibleCells =
    calendarMode === "month"
      ? monthCells
      : calendarMode === "week"
        ? monthCells.slice(0, 7)
        : days.map((day, dayIndex) => ({
            date: day.date || getTripDayDate(startDate, day.day || dayIndex + 1),
            inTrip: true,
            dayIndex
          }));

  return (
    <div className="rounded-[16px] border border-sand-300 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="text-[16px] font-semibold text-cocoa-900">Calendar overview</h3>
        <div className="inline-flex rounded-md border border-sand-300 bg-sand-50 p-1">
          {(["trip", "week", "month"] as const).map((mode) => (
            <button
              key={mode}
              type="button"
              onClick={() => setCalendarMode(mode)}
              className={cn(
                "h-9 rounded px-3 text-[12px] font-semibold capitalize",
                calendarMode === mode ? "bg-white text-cocoa-900" : "text-cocoa-500"
              )}
            >
              {mode}
            </button>
          ))}
        </div>
      </div>
      <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {visibleCells.map((cell, index) => {
          const dayIndex = cell.dayIndex ?? (cell.date ? tripDates.get(cell.date) : undefined);
          const day = dayIndex != null ? days[dayIndex] : undefined;
          const dayNumber = day && dayIndex != null ? day.day || dayIndex + 1 : undefined;
          const date = cell.date;
          const weather = weatherForecast?.days.find((item) => item.date === date);
          const summary = day && dayIndex != null ? dailyScheduleSummary(day, dayIndex, conflicts) : null;
          const anchor = accommodationAnchor(accommodation, date);

          return (
            <button
              key={`${cell.date ?? "blank"}-${index}`}
              type="button"
              disabled={dayIndex == null}
              onClick={() => dayIndex != null && setSelectedDayIndex(dayIndex)}
              className={cn(
                "min-h-36 rounded-[14px] border p-4 text-left transition",
                day
                  ? summary && summary.activityCount > 4
                    ? "border-[#E5C3B6] bg-[#FBF0EB]"
                    : "border-sand-300 bg-sand-50 hover:border-sand-400"
                  : "border-sand-200 bg-white text-cocoa-400"
              )}
            >
              <div className="flex items-start justify-between gap-3">
                <span className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
                  {dayNumber ? `Day ${dayNumber}` : "Free"}
                </span>
                <span className="text-[12px] font-medium text-cocoa-400">{date ?? ""}</span>
              </div>
              {day ? (
                <>
                  <p className="mt-2 line-clamp-2 text-[14px] font-semibold text-cocoa-900">{day.title}</p>
                  <div className="mt-3 flex flex-wrap gap-2 text-[12px] font-semibold">
                    {summary ? <span className="rounded-full bg-white px-2.5 py-1">{summary.activityCount} items</span> : null}
                    {summary ? <span className="rounded-full bg-white px-2.5 py-1">{formatMoney(summary.estimatedCost, currency)}</span> : null}
                    {anchor ? <span className="rounded-full bg-[#EDF3EA] px-2.5 py-1 text-[#38543F]">{anchor}</span> : null}
                    {weather ? <span className="rounded-full bg-white px-2.5 py-1">{weather.condition}</span> : null}
                  </div>
                </>
              ) : (
                <p className="mt-2 text-[13px] text-cocoa-400">No trip activity</p>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function ItemScheduleActions({
  dayIndex,
  item,
  itemIndex,
  onItemUpdate
}: {
  dayIndex: number;
  item: ItineraryItem;
  itemIndex: number;
  onItemUpdate: (dayIndex: number, itemIndex: number, updater: (item: ItineraryItem) => ItineraryItem) => void;
}) {
  const travel = isTravelLike(item);
  if (travel) {
    return <span className="text-[12px] font-semibold text-cocoa-400">Travel block</span>;
  }
  return (
    <div className="flex flex-wrap justify-start gap-1 sm:justify-end">
      <SmallAction label="Earlier" onClick={() => onItemUpdate(dayIndex, itemIndex, (current) => adjustItemStart(current, -15))} />
      <SmallAction label="Later" onClick={() => onItemUpdate(dayIndex, itemIndex, (current) => adjustItemStart(current, 15))} />
      <SmallAction label="Longer" onClick={() => onItemUpdate(dayIndex, itemIndex, (current) => adjustItemDuration(current, 15))} />
      <SmallAction label="Unscheduled" onClick={() => onItemUpdate(dayIndex, itemIndex, unscheduleItem)} />
    </div>
  );
}

function SmallAction({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="h-7 rounded border border-sand-300 bg-white px-2 text-[11px] font-semibold text-cocoa-500 transition hover:border-sand-400 hover:text-cocoa-900"
    >
      {label}
    </button>
  );
}

function TravelDetail({ item, currency }: { item: ItineraryItem; currency: string }) {
  if (!item.transfer) {
    return (
      <p className="mt-1 text-[12px] text-cocoa-400">
        {formatDurationLabel(getItemDurationMinutes(item))}
      </p>
    );
  }
  return (
    <p className="mt-1 text-[12px] text-cocoa-400">
      {item.transfer.from} to {item.transfer.to} · {formatDurationLabel(item.transfer.estimatedDurationMinutes ?? getItemDurationMinutes(item))}
      {item.transfer.estimatedDistanceKm ? ` · ${Math.round(item.transfer.estimatedDistanceKm)} km` : ""}
      {item.transfer.estimatedCost?.amount != null
        ? ` · ${formatMoney(item.transfer.estimatedCost.amount, item.transfer.estimatedCost.currency || currency)}`
        : ""}
    </p>
  );
}

function toneForItem(item: ItineraryItem) {
  if (isAccommodationLike(item)) {
    return "bg-[#EDF3EA] text-[#38543F]";
  }
  if (isTravelLike(item)) {
    return "bg-sand-100 text-cocoa-500";
  }
  const status = normalizeSchedulingStatus(item.schedulingStatus, item);
  if (status === "NeedsReview") {
    return "bg-[#FDF0E3] text-[#96682A]";
  }
  if (status === "Conflict") {
    return "bg-[#FBF0EB] text-[#B3402E]";
  }
  return "bg-[#F7E4DB] text-clay-deep";
}

function readDragPayload(event: DragEvent<HTMLElement>): DragPayload | null {
  try {
    const raw = event.dataTransfer.getData("application/x-itinerary-item");
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as DragPayload;
    if (
      Number.isInteger(parsed.dayIndex) &&
      Number.isInteger(parsed.itemIndex) &&
      parsed.dayIndex >= 0 &&
      parsed.itemIndex >= 0
    ) {
      return parsed;
    }
  } catch {
    return null;
  }
  return null;
}

function cloneItinerary(itinerary: Itinerary): Itinerary {
  return JSON.parse(JSON.stringify(itinerary)) as Itinerary;
}

function accommodationAnchor(accommodation: TripAccommodation | null | undefined, date: string | null | undefined) {
  if (!accommodation || !date) {
    return null;
  }
  if (accommodation.checkInDate === date) {
    return `Check-in · ${accommodation.name}`;
  }
  if (accommodation.checkOutDate === date) {
    return `Check-out · ${accommodation.name}`;
  }
  if (
    accommodation.checkInDate &&
    accommodation.checkOutDate &&
    accommodation.checkInDate < date &&
    accommodation.checkOutDate > date
  ) {
    return accommodation.name;
  }
  return null;
}

function buildMonthCells(startDate: string | null | undefined, tripDayCount: number) {
  const start = startDate ? new Date(`${startDate}T00:00:00`) : null;
  if (!start || Number.isNaN(start.getTime())) {
    return Array.from({ length: tripDayCount }, (_, index) => ({
      date: null,
      inTrip: true,
      dayIndex: index
    }));
  }
  const monthStart = new Date(start);
  monthStart.setDate(1);
  const leading = monthStart.getDay();
  const firstCell = new Date(monthStart);
  firstCell.setDate(monthStart.getDate() - leading);
  const tripDates = new Map<string, number>();
  for (let index = 0; index < tripDayCount; index++) {
    const day = new Date(start);
    day.setDate(start.getDate() + index);
    tripDates.set(day.toISOString().slice(0, 10), index);
  }
  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(firstCell);
    date.setDate(firstCell.getDate() + index);
    const value = date.toISOString().slice(0, 10);
    const dayIndex = tripDates.get(value);
    return {
      date: value,
      inTrip: dayIndex != null,
      dayIndex
    };
  });
}
