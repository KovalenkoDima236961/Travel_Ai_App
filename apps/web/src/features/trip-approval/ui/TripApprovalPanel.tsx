"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import {
  ApproveTripDialog,
  CancelApprovalDialog,
  RequestChangesDialog,
  SubmitForApprovalDialog
} from "./ApprovalDialogs";
import { ApprovalChecklist } from "./ApprovalChecklist";
import { ApprovalStatusBadge } from "./ApprovalStatusBadge";
import { RiskFactorsList, RiskScoreCard, useTripApprovalRisk } from "@/features/approval-risk";
import { handleRiskAction } from "@/lib/approval-risk/action-router";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import {
  useTripApproval,
  useTripApprovalEvents,
  useTripApprovalMutations
} from "../model/useTripApproval";
import { getApiErrorMessage } from "@/shared/api/client";
import type { ApprovalEventType } from "@/entities/approval/model";
import type { RepairMode } from "@/entities/trip-repair/model";

type DialogKind = "submit" | "approve" | "request-changes" | "cancel" | null;

const EVENT_LABELS: Record<ApprovalEventType, string> = {
  submitted: "Submitted for approval",
  approved: "Approved",
  changes_requested: "Changes requested",
  cancelled: "Submission cancelled",
  reset_to_draft: "Reset to draft after an edit"
};

function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "" : date.toLocaleString();
}

