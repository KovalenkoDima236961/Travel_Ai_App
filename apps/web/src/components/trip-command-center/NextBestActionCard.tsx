import { severityClasses } from "./status-ui";
import type { NextBestAction } from "@/types/trip-command-center";

type NextBestActionCardProps = {
  action: NextBestAction;
};

export function NextBestActionCard({ action }: NextBestActionCardProps) {
  return (
    <section className="rounded-[20px] border border-sand-300 bg-white p-6">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Next best action
            </p>
            <span
              className={`rounded-full border px-2.5 py-1 text-[12px] font-semibold ${severityClasses(
                action.severity
              )}`}
            >
              {action.severity}
            </span>
            {action.viewOnly ? (
              <span className="rounded-full border border-[#D6DEE8] bg-[#F4F7FA] px-2.5 py-1 text-[12px] font-semibold text-[#536171]">
                View only
              </span>
            ) : null}
          </div>
          <h2 className="mt-3 font-newsreader text-[30px] font-semibold tracking-[-0.01em] text-cocoa-900">
            {action.title}
          </h2>
          <p className="mt-2 max-w-[720px] text-[15px] leading-[1.65] text-cocoa-600">
            {action.description}
          </p>
          <p className="mt-3 text-[13px] leading-[1.5] text-cocoa-400">
            Why this matters: {action.reason}
          </p>
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          <a
            href={action.href}
            className="inline-flex h-10 items-center justify-center rounded-full bg-clay px-5 text-[14px] font-semibold text-sand-100 shadow-[0_8px_20px_rgba(192,91,59,0.18)] transition hover:bg-clay-dark"
          >
            {action.actionLabel}
          </a>
          <a
            href="#health"
            className="inline-flex h-10 items-center justify-center rounded-full border border-sand-400 bg-white px-5 text-[14px] font-semibold text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
          >
            View all issues
          </a>
        </div>
      </div>
    </section>
  );
}
