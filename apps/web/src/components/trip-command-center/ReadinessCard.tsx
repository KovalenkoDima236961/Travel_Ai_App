import { readinessStatusClasses, readinessStatusLabel } from "./status-ui";
import type { ReadinessCard as ReadinessCardModel } from "@/types/trip-command-center";

type ReadinessCardProps = {
  card: ReadinessCardModel;
};

export function ReadinessCard({ card }: ReadinessCardProps) {
  return (
    <article className="flex min-h-[220px] flex-col rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Readiness
          </p>
          <h3 className="mt-1 text-[17px] font-semibold text-cocoa-900">{card.title}</h3>
        </div>
        <span
          className={`shrink-0 rounded-full border px-2.5 py-1 text-[12px] font-semibold ${readinessStatusClasses(
            card.status
          )}`}
        >
          {readinessStatusLabel[card.status]}
        </span>
      </div>

      {card.score != null ? (
        <div className="mt-4 h-2 overflow-hidden rounded-full bg-sand-200">
          <div
            className="h-full rounded-full bg-[#3E6B5A]"
            style={{ width: `${Math.max(0, Math.min(card.score, 100))}%` }}
          />
        </div>
      ) : null}

      <p className="mt-4 text-[14px] leading-[1.55] text-cocoa-700">{card.summary}</p>
      {card.detail ? (
        <p className="mt-2 text-[13px] leading-[1.5] text-cocoa-400">{card.detail}</p>
      ) : null}

      {card.metrics.length > 0 ? (
        <dl className="mt-4 grid grid-cols-3 gap-2">
          {card.metrics.slice(0, 3).map((metric) => (
            <div key={metric.label} className="rounded-[12px] bg-sand-50 p-3">
              <dt className="truncate text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
                {metric.label}
              </dt>
              <dd className="mt-1 truncate text-[14px] font-semibold text-cocoa-900">
                {metric.value}
              </dd>
            </div>
          ))}
        </dl>
      ) : null}

      <div className="mt-auto flex flex-wrap gap-2 pt-5">
        {card.primaryAction ? (
          <a
            href={card.primaryAction.href}
            className="inline-flex h-9 items-center justify-center rounded-full bg-cocoa-900 px-4 text-[13px] font-semibold text-sand-100 transition hover:bg-cocoa-700"
          >
            {card.primaryAction.label}
          </a>
        ) : null}
        {card.secondaryAction ? (
          <a
            href={card.secondaryAction.href}
            className="inline-flex h-9 items-center justify-center rounded-full border border-sand-400 bg-white px-4 text-[13px] font-semibold text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
          >
            {card.secondaryAction.label}
          </a>
        ) : null}
      </div>
    </article>
  );
}
