"use client";

import Link from "next/link";
import { useState } from "react";

import { ApproveTripDialog, RequestChangesDialog } from "@/components/approvals/ApprovalDialogs";
import { ApprovalStatusBadge } from "@/components/approvals/ApprovalStatusBadge";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { useTripApprovalMutations } from "@/hooks/useTripApproval";
import { useWorkspaceApprovals } from "@/hooks/useWorkspaceApprovals";
import { getApiErrorMessage } from "@/lib/api/client";
import type {
  WorkspaceApprovalQueueItem,
  WorkspaceApprovalStatusFilter
} from "@/types/approval";

const TABS: { value: WorkspaceApprovalStatusFilter; label: string }[] = [
  { value: "pending_approval", label: "Pending" },
  { value: "changes_requested", label: "Changes requested" },
  { value: "draft", label: "Draft" },
  { value: "approved", label: "Approved" },
  { value: "all", label: "All" }
];

function formatMoney(amount: number, currency?: string): string {
  if (!currency) {
    return amount.toFixed(2);
  }
  try {
    return new Intl.NumberFormat(undefined, { style: "currency", currency }).format(amount);
  } catch {
    return `${amount.toFixed(2)} ${currency}`;
  }
}

export function WorkspaceApprovalsQueue({
  workspaceId,
  canManage
}: {
  workspaceId: string;
  canManage: boolean;
}) {
  const [status, setStatus] = useState<WorkspaceApprovalStatusFilter>("pending_approval");
  const query = useWorkspaceApprovals({ workspaceId, status });

  const counts = query.data?.counts;
  const approvals = query.data?.approvals ?? [];

  return (
    <div className="space-y-4">
      {counts ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <CountCard label="Pending" value={counts.pendingApproval} />
          <CountCard label="Changes requested" value={counts.changesRequested} />
          <CountCard label="Draft" value={counts.draft} />
          <CountCard label="Approved" value={counts.approved} />
        </div>
      ) : null}

      <div className="flex flex-wrap gap-2 border-b border-slate-200 pb-2">
        {TABS.map((tab) => (
          <button
            key={tab.value}
            type="button"
            onClick={() => setStatus(tab.value)}
            className={
              status === tab.value
                ? "rounded-md bg-primary-600 px-3 py-1.5 text-sm font-medium text-white"
                : "rounded-md px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-100"
            }
          >
            {tab.label}
          </button>
        ))}
      </div>

      {query.isLoading ? (
        <p className="text-sm text-slate-500">Loading approvals…</p>
      ) : query.isError ? (
        <p className="text-sm text-red-600">Could not load workspace approvals.</p>
      ) : approvals.length === 0 ? (
        <Card>
          <p className="text-sm text-slate-600">
            {status === "pending_approval"
              ? "No trips waiting for approval."
              : "Nothing here yet."}
          </p>
          <p className="mt-1 text-sm text-slate-400">
            Submit workspace trips for review when they are ready.
          </p>
        </Card>
      ) : (
        <ul className="space-y-3">
          {approvals.map((item) => (
            <ApprovalQueueRow key={item.tripId} item={item} canManage={canManage} />
          ))}
        </ul>
      )}
    </div>
  );
}

function CountCard({ label, value }: { label: string; value: number }) {
  return (
    <Card className="p-3">
      <p className="text-xs uppercase tracking-wide text-slate-400">{label}</p>
      <p className="text-2xl font-semibold text-slate-900">{value}</p>
    </Card>
  );
}

function ApprovalQueueRow({
  item,
  canManage
}: {
  item: WorkspaceApprovalQueueItem;
  canManage: boolean;
}) {
  const mutations = useTripApprovalMutations(item.tripId);
  const [dialog, setDialog] = useState<"approve" | "request-changes" | null>(null);

  const showActions = canManage && item.approvalStatus === "pending_approval";

  return (
    <li>
      <Card className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium text-slate-900">{item.title || item.destination}</span>
            <ApprovalStatusBadge status={item.approvalStatus} />
          </div>
          <p className="mt-1 text-sm text-slate-500">
            {item.destination}
            {item.startDate ? ` · ${item.startDate}` : ""}
            {item.submittedAt
              ? ` · submitted ${new Date(item.submittedAt).toLocaleDateString()}`
              : ""}
          </p>
          <p className="mt-1 text-sm text-slate-500">
            Estimated {formatMoney(item.estimatedTotal, item.budgetCurrency)}
            {item.budgetAmount != null
              ? ` of ${formatMoney(item.budgetAmount, item.budgetCurrency)} budget`
              : ""}
            {" · "}
            <span
              className={
                item.checklistStatus === "blocked"
                  ? "text-red-600"
                  : item.checklistStatus === "warning"
                    ? "text-amber-600"
                    : "text-emerald-600"
              }
            >
              {item.checklistStatus === "ok"
                ? "Checklist OK"
                : `${item.warningCount} warning(s)${item.criticalCount ? `, ${item.criticalCount} blocker(s)` : ""}`}
            </span>
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Link
            className={buttonStyles({ variant: "secondary", size: "sm" })}
            href={`/trips/${item.tripId}`}
          >
            Open trip
          </Link>
          {showActions ? (
            <>
              <Button
                size="sm"
                disabled={mutations.approve.isPending}
                onClick={() => setDialog("approve")}
              >
                Approve
              </Button>
              <Button
                size="sm"
                variant="secondary"
                disabled={mutations.requestChanges.isPending}
                onClick={() => setDialog("request-changes")}
              >
                Request changes
              </Button>
            </>
          ) : null}
        </div>
      </Card>

      <ApproveTripDialog
        open={dialog === "approve"}
        isSubmitting={mutations.approve.isPending}
        error={mutations.approve.isError ? getApiErrorMessage(mutations.approve.error) : null}
        onClose={() => setDialog(null)}
        onSubmit={(input) => mutations.approve.mutate(input, { onSuccess: () => setDialog(null) })}
      />
      <RequestChangesDialog
        open={dialog === "request-changes"}
        isSubmitting={mutations.requestChanges.isPending}
        error={
          mutations.requestChanges.isError
            ? getApiErrorMessage(mutations.requestChanges.error)
            : null
        }
        onClose={() => setDialog(null)}
        onSubmit={(input) =>
          mutations.requestChanges.mutate(input, { onSuccess: () => setDialog(null) })
        }
      />
    </li>
  );
}
