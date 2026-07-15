import {
  formatGeneratedAt,
  levelClasses,
  levelLabel,
  scoreBarClass
} from "./health-ui";
import type { TripHealth } from "@/types/trip-health";

export function TripHealthScoreCard({ health }: { health: TripHealth }) {
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Trip Health
          </p>
          <div className="mt-3 flex items-end gap-3">
            <span className="font-newsreader text-[56px] font-semibold leading-none text-cocoa-900">
              {health.score}
            </span>
            <span
              className={`mb-1 inline-flex rounded-full border px-3 py-1 text-[13px] font-semibold ${levelClasses(
                health.level
              )}`}
            >
              {levelLabel[health.level]}
            </span>
          </div>
          <p className="mt-3 max-w-[620px] text-[14px] leading-[1.6] text-cocoa-500">
            {health.summary}
          </p>
        </div>
        <p className="text-[12px] font-medium text-cocoa-400">
          Evaluated {formatGeneratedAt(health.generatedAt)}
        </p>
      </div>
      <div className="mt-5 h-2 overflow-hidden rounded-full bg-sand-200">
        <div
          className={`h-full rounded-full ${scoreBarClass(health.score)}`}
          style={{ width: `${Math.max(0, Math.min(health.score, 100))}%` }}
        />
      </div>
    </section>
  );
}
