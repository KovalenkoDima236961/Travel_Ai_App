"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { WorkspaceBudgetFormDialog } from "@/features/workspace-budget";
import {
  canManageWorkspace,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import {
  useWorkspaceBudgetMutations,
  useWorkspaceBudgets
} from "@/features/workspace-budget";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import type {
  CreateWorkspaceBudgetInput,
  WorkspaceBudget
} from "@/entities/workspace-budget/model";
import { mutationMessage } from "../model/workspaceBudgetsPageModel";
import { BudgetCard } from "./BudgetCard";

export function WorkspaceBudgetsPageContent() {
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

