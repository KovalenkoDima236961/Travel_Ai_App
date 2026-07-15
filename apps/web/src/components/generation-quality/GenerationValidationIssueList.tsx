import type { GenerationValidationIssue } from "@/types/generation-quality";

type GenerationValidationIssueListProps = {
  issues: GenerationValidationIssue[];
  limit?: number;
};

export function GenerationValidationIssueList({
  issues,
  limit = 4
}: GenerationValidationIssueListProps) {
  if (issues.length === 0) {
    return null;
  }

  const visible = issues.slice(0, limit);
  const hiddenCount = issues.length - visible.length;

  return (
    <ul className="space-y-1 text-sm leading-6">
      {visible.map((issue) => (
        <li key={issue.id}>
          <span className="font-medium">{issue.title}</span>
          {issueLocation(issue) ? <span className="opacity-80"> · {issueLocation(issue)}</span> : null}
        </li>
      ))}
      {hiddenCount > 0 ? <li className="opacity-80">+{hiddenCount} more issue(s)</li> : null}
    </ul>
  );
}

function issueLocation(issue: GenerationValidationIssue) {
  if (issue.dayNumber != null && issue.itemIndex != null) {
    return `Day ${issue.dayNumber}, item ${issue.itemIndex + 1}`;
  }
  if (issue.dayNumber != null) {
    return `Day ${issue.dayNumber}`;
  }
  if (issue.routeLegId) {
    return `Route leg ${issue.routeLegId}`;
  }
  return null;
}
