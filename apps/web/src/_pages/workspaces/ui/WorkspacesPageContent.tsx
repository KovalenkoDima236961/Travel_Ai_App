"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";
import { EmptyState } from "@/components/ui";
import { listWorkspaces, workspaceKeys } from "@/lib/api/workspaces";
import { instrumentSans, newsreader } from "./fonts";
import { PlusIcon } from "./icons";
import { WorkspaceCard } from "./WorkspaceCard";
import { WorkspacesHeader } from "./WorkspacesHeader";

const CREATE_CTA =
  "inline-flex h-11 items-center gap-2 rounded-full bg-clay px-5 text-[14.5px] font-semibold text-sand-100 transition hover:bg-clay-dark";

export function WorkspacesPageContent() {
  const emptyT = useTranslations("emptyStates.workspaces");
  const workspacesQuery = useQuery({
    queryKey: workspaceKeys.list(),
    queryFn: listWorkspaces
  });

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <WorkspacesHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the <main> landmark, and nesting a second one is invalid. */}
      <div className="mx-auto max-w-[1280px] px-6 pb-[72px] pt-12 sm:px-10">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <div className="max-w-[640px]">
            <h1 className="font-newsreader text-[38px] font-medium leading-[1.05] tracking-[-0.02em] text-cocoa-900 sm:text-[44px]">
              Workspaces
            </h1>
            <p className="mt-3.5 text-[16px] leading-[1.6] text-cocoa-500">
              Shared planning spaces for groups and teams — with budgets, approvals, and templates.
            </p>
          </div>
          <Link href="/workspaces/new" className={CREATE_CTA}>
            <PlusIcon className="h-4 w-4" />
            Create workspace
          </Link>
        </div>

        {workspacesQuery.isPending ? (
          <div className="mt-8 rounded-[20px] border border-sand-300 bg-white/60 p-7 text-[14.5px] text-cocoa-500">
            Loading workspaces…
          </div>
        ) : null}

        {workspacesQuery.isError ? (
          <div className="mt-8 rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7 text-[14.5px] text-clay-deep">
            {workspacesQuery.error instanceof Error
              ? workspacesQuery.error.message
              : "Could not load workspaces."}
          </div>
        ) : null}

        {workspacesQuery.isSuccess && workspacesQuery.data.length === 0 ? (
          <EmptyState
            className="mt-8 rounded-[20px] border-sand-400 bg-white/60 py-12"
            title={emptyT("title")}
            description={emptyT("description")}
            primaryAction={{ href: "/workspaces/new", label: emptyT("action") }}
          />
        ) : null}

        {workspacesQuery.isSuccess && workspacesQuery.data.length > 0 ? (
          <div className="mt-8 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {workspacesQuery.data.map((workspace) => (
              <WorkspaceCard key={workspace.id} workspace={workspace} />
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}
