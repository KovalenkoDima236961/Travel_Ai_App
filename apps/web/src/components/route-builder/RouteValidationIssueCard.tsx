import type { RouteBuilderIssue } from "@/lib/route-builder/route-validation";

type RouteValidationIssueCardProps = {
  issue: RouteBuilderIssue;
  onAction?: (issue: RouteBuilderIssue) => void;
};

const tones: Record<RouteBuilderIssue["severity"], string> = {
  error: "border-red-200 bg-red-50 text-red-900",
  warning: "border-amber-200 bg-amber-50 text-amber-900",
  info: "border-sky-200 bg-sky-50 text-sky-900"
};

export function RouteValidationIssueCard({ issue, onAction }: RouteValidationIssueCardProps) {
  const content = (
    <>
      <span aria-hidden>{issue.severity === "error" ? "!" : issue.severity === "warning" ? "⚠" : "i"}</span>
      <span className="sr-only">{issue.severity}: </span>
    </>
  );
  return (
    <article id={`route-issue-${safeId(issue.id)}`} className={`rounded-[14px] border p-3 ${tones[issue.severity]}`}>
      <div className="flex items-start gap-3">
        <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-current text-[11px] font-bold">
          {content}
        </span>
        <div className="min-w-0 flex-1">
          <h4 className="text-[13.5px] font-semibold">{issue.title}</h4>
          <p className="mt-1 text-[12.5px] leading-5 opacity-80">{issue.description}</p>
          {issue.action ? (
            issue.action.href && !onAction ? (
              <a className="mt-2 inline-flex text-[12px] font-semibold underline underline-offset-2" href={issue.action.href}>
                {issue.action.label}
              </a>
            ) : (
              <button
                className="mt-2 text-[12px] font-semibold underline underline-offset-2"
                onClick={() => onAction?.(issue)}
                type="button"
              >
                {issue.action.label}
              </button>
            )
          ) : null}
        </div>
      </div>
    </article>
  );
}

function safeId(value: string) {
  return value.replace(/[^a-zA-Z0-9_-]/g, "-");
}
