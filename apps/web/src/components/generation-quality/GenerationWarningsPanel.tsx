import { GenerationValidationIssueList } from "./GenerationValidationIssueList";
import { RepairAttemptsSummary } from "./RepairAttemptsSummary";
import type { GenerationQuality } from "@/types/generation-quality";

type GenerationWarningsPanelProps = {
  quality?: GenerationQuality | null;
  compact?: boolean;
};

export function GenerationWarningsPanel({
  quality,
  compact = false
}: GenerationWarningsPanelProps) {
  if (!quality) {
    return null;
  }

  const issues = quality.remainingIssues ?? [];
  const warnings = quality.warnings ?? [];
  if (issues.length === 0 && warnings.length === 0 && quality.repairAttempts === 0) {
    return null;
  }

  return (
    <div className={compact ? "mt-2 space-y-1" : "mt-3 space-y-2 border-t border-current/15 pt-3"}>
      {warnings.length > 0 ? (
        <ul className="space-y-1 text-sm leading-6">
          {warnings.slice(0, compact ? 2 : 4).map((warning) => (
            <li key={warning}>{warning}</li>
          ))}
        </ul>
      ) : null}
      <GenerationValidationIssueList issues={issues} limit={compact ? 2 : 4} />
      <RepairAttemptsSummary quality={quality} />
    </div>
  );
}
