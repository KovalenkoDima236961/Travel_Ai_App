import type { GroupReadinessTopAction } from "@/types/group-readiness";

export function ReadinessTopActions({ actions }: { actions: GroupReadinessTopAction[] }) {
  if (actions.length === 0) {
    return (
      <div className="rounded-[16px] border border-sand-300 bg-white p-4">
        <p className="text-[13px] font-semibold text-cocoa-900">No urgent group actions</p>
        <p className="mt-1 text-[13px] text-cocoa-500">Group readiness has no high-priority next step.</p>
      </div>
    );
  }

  return (
    <div className="rounded-[16px] border border-sand-300 bg-white p-4">
      <h3 className="text-[14px] font-semibold text-cocoa-900">Next actions</h3>
      <div className="mt-3 space-y-3">
        {actions.slice(0, 4).map((action) => (
          <a
            key={action.id}
            className="block rounded-[12px] border border-sand-300 bg-sand-50 p-3 transition hover:border-clay"
            href={action.href}
          >
            <p className="text-[13px] font-semibold text-cocoa-900">{action.label}</p>
            <p className="mt-1 text-[12.5px] leading-[1.45] text-cocoa-500">{action.description}</p>
          </a>
        ))}
      </div>
    </div>
  );
}

