"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { createWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import { getErrorMessage } from "@/lib/utils";

export default function NewWorkspacePage() {
  return (
    <ProtectedRoute>
      <PageContainer className="max-w-3xl">
        <NewWorkspaceForm />
      </PageContainer>
    </ProtectedRoute>
  );
}

function NewWorkspaceForm() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { setCurrentWorkspace, refreshWorkspaces } = useWorkspaces();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: createWorkspace,
    onSuccess: async (workspace) => {
      await queryClient.invalidateQueries({ queryKey: workspaceKeys.all });
      await refreshWorkspaces();
      setCurrentWorkspace(workspace.id);
      router.push(`/workspaces/${workspace.id}`);
    }
  });

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedName = name.trim();
    if (trimmedName.length < 2 || trimmedName.length > 80) {
      setValidationError("Name must be between 2 and 80 characters.");
      return;
    }
    if (description.trim().length > 500) {
      setValidationError("Description must be at most 500 characters.");
      return;
    }
    setValidationError(null);
    mutation.mutate({
      name: trimmedName,
      description: description.trim() || null
    });
  }

  return (
    <>
      <div className="mb-8">
        <p className="text-sm font-semibold uppercase text-primary-700">New workspace</p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-950">Create workspace</h1>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
          Create a shared planning space for trips that should be available to a group.
        </p>
      </div>

      <Card>
        <form className="space-y-6" onSubmit={handleSubmit}>
          <label className="block">
            <span className="text-sm font-medium text-slate-800">Name</span>
            <span className="mt-2 block">
              <Input
                value={name}
                maxLength={80}
                placeholder="Japan Trip Group"
                onChange={(event) => setName(event.target.value)}
              />
            </span>
          </label>

          <label className="block">
            <span className="text-sm font-medium text-slate-800">Description</span>
            <span className="mt-2 block">
              <Textarea
                value={description}
                maxLength={500}
                placeholder="Planning space for our group trip."
                onChange={(event) => setDescription(event.target.value)}
              />
            </span>
          </label>

          {validationError || mutation.isError ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {validationError ?? getErrorMessage(mutation.error, "Could not create workspace.")}
            </div>
          ) : null}

          <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
            <Button variant="secondary" onClick={() => router.push("/workspaces")}>
              Cancel
            </Button>
            <Button disabled={mutation.isPending} type="submit">
              {mutation.isPending ? "Creating..." : "Create workspace"}
            </Button>
          </div>
        </form>
      </Card>
    </>
  );
}
