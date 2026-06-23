import type { Itinerary, ItineraryDay, ItineraryItem } from "@/types/trip";

export type ItemMoveDirection = "up" | "down";

export function canMoveItemUp(day: ItineraryDay, itemIndex: number) {
  return itemIndex > 0 && itemIndex < day.items.length;
}

export function canMoveItemDown(day: ItineraryDay, itemIndex: number) {
  return itemIndex >= 0 && itemIndex < day.items.length - 1;
}

export function canMoveItemToDay(
  itinerary: Itinerary,
  fromDayIndex: number,
  itemIndex: number,
  toDayIndex: number
) {
  const days = itinerary.days ?? [];
  const fromDay = days[fromDayIndex];
  const toDay = days[toDayIndex];

  if (!fromDay || !toDay || fromDayIndex === toDayIndex) {
    return false;
  }

  return itemIndex >= 0 && itemIndex < fromDay.items.length && fromDay.items.length > 1;
}

export function moveItemWithinDay(
  itinerary: Itinerary,
  dayIndex: number,
  itemIndex: number,
  direction: ItemMoveDirection
): Itinerary {
  const days = itinerary.days ?? [];
  const day = days[dayIndex];
  const targetIndex = direction === "up" ? itemIndex - 1 : itemIndex + 1;

  if (!day || targetIndex < 0 || targetIndex >= day.items.length) {
    return itinerary;
  }

  const nextItems = [...day.items];
  const movedItem = nextItems[itemIndex];

  if (!movedItem) {
    return itinerary;
  }

  nextItems[itemIndex] = nextItems[targetIndex];
  nextItems[targetIndex] = movedItem;

  return {
    ...itinerary,
    days: days.map((currentDay, index) =>
      index === dayIndex ? { ...currentDay, items: nextItems } : currentDay
    )
  };
}

export function moveItemToDay(
  itinerary: Itinerary,
  fromDayIndex: number,
  itemIndex: number,
  toDayIndex: number
): Itinerary {
  if (!canMoveItemToDay(itinerary, fromDayIndex, itemIndex, toDayIndex)) {
    return itinerary;
  }

  const days = itinerary.days ?? [];
  const item = days[fromDayIndex].items[itemIndex];
  const nextDays = days.map((day, index) => {
    if (index === fromDayIndex) {
      return {
        ...day,
        items: day.items.filter((_, currentItemIndex) => currentItemIndex !== itemIndex)
      };
    }

    if (index === toDayIndex) {
      return {
        ...day,
        items: [...day.items, item]
      };
    }

    return day;
  });

  return normalizeItineraryDays({ ...itinerary, days: nextDays });
}

export function normalizeItineraryDays(itinerary: Itinerary): Itinerary {
  return {
    ...itinerary,
    days: (itinerary.days ?? []).map((day, dayIndex) => ({
      ...day,
      day: dayIndex + 1
    }))
  };
}

export function prepareItineraryForEdit(itinerary: Itinerary): Itinerary {
  return {
    ...itinerary,
    days: (itinerary.days ?? []).map((day, dayIndex) => ({
      ...day,
      day: day.day || dayIndex + 1,
      title: day.title ?? "",
      items: (day.items ?? []).map((item) => ({
        ...item,
        time: item.time ?? "",
        type: item.type ?? "",
        name: item.name ?? "",
        note: item.note ?? "",
        estimatedCost: item.estimatedCost ?? null,
        place: item.place ?? null,
        placeEnrichment: item.placeEnrichment ?? null
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
      items: day.items.map((item) => normalizeItemForSave(item))
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
      if (item.place) {
        if (!item.place.provider.trim()) {
          errors.push(`${itemLabel} attached place needs a provider.`);
        }
        if (!item.place.providerPlaceId.trim()) {
          errors.push(`${itemLabel} attached place needs a provider place ID.`);
        }
        if (!item.place.name.trim()) {
          errors.push(`${itemLabel} attached place needs a name.`);
        }
        if (!item.place.address.trim()) {
          errors.push(`${itemLabel} attached place needs an address.`);
        }
      }
    });
  });

  return errors;
}

function normalizeItemForSave(item: ItineraryItem): ItineraryItem {
  return {
    ...item,
    time: item.time.trim(),
    type: item.type.trim(),
    name: item.name.trim(),
    note: typeof item.note === "string" ? item.note.trim() : item.note ?? "",
    estimatedCost: item.estimatedCost ?? null,
    placeEnrichment: item.placeEnrichment ?? null,
    place: item.place
      ? {
          ...item.place,
          provider: item.place.provider.trim(),
          providerPlaceId: item.place.providerPlaceId.trim(),
          name: item.place.name.trim(),
          address: item.place.address.trim(),
          mapUrl: item.place.mapUrl?.trim() || null,
          category: item.place.category?.trim() || null,
          website: item.place.website?.trim() || null
        }
      : null
  };
}
