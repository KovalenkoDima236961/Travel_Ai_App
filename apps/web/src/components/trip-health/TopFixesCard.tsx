import type { TripHealthTopFix } from "@/types/trip-health";

export function TopFixesCard({ fixes }: { fixes: TripHealthTopFix[] }) {
  if (fixes.length === 0) {
    return (
      <section className="rounded-[18px] border border-[#CFE3D3] bg-[#EFF7F1] p-5">
        <h2 className="font-newsreader text-[22px] font-semibold text-[#2F5C3C]">
          This trip looks ready
        </h2>
        <p className="mt-2 text-[14px] leading-[1.6] text-[#42684C]">
          No high-priority fixes are open right now.
        </p>
      </section>
    );
  }
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
        Top Fixes
      </h2>
      <div className="mt-4 flex flex-col gap-3">
        {fixes.map((fix, index) => (
          <a
            key={fix.issueId}
            href={fix.href}
            className="flex items-center justify-between gap-4 rounded-[14px] border border-sand-200 bg-sand-50 p-4 transition hover:border-sand-400 hover:bg-white"
          >
            <span className="flex min-w-0 items-center gap-3">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-cocoa-900 text-[12px] font-semibold text-sand-100">
                {index + 1}
              </span>
              <span className="truncate text-[14px] font-semibold text-cocoa-800">
                {fix.label}
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
