"use client";

import dynamic from "next/dynamic";
import { usePathname } from "next/navigation";
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

  if (publicShareRoute || !isAuthenticated || isLoading || !open) {
    return null;
  }

  return <CommandPaletteController onClose={() => setOpen(false)} user={user} />;
}
