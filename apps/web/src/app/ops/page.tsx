"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { ApiError } from "@/lib/api/client";
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
import type { OpsJob, ProviderQuotaStatus, ProviderQuotaSummary } from "@/types/ops";

const refreshInterval = 20_000;

export default function OpsPage() {
  return (
    <ProtectedRoute>
      <OpsDashboard />
    </ProtectedRoute>
  );
}

function OpsDashboard() {
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<OpsJobFilters>({});
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);

  const summary = useQuery({
    queryKey: opsKeys.summary,
    queryFn: getOpsJobSummary,
    refetchInterval: refreshInterval
  });
  const jobs = useQuery({
    queryKey: opsKeys.jobs(filters),
    queryFn: () => getOpsJobs(filters),
    refetchInterval: refreshInterval
  });
  const selectedJob = useQuery({
    queryKey: opsKeys.job(selectedJobId),
    queryFn: () => getOpsJob(selectedJobId ?? ""),
    enabled: Boolean(selectedJobId),
    refetchInterval: refreshInterval
  });
  const worker = useQuery({
    queryKey: opsKeys.worker,
    queryFn: getWorkerStatus,
    refetchInterval: refreshInterval
  });
  const queues = useQuery({
    queryKey: opsKeys.queues,
    queryFn: getQueuesStatus,
    refetchInterval: refreshInterval
  });
  const dlq = useQuery({
    queryKey: opsKeys.dlq,
    queryFn: () => getDlqMessages(20),
    refetchInterval: refreshInterval
  });
  const providers = useQuery({
    queryKey: opsKeys.providers,
    queryFn: getProviderStatus,
    refetchInterval: refreshInterval
  });
  const providerQuotas = useQuery({
    queryKey: opsKeys.providerQuotas(),
    queryFn: () => getProviderQuotas(),
    refetchInterval: refreshInterval
  });
  const [selectedQuotaProvider, setSelectedQuotaProvider] = useState<string | null>(null);
  const quotaDetail = useQuery({
    queryKey: opsKeys.providerQuotaDetail(selectedQuotaProvider),
    queryFn: () => getProviderQuotaDetail(selectedQuotaProvider ?? ""),
    enabled: Boolean(selectedQuotaProvider),
    refetchInterval: refreshInterval
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
                  <td className="py-3 pr-4">{formatDate(job.createdAt)}</td>
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

function ProviderQuotaCard({
  provider,
  expanded,
  resetAllowed,
  resetPending,
  onToggle,
  onReset
}: {
  provider: ProviderQuotaSummary;
  expanded: boolean;
  resetAllowed: boolean;
  resetPending: boolean;
  onToggle: () => void;
  onReset: () => void;
}) {
  return (
    <div className="rounded-md border border-slate-200 p-3 text-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="font-medium text-slate-950">{provider.provider}</div>
          <div className="text-slate-600">{provider.category}</div>
        </div>
        <QuotaStatusPill status={provider.status} />
      </div>
      <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-600">
        <QuotaMetric label="Requests today" value={String(provider.usedToday)} />
        <QuotaMetric
          label="Daily quota"
          value={provider.dailyQuota > 0 ? String(provider.dailyQuota) : "unlimited"}
        />
        <QuotaMetric
          label="Remaining"
          value={provider.dailyQuota > 0 ? String(provider.remainingToday) : "—"}
        />
        <QuotaMetric
          label="Minute limit"
          value={provider.rateLimitPerMinute > 0 ? String(provider.rateLimitPerMinute) : "unlimited"}
        />
        <QuotaMetric label="Blocked today" value={String(provider.blockedToday)} />
        <QuotaMetric label="Fallback today" value={String(provider.fallbackToday)} />
      </dl>
      {provider.lastBlockedAt ? (
        <div className="mt-2 text-xs text-amber-700">Last blocked {formatDate(provider.lastBlockedAt)}</div>
      ) : null}
      <div className="mt-3 flex flex-wrap gap-2">
        <Button size="sm" variant="secondary" onClick={onToggle}>
          {expanded ? "Hide details" : "View details"}
        </Button>
        {resetAllowed ? (
          <Button size="sm" variant="danger" disabled={resetPending} onClick={onReset}>
            Reset (dev)
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function ProviderQuotaDetailView({
  detail
}: {
  detail: { provider: ProviderQuotaSummary; history: { date: string; usedCount: number; blockedCount: number; fallbackCount: number }[] };
}) {
  return (
    <div className="mt-4 grid gap-4 lg:grid-cols-2">
      <div>
        <div className="text-xs font-semibold uppercase text-slate-500">Operations</div>
        <table className="mt-2 min-w-full text-left text-xs">
          <thead className="border-b border-slate-200 text-slate-500">
            <tr>
              <th className="py-1 pr-3">Operation</th>
              <th className="py-1 pr-3">Used</th>
              <th className="py-1 pr-3">Blocked</th>
              <th className="py-1 pr-3">Fallback</th>
            </tr>
          </thead>
          <tbody>
            {detail.provider.operations.map((op) => (
              <tr key={op.operation} className="border-b border-slate-100">
                <td className="py-1 pr-3 font-mono">{op.operation}</td>
                <td className="py-1 pr-3">{op.usedToday}</td>
                <td className="py-1 pr-3">{op.blockedToday}</td>
                <td className="py-1 pr-3">{op.fallbackToday}</td>
              </tr>
            ))}
            {detail.provider.operations.length === 0 ? (
              <tr>
                <td className="py-2 text-slate-500" colSpan={4}>
                  No usage recorded today.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      <div>
        <div className="text-xs font-semibold uppercase text-slate-500">Last 7 days</div>
        <div className="mt-2 space-y-1 text-xs text-slate-600">
          {detail.history.length === 0 ? (
            <div className="text-slate-500">No usage in the last 7 days.</div>
          ) : (
            detail.history.map((day) => (
              <div key={day.date} className="flex justify-between gap-3">
                <span className="font-mono">{day.date}</span>
                <span>
                  used {day.usedCount} · blocked {day.blockedCount} · fallback {day.fallbackCount}
                </span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function QuotaMetric({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt className="text-slate-500">{label}</dt>
      <dd className="text-right font-medium text-slate-800">{value}</dd>
    </>
  );
}

function QuotaStatusPill({ status }: { status: ProviderQuotaStatus }) {
  const color =
    status === "quota_exceeded"
      ? "bg-red-50 text-red-700"
      : status === "rate_limited_recently" || status === "nearing_quota"
        ? "bg-amber-50 text-amber-700"
        : status === "healthy"
          ? "bg-emerald-50 text-emerald-700"
          : "bg-slate-100 text-slate-700";
  return (
    <span className={`inline-flex rounded-md px-2 py-1 text-xs font-medium ${color}`}>
      {status.replace(/_/g, " ")}
    </span>
  );
}

function SummaryCard({ label, value }: { label: string; value: number }) {
  return (
    <Card className="p-4">
      <div className="text-sm text-slate-500">{label}</div>
      <div className="mt-2 text-2xl font-semibold text-slate-950">{value}</div>
    </Card>
  );
}

function FilterSelect({ label, value, options, onChange }: { label: string; value: string; options: string[]; onChange: (value: string) => void }) {
  return (
    <label className="w-44 text-sm">
      <span className="mb-1 block text-slate-600">{label}</span>
      <Select value={value} onChange={(event) => onChange(event.target.value || undefinedValue())}>
        {options.map((option) => (
          <option key={option} value={option}>{option || "Any"}</option>
        ))}
      </Select>
    </label>
  );
}

function FilterInput({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="w-48 text-sm">
      <span className="mb-1 block text-slate-600">{label}</span>
      <Input value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function JobActions({ job, onView, onRetry, onCancel, onMarkFailed }: { job: OpsJob; onView: () => void; onRetry: () => void; onCancel: () => void; onMarkFailed: () => void }) {
  return (
    <div className="flex flex-wrap gap-2">
      <Button size="sm" variant="secondary" onClick={onView}>View</Button>
      {job.canRetry ? <Button size="sm" variant="secondary" onClick={onRetry}>Retry</Button> : null}
      {job.canCancel ? <Button size="sm" variant="secondary" onClick={onCancel}>Cancel</Button> : null}
      {job.canMarkFailed ? <Button size="sm" variant="danger" onClick={onMarkFailed}>Mark failed</Button> : null}
    </div>
  );
}

function JobDetails({ job }: { job: OpsJob }) {
  const rows = [
    ["Job ID", job.id],
    ["Trip ID", job.tripId],
    ["Requested by", job.requestedByUserId],
    ["Status", job.status],
    ["Expected revision", String(job.expectedItineraryRevision)],
    ["Result revision", job.resultItineraryRevision ?? "-"],
    ["Error", job.errorCode ?? "-"],
    ["Message", job.errorMessage ?? "-"],
    ["Request ID", job.requestId ?? "-"],
    ["Correlation ID", job.correlationId ?? "-"],
    ["Created", formatDate(job.createdAt)],
    ["Updated", formatDate(job.updatedAt)]
  ];
  return (
    <div className="mt-4 grid gap-4 lg:grid-cols-2">
      <dl className="space-y-2 text-sm">
        {rows.map(([label, value]) => (
          <div key={label} className="grid grid-cols-[9rem_minmax(0,1fr)] gap-3">
            <dt className="text-slate-500">{label}</dt>
            <dd className="break-words font-mono text-xs text-slate-800">{value}</dd>
          </div>
        ))}
      </dl>
      <div className="rounded-md border border-slate-200 p-3 text-sm">
        <div className="font-medium text-slate-950">Payload Summary</div>
        <pre className="mt-2 whitespace-pre-wrap break-words text-xs text-slate-700">
          {JSON.stringify(job.payloadSummary ?? {}, null, 2)}
        </pre>
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-slate-500">{label}</dt>
      <dd className="mt-1 font-medium text-slate-950">{value}</dd>
    </div>
  );
}

function StatusPill({ status }: { status?: string }) {
  const color = status === "failed" || status === "down"
    ? "bg-red-50 text-red-700"
    : status === "running" || status === "degraded"
      ? "bg-amber-50 text-amber-700"
      : status === "completed" || status === "healthy"
        ? "bg-emerald-50 text-emerald-700"
        : "bg-slate-100 text-slate-700";
  return <span className={`inline-flex rounded-md px-2 py-1 text-xs font-medium ${color}`}>{status ?? "unknown"}</span>;
}

function withReason(message: string, action: (reason: string) => void) {
  const reason = window.prompt(`${message}\n\nReason:`);
  if (reason?.trim()) {
    action(reason.trim());
  }
}

function invalidateOps(queryClient: ReturnType<typeof useQueryClient>) {
  return queryClient.invalidateQueries({ queryKey: ["ops"] });
}

function shortId(value?: string | null) {
  if (!value) {
    return "-";
  }
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function formatDate(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

function undefinedValue() {
  return "";
}
