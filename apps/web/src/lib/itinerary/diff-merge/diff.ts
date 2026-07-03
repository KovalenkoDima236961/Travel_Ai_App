import { describeItineraryChange } from "@/lib/itinerary/diff-merge/describe";
import {
  generatedItemKey,
  getItemId,
  itemIdentitySignature,
  normalizeDayMetadataForDiff,
  normalizeItemForDiff,
  valuesEqual
} from "@/lib/itinerary/diff-merge/normalize";
import type {
  ChangeOrigin,
  ItineraryChange,
  ItineraryChangeType
} from "@/lib/itinerary/diff-merge/types";
import type { Itinerary, ItineraryDay, ItineraryItem } from "@/types/trip";

type ItemState = {
  key: string;
  index: number;
  item: ItineraryItem;
  normalized: unknown;
};

export function diffItineraries(
  baseItinerary: Itinerary,
  modifiedItinerary: Itinerary,
  origin: ChangeOrigin
): ItineraryChange[] {
  const changes: ItineraryChange[] = [];
  const baseDays = mapDaysByNumber(baseItinerary);
  const modifiedDays = mapDaysByNumber(modifiedItinerary);
  const allDayNumbers = Array.from(
    new Set([...baseDays.keys(), ...modifiedDays.keys()])
  ).sort((left, right) => left - right);

  allDayNumbers.forEach((dayNumber) => {
    const baseDay = baseDays.get(dayNumber);
    const modifiedDay = modifiedDays.get(dayNumber);

    if (!baseDay && modifiedDay) {
      changes.push(makeChange(origin, "day_added", dayNumber, null, null, null, modifiedDay));
      return;
    }

    if (baseDay && !modifiedDay) {
      changes.push(makeChange(origin, "day_removed", dayNumber, null, null, baseDay, null));
      return;
    }

    if (!baseDay || !modifiedDay) {
      return;
    }

    if (
      !valuesEqual(
        normalizeDayMetadataForDiff(baseDay.day, baseDay.index),
        normalizeDayMetadataForDiff(modifiedDay.day, modifiedDay.index)
      )
    ) {
      changes.push(
        makeChange(origin, "day_replaced", dayNumber, null, null, baseDay.day, modifiedDay.day)
      );
      return;
    }

    const itemChanges = diffDayItems(origin, dayNumber, baseDay.day, modifiedDay.day);
    changes.push(...itemChanges);
  });

  return changes;
}

function diffDayItems(
  origin: ChangeOrigin,
  dayNumber: number,
  baseDay: ItineraryDay,
  modifiedDay: ItineraryDay
): ItineraryChange[] {
  const changes: ItineraryChange[] = [];
  const { baseStates, modifiedStates } = buildComparableItemStates(
    dayNumber,
    baseDay.items ?? [],
    modifiedDay.items ?? []
  );
  const baseByKey = new Map(baseStates.map((state) => [state.key, state]));
  const modifiedByKey = new Map(modifiedStates.map((state) => [state.key, state]));
  const commonKeys = baseStates
    .map((state) => state.key)
    .filter((key) => modifiedByKey.has(key));

  if (baseStates.length > 0 && modifiedStates.length > 0 && commonKeys.length === 0) {
    return [
      makeChange(origin, "day_replaced", dayNumber, null, null, baseDay, modifiedDay)
    ];
  }

  baseStates.forEach((baseState) => {
    if (!modifiedByKey.has(baseState.key)) {
      changes.push(
        makeChange(
          origin,
          "item_removed",
          dayNumber,
          baseState.key,
          baseState.index,
          baseState.item,
          null
        )
      );
    }
  });

  modifiedStates.forEach((modifiedState) => {
    const baseState = baseByKey.get(modifiedState.key);
    if (!baseState) {
      changes.push(
        makeChange(
          origin,
          "item_added",
          dayNumber,
          modifiedState.key,
          modifiedState.index,
          null,
          modifiedState.item
        )
      );
      return;
    }

    if (!valuesEqual(baseState.normalized, modifiedState.normalized)) {
      changes.push(
        makeChange(
          origin,
          "item_modified",
          dayNumber,
          modifiedState.key,
          modifiedState.index,
          baseState.item,
          modifiedState.item
        )
      );
    }
  });

  const baseCommonOrder = baseStates
    .map((state) => state.key)
    .filter((key) => modifiedByKey.has(key));
  const modifiedCommonOrder = modifiedStates
    .map((state) => state.key)
    .filter((key) => baseByKey.has(key));

  if (
    baseCommonOrder.length > 1 &&
    modifiedCommonOrder.length > 1 &&
    !sameOrder(baseCommonOrder, modifiedCommonOrder)
  ) {
    changes.push(
      makeChange(
        origin,
        "item_reordered",
        dayNumber,
        null,
        null,
        baseDay.items ?? [],
        modifiedDay.items ?? []
      )
    );
  }

  return changes;
}

