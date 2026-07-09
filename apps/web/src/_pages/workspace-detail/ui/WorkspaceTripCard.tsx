import Link from "next/link";
import type { Trip, TripStatus } from "@/entities/trip/model";
import { formatBudget } from "@/lib/utils";

// The mock's compact workspace-trip card carries a solid-tint status pill.
// Same semantic mapping as the trips-slice card (Ready/Generating/Draft/Failed),
// rendered here as filled pills to match this screen's flatter card.
const STATUS: Record<TripStatus, { label: string; className: string }> = {
  COMPLETED: { label: "Ready", className: "bg-[#EDF3EA] text-[#2F7A57]" },
  PROCESSING: { label: "Generating", className: "bg-[#FDF0E3] text-[#B57F24]" },
  DRAFT: { label: "Draft", className: "bg-[#F4EDE4] text-[#8A7A6A]" },
  FAILED: { label: "Failed", className: "bg-[#FBEDEA] text-[#C0392B]" }
};

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

export function WorkspaceTripCard({ trip }: { trip: Trip }) {
  const status = STATUS[trip.status] ?? STATUS.DRAFT;

  // Build the mock's "May 12 – 15 · 8 travelers · €9,900" meta line from real
  // fields, dropping the budget segment when the trip carries none.
  const meta = [
    formatDateRange(trip.startDate, trip.days),
    `${trip.travelers} ${trip.travelers === 1 ? "traveler" : "travelers"}`,
    trip.budgetAmount != null ? formatBudget(trip.budgetAmount, trip.budgetCurrency) : null
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <Link
      href={`/trips/${trip.id}`}
      className="block rounded-[18px] border border-sand-300 bg-white p-5 shadow-[0_1px_2px_rgba(34,26,20,0.04)] transition duration-200 hover:-translate-y-[2px] hover:shadow-[0_14px_32px_rgba(34,26,20,0.09)]"
    >
      <div className="flex items-baseline justify-between gap-3">
        <h3 className="truncate font-newsreader text-[21px] font-semibold text-cocoa-900">
          {trip.destination}
        </h3>
        <span
          className={`shrink-0 rounded-full px-[11px] py-1 text-[11.5px] font-semibold ${status.className}`}
        >
          {status.label}
        </span>
      </div>
      <p className="mt-2.5 text-[13.5px] text-cocoa-400">{meta}</p>
    </Link>
  );
}
