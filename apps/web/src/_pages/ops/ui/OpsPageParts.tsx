import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import type { OpsJob, ProviderQuotaStatus, ProviderQuotaSummary } from "@/entities/ops/model";
import { formatOpsDate, shortId, undefinedValue } from "../model/opsPageModel";

export function ProviderQuotaCard({
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
          value={provider.dailyQuota > 0 ? String(provider.remainingToday) : "-"}
        />
        <QuotaMetric
          label="Minute limit"
          value={provider.rateLimitPerMinute > 0 ? String(provider.rateLimitPerMinute) : "unlimited"}
        />
        <QuotaMetric label="Blocked today" value={String(provider.blockedToday)} />
        <QuotaMetric label="Fallback today" value={String(provider.fallbackToday)} />
      </dl>
      {provider.lastBlockedAt ? (
        <div className="mt-2 text-xs text-amber-700">
          Last blocked {formatOpsDate(provider.lastBlockedAt)}
        </div>
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

export function ProviderQuotaDetailView({
  detail
}: {
  detail: {
    provider: ProviderQuotaSummary;
    history: { date: string; usedCount: number; blockedCount: number; fallbackCount: number }[];
  };
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

export function SummaryCard({ label, value }: { label: string; value: number }) {
  return (
    <Card className="p-4">
      <div className="text-sm text-slate-500">{label}</div>
      <div className="mt-2 text-2xl font-semibold text-slate-950">{value}</div>
    </Card>
  );
}

export function FilterSelect({
  label,
  value,
  options,
  onChange
}: {
  label: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
}) {
  return (
    <label className="w-44 text-sm">
      <span className="mb-1 block text-slate-600">{label}</span>
      <Select value={value} onChange={(event) => onChange(event.target.value || undefinedValue())}>
        {options.map((option) => (
          <option key={option} value={option}>
            {option || "Any"}
          </option>
        ))}
      </Select>
    </label>
  );
}

export function FilterInput({
  label,
  value,
  onChange
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="w-48 text-sm">
      <span className="mb-1 block text-slate-600">{label}</span>
      <Input value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

export function JobActions({
  job,
  onView,
  onRetry,
  onCancel,
  onMarkFailed
}: {
  job: OpsJob;
  onView: () => void;
  onRetry: () => void;
  onCancel: () => void;
  onMarkFailed: () => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      <Button size="sm" variant="secondary" onClick={onView}>
        View
      </Button>
      {job.canRetry ? (
        <Button size="sm" variant="secondary" onClick={onRetry}>
          Retry
        </Button>
      ) : null}
      {job.canCancel ? (
        <Button size="sm" variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
      ) : null}
      {job.canMarkFailed ? (
        <Button size="sm" variant="danger" onClick={onMarkFailed}>
          Mark failed
        </Button>
      ) : null}
    </div>
  );
}

export function JobDetails({ job }: { job: OpsJob }) {
  const rows = [
    ["Job ID", job.id],
    ["Trip ID", job.tripId],
    ["Workspace ID", job.workspaceId ?? "-"],
    ["Scope", job.scope ?? "-"],
    ["Requested by", job.requestedByUserId],
    ["Status", job.status],
    ["Expected revision", String(job.expectedItineraryRevision)],
    ["Result revision", job.resultItineraryRevision ?? "-"],
    ["Error", job.errorCode ?? "-"],
    ["Message", job.errorMessage ?? "-"],
    ["Request ID", job.requestId ?? "-"],
    ["Correlation ID", job.correlationId ?? "-"],
    ["Created", formatOpsDate(job.createdAt)],
    ["Updated", formatOpsDate(job.updatedAt)]
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

export function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-slate-500">{label}</dt>
      <dd className="mt-1 font-medium text-slate-950">{value}</dd>
    </div>
  );
}

export function StatusPill({ status }: { status?: string }) {
  const color =
    status === "failed" || status === "down"
      ? "bg-red-50 text-red-700"
      : status === "running" || status === "degraded"
        ? "bg-amber-50 text-amber-700"
        : status === "completed" || status === "healthy"
          ? "bg-emerald-50 text-emerald-700"
          : "bg-slate-100 text-slate-700";
  return (
    <span className={`inline-flex rounded-md px-2 py-1 text-xs font-medium ${color}`}>
      {status ?? "unknown"}
    </span>
  );
}
