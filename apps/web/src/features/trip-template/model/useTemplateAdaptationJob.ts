"use client";

import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  getTemplateAdaptationJob,
  templateAdaptationKeys
} from "@/lib/api/template-adaptation";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { TemplateAdaptationJob } from "@/entities/template-adaptation/model";

type UseTemplateAdaptationJobInput = {
  tripId?: string | null;
  jobId?: string | null;
  enabled: boolean;
  onCompleted?: (job: TemplateAdaptationJob) => void;
  onFailed?: (job: TemplateAdaptationJob) => void;
  onCancelled?: (job: TemplateAdaptationJob) => void;
};

const TERMINAL: GenerationJob["status"][] = ["completed", "failed", "cancelled"];

/** Polls a template adaptation job until it reaches a terminal state, exposing
 * `createdTripId` (the draft trip) once known and the adaptation summary once
 * completed. */
export function useTemplateAdaptationJob({
  tripId,
  jobId,
  enabled,
  onCompleted,
  onFailed,
  onCancelled
}: UseTemplateAdaptationJobInput) {
  const lastTerminalJobId = useRef<string | null>(null);

  const query = useQuery({
    queryKey: templateAdaptationKeys.job(tripId ?? "", jobId ?? ""),
    queryFn: () => getTemplateAdaptationJob(tripId ?? "", jobId ?? ""),
    enabled: enabled && Boolean(tripId) && Boolean(jobId),
    refetchInterval: (q) => {
      const status = q.state.data?.status;
      return status === "queued" || status === "running" ? 2500 : false;
    }
  });

  useEffect(() => {
    const job = query.data as TemplateAdaptationJob | undefined;
    if (!job || !TERMINAL.includes(job.status) || lastTerminalJobId.current === job.id) {
      return;
    }
    lastTerminalJobId.current = job.id;
    if (job.status === "completed") {
      onCompleted?.(job);
    } else if (job.status === "failed") {
      onFailed?.(job);
    } else if (job.status === "cancelled") {
      onCancelled?.(job);
    }
  }, [onCancelled, onCompleted, onFailed, query.data]);

  useEffect(() => {
    if (jobId == null) {
      lastTerminalJobId.current = null;
    }
  }, [jobId]);

  const job = (query.data as TemplateAdaptationJob | undefined) ?? null;
  return {
    ...query,
    job,
    createdTripId: job?.tripId ?? tripId ?? null,
    summary: job?.status === "completed" ? (job.resultPayload ?? null) : null
  };
}
