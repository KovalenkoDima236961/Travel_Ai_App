"use client";

import Link from "next/link";
import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { cn } from "@/shared/lib/cn";
import {
  canManageWorkspace,
  formatWorkspaceRole,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import { listTrips, tripKeys } from "@/lib/api/trips";
import {
  getWorkspace,
  listWorkspaceMembers,
  workspaceKeys
} from "@/lib/api/workspaces";
import { formatDate } from "@/lib/utils";
import type { WorkspaceMember } from "@/entities/workspace/model";
import { instrumentSans, newsreader } from "./fonts";
import {
  ArrowLeftIcon,
  ChartBarIcon,
  CheckCircleIcon,
  PlusIcon,
  SettingsIcon,
  TemplatesIcon,
  TripsNavIcon,
  UsersIcon,
  WalletIcon
} from "./icons";
import { WorkspaceDetailHeader } from "./WorkspaceDetailHeader";
import { WorkspaceTripCard } from "./WorkspaceTripCard";

const CONTENT = "mx-auto max-w-[1280px] px-6 pb-[72px] pt-9 sm:px-10";

// The mock's member avatars cycle four warm brand colors. Real members carry no
// color, so pick one deterministically from a stable id (survives reorders).
const MEMBER_BACKGROUNDS = ["#C05B3B", "#3E6B5A", "#96682A", "#7C93A6"];

function memberBackground(seed: string) {
  let hash = 0;
  for (let index = 0; index < seed.length; index += 1) {
    hash = (hash * 31 + seed.charCodeAt(index)) | 0;
  }
  return MEMBER_BACKGROUNDS[Math.abs(hash) % MEMBER_BACKGROUNDS.length];
}

function initials(value: string) {
  const words = value.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return "?";
  }
  if (words.length === 1) {
    return words[0].slice(0, 2).toUpperCase();
  }
  return (words[0][0] + words[1][0]).toUpperCase();
}

function memberName(member: WorkspaceMember) {
  return member.displayName || member.email || member.userId;
}

const NAV_ITEM = "flex items-center gap-[11px] h-10 rounded-xl px-3.5 text-sm transition";
const NAV_ACTIVE = "bg-sand-200 font-semibold text-cocoa-900";
const NAV_IDLE = "font-medium text-cocoa-500 hover:bg-sand-200 hover:text-cocoa-900";

