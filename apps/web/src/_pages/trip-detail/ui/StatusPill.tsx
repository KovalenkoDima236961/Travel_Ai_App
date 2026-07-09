import type { TripStatus } from "@/entities/trip/model";

// Slice-local status pill so the warm Trip Detail screen never pulls in the
// shared slate `TripStatusBadge` (still rendered on un-redesigned pages). Mapping
// follows the redesign convention: COMPLETED=Ready (green), PROCESSING=Generating
// (amber), DRAFT=Draft (muted), FAILED=Failed (red).
const STATUS_STYLES: Record<
  TripStatus,
  { label: string; className: string; dot: string }
> = {
  COMPLETED: {
    label: "Ready",
    className: "bg-[#EDF3EA] text-[#2F7A57]",
    dot: "bg-[#2F7A57]"
  },
  PROCESSING: {
    label: "Generating…",
    className: "bg-[#FDF0E3] text-[#96682A]",
    dot: "bg-[#D9A441]"
  },
  DRAFT: {
    label: "Draft",
    className: "bg-sand-200 text-cocoa-500",
    dot: "bg-cocoa-400"
  },
  FAILED: {
    label: "Failed",
    className: "bg-[#FBF0EB] text-[#B3402E]",
    dot: "bg-[#B3402E]"
  }
};

export function StatusPill({ status }: { status: TripStatus }) {
  const style = STATUS_STYLES[status] ?? STATUS_STYLES.DRAFT;
  return (
    <span
      className={`inline-flex items-center gap-2 rounded-full px-3.5 py-1.5 text-[13px] font-semibold ${style.className}`}
    >
      <span className={`h-[7px] w-[7px] rounded-full ${style.dot}`} />
      {style.label}
    </span>
  );
}
