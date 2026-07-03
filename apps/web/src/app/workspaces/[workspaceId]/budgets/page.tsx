"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import {
  formatAnalyticsDate,
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { WorkspaceBudgetFormDialog } from "@/components/workspace-budgets/WorkspaceBudgetFormDialog";
import {
  canManageWorkspace,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import { useWorkspaceBudgetSummary } from "@/hooks/useWorkspaceBudgetSummary";
import {
  useWorkspaceBudgetMutations,
  useWorkspaceBudgets
} from "@/hooks/useWorkspaceBudgets";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import type {
  CreateWorkspaceBudgetInput,
  WorkspaceBudget
} from "@/types/workspace-budget";

export default function WorkspaceBudgetsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceBudgetsPageContent />
    </ProtectedRoute>
  );
}

function WorkspaceBudgetsPageContent() {
  const params = useParams<{ workspaceId: string }>();
  const workspaceId = params.workspaceId;
  const { setCurrentWorkspace } = useWorkspaces();
  const [dialogMode, setDialogMode] = useState<"create" | "edit" | null>(null);
  const [editingBudget, setEditingBudget] = useState<WorkspaceBudget | null>(null);
  const budgetsQuery = useWorkspaceBudgets({ workspaceId });
  const mutations = useWorkspaceBudgetMutations(workspaceId);

  const workspaceQuery = useQuery({
    queryKey: workspaceKeys.detail(workspaceId),
    queryFn: () => getWorkspace(workspaceId),
    enabled: Boolean(workspaceId)
  });

  useEffect(() => {
    if (workspaceQuery.isSuccess) {
      setCurrentWorkspace(workspaceId);
    }
  }, [setCurrentWorkspace, workspaceId, workspaceQuery.isSuccess]);

  const workspace = workspaceQuery.data ?? null;
  const canManage = workspace ? canManageWorkspace(workspace.currentUserRole) : false;
  const budgets = budgetsQuery.data ?? [];
  const activeBudgets = budgets.filter((budget) => budget.status === "active");
  const archivedBudgets = budgets.filter((budget) => budget.status === "archived");
  const primaryBudget = activeBudgets.find((budget) => budget.isPrimary) ?? null;
  const mutationError = mutationMessage(
    mutations.createBudget.error ??
      mutations.updateBudget.error ??
      mutations.archiveBudget.error ??
      mutations.makePrimary.error
  );

  function openCreateDialog() {
    setEditingBudget(null);
    setDialogMode("create");
  }

  function openEditDialog(budget: WorkspaceBudget) {
    setEditingBudget(budget);
    setDialogMode("edit");
  }

  function closeDialog() {
    setDialogMode(null);
    setEditingBudget(null);
  }

  function submitBudget(input: CreateWorkspaceBudgetInput) {
    if (dialogMode === "edit" && editingBudget) {
      mutations.updateBudget.mutate(
        { budgetId: editingBudget.id, input },
        { onSuccess: closeDialog }
      );
      return;
    }
    mutations.createBudget.mutate(input, { onSuccess: closeDialog });
  }

  function archiveBudget(budget: WorkspaceBudget) {
    const confirmed = window.confirm("Archive this workspace budget? Existing trips will not be changed.");
    if (!confirmed) {
      return;
    }
    const reason = window.prompt("Archive reason") ?? undefined;
    mutations.archiveBudget.mutate({ budgetId: budget.id, reason });
  }

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href={`/workspaces/${workspaceId}`}>
            Back to workspace
          </Link>
          <h1 className="mt-3 text-3xl font-semibold text-slate-950">Workspace budgets</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
            Shared planning limits for workspace trips.
          </p>
        </div>
        {canManage ? (
          <Button onClick={openCreateDialog} type="button">
            Create budget
          </Button>
        ) : null}
      </div>

      {workspaceQuery.isLoading || budgetsQuery.isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading workspace budgets...
        </div>
      ) : null}

      {workspaceQuery.isError || budgetsQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {workspaceQuery.error instanceof Error
            ? workspaceQuery.error.message
            : budgetsQuery.error instanceof Error
              ? budgetsQuery.error.message
              : "Could not load workspace budgets."}
        </div>
      ) : null}

      {primaryBudget ? (
        <section className="mb-8">
          <h2 className="text-xl font-semibold text-slate-950">Primary budget</h2>
          <div className="mt-4">
            <BudgetCard
              budget={primaryBudget}
              canManage={canManage}
              onArchive={archiveBudget}
              onEdit={openEditDialog}
              onMakePrimary={(budget) => mutations.makePrimary.mutate(budget.id)}
            />
          </div>
        </section>
      ) : null}

      <section>
        <div className="flex items-center justify-between gap-4">
          <h2 className="text-xl font-semibold text-slate-950">Active budgets</h2>
          {activeBudgets.length > 0 ? (
            <span className="text-sm text-slate-500">{activeBudgets.length} active</span>
          ) : null}
        </div>
        {activeBudgets.length === 0 && budgetsQuery.isSuccess ? (
          <Card className="mt-4 text-sm text-slate-600">
            {canManage ? "No active workspace budgets yet." : "No workspace budget set."}
          </Card>
        ) : null}
        {activeBudgets.length > 0 ? (
          <div className="mt-4 grid gap-4 lg:grid-cols-2">
            {activeBudgets.map((budget) => (
              <BudgetCard
                budget={budget}
                canManage={canManage}
                key={budget.id}
                onArchive={archiveBudget}
                onEdit={openEditDialog}
                onMakePrimary={(item) => mutations.makePrimary.mutate(item.id)}
              />
            ))}
          </div>
        ) : null}
      </section>

      {archivedBudgets.length > 0 ? (
        <section className="mt-10">
          <h2 className="text-xl font-semibold text-slate-950">Archived budgets</h2>
          <div className="mt-4 grid gap-4 lg:grid-cols-2">
            {archivedBudgets.map((budget) => (
              <BudgetCard
                budget={budget}
                canManage={false}
                key={budget.id}
                onArchive={archiveBudget}
                onEdit={openEditDialog}
                onMakePrimary={(item) => mutations.makePrimary.mutate(item.id)}
              />
            ))}
          </div>
        </section>
      ) : null}

      <WorkspaceBudgetFormDialog
        error={mutationError}
        initialBudget={editingBudget}
        isSubmitting={mutations.createBudget.isPending || mutations.updateBudget.isPending}
        onClose={closeDialog}
        onSubmit={submitBudget}
        open={dialogMode != null}
        submitLabel={dialogMode === "edit" ? "Save changes" : "Create budget"}
        title={dialogMode === "edit" ? "Edit workspace budget" : "Create workspace budget"}
      />
    </PageContainer>
  );
}

