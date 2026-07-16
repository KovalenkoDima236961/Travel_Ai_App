import type { NavigationGroup } from "@/types/trip-command-center";

export function QuickNavigationGrid({ groups }: { groups: NavigationGroup[] }) {
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
        Quick navigation
      </h2>
      <div className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-5">
        {groups.map((group) => (
          <div key={group.id}>
            <h3 className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              {group.label}
            </h3>
            <div className="mt-2 flex flex-col gap-1.5">
              {group.items.map((item) => (
                <a
                  key={`${group.id}:${item.id}:${item.href}`}
                  href={item.href}
                  className="flex min-h-9 items-center justify-between gap-2 rounded-xl px-3 py-2 text-[13px] font-semibold text-cocoa-600 transition hover:bg-sand-100 hover:text-cocoa-900"
                >
                  <span className="truncate">{item.label}</span>
                  {item.badge ? (
                    <span className="shrink-0 rounded-full bg-[#FBF0EB] px-2 py-0.5 text-[11px] text-[#A93624]">
                      {item.badge}
                    </span>
                  ) : null}
                </a>
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
