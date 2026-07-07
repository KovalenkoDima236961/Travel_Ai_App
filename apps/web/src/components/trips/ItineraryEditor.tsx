"use client";

import { useState } from "react";
import { AttachPlaceDialog } from "@/features/place-attachment";
import { ItemCostEditor } from "@/features/trip-budget";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import {
  canMoveItemDown,
  canMoveItemToDay,
  canMoveItemUp,
  moveItemToDay,
  moveItemWithinDay
} from "@/entities/itinerary/model/editor-utils";
import {
  formatOpeningHoursForDay,
  getDayOfWeekMondayBased,
  getTripItemDate
} from "@/entities/itinerary/model/opening-hours-utils";
import type { OpeningHoursInterval } from "@/entities/place/model";
import type { Place } from "@/entities/place/model";
import type { Itinerary, ItineraryItem } from "@/entities/trip/model";

export {
  normalizeItineraryForSave,
  prepareItineraryForEdit,
  validateEditableItinerary
} from "@/entities/itinerary/model/editor-utils";

type ItineraryEditorProps = {
  itinerary: Itinerary;
  destination?: string;
  startDate?: string | null;
  errors?: string[];
  disabled?: boolean;
  onChange: (itinerary: Itinerary) => void;
};

const defaultItem: ItineraryItem = {
  time: "09:00",
  type: "activity",
  name: "",
  note: "",
  estimatedCost: null,
  place: null,
  placeEnrichment: null
};