function BudgetCard({
  budget,
  canManage,
  onEdit,
  onArchive,
  onMakePrimary
}: {
  budget: WorkspaceBudget;
  canManage: boolean;
  onEdit: (budget: WorkspaceBudget) => void;
  onArchive: (budget: WorkspaceBudget) => void;
  onMakePrimary: (budget: WorkspaceBudget) => void;
}) {
  const summaryQuery = useWorkspaceBudgetSummary({
    workspaceId: budget.workspaceId,
    budgetId: budget.id,
    enabled: budget.status === "active"
  });
  const summary = summaryQuery.data?.summary;
  const utilization = summary?.utilizationPercent ?? 0;
  const progress = Math.min(Math.max(utilization, 0), 100);
  const over = (summary?.overBudgetAmount ?? 0) > 0;

  return (
    <Card className="flex h-full flex-col gap-5">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="break-words text-lg font-semibold text-slate-950">{budget.name}</h3>
            {budget.isPrimary ? (
              <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-semibold text-primary-700">
                Primary
              </span>
            ) : null}
            <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700">
              {budget.status}
            </span>
          </div>
          {budget.description ? (
            <p className="mt-2 text-sm leading-6 text-slate-600">{budget.description}</p>
          ) : null}
        </div>
        <div className="text-right text-sm font-semibold text-slate-950">
          {formatPlainMoney(budget.amount, budget.currency)}
        </div>
      </div>

      <div className="grid gap-3 text-sm sm:grid-cols-3">
        <Metric label="Period" value={formatBudgetPeriod(budget)} />
        <Metric
          label="Estimated"
          value={
            summary ? formatAnalyticsMoney(summary.estimatedTotal, budget.currency) : "Loading..."
          }
        />
        <Metric
          label={over ? "Over" : "Remaining"}
          tone={over ? "danger" : "ok"}
          value={
            summary
              ? formatAnalyticsMoney(
                  over ? summary.overBudgetAmount : summary.remainingAmount,
                  budget.currency
                )
              : "Loading..."
          }
        />
      </div>

      {budget.status === "active" ? (
        <div>
          <div className="flex items-center justify-between text-xs font-semibold text-slate-500">
            <span>Utilization</span>
            <span>{formatPercent(summary?.utilizationPercent)}</span>
          </div>
          <div className="mt-2 h-3 rounded-full bg-slate-100">
            <div
              className={over ? "h-3 rounded-full bg-red-600" : "h-3 rounded-full bg-primary-600"}
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      ) : null}

      <div className="mt-auto flex flex-wrap gap-2">
        <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/workspaces/${budget.workspaceId}/budgets/${budget.id}`}>
          View summary
        </Link>
        {canManage && budget.status === "active" ? (
          <>
            <Button onClick={() => onEdit(budget)} size="sm" type="button" variant="secondary">
              Edit
            </Button>
            {!budget.isPrimary ? (
              <Button onClick={() => onMakePrimary(budget)} size="sm" type="button" variant="secondary">
                Make primary
              </Button>
            ) : null}
            <Button onClick={() => onArchive(budget)} size="sm" type="button" variant="danger">
              Archive
            </Button>
          </>
        ) : null}
      </div>
    </Card>
  );
}

function Metric({
  label,
  value,
  tone = "default"
}: {
  label: string;
  value: string;
  tone?: "default" | "ok" | "danger";
}) {
  return (
    <div className="rounded-md bg-slate-50 p-3">
      <p className="text-xs font-semibold uppercase text-slate-500">{label}</p>
      <p
        className={
          tone === "danger"
            ? "mt-1 break-words font-semibold text-red-700"
            : tone === "ok"
              ? "mt-1 break-words font-semibold text-emerald-700"
              : "mt-1 break-words font-semibold text-slate-900"
        }
      >
        {value}
      </p>
    </div>
  );
}

function formatBudgetPeriod(budget: WorkspaceBudget) {
  if (!budget.periodStart && !budget.periodEnd) {
    return "All trips";
  }
  return `${formatAnalyticsDate(budget.periodStart)} - ${formatAnalyticsDate(budget.periodEnd)}`;
}

function mutationMessage(error: unknown) {
  return error instanceof Error ? error.message : null;
}
