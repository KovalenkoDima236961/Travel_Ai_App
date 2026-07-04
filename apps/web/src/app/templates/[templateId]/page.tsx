"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { CreateTripFromTemplateDialog } from "@/components/templates/CreateTripFromTemplateDialog";
import { TemplateItineraryPreview } from "@/components/templates/TemplateItineraryPreview";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useTripTemplate } from "@/hooks/useTripTemplate";
import { useTripTemplateMutations } from "@/hooks/useTripTemplates";
import { formatBudget, formatDate, getErrorMessage } from "@/lib/utils";
import type { TripTemplateDetail } from "@/types/trip-template";

export default function TemplateDetailPage() {
  return (
    <ProtectedRoute>
      <TemplateDetailPageContent />
    </ProtectedRoute>
  );
}

function TemplateDetailPageContent() {
  const params = useParams<{ templateId: string }>();
  const router = useRouter();
  const templateQuery = useTripTemplate(params.templateId);
  const mutations = useTripTemplateMutations();
  const [useDialogOpen, setUseDialogOpen] = useState(false);
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
                <Button onClick={() => setUseDialogOpen(true)} type="button">
                  Use template
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

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <p className="text-xs font-medium text-slate-500">{label}</p>
      <p className="mt-1 break-words text-sm font-semibold text-slate-900">{value}</p>
    </div>
  );
}

function TemplateMetadataDialog({
  open,
  disabled,
  template,
  onClose,
  onSubmit
}: {
  open: boolean;
  disabled: boolean;
  template: TripTemplateDetail;
  onClose: () => void;
  onSubmit: (input: {
    title: string;
    description: string | null;
    destinationHint: string | null;
    defaultCurrency: string | null;
    tags: string[];
  }) => void;
}) {
  const [title, setTitle] = useState(template.title);
  const [description, setDescription] = useState(template.description ?? "");
  const [destinationHint, setDestinationHint] = useState(template.destinationHint ?? "");
  const [defaultCurrency, setDefaultCurrency] = useState(template.defaultCurrency ?? "");
  const [tags, setTags] = useState(template.tags.join(", "));

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <h2 className="text-xl font-semibold text-slate-950">Edit template metadata</h2>
          <Button disabled={disabled} onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>
        <form
          className="mt-6 space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit({
              title,
              description: description.trim() || null,
              destinationHint: destinationHint.trim() || null,
              defaultCurrency: defaultCurrency.trim().toUpperCase() || null,
              tags: tags
                .split(",")
                .map((tag) => tag.trim())
                .filter(Boolean)
            });
          }}
        >
          <label className="block text-sm font-medium text-slate-700">
            Title
            <Input className="mt-2" onChange={(event) => setTitle(event.target.value)} value={title} />
          </label>
          <label className="block text-sm font-medium text-slate-700">
            Description
            <Textarea
              className="mt-2"
              onChange={(event) => setDescription(event.target.value)}
              value={description}
            />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-medium text-slate-700">
              Destination hint
              <Input
                className="mt-2"
                onChange={(event) => setDestinationHint(event.target.value)}
                value={destinationHint}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Default currency
              <Input
                className="mt-2"
                maxLength={3}
                onChange={(event) => setDefaultCurrency(event.target.value)}
                value={defaultCurrency}
              />
            </label>
          </div>
          <label className="block text-sm font-medium text-slate-700">
            Tags
            <Input className="mt-2" onChange={(event) => setTags(event.target.value)} value={tags} />
          </label>
          <div className="flex flex-wrap justify-end gap-2">
            <Button disabled={disabled} onClick={onClose} type="button" variant="secondary">
              Cancel
            </Button>
            <Button disabled={disabled} type="submit">
              {disabled ? "Saving..." : "Save changes"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
