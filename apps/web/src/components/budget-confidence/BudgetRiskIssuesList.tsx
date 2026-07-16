import type { BudgetConfidenceIssue } from "@/types/budget-confidence";

export function BudgetRiskIssuesList({ issues }: { issues: BudgetConfidenceIssue[] }) {
  if (issues.length === 0) {
    return (
      <p className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-800">
        No budget confidence issues detected.
      </p>
    );
  }

  return (
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Risk issues</p>
      <ul className="mt-2 space-y-2">
        {issues.slice(0, 5).map((issue) => (
          <li
            className="rounded-md border border-slate-200 px-3 py-2 text-sm"
            key={issue.id}
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-medium text-slate-950">{issue.title}</span>
              <span className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${severityClasses(issue.severity)}`}>
                {issue.severity}
              </span>
            </div>
            <p className="mt-1 text-xs leading-5 text-slate-600">{issue.description}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}

function severityClasses(severity: BudgetConfidenceIssue["severity"]) {
  switch (severity) {
    case "critical":
      return "border-red-200 bg-red-50 text-red-700";
    case "high":
      return "border-orange-200 bg-orange-50 text-orange-700";
    case "warning":
      return "border-amber-200 bg-amber-50 text-amber-800";
    default:
      return "border-slate-200 bg-slate-50 text-slate-600";
  }
}