export function buildComparableItemStates(
  dayNumber: number,
  baseItems: ItineraryItem[],
  modifiedItems: ItineraryItem[]
) {
  const baseStates = baseItems.map((item, index) => {
    const id = getItemId(item);
    return {
      key: id ? `id:${id}` : generatedItemKey(dayNumber, index, item),
      index,
      item,
      normalized: normalizeItemForDiff(item)
    };
  });
  const baseBySignature = new Map<string, ItemState[]>();
  baseStates.forEach((state) => {
    const signature = itemIdentitySignature(state.item);
    baseBySignature.set(signature, [...(baseBySignature.get(signature) ?? []), state]);
  });

  const usedBaseKeys = new Set<string>();
  const modifiedStates = modifiedItems.map((item, index) => {
    const id = getItemId(item);
    let key = id ? `id:${id}` : null;

    if (!key) {
      const signature = itemIdentitySignature(item);
      const matchingBaseState = (baseBySignature.get(signature) ?? []).find(
        (state) => !usedBaseKeys.has(state.key)
      );
      key = matchingBaseState?.key ?? generatedItemKey(dayNumber, index, item);
    }

    usedBaseKeys.add(key);
    return {
      key,
      index,
      item,
      normalized: normalizeItemForDiff(item)
    };
  });

  return { baseStates, modifiedStates };
}

function mapDaysByNumber(itinerary: Itinerary) {
  const dayMap = new Map<number, { day: ItineraryDay; index: number }>();
  (itinerary.days ?? []).forEach((day, index) => {
    dayMap.set(day.day || index + 1, { day, index });
  });
  return dayMap;
}

function makeChange(
  origin: ChangeOrigin,
  type: ItineraryChangeType,
  dayNumber: number,
  itemKey: string | null,
  itemIndex: number | null,
  before: unknown,
  after: unknown
): ItineraryChange {
  const change = {
    id: `${origin}:${type}:${dayNumber}:${itemKey ?? itemIndex ?? "day"}`,
    origin,
    type,
    dayNumber,
    itemKey,
    itemIndex,
    before: before ?? undefined,
    after: after ?? undefined,
    conflictKey: conflictKeyForChange(type, dayNumber, itemKey, itemIndex)
  };

  return {
    ...change,
    summary: describeItineraryChange(change)
  };
}

function conflictKeyForChange(
  type: ItineraryChangeType,
  dayNumber: number,
  itemKey: string | null,
  itemIndex: number | null
) {
  if (type.startsWith("day_") || type === "item_reordered") {
    return `day:${dayNumber}`;
  }
  if (itemKey) {
    return `day:${dayNumber}:item:${itemKey}`;
  }
  return `day:${dayNumber}:index:${itemIndex ?? 0}`;
}

function sameOrder(left: string[], right: string[]) {
  return left.length === right.length && left.every((key, index) => right[index] === key);
}
