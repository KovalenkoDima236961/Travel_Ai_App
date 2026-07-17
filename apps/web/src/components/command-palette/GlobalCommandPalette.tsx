"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { usePathname, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useEffect, useMemo, useRef, useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { CommandPaletteDialog } from "./CommandPaletteDialog";
import type { PaletteSection } from "./types";
import type { Trip } from "@/entities/trip/model";
import { getTrip, tripKeys } from "@/lib/api/trips";
import { searchGlobal, searchKeys } from "@/lib/api/search";
import {
  commandToResult,
  filterCommands,
  getCommandRegistry,
  type CommandContext
} from "@/lib/command-palette/commands";
import { buildCurrentTripLocalResults } from "@/lib/command-palette/local-search";
import {
  getRecentCommandPaletteItems,
  recordRecentCommandPaletteItem
} from "@/lib/command-palette/recent-items";
import type { SearchResult } from "@/types/search";

const MIN_QUERY_LENGTH = 2;
type CommandPaletteT = (key: string) => string;

export function GlobalCommandPalette() {
  const t = useTranslations("commandPalette");
  const router = useRouter();
  const pathname = usePathname();
  const queryClient = useQueryClient();
  const { isAuthenticated, isLoading, user } = useAuth();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [recentItems, setRecentItems] = useState<SearchResult[]>([]);
  const debouncedQuery = useDebouncedValue(query, 180);
  const currentTripId = useMemo(() => extractCurrentTripId(pathname), [pathname]);
  const publicShareRoute = /^\/share\/[^/]+/.test(pathname ?? "");

  const cachedTrip = currentTripId
    ? queryClient.getQueryData<Trip>(tripKeys.detail(currentTripId))
    : undefined;

  const currentTripQuery = useQuery({
    queryKey: currentTripId ? tripKeys.detail(currentTripId) : ["trips", "detail", "none"],
    queryFn: () => getTrip(currentTripId ?? ""),
    enabled: open && Boolean(currentTripId) && !publicShareRoute,
    initialData: cachedTrip,
    staleTime: 30_000
  });

  const currentTrip = currentTripQuery.data;
  const commandContext = useMemo<CommandContext>(
    () => ({
      currentTripId,
      canEditCurrentTrip: Boolean(currentTrip?.access?.canEdit),
      isOpsAdmin: isOpsAdmin(user?.email)
    }),
    [currentTrip?.access?.canEdit, currentTripId, user?.email]
  );
  const commands = useMemo(() => getCommandRegistry(t), [t]);
  const trimmedQuery = query.trim();
  const debouncedTrimmedQuery = debouncedQuery.trim();
  const backendEnabled =
    open &&
    isAuthenticated &&
    !publicShareRoute &&
    debouncedTrimmedQuery.length >= MIN_QUERY_LENGTH;

  const backendSearch = useQuery({
    queryKey: searchKeys.global({
      q: debouncedTrimmedQuery,
      scope: "all",
      tripId: currentTripId,
      limit: 20
    }),
    queryFn: () =>
      searchGlobal({
        q: debouncedTrimmedQuery,
        scope: "all",
        tripId: currentTripId,
        limit: 20
      }),
    enabled: backendEnabled,
    staleTime: 10_000
  });

  useEffect(() => {
    if (publicShareRoute || isLoading || !isAuthenticated) {
      return;
    }
    function handleShortcut(event: globalThis.KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        previousFocusRef.current = document.activeElement instanceof HTMLElement
          ? document.activeElement
          : null;
        setOpen(true);
      }
    }
    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [isAuthenticated, isLoading, publicShareRoute]);

  useEffect(() => {
    if (!open) {
      return;
    }
    setSelectedIndex(0);
    setRecentItems(getRecentCommandPaletteItems(user?.id));
    window.setTimeout(() => inputRef.current?.focus(), 0);
  }, [open, user?.id]);

  const sections = useMemo(() => {
    if (!open) {
      return [];
    }

    const contextCommands = filterCommands(commands, trimmedQuery, commandContext)
      .map((command) => commandToResult(command, commandContext))
      .filter((item): item is SearchResult => Boolean(item));

    if (trimmedQuery.length === 0) {
      return emptyQuerySections(t, recentItems, contextCommands);
    }

    const localResults = buildCurrentTripLocalResults(currentTrip, trimmedQuery);
    const backendGroups = backendSearch.data?.groups ?? [];
    const sections: PaletteSection[] = [];

    if (contextCommands.length > 0) {
      sections.push({
        title: t("sections.quickActions"),
        kind: "commands",
        items: contextCommands.slice(0, 6)
      });
    }
    if (localResults.length > 0) {
      sections.push({
        title: t("sections.currentTrip"),
        kind: "results",
        items: localResults
      });
    }
    for (const group of backendGroups) {
      const items = dedupeResults(group.items, sections);
      if (items.length > 0) {
        sections.push({ title: group.title, kind: "results", items });
      }
    }
    return sections;
  }, [
    backendSearch.data?.groups,
    commandContext,
    commands,
    currentTrip,
    open,
    recentItems,
    t,
    trimmedQuery
  ]);

  const itemCount = useMemo(
    () => sections.reduce((total, section) => total + section.items.length, 0),
    [sections]
  );

  useEffect(() => {
    setSelectedIndex(0);
  }, [trimmedQuery, itemCount]);

  if (publicShareRoute || !isAuthenticated || isLoading) {
    return null;
  }

  function closePalette() {
    setOpen(false);
    setQuery("");
    window.setTimeout(() => previousFocusRef.current?.focus(), 0);
  }

  function selectItem(item: SearchResult, newTab = false) {
    recordRecentCommandPaletteItem(user?.id, item);
    if (newTab) {
      window.open(item.href, "_blank", "noopener,noreferrer");
    } else {
      router.push(item.href);
    }
    closePalette();
  }

  function moveSelection(delta: number) {
    if (itemCount === 0) {
      setSelectedIndex(0);
      return;
    }
    setSelectedIndex((current) => (current + delta + itemCount) % itemCount);
  }

  return open ? (
    <CommandPaletteDialog
      error={backendSearch.isError}
      inputRef={inputRef}
      labels={{
        title: t("title"),
        placeholder: t("placeholder"),
        inputLabel: t("inputLabel"),
        loading: t("loading"),
        emptyTitle:
          trimmedQuery.length < MIN_QUERY_LENGTH ? t("empty.shortTitle") : t("empty.title"),
        emptyDescription:
          trimmedQuery.length < MIN_QUERY_LENGTH
            ? t("empty.shortDescription")
            : t("empty.description"),
        errorTitle: t("error.title"),
        errorDescription: t("error.description"),
        footer: t("footer")
      }}
      loading={backendSearch.isFetching && debouncedTrimmedQuery.length >= MIN_QUERY_LENGTH}
      onClose={closePalette}
      onMoveSelection={moveSelection}
      onQueryChange={setQuery}
      onSelect={selectItem}
      query={query}
      sections={sections}
      selectedIndex={selectedIndex}
    />
  ) : null;
}

