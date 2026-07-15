"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { tripHealthKeys } from "@/lib/api/trip-health";
import {
  checkChecklistItem,
  checklistKeys,
  createChecklistItem,
  deleteChecklistItem,
  generateTripChecklist,
  reorderChecklistItems,
  uncheckChecklistItem,
  updateChecklistItem
} from "@/lib/api/checklists";
import type {
  ChecklistItemPayload,
  GenerateChecklistRequest,
  UpdateChecklistItemPayload
} from "@/entities/checklist/model";

export function useChecklistMutations(tripId: string) {
  const queryClient = useQueryClient();

  async function invalidateChecklist() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: checklistKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
    ]);
  }

  const generateMutation = useMutation({
    mutationFn: (input: GenerateChecklistRequest) => generateTripChecklist(tripId, input),
    onSuccess: async (data) => {
      queryClient.setQueryData(checklistKeys.detail(tripId), data);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
    }
  });

  const createItemMutation = useMutation({
    mutationFn: (input: ChecklistItemPayload) => createChecklistItem(tripId, input),
    onSuccess: invalidateChecklist
  });

  const updateItemMutation = useMutation({
    mutationFn: ({
      itemId,
      input
    }: {
      itemId: string;
      input: UpdateChecklistItemPayload;
    }) => updateChecklistItem(tripId, itemId, input),
    onSuccess: invalidateChecklist
  });

  const deleteItemMutation = useMutation({
    mutationFn: (itemId: string) => deleteChecklistItem(tripId, itemId),
    onSuccess: invalidateChecklist
  });

  const setCheckedMutation = useMutation({
    mutationFn: ({ itemId, checked }: { itemId: string; checked: boolean }) =>
      checked ? checkChecklistItem(tripId, itemId) : uncheckChecklistItem(tripId, itemId),
    onSuccess: invalidateChecklist
  });

  const reorderMutation = useMutation({
    mutationFn: (itemIds: string[]) => reorderChecklistItems(tripId, itemIds),
    onSuccess: invalidateChecklist
  });

  return {
    generateMutation,
    createItemMutation,
    updateItemMutation,
    deleteItemMutation,
    setCheckedMutation,
    reorderMutation
  };
}
