"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  aiDatasetKeys,
  approveAIDatasetExample,
  getAIDatasetReadiness,
  listAIDatasetExamples,
  rejectAIDatasetExample,
  type AIDatasetExample
} from "@/lib/api/ai-datasets";
import { getErrorMessage } from "@/lib/utils";
import { cn } from "@/shared/lib/cn";
import { shortId, withReason } from "@/_pages/ops/model/opsPageModel";
import {
  CARD_HEADING,
  MONO,
  SMALL_DANGER_BUTTON,
  SMALL_OUTLINE_BUTTON,
  statusPillClass
} from "@/_pages/ops/ui/opsStyles";

const pendingFilters = { reviewStatus: "pending", limit: 8 };

export function AIDatasetCurationPanel() {
  const queryClient = useQueryClient();
  const readiness = useQuery({
    queryKey: aiDatasetKeys.readiness,
    queryFn: getAIDatasetReadiness,
    staleTime: 30_000
  });
  const examples = useQuery({
    queryKey: aiDatasetKeys.examples(pendingFilters),
    queryFn: () => listAIDatasetExamples(pendingFilters),
    staleTime: 30_000
  });
  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: aiDatasetKeys.readiness }),
      queryClient.invalidateQueries({ queryKey: ["ai-datasets", "examples"] })
    ]);
  };
  const approve = useMutation({
    mutationFn: ({ exampleId, reason }: { exampleId: string; reason: string }) =>
      approveAIDatasetExample(exampleId, reason),
    onSuccess: refresh
  });
  const reject = useMutation({
    mutationFn: ({ exampleId, reason }: { exampleId: string; reason: string }) =>
      rejectAIDatasetExample(exampleId, reason),
    onSuccess: refresh
  });
  const rows = examples.data?.examples ?? [];
  const actionPending = approve.isPending || reject.isPending;
  const error = readiness.error ?? examples.error ?? approve.error ?? reject.error;

  return (
    <section className="mt-6 rounded-[20px] border border-sand-300 bg-white px-6 py-6 sm:px-[26px]">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className={CARD_HEADING}>AI dataset curation</h2>
          <p className="mt-1.5 text-[14px] text-cocoa-400">
            Consent, sanitizer, quality, review, split, and export readiness.
          </p>
        </div>
        <span className={statusPillClass(readiness.data?.ready ? "healthy" : "degraded")}>
          {readiness.data?.ready ? "Ready" : "Not ready"}
        </span>
      </div>

      {error ? (
        <p className="mt-4 rounded-xl border border-[#E5C3B6] bg-[#FBF0EB] px-4 py-3 text-[13px] text-[#B3402E]">
          {getErrorMessage(error, "AI dataset curation is unavailable.")}
        </p>
      ) : null}

      <div className="mt-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <DatasetMetric label="Approved" value={readiness.data?.approvedExampleCount ?? 0} />
        <DatasetMetric label="Holdout" value={readiness.data?.holdoutCount ?? 0} />
        <DatasetMetric label="Duplicates" value={readiness.data?.duplicateCount ?? 0} />
        <DatasetMetric label="Sanitization failures" value={readiness.data?.sanitizationFailureCount ?? 0} />
      </div>

      {readiness.data?.blockers.length ? (
        <div className="mt-4 rounded-xl border border-sand-200 bg-sand-50 px-4 py-3">
          <p className="text-[13.5px] font-semibold text-cocoa-900">Readiness blockers</p>
          <ul className="mt-2 space-y-1 text-[13px] text-cocoa-500">
            {readiness.data.blockers.map((blocker) => (
              <li key={blocker}>{blocker}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="mt-5 overflow-x-auto rounded-[14px] border border-sand-200">
        <table className="min-w-full text-left">
          <thead>
            <tr className="bg-sand-50 text-[11.5px] uppercase tracking-[0.04em] text-[#A08D78]">
              <th className="px-4 py-3 font-semibold">Example</th>
              <th className="px-4 py-3 font-semibold">Task</th>
              <th className="px-4 py-3 font-semibold">Source</th>
              <th className="px-4 py-3 font-semibold">Consent</th>
              <th className="px-4 py-3 font-semibold">Sanitizer</th>
              <th className="px-4 py-3 font-semibold">Quality</th>
              <th className="px-4 py-3 text-right font-semibold">Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((example) => (
              <DatasetExampleRow
                disabled={actionPending}
                example={example}
                key={example.id}
                onApprove={() =>
                  withReason("Approve this sanitized training example.", (reason) =>
                    approve.mutate({ exampleId: example.id, reason })
                  )
                }
                onReject={() =>
                  withReason("Reject this training example.", (reason) =>
                    reject.mutate({ exampleId: example.id, reason })
                  )
                }
              />
            ))}
            {!examples.isPending && rows.length === 0 ? (
              <tr className="border-t border-sand-200">
                <td className="px-4 py-8 text-center text-[13px] text-cocoa-400" colSpan={7}>
                  No pending examples.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function DatasetMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl border border-sand-200 bg-sand-50 px-4 py-3">
      <p className="text-[12px] font-semibold uppercase tracking-[0.04em] text-[#A08D78]">
        {label}
      </p>
      <p className="mt-1 text-[22px] font-semibold text-cocoa-900">{value}</p>
    </div>
  );
}

function DatasetExampleRow({
  disabled,
  example,
  onApprove,
  onReject
}: {
  disabled: boolean;
  example: AIDatasetExample;
  onApprove: () => void;
  onReject: () => void;
}) {
  const quality = example.qualityScore == null ? "-" : example.qualityScore.toFixed(2);
  return (
    <tr className="border-t border-sand-200 align-middle">
      <td className={cn("px-4 py-3.5 text-[12.5px] text-cocoa-500", MONO)}>
        {shortId(example.id)}
      </td>
      <td className="px-4 py-3.5 text-[13px] text-cocoa-900">{example.taskType}</td>
      <td className="px-4 py-3.5 text-[13px] text-cocoa-500">{example.sourceType}</td>
      <td className="px-4 py-3.5">
        <span className={statusPillClass(example.consentStatus === "granted" || example.consentStatus === "not_required" ? "healthy" : "degraded")}>
          {example.consentStatus}
        </span>
      </td>
      <td className="px-4 py-3.5">
        <span className={statusPillClass(example.sanitizationStatus === "passed" ? "healthy" : "degraded")}>
          {example.sanitizationStatus}
        </span>
      </td>
      <td className="px-4 py-3.5 text-[13px] text-cocoa-500">
        {example.qualityStatus} · {quality}
      </td>
      <td className="px-4 py-3.5">
        <div className="flex justify-end gap-2">
          <button className={SMALL_OUTLINE_BUTTON} disabled={disabled} onClick={onApprove} type="button">
            Approve
          </button>
          <button className={SMALL_DANGER_BUTTON} disabled={disabled} onClick={onReject} type="button">
            Reject
          </button>
        </div>
      </td>
    </tr>
  );
}
