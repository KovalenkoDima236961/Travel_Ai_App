import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import type { Itinerary, ItineraryItem } from "@/types/trip";

type ItineraryEditorProps = {
  itinerary: Itinerary;
  errors?: string[];
  disabled?: boolean;
  onChange: (itinerary: Itinerary) => void;
};

const defaultItem: ItineraryItem = {
  time: "09:00",
  type: "activity",
  name: "",
  note: "",
  estimatedCost: null
};

export function ItineraryEditor({
  itinerary,
  errors = [],
  disabled = false,
  onChange
}: ItineraryEditorProps) {
  const days = itinerary.days ?? [];

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
    onChange({
      ...itinerary,
      days: days.map((day, index) =>
        index === dayIndex ? { ...day, items: [...day.items, { ...defaultItem }] } : day
      )
    });
  }

  function removeItem(dayIndex: number, itemIndex: number) {
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
    const nextDays = days
      .filter((_, index) => index !== dayIndex)
      .map((day, index) => ({ ...day, day: index + 1 }));
    onChange({ ...itinerary, days: nextDays });
  }

  return (
    <div className="space-y-5">
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

                  <div className="grid gap-2">
                    <label
                      className="text-sm font-medium text-slate-700"
                      htmlFor={`item-cost-${dayIndex}-${itemIndex}`}
                    >
                      Cost
                    </label>
                    <Input
                      disabled={disabled}
                      id={`item-cost-${dayIndex}-${itemIndex}`}
                      min={0}
                      onChange={(event) =>
                        updateItem(dayIndex, itemIndex, {
                          estimatedCost:
                            event.target.value === "" ? null : Number(event.target.value)
                        })
                      }
                      placeholder="0"
                      step="0.01"
                      type="number"
                      value={item.estimatedCost ?? ""}
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

export function prepareItineraryForEdit(itinerary: Itinerary): Itinerary {
  return {
    ...itinerary,
    days: (itinerary.days ?? []).map((day, dayIndex) => ({
      ...day,
      day: day.day || dayIndex + 1,
      title: day.title ?? "",
      items: (day.items ?? []).map((item) => ({
        time: item.time ?? "",
        type: item.type ?? "",
        name: item.name ?? "",
        note: item.note ?? "",
        estimatedCost: item.estimatedCost ?? null
      }))
    }))
  };
}

export function normalizeItineraryForSave(itinerary: Itinerary): Itinerary {
  return {
    ...itinerary,
    days: itinerary.days.map((day, dayIndex) => ({
      ...day,
      day: dayIndex + 1,
      title: day.title.trim(),
      items: day.items.map((item) => ({
        time: item.time.trim(),
        type: item.type.trim(),
        name: item.name.trim(),
        note: typeof item.note === "string" ? item.note.trim() : item.note ?? "",
        estimatedCost: item.estimatedCost ?? null
      }))
    }))
  };
}

export function validateEditableItinerary(itinerary: Itinerary): string[] {
  const errors: string[] = [];

  if (!itinerary.days || itinerary.days.length === 0) {
    return ["Add at least one day."];
  }

  itinerary.days.forEach((day, dayIndex) => {
    const dayLabel = `Day ${dayIndex + 1}`;
    if (!day.title.trim()) {
      errors.push(`${dayLabel} needs a title.`);
    }
    if (!day.items || day.items.length === 0) {
      errors.push(`${dayLabel} needs at least one item.`);
      return;
    }

    day.items.forEach((item, itemIndex) => {
      const itemLabel = `${dayLabel}, item ${itemIndex + 1}`;
      if (!item.time.trim()) {
        errors.push(`${itemLabel} needs a time.`);
      }
      if (!item.type.trim()) {
        errors.push(`${itemLabel} needs a type.`);
      }
      if (!item.name.trim()) {
        errors.push(`${itemLabel} needs a name.`);
      }
      if (item.estimatedCost != null && item.estimatedCost < 0) {
        errors.push(`${itemLabel} cost must be 0 or more.`);
      }
      if (item.estimatedCost != null && Number.isNaN(item.estimatedCost)) {
        errors.push(`${itemLabel} cost must be a valid number.`);
      }
    });
  });

  return errors;
}
