import type { TripLibraryItem } from "@/types/library";
import { TripLibraryCard } from "./TripLibraryCard";

export function TripLibraryGrid({ items, onArchive, onRestore }: { items: TripLibraryItem[]; onArchive: (item: TripLibraryItem) => void; onRestore: (item: TripLibraryItem) => void }) { return <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">{items.map((item) => <TripLibraryCard key={item.trip.id} item={item} onArchive={onArchive} onRestore={onRestore} />)}</div>; }
