import { cn } from "@/shared/lib/cn";
import type { TripLifecycle } from "@/types/library";

export type LibraryTab = "all" | "active" | "completed" | "recaps" | "templates" | "archived";

const tabs: Array<{ id: LibraryTab; label: string }> = [
  { id: "all", label: "All" }, { id: "active", label: "Upcoming / Active" }, { id: "completed", label: "Completed" },
  { id: "recaps", label: "Recaps" }, { id: "templates", label: "Templates" }, { id: "archived", label: "Archived" }
];

export function tabFilters(tab: LibraryTab): { lifecycle?: TripLifecycle | "all" | "active,planning,ready,draft"; hasRecap?: boolean; hasTemplate?: boolean; archived?: boolean } {
  if (tab === "active") return { lifecycle: "active,planning,ready,draft" };
  if (tab === "completed") return { lifecycle: "completed" };
  if (tab === "recaps") return { hasRecap: true };
  if (tab === "templates") return { hasTemplate: true };
  if (tab === "archived") return { lifecycle: "archived", archived: true };
  return { lifecycle: "all" };
}

export function TripLifecycleTabs({ active, onChange }: { active: LibraryTab; onChange: (tab: LibraryTab) => void }) {
  return <div className="flex gap-2 overflow-x-auto pb-1" aria-label="Library categories">{tabs.map((tab) => <button key={tab.id} type="button" onClick={() => onChange(tab.id)} className={cn("whitespace-nowrap rounded-full px-3.5 py-2 text-sm font-medium transition", active === tab.id ? "bg-cocoa-900 text-white" : "bg-white text-cocoa-600 ring-1 ring-sand-300 hover:bg-sand-150")}>{tab.label}</button>)}</div>;
}
