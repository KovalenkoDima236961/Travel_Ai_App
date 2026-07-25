"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  aiTrainingExperimentKeys,
  cancelAITrainingExperiment,
  decideAITrainingExperimentPromotion,
  evaluateAITrainingExperiment,
  getAITrainingExperiment,
  listAITrainingExperiments,
  startAITrainingExperiment,
  validateAITrainingExperiment,
  type AITrainingExperiment
} from "@/lib/api/ai-training-experiments";
import { getErrorMessage } from "@/lib/utils";
import { ApiError } from "@/shared/api/client";
import { cn } from "@/shared/lib/cn";
import { shortId, withReason } from "@/_pages/ops/model/opsPageModel";
import {
  CARD_HEADING,
  MONO,
  SMALL_DANGER_BUTTON,
  SMALL_OUTLINE_BUTTON,
  statusPillClass
} from "@/_pages/ops/ui/opsStyles";

export function AIExperimentsPanel() {
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const experiments = useQuery({
    queryKey: aiTrainingExperimentKeys.all,
    queryFn: listAITrainingExperiments,
    staleTime: 30_000
  });
  const rows = useMemo(
    () => experiments.data?.experiments ?? [],
    [experiments.data?.experiments]
  );
  const activeId = selectedId ?? rows[0]?.id ?? null;
  const detail = useQuery({
    queryKey: aiTrainingExperimentKeys.detail(activeId),
    queryFn: () => getAITrainingExperiment(activeId ?? ""),
    enabled: Boolean(activeId),
    staleTime: 30_000
  });
  const unavailable = [experiments.error, detail.error].some(
    (error) => error instanceof ApiError && (error.status === 403 || error.status === 404)
  );
  const error = experiments.error ?? detail.error;
  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: aiTrainingExperimentKeys.all }),
      queryClient.invalidateQueries({ queryKey: aiTrainingExperimentKeys.detail(activeId) })
    ]);
  };
  const validate = useMutation({ mutationFn: validateAITrainingExperiment, onSuccess: refresh });
  const start = useMutation({ mutationFn: startAITrainingExperiment, onSuccess: refresh });
  const evaluate = useMutation({
    mutationFn: (experimentId: string) => evaluateAITrainingExperiment(experimentId),
    onSuccess: refresh
  });
  const cancel = useMutation({
    mutationFn: ({ experimentId, reason }: { experimentId: string; reason: string }) =>
      cancelAITrainingExperiment(experimentId, reason),
    onSuccess: refresh
  });
  const decide = useMutation({
    mutationFn: ({
      decision,
      experimentId,
      reason
    }: {
      decision: "approve_staging" | "reject";
      experimentId: string;
      reason: string;
    }) => decideAITrainingExperimentPromotion(experimentId, decision, reason),
    onSuccess: refresh
  });
  const selectedExperiment = useMemo(
    () => rows.find((row) => row.id === activeId) ?? rows[0] ?? null,
    [activeId, rows]
  );
  const actionPending =
    validate.isPending || start.isPending || evaluate.isPending || cancel.isPending || decide.isPending;

  return (
    <section className="mt-6 rounded-[20px] border border-sand-300 bg-white px-6 py-6 sm:px-[26px]">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className={CARD_HEADING}>AI fine-tuning experiments</h2>
          <p className="mt-1.5 text-[14px] text-cocoa-400">
            Local adapter experiments, evaluation variants, and manual promotion gates.
          </p>
        </div>
        <span className={statusPillClass(unavailable ? "degraded" : "healthy")}>
          {unavailable ? "Disabled" : "Available"}
        </span>
      </div>

      {error ? (
        <p className="mt-4 rounded-xl border border-[#E5C3B6] bg-[#FBF0EB] px-4 py-3 text-[13px] text-[#B3402E]">
          {getErrorMessage(error, "AI fine-tuning experiments are unavailable.")}
        </p>
      ) : null}

      <div className="mt-5 grid gap-4 xl:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)]">
        <div className="overflow-x-auto rounded-[14px] border border-sand-200">
          <table className="min-w-full text-left">
            <thead>
              <tr className="bg-sand-50 text-[11.5px] uppercase tracking-[0.04em] text-[#A08D78]">
                <th className="px-4 py-3 font-semibold">Experiment</th>
                <th className="px-4 py-3 font-semibold">Dataset</th>
                <th className="px-4 py-3 font-semibold">Method</th>
                <th className="px-4 py-3 font-semibold">Status</th>
                <th className="px-4 py-3 font-semibold">Adapter</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((experiment) => (
                <ExperimentRow
                  active={experiment.id === activeId}
                  experiment={experiment}
                  key={experiment.id}
                  onSelect={() => setSelectedId(experiment.id)}
                />
              ))}
              {!experiments.isPending && rows.length === 0 ? (
                <tr className="border-t border-sand-200">
                  <td className="px-4 py-8 text-center text-[13px] text-cocoa-400" colSpan={5}>
                    No experiments.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>

        <div className="rounded-[14px] border border-sand-200 bg-sand-50 px-4 py-4">
          <h3 className="text-[14px] font-semibold text-cocoa-900">Promotion gates</h3>
          {selectedExperiment ? (
            <>
              <div className="mt-3 grid grid-cols-2 gap-2">
                <GateMetric label="Train" value={selectedExperiment.trainCount} />
                <GateMetric label="Validation" value={selectedExperiment.validationCount} />
                <GateMetric label="Test" value={selectedExperiment.testCount} />
                <GateMetric label="Holdout" value={selectedExperiment.holdoutCount} />
              </div>
              <div className="mt-4 space-y-2 text-[13px] text-cocoa-500">
                {(detail.data?.promotionGates?.failedGates ?? []).length === 0 ? (
                  <p>No failed gates reported.</p>
                ) : (
                  detail.data?.promotionGates?.failedGates.map((gate) => <p key={gate}>{gate}</p>)
                )}
              </div>
              <div className="mt-4 flex flex-wrap gap-2">
                <button
                  className={SMALL_OUTLINE_BUTTON}
                  disabled={actionPending}
                  onClick={() => validate.mutate(selectedExperiment.id)}
                  type="button"
                >
                  Validate
                </button>
                <button
                  className={SMALL_OUTLINE_BUTTON}
                  disabled={actionPending}
                  onClick={() => evaluate.mutate(selectedExperiment.id)}
                  type="button"
                >
                  Evaluate
                </button>
                <button
                  className={SMALL_OUTLINE_BUTTON}
                  disabled={actionPending}
                  onClick={() => start.mutate(selectedExperiment.id)}
                  type="button"
                >
                  Start
                </button>
                <button
                  className={SMALL_OUTLINE_BUTTON}
                  disabled={actionPending}
                  onClick={() =>
                    withReason("Approve this model candidate for staging.", (reason) =>
                      decide.mutate({
                        decision: "approve_staging",
                        experimentId: selectedExperiment.id,
                        reason
                      })
                    )
                  }
                  type="button"
                >
                  Stage
                </button>
                <button
                  className={SMALL_DANGER_BUTTON}
                  disabled={actionPending}
                  onClick={() =>
                    withReason("Cancel this fine-tuning experiment.", (reason) =>
                      cancel.mutate({ experimentId: selectedExperiment.id, reason })
                    )
                  }
                  type="button"
                >
                  Cancel
                </button>
                <button
                  className={SMALL_DANGER_BUTTON}
                  disabled={actionPending}
                  onClick={() =>
                    withReason("Reject this model candidate.", (reason) =>
                      decide.mutate({
                        decision: "reject",
                        experimentId: selectedExperiment.id,
                        reason
                      })
                    )
                  }
                  type="button"
                >
                  Reject
                </button>
              </div>
            </>
          ) : (
            <p className="mt-3 text-[13px] text-cocoa-400">No experiment selected.</p>
          )}
        </div>
      </div>
    </section>
  );
}

