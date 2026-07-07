"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { PageContainer } from "@/components/layout/PageContainer";
import { AdaptTemplateWithAiDialog } from "@/features/trip-template";
import { CreateTripFromTemplateDialog } from "@/features/trip-template";
import { TemplateItineraryPreview } from "@/features/trip-template";
import { Button, buttonStyles } from "@/shared/ui/button";
import { useTripTemplate } from "@/features/trip-template";
import { useTripTemplateMutations } from "@/features/trip-template";
import { formatBudget, formatDate, getErrorMessage } from "@/lib/utils";
import { Fact } from "./Fact";
import { TemplateMetadataDialog } from "./TemplateMetadataDialog";

export function TemplateDetailPageContent() {
  const params = useParams<{ templateId: string }>();
  const router = useRouter();
  const templateQuery = useTripTemplate(params.templateId);
  const mutations = useTripTemplateMutations();
  const [useDialogOpen, setUseDialogOpen] = useState(false);
  const [adaptDialogOpen, setAdaptDialogOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);

  const template = templateQuery.data ?? null;
  const mutationError = getErrorMessage(
    mutations.archiveTemplate.error ??
      mutations.duplicateTemplate.error ??
      mutations.updateTemplate.error,
    ""
  );

  function archiveTemplate() {
    if (!template || !window.confirm("Archive this template?")) {
      return;
    }
    mutations.archiveTemplate.mutate(
      { templateId: template.id },
      {
        onSuccess: () => {
          router.push("/templates");
        }
      }
    );
  }

  function duplicateTemplate() {
    if (!template) {
      return;
    }
    mutations.duplicateTemplate.mutate(
      {
        templateId: template.id,
        input: { title: `Copy of ${template.title}`, visibility: "private" }
      },
      {
        onSuccess: (created) => {
          router.push(`/templates/${created.id}`);
        }
      }
    );
  }

  return (
    <PageContainer>
      {templateQuery.isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading template...
        </div>
      ) : null}

      {templateQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {getErrorMessage(templateQuery.error, "Could not load template.")}
        </div>
      ) : null}

      {template ? (
        <>
          <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href="/templates">
                Back to templates
              </Link>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <h1 className="text-3xl font-semibold text-slate-950">{template.title}</h1>
                <span className="rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700">
                  {template.visibility}
                </span>
                <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700">
                  {template.status}
                </span>
              </div>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
                {template.description || "Reusable itinerary template."}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              {template.access.canUse ? (
                <Button onClick={() => setUseDialogOpen(true)} type="button" variant="secondary">
                  Use template directly
                </Button>
              ) : null}
              {template.access.canUse ? (
                <Button onClick={() => setAdaptDialogOpen(true)} type="button">
                  Adapt with AI
                </Button>
              ) : null}
              {template.access.canEdit ? (
                <Button onClick={() => setEditOpen(true)} type="button" variant="secondary">
                  Edit metadata
                </Button>
              ) : null}
              {template.access.canDuplicate ? (
                <Button onClick={duplicateTemplate} type="button" variant="secondary">
                  Duplicate
                </Button>
              ) : null}
              {template.access.canArchive ? (
                <Button onClick={archiveTemplate} type="button" variant="ghost">
                  Archive
                </Button>
              ) : null}
            </div>
          </div>

          {mutationError ? (
            <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              {mutationError}
            </div>
          ) : null}

          <div className="mb-8 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Fact label="Destination" value={template.destinationHint || "Flexible"} />
            <Fact
              label="Duration"
              value={`${template.durationDays} ${template.durationDays === 1 ? "day" : "days"}`}
            />
            <Fact
              label="Estimate"
              value={formatBudget(
                template.estimatedTotalAmount,
                template.estimatedTotalCurrency || template.defaultCurrency || "EUR"
              )}
            />
            <Fact label="Updated" value={formatDate(template.updatedAt)} />
          </div>

          {template.tags.length > 0 ? (
            <div className="mb-6 flex flex-wrap gap-2">
              {template.tags.map((tag) => (
                <span
                  className="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-medium text-slate-700"
                  key={tag}
                >
                  {tag}
                </span>
              ))}
            </div>
          ) : null}

          <TemplateItineraryPreview
            currency={template.defaultCurrency || template.estimatedTotalCurrency}
            templateJson={template.templateJson}
          />

          <CreateTripFromTemplateDialog
            onClose={() => setUseDialogOpen(false)}
            open={useDialogOpen}
            template={template}
          />
          <AdaptTemplateWithAiDialog
            onClose={() => setAdaptDialogOpen(false)}
            onUseDirectly={() => {
              setAdaptDialogOpen(false);
              setUseDialogOpen(true);
            }}
            open={adaptDialogOpen}
            template={template}
          />
          <TemplateMetadataDialog
            disabled={mutations.updateTemplate.isPending}
            onClose={() => setEditOpen(false)}
            onSubmit={(input) =>
              mutations.updateTemplate.mutate(
                { templateId: template.id, input },
                { onSuccess: () => setEditOpen(false) }
              )
            }
            open={editOpen}
            template={template}
          />
        </>
      ) : null}
    </PageContainer>
  );
}
