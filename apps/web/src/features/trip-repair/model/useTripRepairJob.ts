"use client";

import { useQuery } from "@tanstack/react-query";
import { useDocumentVisibility } from "@/hooks/useDocumentVisibility";
import { getTripRepairJob, tripRepairKeys } from "@/lib/api/trip-repair";

const TERMINAL_STATUSES = new Set(["completed", "failed", "cancelled"]);

type UseTripRepairJobInput = {
  tripId: string;
  jobId?: string | null;
  enabled?: boolean;
};

export function useTripRepairJob({ tripId, jobId, enabled = true }: UseTripRepairJobInput) {
  const documentVisible = useDocumentVisibility();
  return useQuery({
    queryKey: jobId ? tripRepairKeys.job(tripId, jobId) : tripRepairKeys.all(tripId),
    queryFn: () => getTripRepairJob(tripId, jobId ?? ""),
    enabled: enabled && Boolean(tripId) && Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!documentVisible || (status && TERMINAL_STATUSES.has(status))) {
        return false;
      }
      return query.state.dataUpdateCount <= 4 ? 2500 : 5000;
    },
    refetchIntervalInBackground: false
  });
}
