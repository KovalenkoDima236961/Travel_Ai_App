import type { GenerationQuality } from "@/types/generation-quality";

type RepairAttemptsSummaryProps = {
  quality?: GenerationQuality | null;
};

export function RepairAttemptsSummary({ quality }: RepairAttemptsSummaryProps) {
  const attempts = quality?.repairAttemptsLog ?? [];
  if (!quality || quality.repairAttempts === 0 || attempts.length === 0) {
    return null;
  }

  const fixedCount = attempts.reduce((total, attempt) => total + attempt.issuesFixed.length, 0);
  const lastAttempt = attempts[attempts.length - 1];

  return (
    <p className="text-xs leading-5 opacity-80">
      Repair attempted {quality.repairAttempts} time(s), fixed {fixedCount} issue(s)
      {lastAttempt?.aiProviderMode ? ` via ${lastAttempt.aiProviderMode}` : ""}.
    </p>
  );
}
