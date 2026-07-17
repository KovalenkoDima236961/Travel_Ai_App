"use client";

import { ReactNode, useMemo, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/shared/lib/cn";
import { ApiError } from "@/shared/api/client";
import {
  cancelOpsJob,
  discardDlqMessage,
  getDlqMessages,
  getOpsJob,
  getOpsJobs,
  getOpsJobSummary,
  getProviderQuotaDetail,
  getProviderQuotas,
  getProviderStatus,
  getQueuesStatus,
  getWorkerStatus,
  markOpsJobFailed,
  opsKeys,
  requeueDlqMessage,
  resetProviderQuotaDev,
  retryOpsJob,
  type OpsJobFilters
} from "@/lib/api/ops";
import {
  OPS_REFRESH_INTERVAL,
  formatOpsDate,
  invalidateOps,
  shortId,
  withReason
} from "../model/opsPageModel";
import { instrumentSans, jetBrainsMono, newsreader } from "./fonts";
import { OpsHeader } from "./OpsHeader";
import {
  CARD,
  CARD_HEADING,
  MONO,
  OUTLINE_BUTTON,
  SMALL_DANGER_BUTTON,
  SMALL_OUTLINE_BUTTON
} from "./opsStyles";
import {
  FilterInput,
  FilterSelect,
  JobActions,
  JobDetails,
  ProviderQuotaCard,
  ProviderQuotaDetailView,
  StatusPill,
  StatusTile,
  SummaryCard
} from "./OpsPageParts";

function OpsShell({ onRefresh, children }: { onRefresh: () => void; children: ReactNode }) {
  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        jetBrainsMono.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <OpsHeader onRefresh={onRefresh} />
      {/* Content region is a div, not <main> — the root layout already provides
          the <main> landmark, and nesting a second one is invalid. */}
      <div className="mx-auto max-w-[1360px] px-6 pb-[72px] pt-8 sm:px-10">{children}</div>
    </div>
  );
}

function NoticeCard({ title, message }: { title: string; message: string }) {
  return (
    <div className="rounded-[20px] border border-sand-300 bg-white p-7">
      <h1 className="font-newsreader text-[24px] font-semibold text-cocoa-900">{title}</h1>
      <p className="mt-2 text-[14.5px] text-cocoa-500">{message}</p>
    </div>
  );
}

function connectionTile(label: string, ok: boolean | undefined, okLabel: string, badLabel: string) {
  return (
    <StatusTile
      label={label}
      value={ok === undefined ? "—" : ok ? okLabel : badLabel}
      tone={ok === undefined ? "neutral" : ok ? "ok" : "bad"}
    />
  );
}