// TripApprovalPanel renders a workspace trip's approval state, checklist,
// role-aware actions, and history. Personal trips render a short "not required"
// note. For workspace trips this is the single control surface for the approval
// lifecycle.
export function TripApprovalPanel({
  tripId,
  onOpenTripRepair
}: {
  tripId: string;
  onOpenTripRepair?: (repairMode?: RepairMode) => void;
}) {
  const router = useRouter();
  const { online } = useNetworkStatus();
  const approvalQuery = useTripApproval(tripId);
  const eventsQuery = useTripApprovalEvents(tripId);
  const mutations = useTripApprovalMutations(tripId);
  const [dialog, setDialog] = useState<DialogKind>(null);
  const [showHistory, setShowHistory] = useState(false);

  const approval = approvalQuery.data;
  const riskQuery = useTripApprovalRisk(tripId, Boolean(approval?.workspaceId));
  const risk = riskQuery.data;

  if (approvalQuery.isLoading) {
    return (
      <Card id="approval">
        <p className="text-sm text-slate-500">Loading approval status…</p>
      </Card>
    );
  }

  if (approvalQuery.isError || !approval) {
    return (
      <Card id="approval">
        <p className="text-sm text-red-600">Could not load approval status.</p>
      </Card>
    );
  }

  if (approval.status === "not_required" || !approval.workspaceId) {
    return (
      <Card>
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-base font-semibold text-slate-900">Approval</h3>
          <ApprovalStatusBadge status="not_required" />
        </div>
        <p className="mt-2 text-sm text-slate-500">
          Approval is not required for personal trips.
        </p>
      </Card>
    );
  }

  const anyPending =
    mutations.submit.isPending ||
    mutations.approve.isPending ||
    mutations.requestChanges.isPending ||
    mutations.cancel.isPending;

  function closeDialog() {
    setDialog(null);
  }

  return (
    <Card id="approval" className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-base font-semibold text-slate-900">Approval</h3>
        <ApprovalStatusBadge status={approval.status} />
      </div>

      <dl className="grid grid-cols-1 gap-2 text-sm text-slate-600 sm:grid-cols-2">
        {approval.submittedAt ? (
          <div>
            <dt className="text-xs uppercase tracking-wide text-slate-400">Submitted</dt>
            <dd>{formatDateTime(approval.submittedAt)}</dd>
          </div>
        ) : null}
        {approval.approvedAt ? (
          <div>
            <dt className="text-xs uppercase tracking-wide text-slate-400">Approved</dt>
            <dd>{formatDateTime(approval.approvedAt)}</dd>
          </div>
        ) : null}
      </dl>

      {riskQuery.isLoading ? (
        <p className="rounded-md bg-slate-50 px-3 py-2 text-sm text-slate-500">
          Loading approval risk…
        </p>
      ) : riskQuery.isError ? (
        <p className="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-800">
          Approval risk is temporarily unavailable.
        </p>
      ) : risk && risk.status !== "not_applicable" ? (
        <>
          <RiskScoreCard
            risk={risk}
            onAction={(action) =>
              handleRiskAction(action, {
                tripId,
                workspaceId: approval.workspaceId,
                openTripRepair: onOpenTripRepair,
                router
              })
            }
          />
          {risk.status === "medium" || risk.status === "high" || risk.status === "critical" ? (
            <RiskFactorsList
              defaultOpen={risk.status === "critical"}
              factors={risk.factors}
              onAction={(action) =>
                handleRiskAction(action, {
                  tripId,
                  workspaceId: approval.workspaceId,
                  openTripRepair: onOpenTripRepair,
                  router
                })
              }
            />
          ) : null}
        </>
      ) : null}

      {approval.status === "changes_requested" && approval.decisionNote ? (
        <div className="rounded-md border border-orange-200 bg-orange-50 px-3 py-2 text-sm text-orange-800">
          <span className="font-medium">Changes requested:</span> {approval.decisionNote}
        </div>
      ) : null}
      {approval.note && approval.status === "pending_approval" ? (
        <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">
          <span className="font-medium">Submitter note:</span> {approval.note}
        </div>
      ) : null}

      {approval.status === "approved" ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
          Editing this approved trip will move it back to draft and require approval again.
        </div>
      ) : null}
      {approval.status === "pending_approval" ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
          Editing this trip while it is pending will move it back to draft; it will need to be
          resubmitted.
        </div>
      ) : null}

      {approval.checklist ? <ApprovalChecklist checklist={approval.checklist} /> : null}

      {!online ? (
        <p className="rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-500">
          Approval actions require internet.
        </p>
      ) : null}

      <div className="flex flex-wrap gap-2">
        {approval.canSubmit ? (
          <Button disabled={!online || anyPending} onClick={() => setDialog("submit")}>
            Submit for approval
          </Button>
        ) : null}
        {approval.canApprove ? (
          <Button disabled={!online || anyPending} onClick={() => setDialog("approve")}>
            Approve
          </Button>
        ) : null}
        {approval.canRequestChanges ? (
          <Button
            disabled={!online || anyPending}
            variant="secondary"
            onClick={() => setDialog("request-changes")}
          >
            Request changes
          </Button>
        ) : null}
        {approval.canCancel ? (
          <Button
            disabled={!online || anyPending}
            variant="ghost"
            onClick={() => setDialog("cancel")}
          >
            Cancel submission
          </Button>
        ) : null}
      </div>

      <div>
        <button
          className="text-sm font-medium text-primary-700 hover:underline"
          onClick={() => setShowHistory((prev) => !prev)}
          type="button"
        >
          {showHistory ? "Hide approval history" : "Show approval history"}
        </button>
        {showHistory ? (
          <ul className="mt-2 space-y-2">
            {(eventsQuery.data?.events ?? []).map((event) => (
              <li key={event.id} className="text-sm text-slate-600">
                <span className="font-medium text-slate-900">
                  {EVENT_LABELS[event.eventType] ?? event.eventType}
                </span>
                <span className="text-slate-400"> · {formatDateTime(event.createdAt)}</span>
                {event.note ? <p className="text-slate-500">{event.note}</p> : null}
              </li>
            ))}
            {(eventsQuery.data?.events?.length ?? 0) === 0 ? (
              <li className="text-sm text-slate-400">No approval history yet.</li>
            ) : null}
          </ul>
        ) : null}
      </div>

      <SubmitForApprovalDialog
        open={dialog === "submit"}
        checklist={approval.checklist}
        risk={risk}
        isSubmitting={mutations.submit.isPending}
        error={mutations.submit.isError ? getApiErrorMessage(mutations.submit.error) : null}
        onClose={closeDialog}
        onSubmit={(input) => mutations.submit.mutate(input, { onSuccess: closeDialog })}
      />
      <ApproveTripDialog
        open={dialog === "approve"}
        isSubmitting={mutations.approve.isPending}
        error={mutations.approve.isError ? getApiErrorMessage(mutations.approve.error) : null}
        onClose={closeDialog}
        onSubmit={(input) => mutations.approve.mutate(input, { onSuccess: closeDialog })}
      />
      <RequestChangesDialog
        open={dialog === "request-changes"}
        isSubmitting={mutations.requestChanges.isPending}
        error={
          mutations.requestChanges.isError
            ? getApiErrorMessage(mutations.requestChanges.error)
            : null
        }
        onClose={closeDialog}
        onSubmit={(input) => mutations.requestChanges.mutate(input, { onSuccess: closeDialog })}
      />
      <CancelApprovalDialog
        open={dialog === "cancel"}
        isSubmitting={mutations.cancel.isPending}
        error={mutations.cancel.isError ? getApiErrorMessage(mutations.cancel.error) : null}
        onClose={closeDialog}
        onSubmit={(input) => mutations.cancel.mutate(input, { onSuccess: closeDialog })}
      />
    </Card>
  );
}
