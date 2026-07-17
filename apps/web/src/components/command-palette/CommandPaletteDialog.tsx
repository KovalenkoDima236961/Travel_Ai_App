"use client";

import { RefObject, useMemo, type KeyboardEvent } from "react";
import { RecentItemsSection } from "./RecentItemsSection";
import { SearchEmptyState } from "./SearchEmptyState";
import { SearchErrorState } from "./SearchErrorState";
import { SearchInput } from "./SearchInput";
import { SearchLoadingState } from "./SearchLoadingState";
import { SearchResultGroup } from "./SearchResultGroup";
import { SuggestedActionsSection } from "./SuggestedActionsSection";
import type { PaletteSection } from "./types";
import type { SearchResult } from "@/types/search";

type CommandPaletteDialogProps = {
  inputRef: RefObject<HTMLInputElement | null>;
  query: string;
  sections: PaletteSection[];
  selectedIndex: number;
  loading: boolean;
  error: boolean;
  labels: {
    title: string;
    placeholder: string;
    inputLabel: string;
    loading: string;
    emptyTitle: string;
    emptyDescription: string;
    errorTitle: string;
    errorDescription: string;
    footer: string;
  };
  onQueryChange: (value: string) => void;
  onClose: () => void;
  onSelect: (item: SearchResult, newTab?: boolean) => void;
  onMoveSelection: (delta: number) => void;
};

export function CommandPaletteDialog({
  inputRef,
  query,
  sections,
  selectedIndex,
  loading,
  error,
  labels,
  onQueryChange,
  onClose,
  onSelect,
  onMoveSelection
}: CommandPaletteDialogProps) {
  const itemCount = useMemo(
    () => sections.reduce((total, section) => total + section.items.length, 0),
    [sections]
  );
  const sectionStarts = useMemo(() => {
    let next = 0;
    return sections.map((section) => {
      const start = next;
      next += section.items.length;
      return start;
    });
  }, [sections]);

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
      return;
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      onMoveSelection(1);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      onMoveSelection(-1);
      return;
    }
    if (event.key === "Tab") {
      trapTabFocus(event);
      return;
    }
    if (event.key === "Enter" && itemCount > 0) {
      event.preventDefault();
      const item = itemAt(sections, selectedIndex);
      if (item) {
        onSelect(item, event.metaKey || event.ctrlKey);
      }
    }
  }

  return (
    <div className="fixed inset-0 z-50" onKeyDown={handleKeyDown}>
      <button
        aria-label={labels.title}
        className="absolute inset-0 h-full w-full bg-slate-950/40"
        onClick={onClose}
        tabIndex={-1}
        type="button"
      />
      <div
        aria-label={labels.title}
        aria-modal="true"
        className="absolute left-1/2 top-[10vh] flex max-h-[min(74vh,720px)] w-[min(760px,calc(100vw-1.5rem))] -translate-x-1/2 flex-col overflow-hidden rounded-lg border border-slate-200 bg-white shadow-2xl"
        role="dialog"
      >
        <SearchInput
          label={labels.inputLabel}
          onChange={onQueryChange}
          placeholder={labels.placeholder}
          ref={inputRef}
          value={query}
        />
        <div className="min-h-0 flex-1 overflow-y-auto py-1" role="listbox">
          {sections.map((section, index) => {
            const props = {
              section,
              startIndex: sectionStarts[index] ?? 0,
              selectedIndex,
              onSelect
            };
            if (section.kind === "recent") {
              return <RecentItemsSection key={section.title} {...props} />;
            }
            if (section.kind === "suggested") {
              return <SuggestedActionsSection key={section.title} {...props} />;
            }
            return <SearchResultGroup key={section.title} {...props} />;
          })}
          {loading ? <SearchLoadingState label={labels.loading} /> : null}
          {error ? (
            <SearchErrorState
              description={labels.errorDescription}
              title={labels.errorTitle}
            />
          ) : null}
          {!loading && !error && itemCount === 0 ? (
            <SearchEmptyState
              description={labels.emptyDescription}
              title={labels.emptyTitle}
            />
          ) : null}
        </div>
        <div className="border-t border-slate-200 px-4 py-2 text-[11px] text-slate-500">
          {labels.footer}
        </div>
      </div>
    </div>
  );
}

function trapTabFocus(event: KeyboardEvent<HTMLDivElement>) {
  const focusable = Array.from(
    event.currentTarget.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled]), [href], [tabindex]:not([tabindex="-1"])'
    )
  ).filter((element) => element.offsetParent !== null);
  if (focusable.length === 0) {
    return;
  }
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  const active = document.activeElement;
  if (event.shiftKey && active === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && active === last) {
    event.preventDefault();
    first.focus();
  }
}

function itemAt(sections: PaletteSection[], selectedIndex: number) {
  let cursor = 0;
  for (const section of sections) {
    if (selectedIndex < cursor + section.items.length) {
      return section.items[selectedIndex - cursor];
    }
    cursor += section.items.length;
  }
  return null;
}
