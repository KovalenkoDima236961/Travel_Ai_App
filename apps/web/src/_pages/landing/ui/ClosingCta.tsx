import Link from "next/link";
import { ArrowRightIcon } from "./icons";

export function ClosingCta() {
  return (
    <section className="bg-cocoa-900">
      <div className="mx-auto flex max-w-[1240px] flex-wrap items-center justify-between gap-8 px-6 py-[72px] sm:px-10">
        <h2 className="max-w-[560px] text-balance font-newsreader text-[30px] font-medium leading-[1.2] tracking-[-0.015em] text-sand-150 sm:text-[36px]">
          Your next trip is a <em className="italic text-clay-glow">conversation</em> away.
        </h2>
        <Link
          href="/trips/new"
          className="inline-flex h-[52px] items-center gap-2.5 rounded-full bg-clay px-7 text-base font-semibold text-sand-100 transition hover:bg-clay-bright"
        >
          Create your first trip
          <ArrowRightIcon className="h-[17px] w-[17px]" />
        </Link>
      </div>
    </section>
  );
}
