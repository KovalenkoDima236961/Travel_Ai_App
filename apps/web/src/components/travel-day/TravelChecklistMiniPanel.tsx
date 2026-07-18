"use client";

import { useChecklistMutations } from "@/hooks/useChecklistMutations";
import type { TravelDaySummary } from "@/types/travel-day";

export function TravelChecklistMiniPanel({ tripId, checklist, online }: { tripId: string; checklist: TravelDaySummary["checklist"]; online: boolean }) {
  const mutations = useChecklistMutations(tripId);
  const items = [...checklist.overdue, ...checklist.dueToday].slice(0, 4);
  return <section id="checklist" className="rounded-2xl border border-sand-300 bg-white p-4"><div className="flex items-center justify-between"><h2 className="font-semibold text-cocoa-900">Today’s checklist</h2><span className="text-xs text-cocoa-500">{checklist.progress.completed}/{checklist.progress.total}</span></div>{items.length ? <ul className="mt-3 space-y-2">{items.map((item) => <li className="flex items-center gap-2 text-sm text-cocoa-700" key={item.id}><input aria-label={`Complete ${item.title}`} checked={item.checked} disabled={!online || mutations.setCheckedMutation.isPending} onChange={() => mutations.setCheckedMutation.mutate({ itemId: item.id, checked: !item.checked })} type="checkbox"/><span>{item.title}</span></li>)}</ul> : <p className="mt-2 text-sm text-cocoa-500">Nothing due today.</p>}{!online ? <p className="mt-3 text-xs text-cocoa-500">Checklist changes sync when you’re online.</p> : null}</section>;
}
