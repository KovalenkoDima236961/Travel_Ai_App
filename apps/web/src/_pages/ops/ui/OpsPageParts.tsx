import type { OpsJob, ProviderQuotaStatus, ProviderQuotaSummary } from "@/entities/ops/model";
import { cn } from "@/shared/lib/cn";
import { formatOpsDate, undefinedValue } from "../model/opsPageModel";
import {
  MICRO_LABEL,
  MONO,
  OPS_INPUT,
  OPS_SELECT,
  SMALL_DANGER_BUTTON,
  SMALL_OUTLINE_BUTTON,
  quotaPillClass,
  statusPillClass
} from "./opsStyles";

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
    <div className="rounded-xl border border-sand-200 p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-[13.5px] font-semibold text-cocoa-900">{provider.provider}</div>
          <div className="mt-0.5 text-[12.5px] text-cocoa-400">{provider.category}</div>
        </div>
        <QuotaStatusPill status={provider.status} />
      </div>
      <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1.5 text-[12px]">
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
        <div className="mt-2 text-[12px] text-[#96682A]">
          Last blocked {formatOpsDate(provider.lastBlockedAt)}
        </div>
      ) : null}
      <div className="mt-3 flex flex-wrap gap-2">
        <button type="button" className={SMALL_OUTLINE_BUTTON} onClick={onToggle}>
          {expanded ? "Hide details" : "View details"}
        </button>
        {resetAllowed ? (
          <button
            type="button"
            className={SMALL_DANGER_BUTTON}
            disabled={resetPending}
            onClick={onReset}
          >
            Reset (dev)
          </button>
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
    <div className="mt-4 grid gap-5 lg:grid-cols-2">
      <div>
        <div className={MICRO_LABEL}>Operations</div>
        <table className="mt-2 min-w-full text-left text-[12px]">
          <thead className="border-b border-sand-200 text-cocoa-400">
            <tr>
              <th className="py-1.5 pr-3 font-medium">Operation</th>
              <th className="py-1.5 pr-3 font-medium">Used</th>
              <th className="py-1.5 pr-3 font-medium">Blocked</th>
              <th className="py-1.5 pr-3 font-medium">Fallback</th>
            </tr>
          </thead>
          <tbody>
            {detail.provider.operations.map((op) => (
              <tr key={op.operation} className="border-b border-sand-200">
                <td className={cn("py-1.5 pr-3", MONO, "text-cocoa-700")}>{op.operation}</td>
                <td className="py-1.5 pr-3 text-cocoa-700">{op.usedToday}</td>
                <td className="py-1.5 pr-3 text-cocoa-700">{op.blockedToday}</td>
                <td className="py-1.5 pr-3 text-cocoa-700">{op.fallbackToday}</td>
              </tr>
            ))}
            {detail.provider.operations.length === 0 ? (
              <tr>
                <td className="py-2 text-cocoa-400" colSpan={4}>
                  No usage recorded today.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      <div>
        <div className={MICRO_LABEL}>Last 7 days</div>
        <div className="mt-2 space-y-1.5 text-[12px] text-cocoa-500">
          {detail.history.length === 0 ? (
            <div className="text-cocoa-400">No usage in the last 7 days.</div>
          ) : (
            detail.history.map((day) => (
              <div key={day.date} className="flex justify-between gap-3">
                <span className={cn(MONO, "text-cocoa-700")}>{day.date}</span>
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
      <dt className="text-cocoa-400">{label}</dt>
      <dd className="text-right font-semibold text-cocoa-700">{value}</dd>
    </>
  );
}

function QuotaStatusPill({ status }: { status: ProviderQuotaStatus }) {
  return <span className={quotaPillClass(status)}>{status.replace(/_/g, " ")}</span>;
}

export function SummaryCard({
  label,
  value,
  valueClassName
}: {
  label: string;
  value: number;
  valueClassName?: string;
}) {
  return (
    <div className="rounded-2xl border border-sand-300 bg-white px-5 py-[18px]">
      <p className={MICRO_LABEL}>{label}</p>
      <p
        className={cn(
          "mt-2.5 font-newsreader text-[30px] font-semibold text-cocoa-900",
          valueClassName
        )}
      >
        {value}
      </p>
    </div>
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
    <label className="w-40 text-[13px]">
      <span className="mb-1.5 block font-medium text-cocoa-500">{label}</span>
      <select
        className={OPS_SELECT}
        value={value}
        onChange={(event) => onChange(event.target.value || undefinedValue())}
      >
        {options.map((option) => (
          <option key={option} value={option}>
            {option || "Any"}
          </option>
        ))}
      </select>
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
    <label className="w-44 text-[13px]">
      <span className="mb-1.5 block font-medium text-cocoa-500">{label}</span>
      <input className={OPS_INPUT} value={value} onChange={(event) => onChange(event.target.value)} />
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
    <div className="flex flex-wrap justify-end gap-2">
      <button type="button" className={SMALL_OUTLINE_BUTTON} onClick={onView}>
        View
      </button>
      {job.canRetry ? (
        <button type="button" className={SMALL_OUTLINE_BUTTON} onClick={onRetry}>
          Retry
        </button>
      ) : null}
      {job.canCancel ? (
        <button type="button" className={SMALL_OUTLINE_BUTTON} onClick={onCancel}>
          Cancel
        </button>
      ) : null}
      {job.canMarkFailed ? (
        <button type="button" className={SMALL_DANGER_BUTTON} onClick={onMarkFailed}>
          Mark failed
        </button>
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
    <div className="mt-4 grid gap-5 lg:grid-cols-2">
      <dl className="space-y-2 text-[13px]">
        {rows.map(([label, value]) => (
          <div key={label} className="grid grid-cols-[9rem_minmax(0,1fr)] gap-3">
            <dt className="text-cocoa-400">{label}</dt>
            <dd className={cn("break-words text-[12px] text-cocoa-700", MONO)}>{value}</dd>
          </div>
        ))}
      </dl>
      <div className="rounded-xl border border-sand-200 bg-sand-50 p-4">
        <div className="text-[13.5px] font-semibold text-cocoa-900">Payload summary</div>
        <pre className={cn("mt-2 whitespace-pre-wrap break-words text-[12px] text-cocoa-500", MONO)}>
          {JSON.stringify(job.payloadSummary ?? {}, null, 2)}
        </pre>
      </div>
    </div>
  );
}

export function StatusTile({
  label,
  value,
  tone = "neutral"
}: {
  label: string;
  value: string;
  tone?: "ok" | "bad" | "neutral";
}) {
  const color =
    tone === "ok" ? "text-[#2F7A57]" : tone === "bad" ? "text-[#B3402E]" : "text-cocoa-900";
  return (
    <div className="rounded-xl bg-sand-50 px-4 py-3.5">
      <p className="text-[12px] text-[#A08D78]">{label}</p>
      <p className={cn("mt-1.5 text-[14px] font-semibold", color)}>{value}</p>
    </div>
  );
}

export function StatusPill({ status }: { status?: string }) {
  return <span className={statusPillClass(status)}>{status ?? "unknown"}</span>;
}
