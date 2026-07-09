import Link from "next/link";
import {
  ArrowRightIcon,
  BoltIcon,
  CheckCircleIcon,
  SparklesIcon,
  UsersIcon,
  WalletIcon
} from "./icons";

const itinerary = [
  ["08:30", "Vatican Museums"],
  ["13:00", "Lunch in Borgo Pio"],
  ["14:30", "St. Peter's Basilica"]
] as const;

export function LandingHero() {
  return (
    <section className="mx-auto grid max-w-[1240px] items-center gap-12 px-6 pb-20 pt-14 sm:px-10 lg:grid-cols-[minmax(0,1fr)_480px] lg:gap-16 lg:pb-[88px] lg:pt-[72px]">
      <div>
        <p className="text-[12.5px] font-semibold uppercase tracking-[0.14em] text-clay">
          AI-assisted travel planning
        </p>
        <h1 className="mt-5 text-balance font-newsreader text-[44px] font-medium leading-[1.04] tracking-[-0.02em] text-cocoa-900 sm:text-[56px] lg:text-[64px]">
          Plan the trip.
          <br />
          <em className="italic text-clay">Live the story.</em>
        </h1>
        <p className="mt-[26px] max-w-[480px] text-pretty text-lg leading-[1.65] text-cocoa-500">
          Describe where you want to go — the AI drafts a day-by-day itinerary with places, routes,
          weather, and budget. You refine it in one focused workspace.
        </p>
        <div className="mt-9 flex flex-col items-start gap-3.5 sm:flex-row sm:items-center">
          <Link
            href="/trips/new"
            className="inline-flex h-[52px] items-center gap-2.5 rounded-full bg-clay px-7 text-base font-semibold text-sand-100 shadow-[0_10px_24px_rgba(192,91,59,0.28)] transition hover:bg-clay-dark"
          >
            Start planning
            <ArrowRightIcon className="h-[17px] w-[17px]" />
          </Link>
          <Link
            href="/templates"
            className="inline-flex h-[52px] items-center rounded-full border border-sand-400 bg-white/50 px-6 text-base font-medium text-cocoa-900 transition hover:bg-white"
          >
            See a sample itinerary
          </Link>
        </div>
        <ul className="mt-11 flex flex-wrap gap-x-[22px] gap-y-3 text-sm text-cocoa-400">
          <li className="inline-flex items-center gap-2">
            <UsersIcon className="h-[17px] w-[17px] text-clay" />
            Team workspaces
          </li>
          <li className="inline-flex items-center gap-2">
            <WalletIcon className="h-[17px] w-[17px] text-clay" />
            Budgets &amp; cost splitting
          </li>
          <li className="inline-flex items-center gap-2">
            <BoltIcon className="h-[17px] w-[17px] text-clay" />
            Works offline
          </li>
        </ul>
      </div>

      <div className="relative">
        <div className="relative h-[420px] overflow-hidden rounded-[28px] shadow-[0_24px_60px_rgba(34,26,20,0.16)] sm:h-[560px]">
          {/* Hero photo slot — on-brand gradient placeholder; swap for a real destination photo. */}
          <div
            className="h-full w-full"
            style={{
              backgroundImage:
                "radial-gradient(130% 90% at 72% 12%, #F7CDA1 0%, transparent 48%), linear-gradient(165deg, #D98A5A 0%, #B5613C 42%, #5E3722 100%)"
            }}
          />
          <div className="pointer-events-none absolute inset-x-0 bottom-0 h-2/5 bg-gradient-to-t from-cocoa-900/35 to-transparent" />
        </div>

        <div className="absolute bottom-6 left-4 w-[270px] rounded-[18px] border border-sand-300 bg-white p-5 shadow-[0_18px_44px_rgba(34,26,20,0.14)] lg:-left-9 lg:bottom-10">
          <div className="flex items-center justify-between gap-2">
            <p className="font-newsreader text-[19px] font-semibold text-cocoa-900">Rome, day 2</p>
            <span className="inline-flex items-center gap-1.5 rounded-full bg-[#EDF3EA] px-2.5 py-[3px] text-[11.5px] font-semibold text-[#2F7A57]">
              <CheckCircleIcon className="h-3 w-3" />
              On budget
            </span>
          </div>
          <div className="mt-3.5 flex flex-col gap-3">
            {itinerary.map(([time, place]) => (
              <div key={time} className="flex items-center gap-3">
                <span className="w-10 text-xs font-semibold text-cocoa-400">{time}</span>
                <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-clay" />
                <span className="text-[13.5px] font-medium text-cocoa-900">{place}</span>
              </div>
            ))}
          </div>
          <p className="mt-3.5 flex items-center gap-[7px] border-t border-sand-200 pt-3 text-[12.5px] text-cocoa-400">
            <SparklesIcon className="h-3.5 w-3.5 text-clay" />
            Drafted by AI · refined by you
          </p>
        </div>
      </div>
    </section>
  );
}
