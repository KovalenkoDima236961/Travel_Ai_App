"use client";

import { SearchResultItem } from "./SearchResultItem";
import type { SearchResult } from "@/types/search";

type CommandActionItemProps = {
  item: SearchResult;
  active: boolean;
  onSelect: (item: SearchResult, newTab?: boolean) => void;
};

export function CommandActionItem(props: CommandActionItemProps) {
  return <SearchResultItem {...props} />;
}
