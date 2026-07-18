"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/auth/AuthProvider";
import { useTranslations } from "next-intl";

function initialsFromEmail(email: string | undefined) {
  const local = (email ?? "").split("@")[0] ?? "";
  const letters = local.replace(/[^a-zA-Z]/g, "");
  return (letters.slice(0, 2) || "?").toUpperCase();
}

/**
 * Avatar + account menu. The suppressed AppHeader carried logout, so this keeps
 * it reachable (plus a Settings shortcut) behind the avatar the mockup shows.
 */
export function AccountMenu() {
  const translate = useTranslations("navigation");
  const router = useRouter();
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    function handlePointerDown(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  async function handleLogout() {
    setOpen(false);
    await logout();
    router.push("/login");
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label={translate("account")}
        aria-haspopup="menu"
        aria-expanded={open}
        className="inline-flex h-11 w-11 items-center justify-center rounded-full bg-[#3E6B5A] text-[13px] font-semibold text-[#EFF5F1] transition hover:brightness-105 focus:outline-none focus:ring-2 focus:ring-clay/40"
      >
        {initialsFromEmail(user?.email)}
      </button>

      {open ? (
        <div
          role="menu"
          className="absolute right-0 top-11 z-50 w-56 overflow-hidden rounded-2xl border border-sand-300 bg-white shadow-[0_2px_4px_rgba(34,26,20,0.05),0_20px_44px_rgba(34,26,20,0.12)]"
        >
          <div className="border-b border-sand-200 px-4 py-3">
            <p className="text-xs text-cocoa-400">{translate("signedInAs")}</p>
            <p className="mt-0.5 truncate text-[13.5px] font-semibold text-cocoa-900">
              {user?.email ?? translate("yourAccount")}
            </p>
          </div>
          <Link
            href="/settings"
            role="menuitem"
            onClick={() => setOpen(false)}
            className="block px-4 py-2.5 text-[13.5px] font-medium text-cocoa-700 transition hover:bg-sand-150"
          >
            {translate("settings")}
          </Link>
          <Link
            href="/getting-started"
            role="menuitem"
            onClick={() => setOpen(false)}
            className="block px-4 py-2.5 text-[13.5px] font-medium text-cocoa-700 transition hover:bg-sand-150"
          >
            {translate("gettingStarted")}
          </Link>
          <button
            type="button"
            role="menuitem"
            onClick={handleLogout}
            className="block w-full px-4 py-2.5 text-left text-[13.5px] font-medium text-clay-deep transition hover:bg-sand-150"
          >
            {translate("logout")}
          </button>
        </div>
      ) : null}
    </div>
  );
}
