import Link from "next/link";
import type { Trip, TripStatus } from "@/entities/trip/model";
import { formatBudget, formatInterestLabel } from "@/lib/utils";
import { BriefcaseIcon, CalendarIcon, UsersIcon, WalletIcon } from "./icons";
import { TripLifecycleBadge } from "@/components/library/TripLifecycleBadge";

type StatusStyle = { label: string; dot: string; text: string };

// One-off semantic status colors (not in the token palette): ready green,
// generating amber, muted draft, failure red.
const STATUS: Record<TripStatus, StatusStyle> = {
  COMPLETED: { label: "Ready", dot: "#2F7A57", text: "#2F7A57" },
  PROCESSING: { label: "Generating…", dot: "#B57F24", text: "#B57F24" },
  DRAFT: { label: "Draft", dot: "#B09E8A", text: "#6B5D50" },
  FAILED: { label: "Failed", dot: "#C0392B", text: "#C0392B" }
};

// On-brand gradient placeholders standing in for the design's image slots — a
// few warm presets keyed off the destination so cards read as distinct.
const GRADIENTS = [
  "radial-gradient(120% 100% at 20% 10%, #F7CDA1 0%, transparent 55%), linear-gradient(150deg, #D98A5A 0%, #B5613C 100%)",
  "radial-gradient(120% 100% at 80% 15%, #F0D9CC 0%, transparent 55%), linear-gradient(150deg, #C4B5A3 0%, #8F6B4E 100%)",
  "radial-gradient(120% 100% at 30% 20%, #CFE3D6 0%, transparent 55%), linear-gradient(150deg, #6E9C82 0%, #3E6B5A 100%)",
  "radial-gradient(120% 100% at 70% 10%, #F7E4DB 0%, transparent 55%), linear-gradient(150deg, #E0885E 0%, #A84A2E 100%)"
];

function gradientFor(seed: string) {
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash * 31 + seed.charCodeAt(i)) % 997;
  }
  return GRADIENTS[hash % GRADIENTS.length];
}

function formatDateRange(startDate: string | null | undefined, days: number) {
  if (!startDate) {
    return "Dates TBD";
  }
  const start = new Date(startDate);
  if (Number.isNaN(start.getTime())) {
    return "Dates TBD";
  }
  const end = new Date(start);
  end.setDate(start.getDate() + Math.max(0, days - 1));

  const dayFmt = new Intl.DateTimeFormat("en", { month: "short", day: "numeric" });
  const startLabel = dayFmt.format(start);
  if (start.getMonth() === end.getMonth() && start.getFullYear() === end.getFullYear()) {
    return `${startLabel} – ${end.getDate()}`;
  }
  return `${startLabel} – ${dayFmt.format(end)}`;
}

type TripCardProps = {
  trip: Trip;
  workspaceName?: string | null;
};

export function TripCard({ trip, workspaceName }: TripCardProps) {
  const status = STATUS[trip.status] ?? STATUS.DRAFT;
  const interests = trip.interests ?? [];
  const budget =
    trip.budgetAmount != null
      ? formatBudget(trip.budgetAmount, trip.budgetCurrency)
      : "No budget yet";

  return (
    <Link
      href={`/trips/${trip.id}`}
      className="group block overflow-hidden rounded-[20px] border border-sand-300 bg-white shadow-[0_1px_2px_rgba(34,26,20,0.04),0_12px_32px_rgba(34,26,20,0.06)] transition duration-200 hover:-translate-y-[3px] hover:shadow-[0_2px_4px_rgba(34,26,20,0.05),0_20px_44px_rgba(34,26,20,0.11)]"
    >
      <div className="relative h-[190px] bg-sand-200" style={{ backgroundImage: gradientFor(trip.destination) }}>
        <span
          className="pointer-events-none absolute left-3.5 top-3.5 inline-flex items-center gap-1.5 rounded-full bg-white/90 px-3 py-1.5 text-xs font-semibold"
          style={{ color: status.text }}
        >
          <span className="h-[7px] w-[7px] rounded-full" style={{ backgroundColor: status.dot }} />
          {status.label}
        </span>
        {trip.lifecycle ? <span className="pointer-events-none absolute bottom-3.5 left-3.5"><TripLifecycleBadge lifecycle={trip.lifecycle} /></span> : null}
        {trip.workspaceId ? (
          <span className="pointer-events-none absolute right-3.5 top-3.5 inline-flex items-center gap-1.5 rounded-full bg-cocoa-900/[0.78] px-3 py-1.5 text-xs font-semibold text-sand-150">
            <BriefcaseIcon className="h-3 w-3" strokeWidth={1.8} />
            {workspaceName ?? "Workspace"}
          </span>
        ) : null}
      </div>
      <div className="px-[22px] pb-[22px] pt-5">
        <div className="flex items-baseline justify-between gap-3">
          <h2 className="truncate font-newsreader text-[26px] font-semibold tracking-[-0.01em] text-cocoa-900">
            {trip.destination}
          </h2>
          <span className="whitespace-nowrap text-[13.5px] font-medium text-cocoa-400">
            {formatDateRange(trip.startDate, trip.days)}
          </span>
        </div>
        <div className="mt-3 flex flex-wrap gap-x-3.5 gap-y-2 text-[13.5px] text-cocoa-500">
          <span className="inline-flex items-center gap-1.5">
            <CalendarIcon className="h-[15px] w-[15px] text-[#B09E8A]" />
            {trip.days} {trip.days === 1 ? "day" : "days"}
          </span>
          <span className="inline-flex items-center gap-1.5">
            <UsersIcon className="h-[15px] w-[15px] text-[#B09E8A]" />
            {trip.travelers} {trip.travelers === 1 ? "traveler" : "travelers"}
          </span>
          <span className="inline-flex items-center gap-1.5">
            <WalletIcon className="h-[15px] w-[15px] text-[#B09E8A]" />
            {budget}
          </span>
        </div>
        <div className="mt-4 flex flex-wrap gap-[7px]">
          {interests.length > 0 ? (
            <>
              {interests.slice(0, 3).map((interest) => (
                <span
                  key={interest}
                  className="rounded-full bg-sand-150 px-3 py-[5px] text-xs font-medium text-cocoa-500"
                >
                  {formatInterestLabel(interest)}
                </span>
              ))}
              {interests.length > 3 ? (
                <span className="rounded-full bg-sand-150 px-3 py-[5px] text-xs font-medium text-cocoa-500">
                  +{interests.length - 3}
                </span>
              ) : null}
            </>
          ) : (
            <span className="text-[13px] text-cocoa-400">No interests selected</span>
          )}
        </div>
      </div>
    </Link>
  );
}