function ExperimentRow({
  active,
  experiment,
  onSelect
}: {
  active: boolean;
  experiment: AITrainingExperiment;
  onSelect: () => void;
}) {
  return (
    <tr
      className={cn(
        "cursor-pointer border-t border-sand-200 align-middle hover:bg-sand-50",
        active && "bg-[#FFFBF5]"
      )}
      onClick={onSelect}
    >
      <td className="px-4 py-3.5">
        <p className="text-[13px] font-semibold text-cocoa-900">{experiment.name}</p>
        <p className={cn("mt-1 text-[12px] text-cocoa-400", MONO)}>{shortId(experiment.id)}</p>
      </td>
      <td className="px-4 py-3.5 text-[13px] text-cocoa-500">{experiment.datasetVersion}</td>
      <td className="px-4 py-3.5 text-[13px] text-cocoa-500">{experiment.method}</td>
      <td className="px-4 py-3.5">
        <span className={statusPillClass(experiment.status)}>{experiment.status}</span>
      </td>
      <td className={cn("px-4 py-3.5 text-[12.5px] text-cocoa-500", MONO)}>
        {experiment.adapterKey ?? "-"}
      </td>
    </tr>
  );
}

function GateMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl border border-sand-200 bg-white px-3 py-2">
      <p className="text-[11px] font-semibold uppercase tracking-[0.04em] text-[#A08D78]">
        {label}
      </p>
      <p className="mt-1 text-[18px] font-semibold text-cocoa-900">{value}</p>
    </div>
  );
}
