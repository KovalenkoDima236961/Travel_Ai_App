"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { cn } from "@/shared/lib/cn";
import {
  AdaptTemplateWithAiDialog,
  CreateTripFromTemplateDialog,
  useTripTemplate,
  useTripTemplateMutations
} from "@/features/trip-template";
import { formatBudget, formatDate, getErrorMessage } from "@/lib/utils";
import type { TripTemplateVisibility } from "@/entities/trip-template/model";
import { Fact } from "./Fact";
import { instrumentSans, newsreader } from "./fonts";
import { ArrowLeftIcon, SparklesIcon } from "./icons";
import { TemplateDetailHeader } from "./TemplateDetailHeader";
import { TemplateItineraryPreview } from "./TemplateItineraryPreview";
import { TemplateMetadataDialog } from "./TemplateMetadataDialog";

const CONTENT = "mx-auto max-w-[1080px] px-6 pb-[72px] pt-9 sm:px-10";
const PRIMARY_BUTTON =
  "inline-flex h-[42px] items-center gap-2 rounded-full bg-clay px-5 text-[14px] font-semibold text-sand-100 shadow-[0_8px_20px_rgba(192,91,59,0.22)] transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-60";
const OUTLINE_BUTTON =
  "inline-flex h-[42px] items-center rounded-full border border-sand-400 bg-white px-[18px] text-[14px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60";
const GHOST_BUTTON =
  "inline-flex h-[42px] items-center rounded-full px-[18px] text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60";

// Private = warm clay tint, Workspace = muted green (a one-off, not a palette
// token) — mirrors the badge mapping on the redesigned TemplateCard.
const VISIBILITY_PILL: Record<TripTemplateVisibility, string> = {
  private: "bg-clay-tint text-clay-deep",
  workspace: "bg-[#E7EEE9] text-[#3E6B5A]"
};

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
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <TemplateDetailHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the single <main> landmark. */}
      <div className={CONTENT}>
        <Link
          href="/templates"
          className="inline-flex items-center gap-2 text-[14px] font-medium text-clay-deep transition hover:text-clay"
        >
          <ArrowLeftIcon className="h-[15px] w-[15px]" />
          Templates
        </Link>

        {templateQuery.isLoading ? (
          <div className="mt-[18px] rounded-[20px] border border-sand-300 bg-white/60 p-7 text-[14.5px] text-cocoa-500">
            Loading template…
          </div>
        ) : null}

        {templateQuery.isError ? (
          <div className="mt-[18px] rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7 text-[14.5px] text-clay-deep">
            {getErrorMessage(templateQuery.error, "Could not load template.")}
          </div>
        ) : null}

        {template ? (
          <>
            <div className="mt-[18px] flex flex-wrap items-start justify-between gap-6">
              <div className="max-w-[560px]">
                <div className="flex flex-wrap items-center gap-3">
                  <h1 className="font-newsreader text-[40px] font-medium leading-[1.02] tracking-[-0.02em] text-cocoa-900">
                    {template.title}
                  </h1>
                  <span
                    className={cn(
                      "rounded-full px-3 py-[5px] text-[12px] font-semibold capitalize",
                      VISIBILITY_PILL[template.visibility]
                    )}
                  >
                    {template.visibility}
                  </span>
                  {template.status === "archived" ? (
                    <span className="rounded-full bg-sand-200 px-3 py-[5px] text-[12px] font-semibold text-cocoa-500">
                      Archived
                    </span>
                  ) : null}
                </div>
                <p className="mt-3.5 text-[15.5px] leading-[1.6] text-cocoa-500">
                  {template.description || "Reusable itinerary structure."}
                </p>
                {template.tags.length > 0 ? (
                  <div className="mt-4 flex flex-wrap gap-[7px]">
                    {template.tags.map((tag) => (
                      <span
                        key={tag}
                        className="rounded-full border border-sand-300 bg-white px-3 py-[5px] text-[12.5px] font-medium text-cocoa-500"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                ) : null}
              </div>

              <div className="flex flex-wrap items-center gap-2.5">
                {template.access.canUse ? (
                  <button type="button" onClick={() => setUseDialogOpen(true)} className={OUTLINE_BUTTON}>
                    Use directly
                  </button>
                ) : null}
                {template.access.canUse ? (
                  <button
                    type="button"
                    onClick={() => setAdaptDialogOpen(true)}
                    className={PRIMARY_BUTTON}
                  >
                    <SparklesIcon className="h-4 w-4" />
                    Adapt with AI
                  </button>
                ) : null}
                {template.access.canEdit ? (
                  <button type="button" onClick={() => setEditOpen(true)} className={OUTLINE_BUTTON}>
                    Edit metadata
                  </button>
                ) : null}
                {template.access.canDuplicate ? (
                  <button
                    type="button"
                    onClick={duplicateTemplate}
                    disabled={mutations.duplicateTemplate.isPending}
                    className={OUTLINE_BUTTON}
                  >
                    Duplicate
                  </button>
                ) : null}
                {template.access.canArchive ? (
                  <button
                    type="button"
                    onClick={archiveTemplate}
                    disabled={mutations.archiveTemplate.isPending}
                    className={GHOST_BUTTON}
                  >
                    Archive
                  </button>
                ) : null}
              </div>
            </div>

            {mutationError ? (
              <div className="mt-6 rounded-2xl border border-clay/30 bg-clay-tint/50 p-4 text-[14px] text-clay-deep">
                {mutationError}
              </div>
            ) : null}

            <div className="mt-7 grid grid-cols-2 gap-4 md:grid-cols-4">
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

            <h2 className="mt-10 font-newsreader text-[26px] font-semibold text-cocoa-900">
              Itinerary preview
            </h2>
            <TemplateItineraryPreview templateJson={template.templateJson} />

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
      </div>
    </div>
  );
}
