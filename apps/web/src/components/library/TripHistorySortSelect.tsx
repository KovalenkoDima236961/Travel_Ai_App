import type { TripLibrarySort } from "@/types/library";

export function TripHistorySortSelect({ value, onChange }: { value: TripLibrarySort; onChange: (value: TripLibrarySort) => void }) {
  return <label className="flex items-center gap-2 text-sm text-cocoa-600"><span className="sr-only">Sort trips</span><select value={value} onChange={(event) => onChange(event.target.value as TripLibrarySort)} className="rounded-xl border border-sand-300 bg-white px-3 py-2 text-sm text-cocoa-700 outline-none focus:border-clay"><option value="recently_updated">Recently updated</option><option value="trip_date_desc">Trip date: newest</option><option value="trip_date_asc">Trip date: oldest</option><option value="destination">Destination</option><option value="budget_desc">Budget: high to low</option><option value="budget_asc">Budget: low to high</option><option value="completion_rate_desc">Completion rate</option><option value="recap_created_desc">Recap created</option></select></label>;
}
