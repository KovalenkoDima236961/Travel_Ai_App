"use client";

import { cn } from "@/shared/lib/cn";
import type { SearchResult } from "@/types/search";

type SearchResultItemProps = {
  item: SearchResult;
  active: boolean;
  onSelect: (item: SearchResult, newTab?: boolean) => void;
};

export function SearchResultItem({ item, active, onSelect }: SearchResultItemProps) {
  return (
    <button
      aria-selected={active}
      className={cn(
        "grid w-full grid-cols-[2.25rem_minmax(0,1fr)_auto] items-center gap-3 rounded-md px-3 py-2.5 text-left transition",
        active ? "bg-slate-900 text-white" : "text-slate-800 hover:bg-slate-100"
      )}
      onClick={(event) => onSelect(item, event.metaKey || event.ctrlKey)}
      role="option"
      type="button"
    >
      <span
        aria-hidden="true"
        className={cn(
          "flex h-9 w-9 items-center justify-center rounded-md border text-[11px] font-semibold uppercase",
          active ? "border-white/20 bg-white/10 text-white" : "border-slate-200 bg-white text-slate-500"
        )}
      >
        {iconLabel(item.icon, item.type)}
      </span>
      <span className="min-w-0">
        <span className="block truncate text-sm font-semibold">{item.title}</span>
        <span className={cn("mt-0.5 block truncate text-xs", active ? "text-slate-200" : "text-slate-500")}>
          {[item.description, item.context, item.workspaceName].filter(Boolean).join(" · ")}
        </span>
      </span>
      <span
        className={cn(
          "hidden max-w-[9rem] truncate rounded-full px-2 py-0.5 text-[11px] font-medium sm:inline",
          active ? "bg-white/10 text-slate-100" : "bg-slate-100 text-slate-500"
        )}
      >
        {item.category}
      </span>
    </button>
  );
}

function iconLabel(icon: string, type: string) {
  const value = icon || type;
  return value
    .split(/[-_]/)
    .map((part) => part[0])
    .join("")
    .slice(0, 2);
}
