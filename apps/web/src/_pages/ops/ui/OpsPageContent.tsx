"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
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
import {
  FilterInput,
  FilterSelect,
  JobActions,
  JobDetails,
  Metric,
  ProviderQuotaCard,
  ProviderQuotaDetailView,
  StatusPill,
  SummaryCard
} from "./OpsPageParts";

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

  if (forbidden) {
    return (
      <PageContainer className="py-10">
        <Card>
          <h1 className="text-xl font-semibold text-slate-950">Ops Dashboard</h1>
          <p className="mt-2 text-sm text-slate-600">
            You do not have access to Ops Dashboard.
          </p>
        </Card>
      </PageContainer>
    );
  }

  if (disabled) {
    return (
      <PageContainer className="py-10">
        <Card>
          <h1 className="text-xl font-semibold text-slate-950">Ops Dashboard</h1>
          <p className="mt-2 text-sm text-slate-600">Ops Dashboard is disabled.</p>
        </Card>
      </PageContainer>
    );
  }

  return (
    <PageContainer className="space-y-6 py-8">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-slate-950">Ops Dashboard</h1>
          <p className="mt-1 text-sm text-slate-600">Jobs, queues, DLQ, workers, and providers.</p>
        </div>
        <Button
          variant="secondary"
          onClick={() => {
            void invalidateOps(queryClient);
          }}
        >
          Refresh
        </Button>
      </div>

      <section className="grid gap-4 md:grid-cols-3 xl:grid-cols-6">
        <SummaryCard label="Queued" value={summary.data?.countsByStatus.queued ?? 0} />
        <SummaryCard label="Running" value={summary.data?.countsByStatus.running ?? 0} />
        <SummaryCard label="Failed" value={summary.data?.countsByStatus.failed ?? 0} />
        <SummaryCard label="Stale" value={summary.data?.staleRunningCount ?? 0} />
        <SummaryCard label="DLQ" value={dlq.data?.messages.length ?? 0} />
        <SummaryCard label="Providers" value={degradedProviderCount} />
      </section>

      <Card>
        <div className="flex flex-wrap items-end gap-3">
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
          <FilterInput label="Error" value={filters.errorCode ?? ""} onChange={(errorCode) => setFilters((current) => ({ ...current, errorCode }))} />
          <FilterInput label="Trip ID" value={filters.tripId ?? ""} onChange={(tripId) => setFilters((current) => ({ ...current, tripId }))} />
          <FilterInput label="User ID" value={filters.userId ?? ""} onChange={(userId) => setFilters((current) => ({ ...current, userId }))} />
          <Button variant="secondary" onClick={() => setFilters({})}>Clear</Button>
        </div>

        <div className="mt-5 overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-slate-200 text-xs uppercase text-slate-500">
              <tr>
                <th className="py-2 pr-4">Job</th>
                <th className="py-2 pr-4">Type</th>
                <th className="py-2 pr-4">Status</th>
                <th className="py-2 pr-4">Error</th>
                <th className="py-2 pr-4">Trip</th>
                <th className="py-2 pr-4">Scope</th>
                <th className="py-2 pr-4">Created</th>
                <th className="py-2 pr-4">Correlation</th>
                <th className="py-2 pr-4">Actions</th>
              </tr>
            </thead>
            <tbody>
              {(jobs.data?.jobs ?? []).map((job) => (
                <tr key={job.id} className="border-b border-slate-100">
                  <td className="py-3 pr-4 font-mono text-xs">{shortId(job.id)}</td>
                  <td className="py-3 pr-4">{job.jobType}</td>
                  <td className="py-3 pr-4"><StatusPill status={job.status} /></td>
                  <td className="py-3 pr-4">{job.errorCode ?? "-"}</td>
                  <td className="py-3 pr-4 font-mono text-xs">{shortId(job.tripId)}</td>
                  <td className="py-3 pr-4">{job.scope ?? "-"}</td>
                  <td className="py-3 pr-4">{formatOpsDate(job.createdAt)}</td>
                  <td className="py-3 pr-4 font-mono text-xs">{shortId(job.correlationId)}</td>
                  <td className="py-3 pr-4">
                    <JobActions
                      job={job}
                      onView={() => setSelectedJobId(job.id)}
                      onRetry={() => withReason("Retry creates a new queued job.", (reason) => retryMutation.mutate({ jobId: job.id, reason }))}
                      onCancel={() => withReason("Cancel affects this queued job only.", (reason) => cancelMutation.mutate({ jobId: job.id, reason }))}
                      onMarkFailed={() => withReason("Mark failed affects only stale running jobs.", (reason) => markFailedMutation.mutate({ jobId: job.id, reason }))}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {selectedJobId ? (
        <Card>
          <div className="flex items-start justify-between gap-4">
            <h2 className="text-lg font-semibold text-slate-950">Job Details</h2>
            <Button size="sm" variant="ghost" onClick={() => setSelectedJobId(null)}>Close</Button>
          </div>
          {selectedJob.data ? <JobDetails job={selectedJob.data} /> : <p className="mt-3 text-sm text-slate-600">Loading...</p>}
        </Card>
      ) : null}

      <section className="grid gap-6 xl:grid-cols-2">
        <Card>
          <h2 className="text-lg font-semibold text-slate-950">Worker</h2>
          <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
            <Metric label="Healthy" value={worker.data?.healthy ? "yes" : "no"} />
            <Metric label="RabbitMQ" value={worker.data?.rabbitmqConnected ? "connected" : "down"} />
            <Metric label="Postgres" value={worker.data?.dbConnected ? "connected" : "down"} />
            <Metric label="Active jobs" value={String(worker.data?.activeJobs.length ?? 0)} />
          </dl>
        </Card>

        <Card>
          <h2 className="text-lg font-semibold text-slate-950">Queues</h2>
          <div className="mt-4 space-y-3">
            {(queues.data?.queues ?? []).map((queue) => (
              <div key={queue.name} className="rounded-md border border-slate-200 p-3 text-sm">
                <div className="font-medium text-slate-950">{queue.name}</div>
                <div className="mt-1 text-slate-600">
                  ready {queue.messagesReady} · unacked {queue.messagesUnacked} · consumers {queue.consumers}
                </div>
              </div>
            ))}
            {queues.error ? <p className="text-sm text-amber-700">Queue status unavailable.</p> : null}
          </div>
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-2">
        <Card>
          <h2 className="text-lg font-semibold text-slate-950">DLQ Messages</h2>
          <div className="mt-4 space-y-3">
            {(dlq.data?.messages ?? []).map((message) => (
              <div key={message.messageId} className="rounded-md border border-slate-200 p-3 text-sm">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-mono text-xs">{shortId(message.messageId)}</span>
                  <div className="flex gap-2">
                    <Button size="sm" variant="secondary" onClick={() => withReason("Requeue republishes this message to the main queue.", (reason) => requeueMutation.mutate({ messageId: message.messageId, reason }))}>Requeue</Button>
                    <Button size="sm" variant="danger" onClick={() => withReason("Discard removes this DLQ message.", (reason) => discardMutation.mutate({ messageId: message.messageId, reason }))}>Discard</Button>
                  </div>
                </div>
                <div className="mt-2 text-slate-600">
                  {message.jobType ?? "unknown"} · attempts {message.attempts} · {shortId(message.correlationId)}
                </div>
              </div>
            ))}
            {!dlq.data?.messages.length ? <p className="text-sm text-slate-600">No DLQ messages.</p> : null}
          </div>
        </Card>

        <Card>
          <h2 className="text-lg font-semibold text-slate-950">Provider Health</h2>
          <div className="mt-4 space-y-3">
            {(providers.data?.providers ?? []).map((provider) => (
              <div key={provider.name} className="rounded-md border border-slate-200 p-3 text-sm">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <div className="font-medium text-slate-950">{provider.name}</div>
                    <div className="text-slate-600">{provider.activeProvider}</div>
                  </div>
                  <StatusPill status={provider.status} />
                </div>
                <div className="mt-2 text-slate-600">
                  success {provider.recentSuccessCount} · failures {provider.recentFailureCount}
                  {provider.lastErrorCode ? ` · ${provider.lastErrorCode}` : ""}
                </div>
              </div>
            ))}
          </div>
        </Card>
      </section>

      <Card>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">Provider Quotas</h2>
            <p className="mt-1 text-sm text-slate-600">
              Per-provider rate limits and daily quota usage for {providerQuotas.data?.date ?? "today"}.
              {providerQuotas.data && !providerQuotas.data.enabled
                ? " Enforcement is disabled in this environment."
                : ""}
            </p>
          </div>
          <Button
            variant="secondary"
            onClick={() => {
              void queryClient.invalidateQueries({ queryKey: ["ops", "provider-quotas"] });
              void queryClient.invalidateQueries({ queryKey: ["ops", "provider-quota"] });
            }}
          >
            Refresh
          </Button>
        </div>

        {providerQuotas.error ? (
          <p className="mt-4 text-sm text-amber-700">Provider quotas are currently unavailable.</p>
        ) : null}

        <div className="mt-4 grid gap-3 md:grid-cols-2">
          {(providerQuotas.data?.providers ?? []).map((provider) => (
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
          {providerQuotas.data && providerQuotas.data.providers.length === 0 ? (
            <p className="text-sm text-slate-600">No providers configured.</p>
          ) : null}
        </div>

        {selectedQuotaProvider ? (
          <div className="mt-5 rounded-md border border-slate-200 p-4">
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-sm font-semibold text-slate-950">
                {selectedQuotaProvider} — operation breakdown &amp; last 7 days
              </h3>
              <Button size="sm" variant="ghost" onClick={() => setSelectedQuotaProvider(null)}>
                Close
              </Button>
            </div>
            {quotaDetail.data ? (
              <ProviderQuotaDetailView detail={quotaDetail.data} />
            ) : (
              <p className="mt-3 text-sm text-slate-600">Loading...</p>
            )}
          </div>
        ) : null}
      </Card>
    </PageContainer>
  );
}

