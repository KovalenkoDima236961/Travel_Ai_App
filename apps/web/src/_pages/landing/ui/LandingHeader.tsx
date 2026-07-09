import Link from "next/link";
import { GlobeIcon } from "./icons";

export function LandingHeader() {
  return (
    <header className="border-b border-sand-300 bg-sand-50">
      <div className="mx-auto flex max-w-[1240px] items-center justify-between gap-4 px-6 py-[18px] sm:gap-6 sm:px-10">
        <Link href="/" className="flex items-center gap-2.5 text-cocoa-900">
          <span className="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded-full bg-clay text-sand-100">
            <GlobeIcon className="h-[19px] w-[19px]" />
          </span>
          <span className="font-newsreader text-[19px] font-semibold tracking-[-0.01em] sm:text-[21px]">
            Travel AI Planner
          </span>
        </Link>
        <nav className="flex items-center gap-2">
          <Link
            href="/login"
            className="hidden h-10 items-center rounded-full px-4 text-[14.5px] font-medium text-cocoa-700 transition hover:bg-sand-200 hover:text-cocoa-900 sm:inline-flex"
          >
            Log in
          </Link>
          <Link
            href="/register"
            className="inline-flex h-10 items-center rounded-full bg-clay px-5 text-[14.5px] font-semibold text-sand-100 transition hover:bg-clay-dark"
          >
            Get started
          </Link>
        </nav>
      </div>
    </header>
  );
}
