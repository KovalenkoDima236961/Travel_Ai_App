"use client";

import { useQuery } from "@tanstack/react-query";
import { getTripRepairJob, tripRepairKeys } from "@/lib/api/trip-repair";

const TERMINAL_STATUSES = new Set(["completed", "failed", "cancelled"]);

type UseTripRepairJobInput = {
  tripId: string;
  jobId?: string | null;
  enabled?: boolean;
};

export function useTripRepairJob({ tripId, jobId, enabled = true }: UseTripRepairJobInput) {
  return useQuery({
    queryKey: jobId ? tripRepairKeys.job(tripId, jobId) : tripRepairKeys.all(tripId),
    queryFn: () => getTripRepairJob(tripId, jobId ?? ""),
    enabled: enabled && Boolean(tripId) && Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status && TERMINAL_STATUSES.has(status) ? false : 2500;
    }
  });
}
