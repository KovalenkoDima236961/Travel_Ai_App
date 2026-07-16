import { severityClasses } from "./status-ui";
import type { NextBestAction } from "@/types/trip-command-center";

export function TopFixesPanel({ fixes }: { fixes: NextBestAction[] }) {
  if (fixes.length === 0) {
    return (
      <section className="rounded-[18px] border border-[#CFE3D3] bg-[#EFF7F1] p-5">
        <h2 className="font-newsreader text-[22px] font-semibold text-[#2F5C3C]">
          No urgent actions
        </h2>
        <p className="mt-2 text-[14px] leading-[1.6] text-[#42684C]">
          Trip Health is not reporting high-priority fixes right now.
        </p>
      </section>
    );
  }

  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex items-center justify-between gap-3">
        <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">Top fixes</h2>
        <a href="#health" className="text-[13px] font-semibold text-clay hover:text-clay-dark">
          View all issues
        </a>
      </div>
      <div className="mt-4 flex flex-col gap-3">
        {fixes.map((fix, index) => (
          <a
            key={fix.id}
            href={fix.href}
            className="flex items-center justify-between gap-4 rounded-[14px] border border-sand-200 bg-sand-50 p-4 transition hover:border-sand-400 hover:bg-white"
          >
            <span className="flex min-w-0 items-center gap-3">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-cocoa-900 text-[12px] font-semibold text-sand-100">
                {index + 1}
              </span>
              <span className="min-w-0">
                <span className="block truncate text-[14px] font-semibold text-cocoa-800">
                  {fix.title}
                </span>
                <span className="mt-1 inline-flex gap-2">
                  <span
                    className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${severityClasses(
                      fix.severity
                    )}`}
                  >
                    {fix.severity}
                  </span>
                  <span className="rounded-full border border-sand-300 bg-white px-2 py-0.5 text-[11px] font-semibold text-cocoa-500">
                    {fix.category}
                  </span>
                </span>
              </span>
            </span>
            <span aria-hidden className="text-[18px] leading-none text-[#A08D78]">
              -&gt;
            </span>
          </a>
        ))}
      </div>
    </section>
  );
}