function emptyQuerySections(
  t: CommandPaletteT,
  recentItems: SearchResult[],
  commandResults: SearchResult[]
): PaletteSection[] {
  const sections: PaletteSection[] = [];
  if (recentItems.length > 0) {
    sections.push({
      title: t("sections.recent"),
      kind: "recent",
      items: recentItems.slice(0, 6)
    });
  }
  if (commandResults.length > 0) {
    sections.push({
      title: t("sections.suggested"),
      kind: "suggested",
      items: commandResults.slice(0, 8)
    });
  }
  return sections;
}

function dedupeResults(items: SearchResult[], existingSections: PaletteSection[]) {
  const seen = new Set<string>();
  for (const section of existingSections) {
    for (const item of section.items) {
      seen.add(item.href);
      seen.add(item.id);
    }
  }
  return items.filter((item) => {
    if (seen.has(item.href) || seen.has(item.id)) {
      return false;
    }
    seen.add(item.href);
    seen.add(item.id);
    return true;
  });
}

function extractCurrentTripId(pathname: string | null) {
  const match = /^\/trips\/([^/]+)(?:\/.*)?$/.exec(pathname ?? "");
  if (!match || match[1] === "new") {
    return null;
  }
  return match[1];
}

function isOpsAdmin(email?: string | null) {
  const normalizedEmail = email?.trim().toLowerCase();
  if (!normalizedEmail) {
    return false;
  }
  const admins = (process.env.NEXT_PUBLIC_OPS_ADMIN_EMAILS ?? "")
    .split(",")
    .map((item) => item.trim().toLowerCase())
    .filter(Boolean);
  return admins.includes(normalizedEmail);
}

function useDebouncedValue<T>(value: T, delayMs: number) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs);
    return () => window.clearTimeout(timer);
  }, [delayMs, value]);
  return debounced;
}
