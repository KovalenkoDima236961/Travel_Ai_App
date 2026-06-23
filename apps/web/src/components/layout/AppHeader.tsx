"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/auth/AuthProvider";
import { Button, buttonStyles } from "@/components/ui/Button";

export function AppHeader() {
  const router = useRouter();
  const { isAuthenticated, isLoading, logout, user } = useAuth();

  async function handleLogout() {
    await logout();
    router.push("/login");
  }

  return (
    <header className="border-b border-slate-200 bg-white/95">
      <div className="mx-auto flex min-h-16 max-w-6xl flex-wrap items-center justify-between gap-4 px-4 py-3 sm:px-6 lg:px-8">
        <Link className="text-base font-semibold text-slate-950" href="/">
          Travel AI Planner
        </Link>
        <nav className="flex flex-wrap items-center justify-end gap-2">
          {!isLoading && isAuthenticated ? (
            <>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/trips">
                Trips
              </Link>
              <Link className={buttonStyles({ size: "sm" })} href="/trips/new">
                Create Trip
              </Link>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/settings">
                Settings
              </Link>
              <span className="hidden max-w-40 truncate text-sm text-slate-600 lg:inline">
                {user?.email}
              </span>
              <Button size="sm" variant="secondary" onClick={handleLogout}>
                Logout
              </Button>
            </>
          ) : null}
          {!isLoading && !isAuthenticated ? (
            <>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/login">
                Login
              </Link>
              <Link className={buttonStyles({ size: "sm" })} href="/register">
                Register
              </Link>
            </>
          ) : null}
        </nav>
      </div>
    </header>
  );
}
