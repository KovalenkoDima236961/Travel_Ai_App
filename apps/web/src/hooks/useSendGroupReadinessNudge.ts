"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  groupReadinessKeys,
  sendGroupReadinessNudge
} from "@/lib/api/group-readiness";
import { activityKeys } from "@/lib/api/activity";
import { notificationKeys } from "@/lib/api/notifications";
import type { NudgeRequest } from "@/types/group-readiness";

export function useSendGroupReadinessNudge(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: NudgeRequest) => sendGroupReadinessNudge(tripId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupReadinessKeys.detail(tripId) });
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) });
      queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  });
}

