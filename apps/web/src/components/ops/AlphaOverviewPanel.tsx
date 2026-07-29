"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import type {
  OpsJobSummary,
  ProviderQuotaSummary,
  ProviderStatus,
  QueueStatus,
  WorkerStatus
} from "@/entities/ops/model";
import { getOpsFeatureFlags, opsKeys, type OpsFeatureFlag } from "@/lib/api/ops";
import { cn } from "@/shared/lib/cn";
import { CARD, CARD_HEADING, MICRO_LABEL, MONO, SMALL_OUTLINE_BUTTON } from "@/_pages/ops/ui/opsStyles";

const alphaDangerFlags = new Set([
  "real_providers_enabled",
  "calendar_sync_enabled",
  "availability_search_enabled",
  "transport_search_enabled",
  "receipt_ocr_enabled",
  "workspace_approvals_enabled",
  "policy_repair_enabled",
  "web_push_enabled",
  "offline_mode_enabled",
  "route_alternatives_enabled",
  "template_adaptation_enabled",
  "ai_fine_tuning_experiments_enabled",
  "ai_adapter_inference_enabled",
  "ai_adapter_staging_enabled",
  "ai_shadow_evaluation_enabled",
  "ai_candidate_percentage_rollout_enabled"
]);

const alphaControlFlags = [
  "ai_generation_enabled",
  "ai_repair_enabled",
  "public_sharing_enabled",
  "data_exports_enabled",
  "real_providers_enabled",
  "ops_dashboard_enabled"
];

export function AlphaOverviewPanel({
  summary,
  worker,
  queues,
  providers,
  quotas
}: {
  summary?: OpsJobSummary;
  worker?: WorkerStatus;
  queues: QueueStatus[];
  providers: ProviderStatus[];
  quotas: ProviderQuotaSummary[];
}) {
  const flags = useQuery({
    queryKey: opsKeys.featureFlags,
    queryFn: getOpsFeatureFlags,
    staleTime: 30_000
  });

  const flagRows = flags.data?.flags ?? [];
  const dangerousEnabled = flagRows.filter((flag) => alphaDangerFlags.has(flag.key) && flag.value);
  const failedJobs = summary?.countsByStatus.failed ?? 0;
  const queuedJobs = summary?.countsByStatus.queued ?? 0;
  const staleJobs = summary?.staleRunningCount ?? 0;
  const unhealthyProviders = providers.filter((provider) => ["degraded", "down"].includes(provider.status));
  const blockedQuotas = quotas.filter((quota) => ["quota_exceeded", "rate_limited_recently"].includes(quota.status));
  const queueBacklog = queues.reduce((total, queue) => total + queue.messagesReady + queue.messagesUnacked, 0);
  const readiness =
    dangerousEnabled.length > 0 || worker?.healthy === false || staleJobs > 0
      ? "Blocked"
      : failedJobs > 0 || unhealthyProviders.length > 0 || blockedQuotas.length > 0
        ? "Investigate"
        : "Watching";

  return (
    <section className={cn(CARD, "mt-7")} aria-labelledby="alpha-overview-title">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 id="alpha-overview-title" className={CARD_HEADING}>Alpha overview</h2>
        </div>
        <div className="flex flex-wrap gap-2">
          <Link href="/ops/ai-generations" className={SMALL_OUTLINE_BUTTON}>AI traces</Link>
          <a href="#feature-flags" className={SMALL_OUTLINE_BUTTON}>Feature flags</a>
        </div>
      </div>

      <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <AlphaTile label="Readiness signal" value={readiness} tone={readiness === "Blocked" ? "bad" : readiness === "Investigate" ? "warn" : "ok"} />
        <AlphaTile label="Release version" value={process.env.NEXT_PUBLIC_APP_VERSION || "unknown"} mono />
        <AlphaTile label="Worker" value={worker ? (worker.healthy ? "Healthy" : "Down") : "Unknown"} tone={worker?.healthy === false ? "bad" : "ok"} />
        <AlphaTile label="Queue backlog" value={String(queueBacklog)} tone={queueBacklog > 20 ? "warn" : "ok"} />
        <AlphaTile label="Queued jobs" value={String(queuedJobs)} tone={queuedJobs > 20 ? "warn" : "ok"} />
        <AlphaTile label="Failed jobs" value={String(failedJobs)} tone={failedJobs > 0 ? "warn" : "ok"} />
        <AlphaTile label="Stale jobs" value={String(staleJobs)} tone={staleJobs > 0 ? "bad" : "ok"} />
        <AlphaTile label="Provider issues" value={String(unhealthyProviders.length + blockedQuotas.length)} tone={unhealthyProviders.length + blockedQuotas.length > 0 ? "warn" : "ok"} />
      </div>

      <div className="mt-5 grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <div>
          <div className={MICRO_LABEL}>Kill switches</div>
          <div className="mt-2 flex flex-wrap gap-2">
            {alphaControlFlags.map((key) => {
              const flag = flagRows.find((item) => item.key === key);
              return <FlagPill flag={flag} key={key} label={key} />;
            })}
          </div>
        </div>
        <div>
          <div className={MICRO_LABEL}>Latest incidents</div>
          <div className="mt-2 space-y-1.5 text-[12.5px] text-cocoa-500">
            {(summary?.recentFailures ?? []).slice(0, 3).map((failure) => (
              <div key={failure.jobId} className="flex justify-between gap-3">
                <span>{failure.errorCode}</span>
                <span className={cn(MONO, "text-cocoa-400")}>{failure.jobId.slice(0, 8)}</span>
              </div>
            ))}
            {(summary?.recentFailures ?? []).length === 0 ? <span className="text-cocoa-400">No recent job failures.</span> : null}
          </div>
        </div>
      </div>

      {dangerousEnabled.length > 0 ? (
        <div className="mt-5 rounded-lg border border-[#E5C3B6] bg-[#FBF0EB] p-3 text-[13px] text-[#B3402E]">
          Alpha scope drift: {dangerousEnabled.map((flag) => flag.key).join(", ")} enabled.
        </div>
      ) : null}
    </section>
  );
}

function AlphaTile({
  label,
  value,
  tone = "neutral",
  mono = false
}: {
  label: string;
  value: string;
  tone?: "ok" | "warn" | "bad" | "neutral";
  mono?: boolean;
}) {
  const toneClass =
    tone === "bad"
      ? "text-[#B3402E]"
      : tone === "warn"
        ? "text-[#96682A]"
        : tone === "ok"
          ? "text-[#2F7A57]"
          : "text-cocoa-900";
  return (
    <div className="rounded-lg border border-sand-200 bg-[#FFFDFA] p-4">
      <div className={MICRO_LABEL}>{label}</div>
      <div className={cn("mt-2 text-[20px] font-semibold", toneClass, mono && MONO)}>{value}</div>
    </div>
  );
}

function FlagPill({ flag, label }: { flag?: OpsFeatureFlag; label: string }) {
  const active = flag?.value === true;
  const risky = alphaDangerFlags.has(label) && active;
  return (
    <span
      className={cn(
        "rounded-full px-2.5 py-1 text-[11.5px] font-semibold",
        risky
          ? "bg-[#FBF0EB] text-[#B3402E]"
          : active
            ? "bg-[#EDF3EA] text-[#2F7A57]"
            : "bg-[#F4EDE4] text-[#8A7A6A]"
      )}
    >
      {label}: {flag ? (active ? "on" : "off") : "unknown"}
    </span>
  );
}
