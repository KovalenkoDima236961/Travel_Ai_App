"use client";

import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useDocumentVisibility } from "@/hooks/useDocumentVisibility";
import {
  generationJobKeys,
  getGenerationJob
} from "@/lib/api/generation-jobs";
import type { GenerationJob } from "@/entities/generation-job/model";

type UseGenerationJobInput = {
  tripId: string;
  jobId?: string | null;
  enabled: boolean;
  onCompleted?: (job: GenerationJob) => void;
  onFailed?: (job: GenerationJob) => void;
  onCancelled?: (job: GenerationJob) => void;
};

export function useGenerationJob({
  tripId,
  jobId,
  enabled,
  onCompleted,
  onFailed,
  onCancelled
}: UseGenerationJobInput) {
  const lastTerminalJobId = useRef<string | null>(null);
  const documentVisible = useDocumentVisibility();

  const query = useQuery({
    queryKey: generationJobKeys.detail(tripId, jobId ?? ""),
    queryFn: () => getGenerationJob(tripId, jobId ?? ""),
    enabled: enabled && Boolean(tripId) && Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!documentVisible || (status !== "queued" && status !== "running")) {
        return false;
      }
      // Fast feedback for a new job, then a gentler cadence for long-running work.
      return query.state.dataUpdateCount <= 4 ? 2500 : 5000;
    },
    refetchIntervalInBackground: false
  });

  useEffect(() => {
    const job = query.data;
    if (!job || lastTerminalJobId.current === job.id) {
      return;
    }

    if (job.status === "completed") {
      lastTerminalJobId.current = job.id;
      onCompleted?.(job);
    } else if (job.status === "failed") {
      lastTerminalJobId.current = job.id;
      onFailed?.(job);
    } else if (job.status === "cancelled") {
      lastTerminalJobId.current = job.id;
      onCancelled?.(job);
    }
  }, [onCancelled, onCompleted, onFailed, query.data]);

  useEffect(() => {
    if (jobId == null) {
      lastTerminalJobId.current = null;
    }
  }, [jobId]);

  return query;
}
