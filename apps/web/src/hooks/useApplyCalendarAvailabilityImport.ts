"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { applyCalendarAvailabilityImport } from "@/lib/api/calendar-free-busy";
import { notificationKeys } from "@/lib/api/notifications";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { tripAvailabilityKeys } from "@/lib/api/trip-availability";
import type { CalendarImportApplyRequest } from "@/types/calendar-free-busy";

export function useApplyCalendarAvailabilityImport(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CalendarImportApplyRequest) =>
      applyCalendarAvailabilityImport(tripId, input),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripAvailabilityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      ]);
    }
  });
}
