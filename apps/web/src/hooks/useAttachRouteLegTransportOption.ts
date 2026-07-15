"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { approvalRiskKeys } from "@/lib/api/approval-risk";
import { budgetKeys } from "@/lib/api/budget";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { reminderKeys } from "@/lib/api/trip-reminders";
import { tripKeys } from "@/lib/api/trips";
import { attachRouteLegTransportOption } from "@/lib/api/transport";
import type { Trip } from "@/entities/trip/model";
import type { AttachRouteLegTransportOptionInput } from "@/types/transport";

export function useAttachRouteLegTransportOption(tripId: string, legId: string) {
  const queryClient = useQueryClient();
  return useMutation<Trip, Error, AttachRouteLegTransportOptionInput>({
    mutationFn: (input) => attachRouteLegTransportOption(tripId, legId, input),
    onSuccess: async (trip) => {
      queryClient.setQueryData(tripKeys.detail(tripId), trip);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.route(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() }),
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
        queryClient.invalidateQueries({ queryKey: reminderKeys.all }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all })
      ]);
    }
  });
}
