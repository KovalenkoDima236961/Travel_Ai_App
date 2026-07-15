import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { activityKeys } from "@/lib/api/activity";
import { approvalRiskKeys } from "@/lib/api/approval-risk";
import {
  approvalKeys,
  approveTrip,
  cancelTripApproval,
  getTripApproval,
  listTripApprovalEvents,
  requestTripChanges,
  submitTripApproval
} from "@/lib/api/approvals";
import { notificationKeys } from "@/lib/api/notifications";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { tripKeys } from "@/lib/api/trips";
import type {
  ApprovalDecisionInput,
  CancelApprovalInput,
  SubmitApprovalInput
} from "@/entities/approval/model";

export function useTripApproval(tripId: string, enabled = true) {
  return useQuery({
    queryKey: approvalKeys.trip(tripId),
    queryFn: () => getTripApproval(tripId),
    enabled: enabled && Boolean(tripId)
  });
}

export function useTripApprovalEvents(tripId: string, enabled = true) {
  return useQuery({
    queryKey: approvalKeys.tripEvents(tripId),
    queryFn: () => listTripApprovalEvents(tripId),
    enabled: enabled && Boolean(tripId)
  });
}

// useTripApprovalMutations wires submit/approve/request-changes/cancel and
// invalidates the trip, its approval state/history, activity feed, and
// notifications so every dependent view refreshes after an action.
export function useTripApprovalMutations(tripId: string) {
  const queryClient = useQueryClient();

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: approvalKeys.trip(tripId) });
    void queryClient.invalidateQueries({ queryKey: approvalKeys.tripEvents(tripId) });
    void queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) });
    void queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) });
    void queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) });
    void queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) });
    void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    // Workspace approvals queue counts/rows change after any action.
    void queryClient.invalidateQueries({ queryKey: approvalKeys.all });
  }

  return {
    submit: useMutation({
      mutationFn: (input: SubmitApprovalInput) => submitTripApproval(tripId, input),
      onSuccess: invalidate
    }),
    approve: useMutation({
      mutationFn: (input: ApprovalDecisionInput) => approveTrip(tripId, input),
      onSuccess: invalidate
    }),
    requestChanges: useMutation({
      mutationFn: (input: ApprovalDecisionInput) => requestTripChanges(tripId, input),
      onSuccess: invalidate
    }),
    cancel: useMutation({
      mutationFn: (input: CancelApprovalInput) => cancelTripApproval(tripId, input),
      onSuccess: invalidate
    })
  };
}