export function WorkspaceDetailPageContent() {
  const params = useParams<{ workspaceId: string }>();
  const workspaceId = params.workspaceId;
  const { setCurrentWorkspace } = useWorkspaces();

  const workspaceQuery = useQuery({
    queryKey: workspaceKeys.detail(workspaceId),
    queryFn: () => getWorkspace(workspaceId),
    enabled: Boolean(workspaceId)
  });
  const membersQuery = useQuery({
    queryKey: workspaceKeys.members(workspaceId),
    queryFn: () => listWorkspaceMembers(workspaceId),
    enabled: Boolean(workspaceId)
  });
  const tripsQuery = useQuery({
    queryKey: tripKeys.list({ limit: 20, offset: 0, scope: "workspace", workspaceId }),
    queryFn: () => listTrips({ limit: 20, offset: 0, scope: "workspace", workspaceId }),
    enabled: Boolean(workspaceId)
  });

  useEffect(() => {
    if (workspaceQuery.isSuccess) {
      setCurrentWorkspace(workspaceId);
    }
  }, [setCurrentWorkspace, workspaceId, workspaceQuery.isSuccess]);

  const workspace = workspaceQuery.data;
  const canManage = workspace ? canManageWorkspace(workspace.currentUserRole) : false;

  const navItems = [
    { label: "Analytics", href: `/workspaces/${workspaceId}/analytics`, Icon: ChartBarIcon },
    { label: "Budgets", href: `/workspaces/${workspaceId}/budgets`, Icon: WalletIcon },
    { label: "Approvals", href: `/workspaces/${workspaceId}/approvals`, Icon: CheckCircleIcon },
    { label: "Templates", href: `/workspaces/${workspaceId}/templates`, Icon: TemplatesIcon }
  ];

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <WorkspaceDetailHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the <main> landmark, and nesting a second one is invalid. */}
      <div className={CONTENT}>
        <Link
          href="/workspaces"
          className="inline-flex items-center gap-2 text-sm font-medium text-clay-deep transition hover:text-clay"
        >
          <ArrowLeftIcon className="h-[15px] w-[15px]" />
          Workspaces
        </Link>

        {workspaceQuery.isPending ? (
          <div className="mt-6 rounded-[20px] border border-sand-300 bg-white/60 p-7 text-[14.5px] text-cocoa-500">
            Loading workspace…
          </div>
        ) : null}

        {workspaceQuery.isError ? (
          <div className="mt-6 rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7 text-[14.5px] text-clay-deep">
            {workspaceQuery.error instanceof Error
              ? workspaceQuery.error.message
              : "Could not load workspace."}
          </div>
        ) : null}

        {workspace ? (
          <>
            <div className="mt-[18px] flex flex-wrap items-start justify-between gap-6">
              <div className="flex items-center gap-[18px]">
                <span className="flex h-16 w-16 shrink-0 items-center justify-center rounded-[18px] bg-clay font-newsreader text-[26px] font-semibold text-sand-100">
                  {initials(workspace.name)}
                </span>
                <div>
                  <h1 className="font-newsreader text-[40px] font-medium leading-none tracking-[-0.02em] text-cocoa-900">
                    {workspace.name}
                  </h1>
                  <p className="mt-2.5 text-[15px] text-cocoa-500">
                    {workspace.description || "No description yet."}
                  </p>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <span className="rounded-full bg-clay-tint px-3 py-[5px] text-[12.5px] font-semibold text-clay-deep">
                      {formatWorkspaceRole(workspace.currentUserRole)}
                    </span>
                    <span className="rounded-full border border-[#E8DFD3] bg-white px-3 py-[5px] text-[12.5px] font-medium text-cocoa-500">
                      {workspace.memberCount} {workspace.memberCount === 1 ? "member" : "members"}
                    </span>
                    <span className="rounded-full border border-[#E8DFD3] bg-white px-3 py-[5px] text-[12.5px] font-medium text-cocoa-500">
                      Created {formatDate(workspace.createdAt)}
                    </span>
                  </div>
                </div>
              </div>
              {canManage ? (
                <Link
                  href={`/workspaces/${workspace.id}/settings`}
                  className="inline-flex h-[42px] items-center gap-2 rounded-full border border-sand-400 bg-white px-[18px] text-sm font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
                >
                  <SettingsIcon className="h-4 w-4" />
                  Settings
                </Link>
              ) : null}
            </div>

            <div className="mt-7 grid items-start gap-8 lg:grid-cols-[200px_minmax(0,1fr)]">
              <nav className="hidden flex-col gap-0.5 lg:sticky lg:top-[84px] lg:flex">
                <span className={cn(NAV_ITEM, NAV_ACTIVE)}>
                  <TripsNavIcon className="h-[17px] w-[17px] text-clay" />
                  Trips
                </span>
                {navItems.map(({ label, href, Icon }) => (
                  <Link key={label} href={href} className={cn(NAV_ITEM, NAV_IDLE)}>
                    <Icon className="h-[17px] w-[17px] text-[#A08D78]" />
                    {label}
                  </Link>
                ))}
                <a href="#members" className={cn(NAV_ITEM, NAV_IDLE)}>
                  <UsersIcon className="h-[17px] w-[17px] text-[#A08D78]" />
                  Members
                </a>
              </nav>

              <div>
                <div className="flex items-center justify-between gap-4">
                  <h2 className="font-newsreader text-[26px] font-semibold text-cocoa-900">
                    Workspace trips
                  </h2>
                  <Link
                    href="/trips/new"
                    className="inline-flex h-10 items-center gap-2 rounded-full bg-clay px-[18px] text-sm font-semibold text-sand-100 transition hover:bg-clay-dark"
                  >
                    <PlusIcon className="h-[15px] w-[15px]" />
                    New trip
                  </Link>
                </div>

                {tripsQuery.isPending ? (
                  <div className="mt-[18px] rounded-[18px] border border-sand-300 bg-white/60 p-6 text-[14.5px] text-cocoa-500">
                    Loading workspace trips…
                  </div>
                ) : null}

                {tripsQuery.isError ? (
                  <div className="mt-[18px] rounded-[18px] border border-clay/30 bg-clay-tint/50 p-6 text-[14.5px] text-clay-deep">
                    {tripsQuery.error instanceof Error
                      ? tripsQuery.error.message
                      : "Could not load workspace trips."}
                  </div>
                ) : null}

                {tripsQuery.isSuccess && tripsQuery.data.items.length === 0 ? (
                  <div className="mt-[18px] rounded-[18px] border border-dashed border-sand-400 bg-white/60 px-6 py-12 text-center">
                    <p className="text-[14.5px] text-cocoa-400">No trips in this workspace yet.</p>
                    <Link
                      href="/trips/new"
                      className="mt-5 inline-flex h-10 items-center gap-2 rounded-full bg-clay px-[18px] text-sm font-semibold text-sand-100 transition hover:bg-clay-dark"
                    >
                      <PlusIcon className="h-[15px] w-[15px]" />
                      New trip
                    </Link>
                  </div>
                ) : null}

                {tripsQuery.isSuccess && tripsQuery.data.items.length > 0 ? (
                  <div className="mt-[18px] grid gap-5 sm:grid-cols-2">
                    {tripsQuery.data.items.map((trip) => (
                      <WorkspaceTripCard key={trip.id} trip={trip} />
                    ))}
                  </div>
                ) : null}

                <h2
                  id="members"
                  className="mt-10 scroll-mt-[84px] font-newsreader text-[26px] font-semibold text-cocoa-900"
                >
                  Members
                </h2>

                {membersQuery.isPending ? (
                  <div className="mt-[18px] rounded-[18px] border border-sand-300 bg-white/60 p-6 text-[14.5px] text-cocoa-500">
                    Loading members…
                  </div>
                ) : null}

                {membersQuery.isError ? (
                  <div className="mt-[18px] rounded-[18px] border border-clay/30 bg-clay-tint/50 p-6 text-[14.5px] text-clay-deep">
                    {membersQuery.error instanceof Error
                      ? membersQuery.error.message
                      : "Could not load members."}
                  </div>
                ) : null}

                {membersQuery.isSuccess ? (
                  <div className="mt-[18px] overflow-hidden rounded-[18px] border border-sand-300 bg-white">
                    {membersQuery.data.map((member) => {
                      const name = memberName(member);
                      const subline = member.displayName ? member.email : null;
                      return (
                        <div
                          key={member.id}
                          className="flex items-center justify-between gap-4 border-b border-sand-200 px-[22px] py-4 last:border-b-0"
                        >
                          <div className="flex min-w-0 items-center gap-3">
                            <span
                              className="flex h-[38px] w-[38px] shrink-0 items-center justify-center rounded-full text-[12.5px] font-semibold text-sand-100"
                              style={{ background: memberBackground(member.id) }}
                            >
                              {initials(name)}
                            </span>
                            <div className="min-w-0">
                              <p className="truncate text-[14.5px] font-semibold text-cocoa-900">
                                {name}
                              </p>
                              {subline ? (
                                <p className="mt-0.5 truncate text-[12.5px] text-[#A08D78]">
                                  {subline}
                                </p>
                              ) : null}
                            </div>
                          </div>
                          <span
                            className={cn(
                              "shrink-0 rounded-full px-3 py-1 text-xs font-semibold",
                              member.role === "owner"
                                ? "bg-clay-tint text-clay-deep"
                                : "border border-[#E8DFD3] bg-white font-medium text-cocoa-500"
                            )}
                          >
                            {formatWorkspaceRole(member.role)}
                          </span>
                        </div>
                      );
                    })}
                    {membersQuery.data.length === 0 ? (
                      <div className="px-[22px] py-6 text-[14.5px] text-cocoa-400">
                        No members yet.
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </div>
            </div>
          </>
        ) : null}
      </div>
    </div>
  );
}