export function ItineraryEditor({
  itinerary,
  destination,
  startDate,
  errors = [],
  disabled = false,
  onChange
}: ItineraryEditorProps) {
  const days = itinerary.days ?? [];
  const tripCurrency = (itinerary.currency ?? "EUR").toUpperCase();
  const [moveTargets, setMoveTargets] = useState<Record<string, number>>({});
  const [reorderMessage, setReorderMessage] = useState<string | null>(null);
  const [attachTarget, setAttachTarget] = useState<{ dayIndex: number; itemIndex: number } | null>(
    null
  );
  const attachTargetItem =
    attachTarget == null ? null : days[attachTarget.dayIndex]?.items[attachTarget.itemIndex] ?? null;

  function moveTargetKey(dayIndex: number, itemIndex: number) {
    return `${dayIndex}-${itemIndex}`;
  }

  function getMoveTargetDayIndex(dayIndex: number, itemIndex: number) {
    return moveTargets[moveTargetKey(dayIndex, itemIndex)] ?? dayIndex;
  }

  function resetMoveTargets() {
    setMoveTargets({});
  }

  function updateDay(dayIndex: number, updates: Partial<Itinerary["days"][number]>) {
    onChange({
      ...itinerary,
      days: days.map((day, index) => (index === dayIndex ? { ...day, ...updates } : day))
    });
  }

  function updateItem(dayIndex: number, itemIndex: number, updates: Partial<ItineraryItem>) {
    onChange({
      ...itinerary,
      days: days.map((day, index) => {
        if (index !== dayIndex) {
          return day;
        }
        return {
          ...day,
          items: day.items.map((item, innerIndex) =>
            innerIndex === itemIndex ? { ...item, ...updates } : item
          )
        };
      })
    });
  }

  function addItem(dayIndex: number) {
    resetMoveTargets();
    setReorderMessage(null);
    onChange({
      ...itinerary,
      days: days.map((day, index) =>
        index === dayIndex ? { ...day, items: [...day.items, { ...defaultItem }] } : day
      )
    });
  }

  function removeItem(dayIndex: number, itemIndex: number) {
    resetMoveTargets();
    setReorderMessage(null);
    onChange({
      ...itinerary,
      days: days.map((day, index) =>
        index === dayIndex
          ? { ...day, items: day.items.filter((_, innerIndex) => innerIndex !== itemIndex) }
          : day
      )
    });
  }

  function addDay() {
    resetMoveTargets();
    setReorderMessage(null);
    onChange({
      ...itinerary,
      days: [
        ...days,
        {
          day: days.length + 1,
          title: `Day ${days.length + 1}`,
          items: [{ ...defaultItem }]
        }
      ]
    });
  }

  function removeDay(dayIndex: number) {
    resetMoveTargets();
    setReorderMessage(null);
    const nextDays = days
      .filter((_, index) => index !== dayIndex)
      .map((day, index) => ({ ...day, day: index + 1 }));
    onChange({ ...itinerary, days: nextDays });
  }

  function moveItem(dayIndex: number, itemIndex: number, direction: "up" | "down") {
    resetMoveTargets();
    setReorderMessage(null);
    onChange(moveItemWithinDay(itinerary, dayIndex, itemIndex, direction));
  }

  function updateMoveTarget(dayIndex: number, itemIndex: number, targetDayIndex: number) {
    setReorderMessage(null);
    setMoveTargets((currentTargets) => ({
      ...currentTargets,
      [moveTargetKey(dayIndex, itemIndex)]: targetDayIndex
    }));
  }

  function moveItemAcrossDays(dayIndex: number, itemIndex: number) {
    const targetDayIndex = getMoveTargetDayIndex(dayIndex, itemIndex);

    if (targetDayIndex === dayIndex) {
      return;
    }

    if (!canMoveItemToDay(itinerary, dayIndex, itemIndex, targetDayIndex)) {
      setReorderMessage("A day must contain at least one item.");
      return;
    }

    resetMoveTargets();
    setReorderMessage(null);
    onChange(moveItemToDay(itinerary, dayIndex, itemIndex, targetDayIndex));
  }

  function openAttachDialog(dayIndex: number, itemIndex: number) {
    setAttachTarget({ dayIndex, itemIndex });
  }

  function closeAttachDialog() {
    setAttachTarget(null);
  }

  function attachPlace(place: Place) {
    if (attachTarget == null) {
      return;
    }
    updateItem(attachTarget.dayIndex, attachTarget.itemIndex, { place, placeEnrichment: null });
  }

  function updateItemCost(
    dayIndex: number,
    itemIndex: number,
    estimatedCost: ItineraryItem["estimatedCost"]
  ) {
    const current = days[dayIndex]?.items[itemIndex]?.priceEnrichment;
    updateItem(dayIndex, itemIndex, {
      estimatedCost,
      priceEnrichment: current
        ? {
            ...current,
            reviewStatus: estimatedCost ? "changed" : "removed"
          }
        : current
    });
  }

  return (
    <div className="space-y-5">
      <AttachPlaceDialog
        destination={destination ?? itinerary.destination}
        initialQuery={attachTargetItem?.name ?? ""}
        onClose={closeAttachDialog}
        onSelect={attachPlace}
        open={attachTarget != null}
      />

      {errors.length > 0 ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          <p className="font-semibold">Fix these before saving:</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            {errors.map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        </div>
      ) : null}

      {reorderMessage ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          {reorderMessage}
        </div>
      ) : null}

      <div className="flex flex-col gap-3 rounded-lg border border-slate-200 bg-white p-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-950">Edit itinerary</h2>
        </div>
        <Button disabled={disabled} onClick={addDay} type="button" variant="secondary">
          Add day
        </Button>
      </div>

      <datalist id="itinerary-item-types">
        <option value="activity" />
        <option value="place" />
        <option value="food" />
        <option value="transport" />
        <option value="rest" />
      </datalist>

      {days.map((day, dayIndex) => (
        <section key={`${day.day}-${dayIndex}`} className="rounded-lg border border-slate-200 bg-white p-5">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div className="grid flex-1 gap-2">
              <label className="text-sm font-medium text-slate-700" htmlFor={`day-title-${dayIndex}`}>
                Day {dayIndex + 1} title
              </label>
              <Input
                disabled={disabled}
                id={`day-title-${dayIndex}`}
                onChange={(event) => updateDay(dayIndex, { title: event.target.value })}
                value={day.title}
              />
            </div>
            {days.length > 1 ? (
              <Button
                disabled={disabled}
                onClick={() => removeDay(dayIndex)}
                type="button"
                variant="danger"
              >
                Remove day
              </Button>
            ) : null}
          </div>

          <div className="mt-5 space-y-4">
            {day.items.map((item, itemIndex) => (
              <div
                key={`${dayIndex}-${itemIndex}`}
                className="border-t border-slate-200 pt-4 first:border-t-0 first:pt-0"
              >
                <div className="grid gap-4 lg:grid-cols-[7rem_9rem_minmax(0,1fr)_8rem_auto]">
                  <div className="grid gap-2">
                    <label
                      className="text-sm font-medium text-slate-700"
                      htmlFor={`item-time-${dayIndex}-${itemIndex}`}
                    >
                      Time
                    </label>
                    <Input
                      disabled={disabled}
                      id={`item-time-${dayIndex}-${itemIndex}`}
                      onChange={(event) =>
                        updateItem(dayIndex, itemIndex, { time: event.target.value })
                      }
                      placeholder="09:00"
                      value={item.time}
                    />
                  </div>

                  <div className="grid gap-2">
                    <label
                      className="text-sm font-medium text-slate-700"
                      htmlFor={`item-type-${dayIndex}-${itemIndex}`}
                    >
                      Type
                    </label>
                    <Input
                      disabled={disabled}
                      id={`item-type-${dayIndex}-${itemIndex}`}
                      list="itinerary-item-types"
                      onChange={(event) =>
                        updateItem(dayIndex, itemIndex, { type: event.target.value })
                      }
                      value={item.type}
                    />
                  </div>

                  <div className="grid gap-2">
                    <label
                      className="text-sm font-medium text-slate-700"
                      htmlFor={`item-name-${dayIndex}-${itemIndex}`}
                    >
                      Name
                    </label>
                    <Input
                      disabled={disabled}
                      id={`item-name-${dayIndex}-${itemIndex}`}
                      onChange={(event) =>
                        updateItem(dayIndex, itemIndex, { name: event.target.value })
                      }
                      value={item.name}
                    />
                  </div>

                  <div className="flex items-end">
                    <Button
                      disabled={disabled}
                      onClick={() => removeItem(dayIndex, itemIndex)}
                      type="button"
                      variant="danger"
                    >
                      Remove
                    </Button>
                  </div>
                </div>

                <div className="mt-4 rounded-md border border-slate-200 bg-white p-3">
                  <p className="mb-3 text-sm font-medium text-slate-700">Estimated cost</p>
                  <ItemCostEditor
                    cost={item.estimatedCost}
                    disabled={disabled}
                    idPrefix={`item-cost-${dayIndex}-${itemIndex}`}
                    onChange={(estimatedCost) => updateItemCost(dayIndex, itemIndex, estimatedCost)}
                    tripCurrency={tripCurrency}
                  />
                </div>

                <div className="mt-4 rounded-md border border-slate-200 bg-slate-50 p-3">
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                      <p className="text-xs font-semibold uppercase text-slate-500">
                        Attached place
                      </p>
                      {item.place ? (
                        <div className="mt-2 space-y-1 text-sm">
                          <p className="font-semibold text-slate-950">{item.place.name}</p>
                          <p className="leading-5 text-slate-600">{item.place.address}</p>
                          <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs font-medium text-slate-500">
                            <span>
                              Provider: {formatPlaceCategory(item.place.provider || "unknown")}
                            </span>
                            {item.place.rating != null ? (
                              <span>
                                Rating {item.place.rating}
                                {item.place.ratingCount != null
                                  ? ` (${item.place.ratingCount.toLocaleString()})`
                                  : ""}
                              </span>
                            ) : null}
                            {item.place.category ? (
                              <span>{formatPlaceCategory(item.place.category)}</span>
                            ) : null}
                            {item.place.mapUrl ? (
                              <a
                                className="text-primary-700 hover:text-primary-600"
                                href={item.place.mapUrl}
                                rel="noreferrer"
                                target="_blank"
                              >
                                Open map
                              </a>
                            ) : null}
                          </div>
                          <p className="text-xs text-slate-500">
                            Place changes are saved when you save the itinerary.
                          </p>
                          <OpeningHoursEditorSummary
                            dayNumber={day.day || dayIndex + 1}
                            openingHours={item.place.openingHours}
                            startDate={startDate}
                          />
                        </div>
                      ) : (
                        <p className="mt-2 text-sm text-slate-600">
                          No real place attached.
                        </p>
                      )}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        disabled={disabled}
                        onClick={() => openAttachDialog(dayIndex, itemIndex)}
                        size="sm"
                        type="button"
                        variant="secondary"
                      >
                        {item.place ? "Change place" : "Attach real place"}
                      </Button>
                      {item.place ? (
                        <Button
                          disabled={disabled}
                          onClick={() =>
                            updateItem(dayIndex, itemIndex, { place: null, placeEnrichment: null })
                          }
                          size="sm"
                          type="button"
                          variant="ghost"
                        >
                          Remove place
                        </Button>
                      ) : null}
                    </div>
                  </div>
                </div>

                <div className="mt-4 rounded-md border border-slate-200 bg-slate-50 p-3">
                  <p className="text-xs font-semibold uppercase text-slate-500">Reorder</p>
                  <div className="mt-3 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                    <div className="flex flex-wrap gap-2">
                      <Button
                        aria-label="Move item up"
                        disabled={disabled || !canMoveItemUp(day, itemIndex)}
                        onClick={() => moveItem(dayIndex, itemIndex, "up")}
                        size="sm"
                        title="Move item up"
                        type="button"
                        variant="secondary"
                      >
                        Up
                      </Button>
                      <Button
                        aria-label="Move item down"
                        disabled={disabled || !canMoveItemDown(day, itemIndex)}
                        onClick={() => moveItem(dayIndex, itemIndex, "down")}
                        size="sm"
                        title="Move item down"
                        type="button"
                        variant="secondary"
                      >
                        Down
                      </Button>
                    </div>

                    {days.length > 1 ? (
                      <div className="grid gap-2 sm:grid-cols-[minmax(8rem,1fr)_auto] sm:items-end">
                        <div className="grid gap-2">
                          <label
                            className="text-sm font-medium text-slate-700"
                            htmlFor={`move-item-day-${dayIndex}-${itemIndex}`}
                          >
                            Move to day
                          </label>
                          <Select
                            className="h-9"
                            disabled={disabled}
                            id={`move-item-day-${dayIndex}-${itemIndex}`}
                            onChange={(event) =>
                              updateMoveTarget(dayIndex, itemIndex, Number(event.target.value))
                            }
                            value={getMoveTargetDayIndex(dayIndex, itemIndex)}
                          >
                            {days.map((targetDay, targetDayIndex) => (
                              <option key={`${targetDay.day}-${targetDayIndex}`} value={targetDayIndex}>
                                Day {targetDayIndex + 1}
                              </option>
                            ))}
                          </Select>
                        </div>
                        <Button
                          disabled={
                            disabled ||
                            getMoveTargetDayIndex(dayIndex, itemIndex) === dayIndex
                          }
                          onClick={() => moveItemAcrossDays(dayIndex, itemIndex)}
                          size="sm"
                          type="button"
                          variant="secondary"
                        >
                          Move
                        </Button>
                      </div>
                    ) : null}
                  </div>
                </div>

                <div className="mt-4 grid gap-2">
                  <label
                    className="text-sm font-medium text-slate-700"
                    htmlFor={`item-note-${dayIndex}-${itemIndex}`}
                  >
                    Note
                  </label>
                  <Textarea
                    className="min-h-20"
                    disabled={disabled}
                    id={`item-note-${dayIndex}-${itemIndex}`}
                    onChange={(event) =>
                      updateItem(dayIndex, itemIndex, { note: event.target.value })
                    }
                    value={item.note ?? ""}
                  />
                </div>
              </div>
            ))}
          </div>

          <Button
            className="mt-5"
            disabled={disabled}
            onClick={() => addItem(dayIndex)}
            type="button"
            variant="secondary"
          >
            Add item
          </Button>
        </section>
      ))}
    </div>
  );
}

function formatPlaceCategory(value: string) {
  return value
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function OpeningHoursEditorSummary({
  openingHours,
  startDate,
  dayNumber
}: {
  openingHours?: OpeningHoursInterval[] | null;
  startDate?: string | null;
  dayNumber: number;
}) {
  if (!openingHours || openingHours.length === 0 || !startDate) {
    return <p className="text-xs text-slate-500">Opening hours unknown</p>;
  }

  const itemDate = getTripItemDate(startDate, dayNumber);
  if (!itemDate) {
    return <p className="text-xs text-slate-500">Opening hours unknown</p>;
  }

  const dayOfWeek = getDayOfWeekMondayBased(itemDate);
  return (
    <p className="text-xs text-slate-500">
      Opening hours on this day: {formatOpeningHoursForDay(openingHours, dayOfWeek)}
    </p>
  );
}
