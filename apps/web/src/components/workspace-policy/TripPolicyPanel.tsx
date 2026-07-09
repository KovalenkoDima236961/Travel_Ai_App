"use client";

import Link from "next/link";

import { useTripPolicyEvaluation } from "@/hooks/useTripPolicyEvaluation";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import type { PolicyEvaluationResult, PolicySuggestedAction } from "@/types/workspace-policy";
import { PolicyStatusBadge } from "./PolicyStatusBadge";

export function TripPolicyPanel({ tripId }: { tripId: string }) {
  const { query, evaluate } = useTripPolicyEvaluation(tripId);
  const evaluation = query.data;

  if (query.isLoading) {
    return <Card><p className="text-sm text-slate-500">Checking workspace policy…</p></Card>;
  }
  if (query.isError || !evaluation) {
    return (
      <Card>
        <p className="text-sm text-red-700">Workspace policy could not be evaluated.</p>
        <Button className="mt-3" variant="secondary" onClick={() => query.refetch()}>
          Try again
        </Button>
      </Card>
    );
  }

  const visible = evaluation.results.filter((result) => result.status !== "passed");
  const groups = (["blocking", "warning", "info"] as const)
    .map((severity) => ({
      severity,
      results: visible.filter((result) => result.severity === severity)
    }))
    .filter((group) => group.results.length > 0);

  return (
    <Card className="space-y-4" id="workspace-policy">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold text-slate-950">Workspace policy</h3>
          <p className="mt-1 text-sm text-slate-500">
            Deterministic checks are authoritative; AI guidance can still need review.
          </p>
        </div>
        <PolicyStatusBadge status={evaluation.status} />
      </div>

      {evaluation.status === "not_applicable" ? (
        <p className="text-sm text-slate-600">
          {evaluation.notApplicableReason === "personal_trip"
            ? "No workspace policy applies to personal trips."
            : "This workspace has no active planning policy."}
        </p>
      ) : (
        <div className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-5">
          <Metric label="Checked" value={evaluation.summary.rulesChecked} />
          <Metric label="Passed" value={evaluation.summary.passedCount} />
          <Metric label="Info" value={evaluation.summary.infoCount} />
          <Metric label="Warnings" value={evaluation.summary.warningCount} />
          <Metric label="Blocking" value={evaluation.summary.blockingCount} />
        </div>
      )}

      {groups.map((group) => (
        <section key={group.severity}>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            {group.severity}
          </h4>
          <ul className="mt-2 space-y-3">
            {group.results.map((result) => (
              <PolicyResult key={result.ruleKey} result={result} tripId={tripId} />
            ))}
          </ul>
        </section>
      ))}

      {evaluation.warnings.length > 0 ? (
        <ul className="rounded-md bg-amber-50 p-3 text-sm text-amber-900">
          {evaluation.warnings.map((warning) => <li key={warning}>{warning}</li>)}
        </ul>
      ) : null}

      <Button
        disabled={evaluate.isPending}
        variant="secondary"
        onClick={() => evaluate.mutate()}
      >
        {evaluate.isPending ? "Checking…" : "Re-check policy"}
      </Button>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md bg-slate-50 p-3">
      <div className="text-lg font-semibold text-slate-950">{value}</div>
      <div className="text-xs text-slate-500">{label}</div>
    </div>
  );
}

function PolicyResult({
  result,
  tripId
}: {
  result: PolicyEvaluationResult;
  tripId: string;
}) {
  return (
    <li className="rounded-md border border-slate-200 p-3">
      <p className="font-medium text-slate-950">{result.title}</p>
      <p className="mt-1 text-sm text-slate-600">{result.message}</p>
      {result.affectedItems.length > 0 ? (
        <ul className="mt-2 text-xs text-slate-500">
          {result.affectedItems.map((item, index) => (
            <li key={`${item.dayNumber ?? "trip"}-${item.itemIndex ?? index}`}>
              {item.dayNumber ? `Day ${item.dayNumber}` : "Trip"}
              {item.name ? ` · ${item.name}` : ""}
              {item.amount != null ? ` · ${item.amount} ${item.currency ?? ""}` : ""}
            </li>
          ))}
        </ul>
      ) : null}
      <div className="mt-3 flex flex-wrap gap-2">
        {result.suggestedActions.map((action) => (
          <PolicyAction
            key={`${action.type}-${action.dayNumber ?? ""}-${action.itemIndex ?? ""}`}
            action={action}
            tripId={tripId}
          />
        ))}
      </div>
    </li>
  );
}

function PolicyAction({
  action,
  tripId
}: {
  action: PolicySuggestedAction;
  tripId: string;
}) {
  const href = actionHref(action, tripId);
  if (!href) {
    return <span className="text-xs text-primary-700">{action.label}</span>;
  }
  return <Link className="text-xs font-medium text-primary-700 hover:underline" href={href}>
    {action.label}
  </Link>;
}

function actionHref(action: PolicySuggestedAction, tripId: string): string | null {
  if (action.type === "open_trip_analytics") {
    return `/trips/${tripId}/analytics`;
  }
  if (action.type === "open_budget_optimization") {
    return `/trips/${tripId}?optimizeBudget=1#workspace-policy`;
  }
  if (action.dayNumber != null) {
    return `/trips/${tripId}#day-${action.dayNumber}`;
  }
  if (["set_trip_budget", "open_cost_splitting", "open_accommodation"].includes(action.type)) {
    return `/trips/${tripId}`;
  }
  return null;
}
