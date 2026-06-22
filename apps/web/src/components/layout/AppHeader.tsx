import Link from "next/link";
import { buttonStyles } from "@/components/ui/Button";

export function AppHeader() {
  return (
    <header className="border-b border-slate-200 bg-white/95">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <Link className="text-base font-semibold text-slate-950" href="/">
          Travel AI Planner
        </Link>
        <nav className="flex items-center gap-2">
          <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/trips">
            Trips
          </Link>
          <Link className={buttonStyles({ size: "sm" })} href="/trips/new">
            Create
          </Link>
        </nav>
      </div>
    </header>
  );
}
