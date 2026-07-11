import { apiFetch } from "@/shared/api/client";
import type {
  ChecklistItemPayload,
  ChecklistViewResponse,
  GenerateChecklistRequest,
  TripChecklistItem,
  UpdateChecklistItemPayload
} from "@/entities/checklist/model";

export const checklistKeys = {
  all: ["trip-checklists"] as const,
  detail: (tripId: string) => [...checklistKeys.all, tripId] as const
};

export function getTripChecklist(tripId: string) {
  return apiFetch<ChecklistViewResponse>(`/trips/${tripId}/checklist`);
}

export function generateTripChecklist(tripId: string, input: GenerateChecklistRequest = {}) {
  return apiFetch<ChecklistViewResponse>(`/trips/${tripId}/checklist/generate`, {
    method: "POST",
    body: JSON.stringify(cleanGeneratePayload(input))
  });
}

export function createChecklistItem(tripId: string, input: ChecklistItemPayload) {
  return apiFetch<TripChecklistItem>(`/trips/${tripId}/checklist/items`, {
    method: "POST",
    body: JSON.stringify(cleanItemPayload(input))
  });
}

export function updateChecklistItem(
  tripId: string,
  itemId: string,
  input: UpdateChecklistItemPayload
) {
  return apiFetch<TripChecklistItem>(`/trips/${tripId}/checklist/items/${itemId}`, {
    method: "PATCH",
    body: JSON.stringify(cleanUpdatePayload(input))
  });
}

export function deleteChecklistItem(tripId: string, itemId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/checklist/items/${itemId}`, {
    method: "DELETE"
  });
}

export function checkChecklistItem(tripId: string, itemId: string) {
  return apiFetch<TripChecklistItem>(`/trips/${tripId}/checklist/items/${itemId}/check`, {
    method: "POST"
  });
}

export function uncheckChecklistItem(tripId: string, itemId: string) {
  return apiFetch<TripChecklistItem>(`/trips/${tripId}/checklist/items/${itemId}/uncheck`, {
    method: "POST"
  });
}

export function reorderChecklistItems(tripId: string, itemIds: string[]) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/checklist/reorder`, {
    method: "POST",
    body: JSON.stringify({ itemIds })
  });
}

function cleanGeneratePayload(input: GenerateChecklistRequest) {
  return {
    mode: input.mode ?? "full",
    categories: input.categories ?? [],
    instructions: input.instructions?.trim() || undefined,
    preserveCheckedItems: input.preserveCheckedItems ?? true,
    preserveManualItems: input.preserveManualItems ?? true,
    replaceAiItems: input.replaceAiItems ?? false,
    outputLanguage: input.outputLanguage ?? "en"
  };
}

function cleanItemPayload(input: ChecklistItemPayload) {
  return removeUndefined({
    title: input.title.trim(),
    description: emptyToUndefined(input.description),
    category: input.category,
    itemType: input.itemType ?? "packing",
    priority: input.priority ?? "medium",
    quantity: input.quantity ?? undefined,
    assignedToUserId: emptyToUndefined(input.assignedToUserId),
    dueDate: emptyToUndefined(input.dueDate),
    reason: emptyToUndefined(input.reason),
    relatedDayNumber: input.relatedDayNumber ?? undefined,
    relatedItemIndex: input.relatedItemIndex ?? undefined,
    relatedItemId: emptyToUndefined(input.relatedItemId),
    metadata: input.metadata
  });
}

function cleanUpdatePayload(input: UpdateChecklistItemPayload) {
  return removeUndefined({
    title: input.title?.trim(),
    description: emptyToUndefined(input.description),
    clearDescription: input.clearDescription || undefined,
    category: input.category,
    itemType: input.itemType,
    priority: input.priority,
    quantity: input.quantity ?? undefined,
    clearQuantity: input.clearQuantity || undefined,
    assignedToUserId: emptyToUndefined(input.assignedToUserId),
    clearAssignee: input.clearAssignee || undefined,
    dueDate: emptyToUndefined(input.dueDate),
    clearDueDate: input.clearDueDate || undefined,
    reason: emptyToUndefined(input.reason),
    clearReason: input.clearReason || undefined,
    relatedDayNumber: input.relatedDayNumber ?? undefined,
    clearRelatedDay: input.clearRelatedDay || undefined,
    relatedItemIndex: input.relatedItemIndex ?? undefined,
    clearRelatedIndex: input.clearRelatedIndex || undefined,
    relatedItemId: emptyToUndefined(input.relatedItemId),
    clearRelatedItem: input.clearRelatedItem || undefined,
    sortOrder: input.sortOrder ?? undefined,
    metadata: input.metadata
  });
}

function emptyToUndefined(value: string | null | undefined) {
  if (value == null) {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function removeUndefined<T extends Record<string, unknown>>(value: T) {
  return Object.fromEntries(
    Object.entries(value).filter(([, entry]) => entry !== undefined)
  ) as Partial<T>;
}

