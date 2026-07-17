"use client";

import { SearchResultGroup } from "./SearchResultGroup";
import type { PaletteSection } from "./types";
import type { SearchResult } from "@/types/search";

type RecentItemsSectionProps = {
  section: PaletteSection;
  startIndex: number;
  selectedIndex: number;
  onSelect: (item: SearchResult, newTab?: boolean) => void;
};

export function RecentItemsSection(props: RecentItemsSectionProps) {
  return <SearchResultGroup {...props} />;
}
