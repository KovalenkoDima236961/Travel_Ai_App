import type { SearchResult } from "@/types/search";

export type PaletteSectionKind = "recent" | "suggested" | "commands" | "results";

export type PaletteSection = {
  title: string;
  kind: PaletteSectionKind;
  items: SearchResult[];
};
