import type { NavigationGroup } from "@/types/trip-command-center";
import { normalizeTripWorkspaceHref } from "@/lib/trip-workspace/navigation";

export function QuickNavigationGrid({ groups }: { groups: NavigationGroup[] }) {
  const quickItems = groups
    .flatMap((group) => group.items)
    .filter((item) => ["itinerary", "budget", "collaborators", "checklist"].includes(item.id))
    .filter((item, index, items) => items.findIndex((candidate) => candidate.id === item.id) === index)
    .slice(0, 4);
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
        Quick navigation
      </h2>
      <div className="mt-4 grid gap-2 sm:grid-cols-2">
        {quickItems.map((item) => (
          <a
            key={`${item.id}:${item.href}`}
            href={normalizeTripWorkspaceHref(item.href)}
            className="flex min-h-11 items-center justify-between gap-2 rounded-xl border border-sand-200 bg-sand-50 px-3 py-2 text-[13px] font-semibold text-cocoa-600 transition hover:border-sand-400 hover:bg-white hover:text-cocoa-900"
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
    </section>
  );
}
