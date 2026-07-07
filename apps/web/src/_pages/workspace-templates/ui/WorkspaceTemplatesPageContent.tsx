"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { PageContainer } from "@/components/layout/PageContainer";
import { CreateTripFromTemplateDialog } from "@/features/trip-template";
import { TripTemplateCard } from "@/features/trip-template";
import { buttonStyles } from "@/shared/ui/button";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { useTripTemplateMutations } from "@/features/trip-template";
import { useWorkspaceTemplates } from "@/features/trip-template";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import { getErrorMessage } from "@/lib/utils";
import type { TripTemplate } from "@/entities/trip-template/model";

export function WorkspaceTemplatesPageContent() {
  const params = useParams<{ workspaceId: string }>();
  const workspaceId = params.workspaceId;
  const { setCurrentWorkspace } = useWorkspaces();
  const [selectedTemplate, setSelectedTemplate] = useState<TripTemplate | null>(null);
  const workspaceQuery = useQuery({
    queryKey: workspaceKeys.detail(workspaceId),
    queryFn: () => getWorkspace(workspaceId),
    enabled: Boolean(workspaceId)
  });
  const templatesQuery = useWorkspaceTemplates(workspaceId, { status: "active", limit: 50 });
  const mutations = useTripTemplateMutations();
  const templates = templatesQuery.data?.templates ?? [];

  useEffect(() => {
    if (workspaceQuery.isSuccess) {
      setCurrentWorkspace(workspaceId);
    }
  }, [setCurrentWorkspace, workspaceId, workspaceQuery.isSuccess]);

  function archiveTemplate(template: TripTemplate) {
    if (!window.confirm("Archive this workspace template?")) {
      return;
    }
    mutations.archiveTemplate.mutate({ templateId: template.id });
  }

  function duplicateTemplate(template: TripTemplate) {
    mutations.duplicateTemplate.mutate({
      templateId: template.id,
      input: {
        title: `Copy of ${template.title}`,
        visibility: "private"
      }
    });
  }

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href={`/workspaces/${workspaceId}`}>
            Back to workspace
          </Link>
          <h1 className="mt-3 text-3xl font-semibold text-slate-950">Workspace templates</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
            Save a workspace trip as a template to reuse it.
          </p>
        </div>
        <Link className={buttonStyles({ variant: "secondary" })} href="/templates">
          All templates
        </Link>
      </div>

      {workspaceQuery.isLoading || templatesQuery.isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading workspace templates...
        </div>
      ) : null}

      {workspaceQuery.isError || templatesQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {getErrorMessage(
            workspaceQuery.error ?? templatesQuery.error,
            "Could not load workspace templates."
          )}
        </div>
      ) : null}

      {templatesQuery.isSuccess && templates.length === 0 ? (
        <div className="rounded-lg border border-slate-200 bg-white p-8 text-center">
          <h2 className="text-lg font-semibold text-slate-950">No workspace templates yet</h2>
          <p className="mt-2 text-sm text-slate-600">
            Save a workspace trip as a template to reuse it.
          </p>
        </div>
      ) : null}

      {templates.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {templates.map((template) => (
            <TripTemplateCard
              key={template.id}
              onArchive={archiveTemplate}
              onDuplicate={duplicateTemplate}
              onUse={setSelectedTemplate}
              template={template}
            />
          ))}
        </div>
      ) : null}

      <CreateTripFromTemplateDialog
        onClose={() => setSelectedTemplate(null)}
        open={Boolean(selectedTemplate)}
        template={selectedTemplate}
      />
    </PageContainer>
  );
}
