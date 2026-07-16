import { ReadinessCategoryBadge } from "./ReadinessCategoryBadge";
import { severityClasses } from "./readiness-ui";
import type { ReadinessItem } from "@/types/group-readiness";

type ReadinessIssueListProps = {
  items: ReadinessItem[];
};

export function ReadinessIssueList({ items }: ReadinessIssueListProps) {
  if (items.length === 0) {
    return <p className="text-[13px] text-cocoa-500">No open readiness items.</p>;
  }
  return (
    <div className="space-y-2">
      {items.map((item) => (
        <div key={`${item.id}:${item.category}`} className="rounded-[14px] border border-sand-300 bg-white p-3">
          <div className="flex flex-wrap items-center gap-2">
            <ReadinessCategoryBadge category={item.category} />
            <span
              className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${severityClasses(
                item.severity
              )}`}
            >
              {item.severity}
            </span>
            <span className="text-[12px] text-cocoa-400">{item.status.replaceAll("_", " ")}</span>
          </div>
          <p className="mt-2 text-[14px] font-semibold text-cocoa-900">{item.title}</p>
          <p className="mt-1 text-[13px] leading-[1.5] text-cocoa-500">{item.description}</p>
          {item.action ? (
            <a
              href={item.action.href}
              className="mt-3 inline-flex text-[13px] font-semibold text-clay hover:text-clay-dark"
            >
              {item.action.label}
            </a>
          ) : null}
        </div>
      ))}
    </div>
  );
}

