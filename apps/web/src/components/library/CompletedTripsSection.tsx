import type { TripLibraryItem } from "@/types/library";
import { TripLibraryGrid } from "./TripLibraryGrid";
export function CompletedTripsSection({ items, onArchive, onRestore }: { items: TripLibraryItem[]; onArchive: (item: TripLibraryItem) => void; onRestore: (item: TripLibraryItem) => void }) { return <section><h2 className="font-newsreader text-2xl font-semibold text-cocoa-900">Completed trips</h2><div className="mt-4"><TripLibraryGrid items={items} onArchive={onArchive} onRestore={onRestore}/></div></section>; }
