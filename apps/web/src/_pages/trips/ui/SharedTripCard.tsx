import Link from "next/link";
import type { TripStatus } from "@/entities/trip/model";
import type { SharedTripSummary } from "@/entities/collaboration/model";
import { formatDate } from "@/lib/utils";
import { CalendarIcon, UsersIcon } from "./icons";

const STATUS_LABEL: Record<TripStatus, { label: string; dot: string; text: string }> = {
  COMPLETED: { label: "Ready", dot: "#2F7A57", text: "#2F7A57" },
  PROCESSING: { label: "Generating…", dot: "#B57F24", text: "#B57F24" },
  DRAFT: { label: "Draft", dot: "#B09E8A", text: "#6B5D50" },
  FAILED: { label: "Failed", dot: "#C0392B", text: "#C0392B" }
};

export function SharedTripCard({ trip }: { trip: SharedTripSummary }) {
  const status = STATUS_LABEL[trip.status] ?? STATUS_LABEL.DRAFT;

  return (
    <Link
      href={`/trips/${trip.id}`}
      className="block rounded-[20px] border border-sand-300 bg-white p-6 shadow-[0_1px_2px_rgba(34,26,20,0.04),0_12px_32px_rgba(34,26,20,0.06)] transition duration-200 hover:-translate-y-[3px] hover:shadow-[0_2px_4px_rgba(34,26,20,0.05),0_20px_44px_rgba(34,26,20,0.11)]"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate font-newsreader text-[22px] font-semibold tracking-[-0.01em] text-cocoa-900">
            {trip.destination}
          </h3>
          <p className="mt-1 text-[13.5px] text-cocoa-400">
            {trip.updatedAt ? `Updated ${formatDate(trip.updatedAt)}` : "Shared trip"}
          </p>
        </div>
        <span
          className="inline-flex shrink-0 items-center gap-1.5 rounded-full bg-sand-150 px-3 py-1.5 text-xs font-semibold"
          style={{ color: status.text }}
        >
          <span className="h-[7px] w-[7px] rounded-full" style={{ backgroundColor: status.dot }} />
          {status.label}
        </span>
      </div>
      <div className="mt-4 flex flex-wrap gap-x-3.5 gap-y-2 text-[13.5px] text-cocoa-500">
        <span className="inline-flex items-center gap-1.5">
          <CalendarIcon className="h-[15px] w-[15px] text-[#B09E8A]" />
          {trip.days} {trip.days === 1 ? "day" : "days"}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <UsersIcon className="h-[15px] w-[15px] text-[#B09E8A]" />
          {trip.role === "editor" ? "Editor" : "Viewer"}
        </span>
      </div>
    </Link>
  );
}
