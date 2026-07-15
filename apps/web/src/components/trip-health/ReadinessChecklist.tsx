import { severityRank } from "./health-ui";
import type { TripHealthIssue } from "@/types/trip-health";

export function ReadinessChecklist({ issues }: { issues: TripHealthIssue[] }) {
  const critical = issues.filter((issue) => issue.severity === "critical").length;
  const high = issues.filter((issue) => issue.severity === "high").length;
  const warning = issues.filter((issue) => issue.severity === "warning").length;
  const actionable = issues.filter((issue) => Boolean(issue.action?.href)).length;
  const topSeverity = issues.reduce(
    (current, issue) =>
      severityRank[issue.severity] > severityRank[current] ? issue.severity : current,
    "info" as TripHealthIssue["severity"]
  );

  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
        Readiness Checklist
      </h2>
      <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <ReadinessMetric label="Critical" value={critical} ok={critical === 0} />
        <ReadinessMetric label="High" value={high} ok={high === 0} />
        <ReadinessMetric label="Warnings" value={warning} ok={warning === 0} />
        <ReadinessMetric label="Actionable" value={actionable} ok={actionable === 0} />
      </div>
      <p className="mt-4 text-[13px] text-cocoa-400">
        Highest open severity: {topSeverity}
      </p>
    </section>
  );
}

function ReadinessMetric({
  label,
  value,
  ok
}: {
  label: string;
  value: number;
  ok: boolean;
}) {
  return (
    <div className="rounded-[14px] border border-sand-200 bg-sand-50 p-4">
      <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
        {label}
      </p>
      <p className={`mt-2 text-[24px] font-semibold ${ok ? "text-[#2F5C3C]" : "text-cocoa-900"}`}>
        {value}
      </p>
    </div>
  );
}
