import { ReadinessScoreBadge } from "./ReadinessScoreBadge";
import type { GroupReadiness } from "@/types/group-readiness";

type GroupReadinessCardProps = {
  readiness?: GroupReadiness | null;
  loading?: boolean;
};

export function GroupReadinessCard({ readiness, loading = false }: GroupReadinessCardProps) {
  if (loading && !readiness) {
    return (
      <article className="rounded-[18px] border border-sand-300 bg-white p-5">
        <p className="text-[14px] text-cocoa-500">Checking group readiness...</p>
      </article>
    );
  }
  if (!readiness) {
    return null;
  }
  const readyCount = readiness.members.filter((member) => member.level === "ready").length;
  const attentionCount = readiness.members.length - readyCount;
  const topMember = readiness.members.find((member) => member.items.length > 0);

  return (
    <article className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Group readiness
          </p>
          <h3 className="mt-1 text-[18px] font-semibold text-cocoa-900">Who is ready</h3>
        </div>
        <ReadinessScoreBadge level={readiness.level} score={readiness.score} />
      </div>
      <div className="mt-4 h-2 overflow-hidden rounded-full bg-sand-200">
        <div
          className="h-full rounded-full bg-[#3E6B5A]"
          style={{ width: `${Math.max(0, Math.min(readiness.score, 100))}%` }}
        />
      </div>
      <p className="mt-4 text-[14px] leading-[1.55] text-cocoa-700">{readiness.summary}</p>
      <dl className="mt-4 grid grid-cols-3 gap-2">
        <div className="rounded-[12px] bg-sand-50 p-3">
          <dt className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Ready
          </dt>
          <dd className="mt-1 text-[14px] font-semibold text-cocoa-900">
            {readyCount}/{readiness.members.length}
          </dd>
        </div>
        <div className="rounded-[12px] bg-sand-50 p-3">
          <dt className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Attention
          </dt>
          <dd className="mt-1 text-[14px] font-semibold text-cocoa-900">{attentionCount}</dd>
        </div>
        <div className="rounded-[12px] bg-sand-50 p-3">
          <dt className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Actions
          </dt>
          <dd className="mt-1 text-[14px] font-semibold text-cocoa-900">
            {readiness.topActions.length}
          </dd>
        </div>
      </dl>
      {topMember ? (
        <p className="mt-3 text-[13px] leading-[1.5] text-cocoa-500">
          {topMember.displayName} has readiness items waiting.
        </p>
      ) : null}
      <a
        href="#group-readiness"
        className="mt-5 inline-flex h-9 items-center justify-center rounded-full bg-cocoa-900 px-4 text-[13px] font-semibold text-sand-100 transition hover:bg-cocoa-700"
      >
        Open group readiness
      </a>
    </article>
  );
}

