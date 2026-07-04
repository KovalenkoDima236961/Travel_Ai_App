"use client";

import { useMemo, useState } from "react";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { CreateTripFromTemplateDialog } from "@/components/templates/CreateTripFromTemplateDialog";
import { TripTemplateCard } from "@/components/templates/TripTemplateCard";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useTripTemplateMutations, useTripTemplates } from "@/hooks/useTripTemplates";
import { getErrorMessage } from "@/lib/utils";
import type { TripTemplate, TripTemplateVisibility } from "@/types/trip-template";

export default function TemplatesPage() {
  return (
    <ProtectedRoute>
      <TemplatesPageContent />
    </ProtectedRoute>
  );
}

function TemplatesPageContent() {
  const [visibility, setVisibility] = useState<TripTemplateVisibility | "all">("all");
  const [search, setSearch] = useState("");
  const [tag, setTag] = useState("");
  const [selectedTemplate, setSelectedTemplate] = useState<TripTemplate | null>(null);
  const params = useMemo(
    () => ({
      visibility,
      q: search.trim() || undefined,
      tag: tag.trim() || undefined,
      status: "active" as const,
      limit: 50,
      offset: 0
    }),
    [search, tag, visibility]
  );
  const templatesQuery = useTripTemplates(params);
  const mutations = useTripTemplateMutations();
  const templates = templatesQuery.data?.templates ?? [];
  const mutationError = getErrorMessage(
    mutations.archiveTemplate.error ?? mutations.duplicateTemplate.error,
    ""
  );

  function archiveTemplate(template: TripTemplate) {
    if (!window.confirm("Archive this template? It will disappear from active lists.")) {
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
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase text-primary-700">Templates</p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">Template library</h1>
          <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
            Reuse private and workspace itinerary structures without copying live availability or share data.
          </p>
        </div>
      </div>

      <div className="mt-6 grid gap-4 rounded-lg border border-slate-200 bg-white p-4 lg:grid-cols-[auto_minmax(0,1fr)_14rem]">
        <div className="flex flex-wrap gap-2">
          {(["all", "private", "workspace"] as const).map((value) => (
            <Button
              key={value}
              onClick={() => setVisibility(value)}
              size="sm"
              type="button"
              variant={visibility === value ? "primary" : "secondary"}
            >
              {value === "all" ? "All accessible" : value === "private" ? "Private" : "Workspace"}
            </Button>
          ))}
        </div>
        <Input
          aria-label="Search templates"
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Search title or destination"
          value={search}
        />
        <Input
          aria-label="Tag filter"
          onChange={(event) => setTag(event.target.value)}
          placeholder="Tag"
          value={tag}
        />
      </div>

      {templatesQuery.isLoading ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading templates...
        </div>
      ) : null}

      {templatesQuery.isError ? (
        <div className="mt-8 rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {getErrorMessage(templatesQuery.error, "Could not load templates.")}
        </div>
      ) : null}

      {mutationError ? (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          {mutationError}
        </div>
      ) : null}

      {templatesQuery.isSuccess && templates.length === 0 ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-8 text-center">
          <h2 className="text-lg font-semibold text-slate-950">No templates yet</h2>
          <p className="mt-2 text-sm text-slate-600">
            Save a trip as a template to reuse it later.
          </p>
        </div>
      ) : null}

      {templates.length > 0 ? (
        <div className="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
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
