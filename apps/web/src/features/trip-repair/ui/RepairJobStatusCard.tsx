"use client";

import { Button } from "@/shared/ui/button";
import { cn } from "@/lib/utils";
import type { GenerationJob } from "@/entities/generation-job/model";

type RepairJobStatusCardProps = {
  job: GenerationJob;
  onCancel?: (job: GenerationJob) => Promise<void> | void;
  cancelling?: boolean;
};

export function RepairJobStatusCard({
  job,
  onCancel,
  cancelling = false
}: RepairJobStatusCardProps) {
  return (
    <div className="rounded-md border border-amber-200 bg-amber-50 p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-semibold text-amber-950">AI repair job</p>
            <span
              className={cn(
                "rounded-full border px-2.5 py-1 text-xs font-medium capitalize",
                job.status === "queued" && "border-amber-200 bg-white text-amber-800",
                job.status === "running" && "border-blue-200 bg-blue-50 text-blue-700",
                job.status === "completed" && "border-emerald-200 bg-emerald-50 text-emerald-700",
                job.status === "failed" && "border-red-200 bg-red-50 text-red-700",
                job.status === "cancelled" && "border-slate-200 bg-white text-slate-600"
              )}
            >
              {job.status}
            </span>
          </div>
          <p className="mt-1 text-sm text-amber-800">
            {job.status === "queued"
              ? "Queued for policy-aware repair."
              : job.status === "running"
                ? "Building a reviewable repair proposal."
                : job.errorMessage || "Repair job finished."}
          </p>
        </div>
        {job.status === "queued" && onCancel ? (
          <Button
            disabled={cancelling}
            onClick={() => onCancel(job)}
            size="sm"
            type="button"
            variant="secondary"
          >
            {cancelling ? "Cancelling..." : "Cancel"}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
