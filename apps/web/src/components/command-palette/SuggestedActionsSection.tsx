"use client";

import { SearchResultGroup } from "./SearchResultGroup";
import type { PaletteSection } from "./types";
import type { SearchResult } from "@/types/search";

type SuggestedActionsSectionProps = {
  section: PaletteSection;
  startIndex: number;
  selectedIndex: number;
  onSelect: (item: SearchResult, newTab?: boolean) => void;
};

export function SuggestedActionsSection(props: SuggestedActionsSectionProps) {
  return <SearchResultGroup {...props} />;
}