export function OpsPageContent() {
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<OpsJobFilters>({});
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);

  const summary = useQuery({
    queryKey: opsKeys.summary,
    queryFn: getOpsJobSummary,
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const jobs = useQuery({
    queryKey: opsKeys.jobs(filters),
    queryFn: () => getOpsJobs(filters),
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const selectedJob = useQuery({
    queryKey: opsKeys.job(selectedJobId),
    queryFn: () => getOpsJob(selectedJobId ?? ""),
    enabled: Boolean(selectedJobId),
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const worker = useQuery({
    queryKey: opsKeys.worker,
    queryFn: getWorkerStatus,
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const queues = useQuery({
    queryKey: opsKeys.queues,
    queryFn: getQueuesStatus,
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const dlq = useQuery({
    queryKey: opsKeys.dlq,
    queryFn: () => getDlqMessages(20),
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const providers = useQuery({
    queryKey: opsKeys.providers,
    queryFn: getProviderStatus,
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const providerQuotas = useQuery({
    queryKey: opsKeys.providerQuotas(),
    queryFn: () => getProviderQuotas(),
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const [selectedQuotaProvider, setSelectedQuotaProvider] = useState<string | null>(null);
  const quotaDetail = useQuery({
    queryKey: opsKeys.providerQuotaDetail(selectedQuotaProvider),
    queryFn: () => getProviderQuotaDetail(selectedQuotaProvider ?? ""),
    enabled: Boolean(selectedQuotaProvider),
    refetchInterval: OPS_REFRESH_INTERVAL
  });
  const resetQuotaMutation = useMutation({
    mutationFn: (provider: string) => resetProviderQuotaDev(provider),
    onSuccess: () => invalidateOps(queryClient)
  });

  const forbidden = [summary.error, jobs.error, worker.error, providers.error].some(
    (error) => error instanceof ApiError && error.status === 403
  );
  const disabled = [summary.error, jobs.error].some(
    (error) => error instanceof ApiError && error.status === 404
  );

  const retryMutation = useMutation({
    mutationFn: ({ jobId, reason }: { jobId: string; reason: string }) =>
      retryOpsJob(jobId, reason),
    onSuccess: () => invalidateOps(queryClient)
  });
  const cancelMutation = useMutation({
    mutationFn: ({ jobId, reason }: { jobId: string; reason: string }) =>
      cancelOpsJob(jobId, reason),
    onSuccess: () => invalidateOps(queryClient)
  });
  const markFailedMutation = useMutation({
    mutationFn: ({ jobId, reason }: { jobId: string; reason: string }) =>
      markOpsJobFailed(jobId, reason),
    onSuccess: () => invalidateOps(queryClient)
  });
  const requeueMutation = useMutation({
    mutationFn: ({ messageId, reason }: { messageId: string; reason: string }) =>
      requeueDlqMessage(messageId, reason),
    onSuccess: () => invalidateOps(queryClient)
  });
  const discardMutation = useMutation({
    mutationFn: ({ messageId, reason }: { messageId: string; reason: string }) =>
      discardDlqMessage(messageId, reason),
    onSuccess: () => invalidateOps(queryClient)
  });

  const degradedProviderCount = useMemo(
    () =>
      providers.data?.providers.filter((provider) =>
        ["degraded", "down"].includes(provider.status)
      ).length ?? 0,
    [providers.data]
  );

  const refreshAll = () => {
    void invalidateOps(queryClient);
  };

  if (forbidden) {
    return (
      <OpsShell onRefresh={refreshAll}>
        <NoticeCard
          title="Ops Dashboard"
          message="You do not have access to the Ops Dashboard."
        />
      </OpsShell>
    );
  }

  if (disabled) {
    return (
      <OpsShell onRefresh={refreshAll}>
        <NoticeCard title="Ops Dashboard" message="The Ops Dashboard is disabled." />
      </OpsShell>
    );
  }

  const jobRows = jobs.data?.jobs ?? [];
  const queueRows = queues.data?.queues ?? [];
  const dlqMessages = dlq.data?.messages ?? [];
  const dlqCount = dlqMessages.length;
  const providerRows = providers.data?.providers ?? [];
  const quotaRows = providerQuotas.data?.providers ?? [];

  return (
    <OpsShell onRefresh={refreshAll}>
      <div>
        <h1 className="font-newsreader text-[34px] font-medium tracking-[-0.02em] text-cocoa-900">
          Jobs, queues &amp; providers
        </h1>
        <p className="mt-2 text-[14.5px] text-cocoa-400">Live status of generation jobs, message queues, the DLQ, workers, and external providers.</p>
        <Link href="/ops/ai-generations" className="mt-3 inline-flex text-[13px] font-semibold text-clay hover:text-cocoa-900">View AI generation traces →</Link>
      </div>

      <section className="mt-7 grid grid-cols-2 gap-3.5 sm:grid-cols-3 xl:grid-cols-6">
        <SummaryCard label="Queued" value={summary.data?.countsByStatus.queued ?? 0} />
        <SummaryCard
          label="Running"
          value={summary.data?.countsByStatus.running ?? 0}
          valueClassName="text-[#4E6E86]"
        />
        <SummaryCard
          label="Failed"
          value={summary.data?.countsByStatus.failed ?? 0}
          valueClassName="text-[#B3402E]"
        />
        <SummaryCard label="Stale" value={summary.data?.staleRunningCount ?? 0} />
        <SummaryCard label="DLQ" value={dlqCount} valueClassName="text-[#B3402E]" />
        <SummaryCard
          label="Providers"
          value={degradedProviderCount}
          valueClassName="text-[#96682A]"
        />
      </section>

      <section className={cn(CARD, "mt-6")}>
        <h2 className={CARD_HEADING}>Recent jobs</h2>
        <div className="mt-4 flex flex-wrap items-end gap-3">
          <FilterSelect
            label="Status"
            value={filters.status ?? ""}
            onChange={(status) => setFilters((current) => ({ ...current, status }))}
            options={["", "queued", "running", "completed", "failed", "cancelled"]}
          />
          <FilterSelect
            label="Type"
            value={filters.jobType ?? ""}
            onChange={(jobType) => setFilters((current) => ({ ...current, jobType }))}
            options={[
              "",
              "full_generation",
              "day_regeneration",
              "item_regeneration",
              "quality_improvement_day",
              "quality_improvement_item",
              "budget_optimization_day"
            ]}
          />
          <FilterInput
            label="Error"
            value={filters.errorCode ?? ""}
            onChange={(errorCode) => setFilters((current) => ({ ...current, errorCode }))}
          />
          <FilterInput
            label="Trip ID"
            value={filters.tripId ?? ""}
            onChange={(tripId) => setFilters((current) => ({ ...current, tripId }))}
          />
          <FilterInput
            label="User ID"
            value={filters.userId ?? ""}
            onChange={(userId) => setFilters((current) => ({ ...current, userId }))}
          />
          <button type="button" className={OUTLINE_BUTTON} onClick={() => setFilters({})}>
            Clear
          </button>
        </div>

        <div className="mt-4 overflow-x-auto rounded-[14px] border border-sand-200">
          <table className="min-w-full text-left">
            <thead>
              <tr className="bg-sand-50 text-[11.5px] uppercase tracking-[0.04em] text-[#A08D78]">
                <th className="px-4 py-3 font-semibold">Job</th>
                <th className="px-4 py-3 font-semibold">Type</th>
                <th className="px-4 py-3 font-semibold">Status</th>
                <th className="px-4 py-3 font-semibold">Error</th>
                <th className="px-4 py-3 font-semibold">Trip</th>
                <th className="px-4 py-3 font-semibold">Scope</th>
                <th className="px-4 py-3 font-semibold">Created</th>
                <th className="px-4 py-3 font-semibold">Correlation</th>
                <th className="px-4 py-3 text-right font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody>
              {jobRows.map((job) => (
                <tr key={job.id} className="border-t border-sand-200 align-middle">
                  <td className={cn("px-4 py-3.5 text-[12.5px] text-cocoa-500", MONO)}>
                    {shortId(job.id)}
                  </td>
                  <td className="px-4 py-3.5 text-[13px] text-cocoa-900">{job.jobType}</td>
                  <td className="px-4 py-3.5">
                    <StatusPill status={job.status} />
                  </td>
                  <td className={cn("px-4 py-3.5 text-[12px] text-cocoa-400", MONO)}>
                    {job.errorCode ?? "—"}
                  </td>
                  <td className={cn("px-4 py-3.5 text-[12.5px] text-cocoa-500", MONO)}>
                    {shortId(job.tripId)}
                  </td>
                  <td className="px-4 py-3.5 text-[13px] text-cocoa-500">{job.scope ?? "—"}</td>
                  <td className="px-4 py-3.5 text-[12.5px] text-cocoa-400">
                    {formatOpsDate(job.createdAt)}
                  </td>
                  <td className={cn("px-4 py-3.5 text-[12.5px] text-cocoa-500", MONO)}>
                    {shortId(job.correlationId)}
                  </td>
                  <td className="px-4 py-3.5">
                    <JobActions
                      job={job}
                      onView={() => setSelectedJobId(job.id)}
                      onRetry={() =>
                        withReason("Retry creates a new queued job.", (reason) =>
                          retryMutation.mutate({ jobId: job.id, reason })
                        )
                      }
                      onCancel={() =>
                        withReason("Cancel affects this queued job only.", (reason) =>
                          cancelMutation.mutate({ jobId: job.id, reason })
                        )
                      }
                      onMarkFailed={() =>
                        withReason("Mark failed affects only stale running jobs.", (reason) =>
                          markFailedMutation.mutate({ jobId: job.id, reason })
                        )
                      }
                    />
                  </td>
                </tr>
              ))}
              {jobs.data && jobRows.length === 0 ? (
                <tr className="border-t border-sand-200">
                  <td colSpan={9} className="px-4 py-8 text-center text-[13px] text-cocoa-400">
                    No jobs match these filters.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>

      {selectedJobId ? (
        <section className={cn(CARD, "mt-6")}>
          <div className="flex items-start justify-between gap-4">
            <h2 className={CARD_HEADING}>Job details</h2>
            <button
              type="button"
              className={SMALL_OUTLINE_BUTTON}
              onClick={() => setSelectedJobId(null)}
            >
              Close
            </button>
          </div>
          {selectedJob.data ? (
            <JobDetails job={selectedJob.data} />
          ) : (
            <p className="mt-3 text-[14px] text-cocoa-400">Loading…</p>
          )}
        </section>
      ) : null}

      <section className="mt-6 grid gap-6 xl:grid-cols-2">
        <div className={CARD}>
          <h2 className={CARD_HEADING}>Worker &amp; queues</h2>
          <div className="mt-[18px] grid grid-cols-2 gap-3">
            {connectionTile("Worker", worker.data?.healthy, "Healthy", "Down")}
            {connectionTile("RabbitMQ", worker.data?.rabbitmqConnected, "Connected", "Down")}
            {connectionTile("Postgres", worker.data?.dbConnected, "Connected", "Down")}
            <StatusTile label="Active jobs" value={String(worker.data?.activeJobs.length ?? 0)} />
          </div>
          <div className="mt-4 flex flex-col gap-2.5">
            {queueRows.map((queue) => (
              <div key={queue.name} className="rounded-xl border border-sand-200 px-4 py-3">
                <p className="text-[13.5px] font-semibold text-cocoa-900">{queue.name}</p>
                <p className="mt-1 text-[12.5px] text-cocoa-400">
                  ready {queue.messagesReady} · unacked {queue.messagesUnacked} · consumers{" "}
                  {queue.consumers}
                </p>
              </div>
            ))}
            {queues.error ? (
              <p className="text-[12.5px] text-[#96682A]">Queue status unavailable.</p>
            ) : null}
          </div>
        </div>

        <div className={CARD}>
          <h2 className={CARD_HEADING}>Provider health</h2>
          <div className="mt-[18px] flex flex-col gap-2.5">
            {providerRows.map((provider) => (
              <div
                key={provider.name}
                className="flex items-center justify-between gap-3 rounded-xl border border-sand-200 px-4 py-3"
              >
                <div>
                  <p className="text-[13.5px] font-semibold text-cocoa-900">{provider.name}</p>
                  <p className="mt-1 text-[12.5px] text-cocoa-400">
                    {provider.activeProvider} · success {provider.recentSuccessCount} · failures{" "}
                    {provider.recentFailureCount}
                    {provider.lastErrorCode ? ` · ${provider.lastErrorCode}` : ""}
                  </p>
                </div>
                <StatusPill status={provider.status} />
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className={cn(CARD, "mt-6")}>
        <div className="flex flex-wrap items-center justify-between gap-4">
          <h2 className={CARD_HEADING}>Dead-letter queue</h2>
          <span className="rounded-full bg-[#FBF0EB] px-3 py-1 text-[12px] font-semibold text-[#B3402E]">
            {dlqCount} {dlqCount === 1 ? "message" : "messages"}
          </span>
        </div>
        <div className="mt-4 flex flex-col gap-2.5">
          {dlqMessages.map((message) => (
            <div
              key={message.messageId}
              className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-sand-200 px-4 py-3.5"
            >
              <div>
                <p className={cn("text-[12.5px] text-cocoa-900", MONO)}>
                  {shortId(message.messageId)}
                </p>
                <p className="mt-1 text-[12.5px] text-cocoa-400">
                  {message.jobType ?? "unknown"} · attempts {message.attempts} ·{" "}
                  {shortId(message.correlationId)}
                </p>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  className={SMALL_OUTLINE_BUTTON}
                  onClick={() =>
                    withReason("Requeue republishes this message to the main queue.", (reason) =>
                      requeueMutation.mutate({ messageId: message.messageId, reason })
                    )
                  }
                >
                  Requeue
                </button>
                <button
                  type="button"
                  className={SMALL_DANGER_BUTTON}
                  onClick={() =>
                    withReason("Discard removes this DLQ message.", (reason) =>
                      discardMutation.mutate({ messageId: message.messageId, reason })
                    )
                  }
                >
                  Discard
                </button>
              </div>
            </div>
          ))}
          {dlqCount === 0 ? (
            <p className="text-[13px] text-cocoa-400">No DLQ messages.</p>
          ) : null}
        </div>
      </section>

      <section className={cn(CARD, "mt-6")}>
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 className={CARD_HEADING}>Provider quotas</h2>
            <p className="mt-1.5 text-[14px] text-cocoa-400">
              Per-provider rate limits and daily quota usage for{" "}
              {providerQuotas.data?.date ?? "today"}.
              {providerQuotas.data && !providerQuotas.data.enabled
                ? " Enforcement is disabled in this environment."
                : ""}
            </p>
          </div>
          <button
            type="button"
            className={OUTLINE_BUTTON}
            onClick={() => {
              void queryClient.invalidateQueries({ queryKey: ["ops", "provider-quotas"] });
              void queryClient.invalidateQueries({ queryKey: ["ops", "provider-quota"] });
            }}
          >
            Refresh
          </button>
        </div>

        {providerQuotas.error ? (
          <p className="mt-4 text-[13px] text-[#96682A]">
            Provider quotas are currently unavailable.
          </p>
        ) : null}

        <div className="mt-4 grid gap-3 md:grid-cols-2">
          {quotaRows.map((provider) => (
            <ProviderQuotaCard
              key={`${provider.category}-${provider.provider}`}
              provider={provider}
              expanded={selectedQuotaProvider === provider.provider}
              resetAllowed={Boolean(providerQuotas.data?.resetAllowed)}
              resetPending={resetQuotaMutation.isPending}
              onToggle={() =>
                setSelectedQuotaProvider((current) =>
                  current === provider.provider ? null : provider.provider
                )
              }
              onReset={() => {
                if (
                  window.confirm(
                    `Reset today's ${provider.provider} quota counters? This is a dev-only action.`
                  )
                ) {
                  resetQuotaMutation.mutate(provider.provider);
                }
              }}
            />
          ))}
          {providerQuotas.data && quotaRows.length === 0 ? (
            <p className="text-[13px] text-cocoa-400">No providers configured.</p>
          ) : null}
        </div>

        {selectedQuotaProvider ? (
          <div className="mt-5 rounded-xl border border-sand-200 p-4">
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-[13.5px] font-semibold text-cocoa-900">
                {selectedQuotaProvider} — operation breakdown &amp; last 7 days
              </h3>
              <button
                type="button"
                className={SMALL_OUTLINE_BUTTON}
                onClick={() => setSelectedQuotaProvider(null)}
              >
                Close
              </button>
            </div>
            {quotaDetail.data ? (
              <ProviderQuotaDetailView detail={quotaDetail.data} />
            ) : (
              <p className="mt-3 text-[14px] text-cocoa-400">Loading…</p>
            )}
          </div>
        ) : null}
      </section>
    </OpsShell>
  );
}
