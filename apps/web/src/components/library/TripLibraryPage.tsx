"use client";

import { useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { useArchiveTrip, useRestoreTrip, useTripLibrary, useTripLibraryFilters, useTripLibraryInsights } from "@/hooks/useTripLibrary";
import type { TripLibraryFilters, TripLibraryItem, TripLibrarySort } from "@/types/library";
import { ArchivedTripsSection } from "./ArchivedTripsSection";
import { CompletedTripsSection } from "./CompletedTripsSection";
import { LibraryEmptyState } from "./LibraryEmptyState";
import { LibraryErrorState } from "./LibraryErrorState";
import { LibrarySkeleton } from "./LibrarySkeleton";
import { TripArchiveDialog } from "./TripArchiveDialog";
import { TripHistoryFilters } from "./TripHistoryFilters";
import { TripHistoryInsights } from "./TripHistoryInsights";
import { TripHistorySortSelect } from "./TripHistorySortSelect";
import { TripLibraryGrid } from "./TripLibraryGrid";
import { TripLibraryHeader } from "./TripLibraryHeader";
import { TripLifecycleTabs, tabFilters, type LibraryTab } from "./TripLifecycleTabs";
import { TripRestoreDialog } from "./TripRestoreDialog";

export function TripLibraryPage() {
  const searchParams = useSearchParams();
  const { workspaces } = useWorkspaces();
  const [tab, setTab] = useState<LibraryTab>(() => searchParams.get("lifecycle") === "archived" ? "archived" : "all");
  const [baseFilters, setBaseFilters] = useState<TripLibraryFilters>(useTripLibraryFilters({ limit: 30 }));
  const [archiveTarget, setArchiveTarget] = useState<TripLibraryItem | null>(null);
  const [restoreTarget, setRestoreTarget] = useState<TripLibraryItem | null>(null);
  const filters = useMemo(() => ({ ...baseFilters, ...tabFilters(tab) }), [baseFilters, tab]);
  const library = useTripLibrary(filters);
  const insights = useTripLibraryInsights({ year: baseFilters.year });
  const archive = useArchiveTrip();
  const restore = useRestoreTrip();
  const items = library.data?.items ?? [];

  async function confirmArchive(reason?: string) {
    if (!archiveTarget) return;
    await archive.mutateAsync({ tripId: archiveTarget.trip.id, input: { reason } });
    setArchiveTarget(null);
  }
  async function confirmRestore() {
    if (!restoreTarget) return;
    await restore.mutateAsync({ tripId: restoreTarget.trip.id });
    setRestoreTarget(null);
  }

  return <div className="min-h-screen bg-sand-50 text-cocoa-700"><TripLibraryHeader /><div className="mx-auto max-w-[1280px] px-6 pb-16 pt-11 sm:px-10"><div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between"><div><p className="text-sm font-semibold uppercase tracking-[0.14em] text-clay">Your history</p><h1 className="mt-2 font-newsreader text-[42px] font-semibold leading-none text-cocoa-900">Travel library</h1><p className="mt-3 text-[15px] text-cocoa-500">Browse completed, archived, and reusable trips.</p></div><div className="grid grid-cols-2 gap-2 sm:grid-cols-4"><Stat label="Completed" value={library.data?.summary.completed ?? 0}/><Stat label="Archived" value={library.data?.summary.archived ?? 0}/><Stat label="Recaps" value={library.data?.summary.withRecaps ?? 0}/><Stat label="Templates" value={library.data?.summary.withTemplates ?? 0}/></div></div><div className="mt-9"><TripLifecycleTabs active={tab} onChange={setTab}/></div><div className="mt-5 rounded-2xl border border-sand-300 bg-white/70 p-3"><div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between"><TripHistoryFilters filters={baseFilters} years={library.data?.filters.availableYears ?? []} destinations={library.data?.filters.availableDestinations ?? []} workspaces={workspaces} onChange={(next) => setBaseFilters((current) => ({ ...current, ...next }))}/><TripHistorySortSelect value={(baseFilters.sort ?? "recently_updated") as TripLibrarySort} onChange={(sort) => setBaseFilters((current) => ({ ...current, sort }))}/></div></div><div className="mt-6"><TripHistoryInsights insights={insights.data}/></div><div className="mt-7">{library.isPending ? <LibrarySkeleton/> : null}{library.isError ? <LibraryErrorState message={library.error instanceof Error ? library.error.message : undefined}/> : null}{library.isSuccess && !items.length ? <LibraryEmptyState mode={tab === "archived" ? "archived" : tab === "completed" ? "completed" : baseFilters.q || baseFilters.destination || baseFilters.year ? "results" : "default"} /> : null}{library.isSuccess && items.length ? tab === "completed" ? <CompletedTripsSection items={items} onArchive={setArchiveTarget} onRestore={setRestoreTarget}/> : tab === "archived" ? <ArchivedTripsSection items={items} onArchive={setArchiveTarget} onRestore={setRestoreTarget}/> : <TripLibraryGrid items={items} onArchive={setArchiveTarget} onRestore={setRestoreTarget}/> : null}</div></div>{archiveTarget ? <TripArchiveDialog destination={archiveTarget.trip.destination} pending={archive.isPending} onConfirm={confirmArchive} onClose={() => setArchiveTarget(null)}/> : null}{restoreTarget ? <TripRestoreDialog destination={restoreTarget.trip.destination} pending={restore.isPending} onConfirm={confirmRestore} onClose={() => setRestoreTarget(null)}/> : null}</div>;
}

function Stat({ label, value }: { label: string; value: number }) { return <div className="rounded-xl bg-white px-3 py-2.5 text-center ring-1 ring-sand-300"><p className="text-lg font-semibold text-cocoa-900">{value}</p><p className="text-xs text-cocoa-500">{label}</p></div>; }
