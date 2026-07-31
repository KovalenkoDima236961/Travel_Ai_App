"use client";

import dynamic from "next/dynamic";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";

const CommandPaletteController = dynamic(
  () =>
    import("./CommandPaletteController").then((module) => module.CommandPaletteController),
  { ssr: false, loading: () => null }
);

/**
 * Keeps the global shortcut cheap. The command registry, trip lookup, and search
 * renderer are requested only after Cmd/Ctrl+K is pressed.
 */
export function GlobalCommandPalette() {
  const pathname = usePathname();
  const t = useTranslations("commandPalette");
  const { isAuthenticated, isLoading, user } = useAuth();
  const [open, setOpen] = useState(false);
  const publicShareRoute = /^\/share\/[^/]+/.test(pathname ?? "");

  useEffect(() => {
    if (publicShareRoute || isLoading || !isAuthenticated) {
      setOpen(false);
      return;
    }

    function handleShortcut(event: globalThis.KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen(true);
      }
    }

    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [isAuthenticated, isLoading, publicShareRoute]);

  if (publicShareRoute || !isAuthenticated || isLoading) {
    return null;
  }

  return (
    <>
      {!open ? (
        <button
          aria-label={t("title")}
          className="fixed bottom-4 right-4 z-40 flex h-12 w-12 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-900 shadow-lg shadow-slate-950/10 transition hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2 md:hidden"
          onClick={() => setOpen(true)}
          type="button"
        >
          <SearchGlyph />
        </button>
      ) : null}
      {open ? <CommandPaletteController onClose={() => setOpen(false)} user={user} /> : null}
    </>
  );
}

function SearchGlyph() {
  return (
    <svg
      aria-hidden="true"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth="2"
      viewBox="0 0 24 24"
    >
      <circle cx="11" cy="11" r="7" />
      <path d="m16.5 16.5 4 4" />
    </svg>
  );
}
