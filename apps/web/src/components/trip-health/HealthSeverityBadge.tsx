import { severityClasses, severityLabel } from "./health-ui";
import type { TripHealthIssueSeverity } from "@/types/trip-health";

export function HealthSeverityBadge({ severity }: { severity: TripHealthIssueSeverity }) {
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-1 text-[12px] font-semibold ${severityClasses(
        severity
      )}`}
    >
      {severityLabel[severity]}
    </span>
  );
}
