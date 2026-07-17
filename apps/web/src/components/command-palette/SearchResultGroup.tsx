"use client";

import { CommandActionItem } from "./CommandActionItem";
import { SearchResultItem } from "./SearchResultItem";
import type { PaletteSection } from "./types";
import type { SearchResult } from "@/types/search";

type SearchResultGroupProps = {
  section: PaletteSection;
  startIndex: number;
  selectedIndex: number;
  onSelect: (item: SearchResult, newTab?: boolean) => void;
};

export function SearchResultGroup({
  section,
  startIndex,
  selectedIndex,
  onSelect
}: SearchResultGroupProps) {
  if (section.items.length === 0) {
    return null;
  }

  return (
    <section className="px-2 py-2">
      <h3 className="px-3 pb-1 text-[11px] font-semibold uppercase tracking-normal text-slate-500">
        {section.title}
      </h3>
      <div className="space-y-1">
        {section.items.map((item, index) => {
          const absoluteIndex = startIndex + index;
          const active = selectedIndex === absoluteIndex;
          const Component = item.type === "command" || item.type === "ops_page"
            ? CommandActionItem
            : SearchResultItem;
          return (
            <Component
              active={active}
              item={item}
              key={item.id}
              onSelect={onSelect}
            />
          );
        })}
      </div>
    </section>
  );
}
