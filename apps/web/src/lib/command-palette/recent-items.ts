import type { SearchResult } from "@/types/search";

export type RecentCommandPaletteItem = SearchResult & {
  openedAt: string;
};

const MAX_RECENT_ITEMS = 20;
const STORAGE_PREFIX = "command-palette:recent:";

export function getRecentCommandPaletteItems(userId?: string | null): RecentCommandPaletteItem[] {
  if (!userId || typeof window === "undefined") {
    return [];
  }
  try {
    const raw = window.localStorage.getItem(storageKey(userId));
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw) as RecentCommandPaletteItem[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .filter((item) => item && typeof item.id === "string" && typeof item.href === "string")
      .slice(0, MAX_RECENT_ITEMS);
  } catch {
    return [];
  }
}

export function recordRecentCommandPaletteItem(
  userId: string | null | undefined,
  item: SearchResult
) {
  if (!userId || typeof window === "undefined") {
    return;
  }
  const current = getRecentCommandPaletteItems(userId).filter(
    (candidate) => candidate.id !== item.id
  );
  const next: RecentCommandPaletteItem[] = [
    { ...item, openedAt: new Date().toISOString() },
    ...current
  ].slice(0, MAX_RECENT_ITEMS);
  try {
    window.localStorage.setItem(storageKey(userId), JSON.stringify(next));
  } catch {
    // Recent items are best-effort local convenience data.
  }
}

export function clearCommandPaletteRecentItems(userId?: string | null) {
  if (!userId || typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.removeItem(storageKey(userId));
  } catch {
    // Ignore storage failures during logout/session cleanup.
  }
}

function storageKey(userId: string) {
  return `${STORAGE_PREFIX}${userId}`;
}
