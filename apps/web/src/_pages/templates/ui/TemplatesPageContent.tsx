"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";
import { EmptyState } from "@/components/ui";
import { useTripTemplates } from "@/features/trip-template";
import { getErrorMessage } from "@/lib/utils";
import type { TripTemplateVisibility } from "@/entities/trip-template/model";
import { instrumentSans, newsreader } from "./fonts";
import { SearchIcon, TagIcon } from "./icons";
import { TemplateCard } from "./TemplateCard";
import { TemplatesHeader } from "./TemplatesHeader";
import { RecommendedTemplatesSection } from "@/components/personalization";
import { useRecommendedTemplates } from "@/hooks/usePersonalization";

const FILTERS: { value: TripTemplateVisibility | "all"; label: string }[] = [
  { value: "all", label: "All accessible" },
  { value: "private", label: "Private" },
  { value: "workspace", label: "Workspace" }
];

export function TemplatesPageContent() {
  const emptyT = useTranslations("emptyStates.templates");
  const [visibility, setVisibility] = useState<TripTemplateVisibility | "all">("all");
  const [search, setSearch] = useState("");
  const [tag, setTag] = useState("");
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
  const recommendedQuery = useRecommendedTemplates();
  const templates = templatesQuery.data?.templates ?? [];

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <TemplatesHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the <main> landmark, and nesting a second one is invalid. */}
      <div className="mx-auto max-w-[1280px] px-4 pb-[72px] pt-8 sm:px-10 sm:pt-12">
        <div className="max-w-[640px]">
          <h1 className="font-newsreader text-[38px] font-medium leading-[1.05] tracking-[-0.02em] text-cocoa-900 sm:text-[44px]">
            Template library
          </h1>
          <p className="mt-3.5 text-[16px] leading-[1.6] text-cocoa-500">
            Reuse private and workspace itinerary structures — adapt any of them to a new city with AI.
          </p>
        </div>

        <div className="mt-8 flex flex-wrap items-center justify-between gap-4">
          <div className="flex max-w-full overflow-x-auto rounded-full border border-sand-300 bg-white p-1 [scrollbar-width:thin]">
            {FILTERS.map((filter) => {
              const active = visibility === filter.value;
              return (
                <button
                  key={filter.value}
                  type="button"
                  onClick={() => setVisibility(filter.value)}
                  className={cn(
                    "h-11 shrink-0 rounded-full px-4 text-[13.5px] transition",
                    active
                      ? "bg-cocoa-900 font-semibold text-sand-150"
                      : "font-medium text-cocoa-500 hover:bg-sand-200"
                  )}
                >
                  {filter.label}
                </button>
              );
            })}
          </div>

          <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center">
            <div className="flex h-11 w-full items-center gap-2.5 rounded-full border border-sand-400 bg-white px-4 sm:w-auto">
              <TagIcon className="h-4 w-4 text-[#A08D78]" />
              <input
                aria-label="Filter by tag"
                placeholder="Filter by tag"
                value={tag}
                onChange={(event) => setTag(event.target.value)}
                className="w-[130px] border-none bg-transparent text-[14.5px] text-cocoa-900 outline-none placeholder:text-cocoa-400"
              />
            </div>
            <div className="flex h-11 w-full items-center gap-2.5 rounded-full border border-sand-400 bg-white px-4 sm:min-w-[300px]">
              <SearchIcon className="h-[17px] w-[17px] text-[#A08D78]" />
              <input
                aria-label="Search templates"
                placeholder="Search title or destination"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                className="flex-1 border-none bg-transparent text-[14.5px] text-cocoa-900 outline-none placeholder:text-cocoa-400"
              />
            </div>
          </div>
        </div>

        {recommendedQuery.data?.items?.length ? <div className="mt-8"><RecommendedTemplatesSection items={recommendedQuery.data.items} /></div> : null}

        {templatesQuery.isPending ? (
          <div className="mt-8 rounded-[20px] border border-sand-300 bg-white/60 p-7 text-[14.5px] text-cocoa-500">
            Loading templates…
          </div>
        ) : null}

        {templatesQuery.isError ? (
          <div className="mt-8 rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7 text-[14.5px] text-clay-deep">
            {getErrorMessage(templatesQuery.error, "Could not load templates.")}
          </div>
        ) : null}

        {templatesQuery.isSuccess && templates.length === 0 ? (
          <EmptyState
            className="mt-8 rounded-[20px] border-sand-400 bg-white/60 py-12"
            title={emptyT("title")}
            description={emptyT("description")}
            primaryAction={{ href: "/trips", label: emptyT("action") }}
          />
        ) : null}

        {templates.length > 0 ? (
          <div className="mt-8 grid gap-6 sm:grid-cols-2 xl:grid-cols-3">
            {templates.map((template) => (
              <TemplateCard key={template.id} template={template} />
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}
